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
	SyncMode   string
	SyncPolicy string
	CycleMode  string

	// SyncTask 同步任务
	SyncTask struct {
		// Name 任务名称
		Name string `json:"name"`
		// Id 任务ID
		Id string `json:"id"`
		// UserId 账号ID
		UserId string `json:"userId"`
		// DriveName 网盘名称，backup-备份盘，resource-资源盘
		DriveName string `json:"driveName"`
		// DriveId 网盘ID，目前支持文件网盘
		DriveId string `json:"-"`
		// LocalFolderPath 本地目录
		LocalFolderPath string `json:"localFolderPath"`
		// PanFolderPath 云盘目录
		PanFolderPath string `json:"panFolderPath"`
		// Mode 备份模式
		Mode SyncMode `json:"mode"`
		// Policy 备份策略
		Policy SyncPolicy `json:"policy"`
		// CycleMode 循环模式，OneTime-运行一次，InfiniteLoop-无限循环模式
		CycleModeType CycleMode `json:"cycleModeType"`
		// Priority 优先级选项
		Priority SyncPriorityOption `json:"-"`
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
		panClient *config.PanClient

		syncOption SyncOption

		fileActionTaskManager *FileActionTaskManager
		resourceMutex         *sync.Mutex
		scanLoopIsDone        bool // 本次扫描对比文件进程是否已经完成

		plugin      plugins.Plugin
		pluginMutex *sync.Mutex
	}
)

const (
	// Upload 上传，即备份本地文件到云盘
	Upload SyncMode = "upload"
	// Download 下载，即备份云盘文件到本地
	Download SyncMode = "download"
	// SyncTwoWay 双向同步，本地和云盘文件完全保持一致
	SyncTwoWay SyncMode = "sync"

	// SyncPolicyExclusive 备份策略，排他备份，保证本地和云盘一比一备份，目标目录多余的文件会被删除
	SyncPolicyExclusive SyncPolicy = "exclusive"
	// SyncPolicyIncrement 备份策略，增量备份，只会增量备份文件，目标目录多余的（旧的）文件不会被删除
	SyncPolicyIncrement SyncPolicy = "increment"

	// CycleOneTime 只运行一次
	CycleOneTime CycleMode = "OneTime"
	// CycleInfiniteLoop 无限循环模式
	CycleInfiniteLoop CycleMode = "InfiniteLoop"
)

func (t *SyncTask) NameLabel() string {
	return t.Name + "(" + t.Id + ")"
}

func (t *SyncTask) String() string {
	builder := &strings.Builder{}
	builder.WriteString("任务: " + t.NameLabel() + "\n")
	mode := "双向备份"
	if t.Mode == Upload {
		mode = "备份本地文件（上传）"
	}
	if t.Mode == Download {
		mode = "备份云盘文件（下载）"
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
	policy := ""
	if t.Policy == SyncPolicyExclusive {
		if t.Mode == Upload {
			policy = "排他备份（上传&删除）"
		} else if t.Mode == Download {
			policy = "排他备份（下载&删除）"
		}
	}
	if t.Policy == SyncPolicyIncrement {
		if t.Mode == Upload {
			policy = "增量备份（只上传）"
		} else if t.Mode == Download {
			policy = "增量备份（只下载）"
		}
	}
	builder.WriteString("同步策略: " + policy + "\n")
	builder.WriteString("本地目录: " + t.LocalFolderPath + "\n")
	builder.WriteString("云盘目录: " + t.PanFolderPath + "\n")
	driveName := "备份盘"
	if strings.ToLower(t.DriveName) == "resource" {
		driveName = "资源盘"
	}
	builder.WriteString("目标网盘: " + driveName + "\n")
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
func (t *SyncTask) Start() error {
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
	if _, er := t.panClient.OpenapiPanClient().FileInfoByPath(t.DriveId, t.PanFolderPath); er != nil {
		if er.Code == apierror.ApiCodeFileNotFoundCode {
			t.panClient.OpenapiPanClient().MkdirByFullPath(t.DriveId, t.PanFolderPath)
		}
	}

	// setup sync db file
	t.setupDb()

	if t.fileActionTaskManager == nil {
		t.fileActionTaskManager = NewFileActionTaskManager(t)
	}
	if t.resourceMutex == nil {
		t.resourceMutex = &sync.Mutex{}
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

	// 初始化文件执行进程
	if e := t.fileActionTaskManager.InitMgr(); e != nil {
		return e
	}

	// 策略
	if t.Policy == "" {
		t.Policy = SyncPolicyIncrement
	}

	// 启动文件扫描进程
	t.SetScanLoopFlag(false)
	if t.Mode == Upload {
		go t.scanLocalFile(t.ctx)
	} else if t.Mode == Download {
		go t.scanPanFile(t.ctx)
	} else {
		return fmt.Errorf("异常：暂不支持该模式。")
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

// IsScanLoopDone 获取文件扫描进程状态
func (t *SyncTask) IsScanLoopDone() bool {
	t.resourceMutex.Lock()
	defer t.resourceMutex.Unlock()
	return t.scanLoopIsDone
}

// SetScanLoopFlag 设置文件扫描进程状态标记
func (t *SyncTask) SetScanLoopFlag(done bool) {
	t.resourceMutex.Lock()
	defer t.resourceMutex.Unlock()
	t.scanLoopIsDone = done
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
			if t.Mode == Download {
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

// scanLocalFile 本地文件扫描进程。上传备份模式是以本地文件为扫描对象，并对比云盘端对应目录文件，以决定是否需要上传新文件到云盘
func (t *SyncTask) scanLocalFile(ctx context.Context) {
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

	// 文件夹队列
	folderQueue := collection.NewFifoQueue()
	rootFolder, err := os.Stat(t.LocalFolderPath)
	if err != nil {
		return
	}
	folderQueue.Push(&folderItem{
		fileInfo: rootFolder,
		path:     t.LocalFolderPath,
	})
	delayTimeCount := int64(0)

	for {
		select {
		case <-ctx.Done():
			// cancel routine & done
			logger.Verboseln("local file routine done, exit loop")
			return
		default:
			// 采用广度优先遍历(BFS)进行文件遍历
			if delayTimeCount > 0 {
				time.Sleep(1 * time.Second)
				delayTimeCount -= 1
				continue
			} else if delayTimeCount == 0 {
				// 确认文件执行进程是否已完成
				if !t.fileActionTaskManager.IsExecuteLoopIsDone() {
					time.Sleep(1 * time.Second)
					continue // 需要等待文件上传进程完成才能开启新一轮扫描
				}
				delayTimeCount -= 1
				logger.Verboseln("start scan local file process at ", utils.NowTimeStr())
				t.SetScanLoopFlag(false)
				t.fileActionTaskManager.StartFileActionTaskExecutor()
				PromptPrintln("开始进行文件扫描...")
			}

			obj := folderQueue.Pop()
			if obj == nil {
				// 没有其他文件夹需要扫描了，已完成了一次全量文件夹的扫描了
				t.SetScanLoopFlag(true)

				if t.CycleModeType == CycleOneTime {
					// 只运行一次，全盘扫描一次后退出任务循环
					logger.Verboseln("file scan task is finish, exit normally")
					return
				}

				// 无限循环模式，继续下一次扫描
				folderQueue.Push(&folderItem{
					fileInfo: rootFolder,
					path:     t.LocalFolderPath,
				})
				delayTimeCount = TimeSecondsOfOneMinute
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
			localFileScanList := LocalFileList{}
			localFileAppendList := LocalFileList{}
			for _, file := range files { // 逐个确认目录下面的每个文件的情况
				if strings.HasSuffix(file.Name(), DownloadingFileSuffix) {
					// 下载中的文件，跳过
					continue
				}

				// 检查JS插件
				localFile := newLocalFileItem(file, item.path+"/"+file.Name())
				if t.skipLocalFile(localFile) {
					PromptPrintln("插件禁止扫描本地文件: " + localFile.Path)
					continue
				}

				// 跳过软链接文件
				if IsSymlinkFile(file) {
					logger.Verboseln("软链接文件，跳过：" + item.path + "/" + file.Name())
					continue
				}

				PromptPrintln("扫描到本地文件：" + item.path + "/" + file.Name())
				// 文件夹需要增加到扫描队列
				if file.IsDir() {
					folderQueue.Push(&folderItem{
						fileInfo: file,
						path:     item.path + "/" + file.Name(),
					})
				}

				// 查询本地扫描数据库
				localFileInDb, _ := t.localFileDb.Get(localFile.Path)
				if localFileInDb == nil {
					// 记录不存在，直接增加到本地数据库队列
					localFileAppendList = append(localFileAppendList, localFile)
				} else {
					// 记录存在，查看文件SHA1是否更改
					if localFile.UpdateTimeUnix() == localFileInDb.UpdateTimeUnix() && localFile.FileSize == localFileInDb.FileSize {
						// 文件大小没变，文件修改时间没变，假定文件内容也没变
						localFile.Sha1Hash = localFileInDb.Sha1Hash
					} else {
						// 文件已修改，更新文件信息到扫描数据库
						localFileInDb.Sha1Hash = localFile.Sha1Hash
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
				}
				localFileScanList = append(localFileScanList, localFile)
			}
			if len(localFileAppendList) > 0 {
				//fmt.Println(utils.ObjectToJsonStr(localFileAppendList))
				if _, er := t.localFileDb.AddFileList(localFileAppendList); er != nil {
					logger.Verboseln("add new files to local file db error {}", er)
				}
			}

			// 获取云盘对应目录下的文件清单
			panFileInfo, er := t.panClient.OpenapiPanClient().FileInfoByPath(t.DriveId, GetPanFileFullPathFromLocalPath(item.path, t.LocalFolderPath, t.PanFolderPath))
			if er != nil {
				logger.Verboseln("query pan file info error: ", er)
				// do nothing
				continue
			}
			panFileList, er2 := t.panClient.OpenapiPanClient().FileListGetAll(&aliyunpan.FileListParam{
				DriveId:      t.DriveId,
				ParentFileId: panFileInfo.FileId,
			}, 1500) // 延迟时间避免触发风控
			if er2 != nil {
				logger.Verboseln("query pan file list error: ", er)
				continue
			}
			panFileScanList := PanFileList{}
			for _, pf := range panFileList {
				pf.Path = path.Join(GetPanFileFullPathFromLocalPath(item.path, t.LocalFolderPath, t.PanFolderPath), pf.FileName)
				panFileScanList = append(panFileScanList, NewPanFileItem(pf))
			}

			// 对比文件
			t.fileActionTaskManager.doFileDiffRoutine(localFileScanList, panFileScanList)
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
			if t.Mode == Upload {
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

// scanPanFile 云盘文件循环扫描进程。下载备份模式是以云盘文件为扫描对象，并对比本地对应目录文件，以决定是否需要下载新文件到本地
func (t *SyncTask) scanPanFile(ctx context.Context) {
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
	fi, err := t.panClient.OpenapiPanClient().FileInfoByPath(t.DriveId, fullPath)
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
	delayTimeCount := int64(0)

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
				// 确认文件执行进程是否已完成
				if !t.fileActionTaskManager.IsExecuteLoopIsDone() {
					time.Sleep(1 * time.Second)
					continue // 需要等待文件上传进程完成才能开启新一轮扫描
				}
				delayTimeCount -= 1
				logger.Verboseln("start scan pan file process at ", utils.NowTimeStr())
				t.SetScanLoopFlag(false)
				t.fileActionTaskManager.StartFileActionTaskExecutor()
				PromptPrintln("开始进行文件扫描...")
			}
			obj := folderQueue.Pop()
			if obj == nil {
				// 没有其他文件夹需要扫描了，已完成了一次全量文件夹的扫描了
				t.SetScanLoopFlag(true)

				if t.CycleModeType == CycleOneTime {
					// 只运行一次，全盘扫描一次后退出任务循环
					logger.Verboseln("pan file scan task is finish, exit normally")
					return
				}

				// 无限循环模式，继续下一次扫描
				folderQueue.Push(rootPanFile)
				delayTimeCount = TimeSecondsOfOneMinute
				continue
			}
			item := obj.(*aliyunpan.FileEntity)
			files, err1 := t.panClient.OpenapiPanClient().FileListGetAll(&aliyunpan.FileListParam{
				DriveId:      t.DriveId,
				ParentFileId: item.FileId,
			}, 1500) // 延迟时间避免触发风控
			if err1 != nil {
				// 下一轮重试
				folderQueue.Push(item)
				time.Sleep(10 * time.Second)
				continue
			}
			panFileScanList := PanFileList{}
			for _, file := range files {
				file.Path = path.Join(item.Path, file.FileName)
				panFile := NewPanFileItem(file)

				// 检查JS插件
				if t.skipPanFile(panFile) {
					PromptPrintln("插件禁止扫描云盘文件: " + panFile.Path)
					continue
				}

				PromptPrintln("扫描到云盘文件：" + file.Path)
				panFile.ScanTimeAt = utils.NowTimeStr()
				panFileScanList = append(panFileScanList, panFile)
				logger.Verboseln("scan pan file: ", utils.ObjectToJsonStr(panFile, false))

				if file.IsFolder() {
					folderQueue.Push(file)
				}
			}
			if len(panFileScanList) == 0 {
				// empty dir
				continue
			}

			// 获取本地对应目录下的文件清单
			localFolderPath := GetLocalFileFullPathFromPanPath(item.Path, t.LocalFolderPath, t.PanFolderPath)
			localFiles, err2 := ioutil.ReadDir(localFolderPath)
			if err2 != nil {
				logger.Verboseln("query local file list error: ", err2)
				continue
			}
			localFileScanList := LocalFileList{}
			for _, file := range localFiles { // 逐个确认目录下面的每个文件的情况
				if strings.HasSuffix(file.Name(), DownloadingFileSuffix) {
					// 下载中的文件，跳过
					continue
				}
				localFile := newLocalFileItem(file, localFolderPath+"/"+file.Name())
				logger.Verboseln("扫描到本地文件：" + localFile.Path)

				// 查询本地扫描数据库
				localFileInDb, _ := t.localFileDb.Get(localFile.Path)
				if localFileInDb != nil {
					// 记录存在，查看文件SHA1是否更改
					if localFile.UpdateTimeUnix() == localFileInDb.UpdateTimeUnix() && localFile.FileSize == localFileInDb.FileSize {
						// 文件大小没变，文件修改时间没变，假定文件内容也没变
						localFile.Sha1Hash = localFileInDb.Sha1Hash
					} else {
						// 文件已修改，更新文件信息到扫描数据库
						localFileInDb.Sha1Hash = localFile.Sha1Hash
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
				}
				localFileScanList = append(localFileScanList, localFile)
			}

			// 对比文件
			t.fileActionTaskManager.doFileDiffRoutine(localFileScanList, panFileScanList)
		}
	}
}

// IsExecuteLoopIsDone 判断任务执行是否已经全部完成
func (t *SyncTask) IsExecuteLoopIsDone() bool {
	return t.fileActionTaskManager.IsExecuteLoopIsDone()
}
