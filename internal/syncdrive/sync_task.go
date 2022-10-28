package syncdrive

import (
	"context"
	"fmt"
	"github.com/tickstep/aliyunpan-api/aliyunpan"
	"github.com/tickstep/aliyunpan-api/aliyunpan/apierror"
	"github.com/tickstep/aliyunpan/internal/config"
	"github.com/tickstep/aliyunpan/internal/plugins"
	"github.com/tickstep/aliyunpan/internal/utils"
	"github.com/tickstep/aliyunpan/internal/waitgroup"
	"github.com/tickstep/aliyunpan/library/collection"
	"github.com/tickstep/library-go/logger"
	"io/ioutil"
	"os"
	"path"
	"strings"
	"sync"
	"time"
)

type (
	TaskStep string
	SyncMode string

	// SyncTask 同步任务
	SyncTask struct {
		// Name 任务名称
		Name string `json:"name"`
		// Id 任务ID
		Id string `json:"id"`
		// UserId 账号ID
		UserId string `json:"userId"`
		// DriveId 网盘ID，目前支持文件网盘
		DriveId string `json:"-"`
		// LocalFolderPath 本地目录
		LocalFolderPath string `json:"localFolderPath"`
		// PanFolderPath 云盘目录
		PanFolderPath string `json:"panFolderPath"`
		// Mode 同步模式
		Mode SyncMode `json:"mode"`
		// Priority 优先级选项
		Priority SyncPriorityOption `json:"priority"`
		// LastSyncTime 上一次同步时间
		LastSyncTime string `json:"lastSyncTime"`

		syncDbFolderPath string
		localFileDb      LocalSyncDb
		panFileDb        PanSyncDb
		syncFileDb       SyncFileDb

		wg         *waitgroup.WaitGroup
		ctx        context.Context
		cancelFunc context.CancelFunc

		panUser   *config.PanUser
		panClient *aliyunpan.PanClient

		syncOption SyncOption

		fileActionTaskManager *FileActionTaskManager

		plugin      plugins.Plugin
		pluginMutex *sync.Mutex
	}
)

const (
	// UploadOnly 单向上传，即备份本地文件到云盘
	UploadOnly SyncMode = "upload"
	// DownloadOnly 只下载，即备份云盘文件到本地
	DownloadOnly SyncMode = "download"
	// SyncTwoWay 双向同步，本地和云盘文件完全保持一致
	SyncTwoWay SyncMode = "sync"

	// StepScanFile 任务步骤，扫描文件建立同步数据库
	StepScanFile TaskStep = "scan"
	// StepDiffFile 任务步骤，对比文件
	StepDiffFile TaskStep = "diff"
	// StepSyncFile 任务步骤，同步文件
	StepSyncFile TaskStep = "sync"
)

func (t *SyncTask) NameLabel() string {
	return t.Name + "(" + t.Id + ")"
}

func (t *SyncTask) String() string {
	builder := &strings.Builder{}
	builder.WriteString("任务: " + t.NameLabel() + "\n")
	mode := "双向备份"
	if t.Mode == UploadOnly {
		mode = "备份本地文件（只上传）"
	}
	if t.Mode == DownloadOnly {
		mode = "备份云盘文件（只下载）"
	}
	builder.WriteString("同步模式: " + mode + "\n")
	if t.Mode == SyncTwoWay {
		priority := "时间优先"
		if t.syncOption.SyncPriority == SyncPriorityLocalFirst {
			priority = "本地文件优先"
		} else if t.syncOption.SyncPriority == SyncPriorityPanFirst {
			priority = "网盘文件优先"
		} else {
			priority = "时间优先"
		}
		builder.WriteString("优先选项: " + priority + "\n")
	}
	builder.WriteString("本地目录: " + t.LocalFolderPath + "\n")
	builder.WriteString("云盘目录: " + t.PanFolderPath + "\n")
	return builder.String()
}

func (t *SyncTask) setupDb() error {
	t.localFileDb = NewLocalSyncDb(t.localSyncDbFullPath())
	t.panFileDb = NewPanSyncDb(t.panSyncDbFullPath())
	t.syncFileDb = NewSyncFileDb(t.syncFileDbFullPath())
	if _, e := t.localFileDb.Open(); e != nil {
		return e
	}
	if _, e := t.panFileDb.Open(); e != nil {
		return e
	}
	if _, e := t.syncFileDb.Open(); e != nil {
		return e
	}
	return nil
}

// Start 启动同步任务
// 扫描本地和云盘文件信息并存储到本地数据库
func (t *SyncTask) Start(step TaskStep) error {
	if t.ctx != nil {
		return fmt.Errorf("task have starting")
	}

	// 同步模式下，如果本地磁盘有问题，会导致云盘备份删除文件，需要验证本地磁盘可靠性以避免误删云盘文件
	if t.Mode == SyncTwoWay {
		// check local disk unplug issue
		fi, err := os.Stat(t.localSyncDbFullPath())
		if err == nil && fi.Size() > 0 { // local db existed
			// check local sync dir state
			if b, e := utils.PathExists(t.LocalFolderPath); e == nil {
				if !b {
					// maybe the local disk unplug, check
					return fmt.Errorf("异常：本地同步目录不存在，本任务已经停止。如需继续同步请手动删除同步数据库再重试。")
				}
			}
		}
	}

	// check root dir & init
	if b, e := utils.PathExists(t.LocalFolderPath); e == nil {
		if !b {
			// create local root folder
			os.MkdirAll(t.LocalFolderPath, 0755)
		}
	}
	if _, er := t.panClient.FileInfoByPath(t.DriveId, t.PanFolderPath); er != nil {
		if er.Code == apierror.ApiCodeFileNotFoundCode {
			t.panClient.MkdirByFullPath(t.DriveId, t.PanFolderPath)
		}
	}

	// setup sync db file
	t.setupDb()

	if t.fileActionTaskManager == nil {
		t.fileActionTaskManager = NewFileActionTaskManager(t)
	}

	if t.plugin == nil {
		pluginManger := plugins.NewPluginManager(config.GetPluginDir())
		t.plugin, _ = pluginManger.GetPlugin()
	}
	if t.pluginMutex == nil {
		t.pluginMutex = &sync.Mutex{}
	}

	t.wg = waitgroup.NewWaitGroup(0)

	var cancel context.CancelFunc
	t.ctx, cancel = context.WithCancel(context.Background())
	t.cancelFunc = cancel

	go t.scanLocalFile(t.ctx, step == StepScanFile)
	go t.scanPanFile(t.ctx, step == StepScanFile)

	// 如果只扫描文件建立同步数据库，则无需进行文件对比这一步
	if step != StepScanFile {
		// start file sync manager
		if e := t.fileActionTaskManager.Start(); e != nil {
			return e
		}
	} else {
		// delay
		time.Sleep(1 * time.Second)
	}
	return nil
}

// WaitToStop 等待异步任务执行完成后停止
func (t *SyncTask) WaitToStop() error {
	// wait for finished
	t.wg.Wait()

	t.Stop()
	return nil
}

// Stop 停止同步任务
func (t *SyncTask) Stop() error {
	if t.ctx == nil {
		return nil
	}
	// cancel all sub task & process
	t.cancelFunc()

	// wait for finished
	t.wg.Wait()

	t.ctx = nil
	t.cancelFunc = nil

	// stop file action task (block routine)
	t.fileActionTaskManager.Stop()

	// release resources
	if t.localFileDb != nil {
		t.localFileDb.Close()
	}
	if t.panFileDb != nil {
		t.panFileDb.Close()
	}
	if t.syncFileDb != nil {
		t.syncFileDb.Close()
	}

	// record the sync time
	t.LastSyncTime = utils.NowTimeStr()
	return nil
}

// panSyncDbFullPath 云盘文件数据库
func (t *SyncTask) panSyncDbFullPath() string {
	dir := path.Join(t.syncDbFolderPath, t.Id)
	if b, _ := utils.PathExists(dir); !b {
		os.MkdirAll(dir, 0755)
	}
	return path.Join(dir, "pan.bolt")
}

// localSyncDbFullPath 本地文件数据库
func (t *SyncTask) localSyncDbFullPath() string {
	dir := path.Join(t.syncDbFolderPath, t.Id)
	if b, _ := utils.PathExists(dir); !b {
		os.MkdirAll(dir, 0755)
	}
	return path.Join(dir, "local.bolt")
}

// syncFileDbFullPath 同步文件过程数据库
func (t *SyncTask) syncFileDbFullPath() string {
	dir := path.Join(t.syncDbFolderPath, t.Id)
	if b, _ := utils.PathExists(dir); !b {
		os.MkdirAll(dir, 0755)
	}
	return path.Join(dir, "sync.bolt")
}

func newLocalFileItem(file os.FileInfo, fullPath string) *LocalFileItem {
	ft := "file"
	if file.IsDir() {
		ft = "folder"
	}
	return &LocalFileItem{
		FileName:      file.Name(),
		FileSize:      file.Size(),
		FileType:      ft,
		CreatedAt:     file.ModTime().Format("2006-01-02 15:04:05"),
		UpdatedAt:     file.ModTime().Format("2006-01-02 15:04:05"),
		FileExtension: path.Ext(file.Name()),
		Sha1Hash:      "",
		Path:          fullPath,
		ScanTimeAt:    utils.NowTimeStr(),
		ScanStatus:    ScanStatusNormal,
	}
}

// discardLocalFileDb 清理本地数据库中无效的数据项
func (t *SyncTask) discardLocalFileDb(filePath string, startTimeUnix int64) bool {
	files, e := t.localFileDb.GetFileList(filePath)
	r := false
	if e != nil {
		return r
	}
	for _, file := range files {
		if file.ScanTimeAt == "" || file.ScanTimeUnix() < startTimeUnix {
			if t.Mode == DownloadOnly {
				// delete discard local file info directly
				t.localFileDb.Delete(file.Path)
				logger.Verboseln("label discard local file from DB: ", utils.ObjectToJsonStr(file, false))
			} else {
				// label file discard
				file.ScanStatus = ScanStatusDiscard
				t.localFileDb.Update(file)
				logger.Verboseln("label local file discard: ", utils.ObjectToJsonStr(file, false))
			}
			r = true
		} else {
			if file.IsFolder() {
				b := t.discardLocalFileDb(file.Path, startTimeUnix)
				if b {
					r = b
				}
			}
		}
	}
	return r
}

func (t *SyncTask) skipLocalFile(file *LocalFileItem) bool {
	// 插件回调
	pluginParam := &plugins.SyncScanLocalFilePrepareParams{
		LocalFilePath:      file.Path,
		LocalFileName:      file.FileName,
		LocalFileSize:      file.FileSize,
		LocalFileType:      file.FileType,
		LocalFileUpdatedAt: file.UpdatedAt,
		DriveId:            t.DriveId,
	}
	t.pluginMutex.Lock()
	defer t.pluginMutex.Unlock()
	if result, er := t.plugin.SyncScanLocalFilePrepareCallback(plugins.GetContext(t.panUser), pluginParam); er == nil && result != nil {
		if strings.Compare("no", result.SyncScanLocalApproved) == 0 {
			// skip this file
			return true
		}
	}
	return false
}

// scanLocalFile 本地文件循环扫描进程
func (t *SyncTask) scanLocalFile(ctx context.Context, scanFileOnly bool) {
	t.wg.AddDelta()
	defer t.wg.Done()

	type folderItem struct {
		fileInfo os.FileInfo
		path     string
	}

	// init the root folders info
	pathParts := strings.Split(strings.ReplaceAll(t.LocalFolderPath, "\\", "/"), "/")
	fullPath := ""
	for _, p := range pathParts {
		if p == "" {
			continue
		}
		if strings.Contains(p, ":") {
			// windows volume label, e.g: C:/ D:/
			fullPath += p
			continue
		}
		fullPath += "/" + p
		fi, err := os.Stat(fullPath)
		if err != nil {
			// may be permission deny
			continue
		}
		t.localFileDb.Add(newLocalFileItem(fi, fullPath))
	}

	folderQueue := collection.NewFifoQueue()
	rootFolder, err := os.Stat(t.LocalFolderPath)
	if err != nil {
		return
	}
	folderQueue.Push(&folderItem{
		fileInfo: rootFolder,
		path:     t.LocalFolderPath,
	})
	startTimeOfThisLoop := time.Now().Unix()
	delayTimeCount := int64(0)
	isLocalFolderModify := false

	// 触发一次文件对比任务
	t.fileActionTaskManager.AddLocalFolderModifyCount()

	for {
		select {
		case <-ctx.Done():
			// cancel routine & done
			logger.Verboseln("local file routine done")
			return
		default:
			// 采用广度优先遍历(BFS)进行文件遍历
			if delayTimeCount > 0 {
				time.Sleep(1 * time.Second)
				delayTimeCount -= 1
				continue
			} else if delayTimeCount == 0 {
				delayTimeCount -= 1
				startTimeOfThisLoop = time.Now().Unix()
				logger.Verboseln("do scan local file process at ", utils.NowTimeStr())
			}

			// check local sync dir
			if t.Mode == SyncTwoWay {
				// check local disk unplug issue
				if b, e := utils.PathExists(t.LocalFolderPath); e == nil {
					if !b {
						// maybe the local disk unplug, check
						fmt.Println("异常：本地同步目录不存在，本任务已经停止。如需继续同步请手动删除同步数据库再重试。")
						return
					}
				}
			}

			obj := folderQueue.Pop()
			if obj == nil {
				// label discard file from DB
				if t.discardLocalFileDb(t.LocalFolderPath, startTimeOfThisLoop) {
					logger.Verboseln("notify local folder modify, need to do file action task")
					t.fileActionTaskManager.AddLocalFolderModifyCount()
					t.fileActionTaskManager.AddPanFolderModifyCount()
					isLocalFolderModify = false // 重置标记
				}

				if scanFileOnly {
					// 全盘扫描文件一次，建立好文件同步数据库，然后退出
					return
				}

				// restart scan loop over again
				folderQueue.Push(&folderItem{
					fileInfo: rootFolder,
					path:     t.LocalFolderPath,
				})
				delayTimeCount = TimeSecondsOf30Seconds
				if isLocalFolderModify {
					logger.Verboseln("notify local folder modify, need to do file action task")
					t.fileActionTaskManager.AddLocalFolderModifyCount()
					isLocalFolderModify = false // 重置标记
				}
				continue
			}
			item := obj.(*folderItem)
			files, err1 := ioutil.ReadDir(item.path)
			if err1 != nil {
				continue
			}
			if len(files) == 0 {
				continue
			}
			localFileAppendList := LocalFileList{}
			for _, file := range files {
				if strings.HasSuffix(file.Name(), DownloadingFileSuffix) {
					// 下载中文件，跳过
					continue
				}

				localFile := newLocalFileItem(file, item.path+"/"+file.Name())
				if t.skipLocalFile(localFile) {
					fmt.Println("插件禁止扫描本地文件: ", localFile.Path)
					continue
				}

				if scanFileOnly {
					fmt.Println("扫描本地文件：" + item.path + "/" + file.Name())
				}

				localFileInDb, _ := t.localFileDb.Get(localFile.Path)
				if localFileInDb == nil {
					// append
					localFile.ScanTimeAt = utils.NowTimeStr()
					localFileAppendList = append(localFileAppendList, localFile)
					logger.Verboseln("add local file to db: ", utils.ObjectToJsonStr(localFile, false))
					isLocalFolderModify = true
				} else {
					// update newest info into DB
					if localFile.UpdateTimeUnix() > localFileInDb.UpdateTimeUnix() || localFile.FileSize != localFileInDb.FileSize {
						localFileInDb.Sha1Hash = ""
						isLocalFolderModify = true
					}

					localFileInDb.UpdatedAt = localFile.UpdatedAt
					localFileInDb.CreatedAt = localFile.CreatedAt
					localFileInDb.FileSize = localFile.FileSize
					localFileInDb.FileType = localFile.FileType
					localFileInDb.ScanTimeAt = utils.NowTimeStr()
					localFileInDb.ScanStatus = ScanStatusNormal
					logger.Verboseln("update local file to db: ", utils.ObjectToJsonStr(localFileInDb, false))
					if _, er := t.localFileDb.Update(localFileInDb); er != nil {
						logger.Verboseln("local db update error ", er)
					}
				}

				// for next term scan
				if file.IsDir() {
					folderQueue.Push(&folderItem{
						fileInfo: file,
						path:     item.path + "/" + file.Name(),
					})
				}
			}

			if len(localFileAppendList) > 0 {
				//fmt.Println(utils.ObjectToJsonStr(localFileAppendList))
				if _, er := t.localFileDb.AddFileList(localFileAppendList); er != nil {
					logger.Verboseln("add files to local file db error {}", er)
				}
			}
			time.Sleep(500 * time.Millisecond)
		}
	}
}

// discardPanFileDb 清理云盘数据库中无效的数据项
func (t *SyncTask) discardPanFileDb(filePath string, startTimeUnix int64) bool {
	files, e := t.panFileDb.GetFileList(filePath)
	r := false
	if e != nil {
		return r
	}
	for _, file := range files {
		if file.ScanTimeUnix() < startTimeUnix {
			if t.Mode == UploadOnly {
				// delete discard pan file info directly
				t.panFileDb.Delete(file.Path)
				logger.Verboseln("delete discard pan file from DB: ", utils.ObjectToJsonStr(file, false))
			} else {
				// label file discard
				file.ScanStatus = ScanStatusDiscard
				t.panFileDb.Update(file)
				logger.Verboseln("label pan file discard: ", utils.ObjectToJsonStr(file, false))
			}
			r = true
		} else {
			if file.IsFolder() {
				b := t.discardPanFileDb(file.Path, startTimeUnix)
				if b {
					r = b
				}
			}
		}
	}
	return r
}

func (t *SyncTask) skipPanFile(file *PanFileItem) bool {
	// 插件回调
	pluginParam := &plugins.SyncScanPanFilePrepareParams{
		DriveId:            file.DriveId,
		DriveFileName:      file.FileName,
		DriveFilePath:      file.Path,
		DriveFileSha1:      file.Sha1Hash,
		DriveFileSize:      file.FileSize,
		DriveFileType:      file.FileType,
		DriveFileUpdatedAt: file.UpdatedAt,
	}
	t.pluginMutex.Lock()
	defer t.pluginMutex.Unlock()
	if result, er := t.plugin.SyncScanPanFilePrepareCallback(plugins.GetContext(t.panUser), pluginParam); er == nil && result != nil {
		if strings.Compare("no", result.SyncScanPanApproved) == 0 {
			// skip this file
			return true
		}
	}
	return false
}

// scanPanFile 云盘文件循环扫描进程
func (t *SyncTask) scanPanFile(ctx context.Context, scanFileOnly bool) {
	t.wg.AddDelta()
	defer t.wg.Done()

	// init the root folders info
	pathParts := strings.Split(strings.ReplaceAll(t.PanFolderPath, "\\", "/"), "/")
	fullPath := ""
	for _, p := range pathParts {
		if p == "" {
			continue
		}
		fullPath += "/" + p
	}
	fi, err := t.panClient.FileInfoByPath(t.DriveId, fullPath)
	if err != nil {
		return
	}
	pFile := NewPanFileItem(fi)
	pFile.ScanTimeAt = utils.NowTimeStr()
	t.panFileDb.Add(pFile)
	time.Sleep(200 * time.Millisecond)

	folderQueue := collection.NewFifoQueue()
	rootPanFile := fi
	folderQueue.Push(rootPanFile)
	startTimeOfThisLoop := time.Now().Unix()
	delayTimeCount := int64(0)
	isPanFolderModify := false

	// 触发一次文件对比任务
	t.fileActionTaskManager.AddPanFolderModifyCount()

	for {
		select {
		case <-ctx.Done():
			// cancel routine & done
			logger.Verboseln("pan file routine done")
			return
		default:
			// 采用广度优先遍历(BFS)进行文件遍历
			if delayTimeCount > 0 {
				time.Sleep(1 * time.Second)
				delayTimeCount -= 1
				continue
			} else if delayTimeCount == 0 {
				delayTimeCount -= 1
				startTimeOfThisLoop = time.Now().Unix()
				logger.Verboseln("do scan pan file process at ", utils.NowTimeStr())
			}
			obj := folderQueue.Pop()
			if obj == nil {
				// label discard file from DB
				if t.discardPanFileDb(t.PanFolderPath, startTimeOfThisLoop) {
					logger.Verboseln("notify pan folder modify, need to do file action task")
					t.fileActionTaskManager.AddPanFolderModifyCount()
					t.fileActionTaskManager.AddLocalFolderModifyCount()
					isPanFolderModify = false
				}

				if scanFileOnly {
					// 全盘扫描文件一次，建立好文件同步数据库，然后退出
					return
				}

				// restart scan loop over again
				folderQueue.Push(rootPanFile)
				delayTimeCount = TimeSecondsOf2Minute
				if isPanFolderModify {
					logger.Verboseln("notify pan folder modify, need to do file action task")
					t.fileActionTaskManager.AddPanFolderModifyCount()
					isPanFolderModify = false
				}
				continue
			}
			item := obj.(*aliyunpan.FileEntity)
			files, err1 := t.panClient.FileListGetAll(&aliyunpan.FileListParam{
				DriveId:      t.DriveId,
				ParentFileId: item.FileId,
			}, 1500) // 延迟时间避免触发风控
			if err1 != nil {
				// retry next term
				folderQueue.Push(item)
				time.Sleep(10 * time.Second)
				continue
			}
			panFileList := PanFileList{}
			for _, file := range files {
				file.Path = path.Join(item.Path, file.FileName)
				panFile := NewPanFileItem(file)
				if t.skipPanFile(panFile) {
					logger.Verboseln("插件禁止扫描云盘文件: ", panFile.Path)
					continue
				}
				if scanFileOnly {
					fmt.Println("扫描云盘文件：" + file.Path)
				}

				panFileInDb, _ := t.panFileDb.Get(file.Path)
				if panFileInDb == nil {
					// append
					panFile.ScanTimeAt = utils.NowTimeStr()
					panFileList = append(panFileList, panFile)
					logger.Verboseln("add pan file to db: ", utils.ObjectToJsonStr(panFile, false))
					isPanFolderModify = true
				} else {
					// update newest info into DB
					if strings.ToLower(file.ContentHash) != strings.ToLower(panFileInDb.Sha1Hash) {
						isPanFolderModify = true

						panFileInDb.DomainId = file.DomainId
						panFileInDb.FileId = file.FileId
						panFileInDb.FileType = file.FileType
						panFileInDb.Category = file.Category
						panFileInDb.Crc64Hash = file.Crc64Hash
						panFileInDb.Sha1Hash = file.ContentHash
						panFileInDb.FileSize = file.FileSize
						panFileInDb.UpdatedAt = file.UpdatedAt
						panFileInDb.CreatedAt = file.CreatedAt
					}
					// update scan time
					panFileInDb.ScanTimeAt = utils.NowTimeStr()
					panFileInDb.ScanStatus = ScanStatusNormal
					logger.Verboseln("update pan file to db: ", utils.ObjectToJsonStr(panFileInDb, false))
					t.panFileDb.Update(panFileInDb)
				}

				if file.IsFolder() {
					folderQueue.Push(file)
				}
			}
			if len(panFileList) > 0 {
				if _, er := t.panFileDb.AddFileList(panFileList); er != nil {
					logger.Verboseln("add files to pan file db error {}", er)
				}
			}
			time.Sleep(5 * time.Second) // 延迟避免触发风控
		}
	}
}
