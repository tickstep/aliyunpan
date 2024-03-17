package syncdrive

import (
	"context"
	"fmt"
	mapset "github.com/deckarep/golang-set"
	"github.com/tickstep/aliyunpan/internal/config"
	"github.com/tickstep/aliyunpan/internal/plugins"
	"github.com/tickstep/aliyunpan/internal/utils"
	"github.com/tickstep/aliyunpan/internal/waitgroup"
	"github.com/tickstep/aliyunpan/library/collection"
	"github.com/tickstep/library-go/logger"
	"os"
	"path"
	"strings"
	"sync"
	"time"
)

type (
	FileActionTaskList []*FileActionTask

	FileActionTaskManager struct {
		mutex            *sync.Mutex
		localCreateMutex *sync.Mutex
		panCreateMutex   *sync.Mutex

		task       *SyncTask
		wg         *waitgroup.WaitGroup
		ctx        context.Context
		cancelFunc context.CancelFunc

		fileInProcessQueue *collection.Queue
		syncOption         SyncOption

		resourceModifyMutex *sync.Mutex
		executeLoopIsDone   bool // 文件执行进程是否已经完成

		panUser *config.PanUser

		// 插件
		plugin      plugins.Plugin
		pluginMutex *sync.Mutex
	}

	localFileSet struct {
		items           LocalFileList
		localFolderPath string
	}
	panFileSet struct {
		items         PanFileList
		panFolderPath string
	}
)

func NewFileActionTaskManager(task *SyncTask) *FileActionTaskManager {
	return &FileActionTaskManager{
		mutex:            &sync.Mutex{},
		localCreateMutex: &sync.Mutex{},
		panCreateMutex:   &sync.Mutex{},
		task:             task,

		fileInProcessQueue: collection.NewFifoQueue(),
		syncOption:         task.syncOption,

		resourceModifyMutex: &sync.Mutex{},
		executeLoopIsDone:   true,

		panUser: task.panUser,
	}
}

// IsExecuteLoopIsDone 获取文件执行进程状态
func (f *FileActionTaskManager) IsExecuteLoopIsDone() bool {
	f.resourceModifyMutex.Lock()
	defer f.resourceModifyMutex.Unlock()
	return f.executeLoopIsDone
}

// SetExecuteLoopFlag 设置文件执行进程状态标记
func (f *FileActionTaskManager) setExecuteLoopFlag(done bool) {
	f.resourceModifyMutex.Lock()
	defer f.resourceModifyMutex.Unlock()
	f.executeLoopIsDone = done
}

// InitMgr 初始化文件动作任务管理进程
func (f *FileActionTaskManager) InitMgr() error {
	if f.ctx != nil {
		return fmt.Errorf("task have starting")
	}
	f.wg = waitgroup.NewWaitGroup(0)

	var cancel context.CancelFunc
	f.ctx, cancel = context.WithCancel(context.Background())
	f.cancelFunc = cancel

	if f.plugin == nil {
		pluginManger := plugins.NewPluginManager(config.GetPluginDir())
		f.plugin, _ = pluginManger.GetPlugin()
	}
	if f.pluginMutex == nil {
		f.pluginMutex = &sync.Mutex{}
	}

	return nil
}

func (f *FileActionTaskManager) Stop() error {
	if f.ctx == nil {
		return nil
	}
	// cancel all sub task & process
	f.cancelFunc()

	// wait for finished
	f.wg.Wait()

	f.ctx = nil
	f.cancelFunc = nil

	return nil
}

func (f *FileActionTaskManager) StartFileActionTaskExecutor() error {
	logger.Verboseln("start file execute task at ", utils.NowTimeStr())
	f.setExecuteLoopFlag(false)
	go f.fileActionTaskExecutor(f.ctx)
	return nil
}

// getPanPathFromLocalPath 通过本地文件路径获取网盘文件的对应路径
func (f *FileActionTaskManager) getPanPathFromLocalPath(localPath string) string {
	localPath = strings.ReplaceAll(localPath, "\\", "/")
	localRootPath := path.Clean(strings.ReplaceAll(f.task.LocalFolderPath, "\\", "/"))

	relativePath := strings.TrimPrefix(localPath, localRootPath)
	panPath := path.Join(path.Clean(f.task.PanFolderPath), relativePath)
	return strings.ReplaceAll(panPath, "\\", "/")
}

// getLocalPathFromPanPath 通过网盘文件路径获取对应的本地文件的对应路径
func (f *FileActionTaskManager) getLocalPathFromPanPath(panPath string) string {
	panPath = strings.ReplaceAll(panPath, "\\", "/")
	panRootPath := path.Clean(strings.ReplaceAll(f.task.PanFolderPath, "\\", "/"))

	relativePath := strings.TrimPrefix(panPath, panRootPath)
	return path.Join(path.Clean(f.task.LocalFolderPath), relativePath)
}

// doFileDiffRoutine 对比本地-云盘文件目录，决定哪些文件需要上传，哪些需要下载
func (f *FileActionTaskManager) doFileDiffRoutine(localFiles LocalFileList, panFiles PanFileList) {
	// empty loop
	if len(panFiles) == 0 && len(localFiles) == 0 {
		time.Sleep(100 * time.Millisecond)
		return
	}

	localFilesSet := &localFileSet{
		items:           localFiles,
		localFolderPath: f.task.LocalFolderPath,
	}
	panFilesSet := &panFileSet{
		items:         panFiles,
		panFolderPath: f.task.PanFolderPath,
	}
	localFilesNeedToUpload := localFilesSet.Difference(panFilesSet)                       // 差集
	panFilesNeedToDownload := panFilesSet.Difference(localFilesSet)                       // 补集
	localFilesNeedToCheck, panFilesNeedToCheck := localFilesSet.Intersection(panFilesSet) // 交集

	// download file from pan drive
	if panFilesNeedToDownload != nil {
		for _, file := range panFilesNeedToDownload {
			if f.task.Mode == DownloadOnly {
				syncItem := &SyncFileItem{
					Action:            SyncFileActionDownload,
					Status:            SyncFileStatusCreate,
					LocalFile:         nil,
					PanFile:           file,
					StatusUpdateTime:  "",
					PanFolderPath:     f.task.PanFolderPath,
					LocalFolderPath:   f.task.LocalFolderPath,
					DriveId:           f.task.DriveId,
					DownloadBlockSize: f.syncOption.FileDownloadBlockSize,
					UploadBlockSize:   f.syncOption.FileUploadBlockSize,
				}

				if file.IsFolder() {
					// 创建本地文件夹，这样就可以同步空文件夹
					f.createLocalFolder(file)
				} else {
					// 文件，进入下载队列
					fileActionTask := &FileActionTask{
						syncItem: syncItem,
					}
					f.addToSyncDb(fileActionTask)
				}
			}
		}
	}

	// upload file to pan drive
	if localFilesNeedToUpload != nil {
		for _, file := range localFilesNeedToUpload {
			if f.task.Mode == UploadOnly {
				// check local file modified or not
				if file.IsFile() {
					if f.syncOption.LocalFileModifiedCheckIntervalSec > 0 {
						time.Sleep(time.Duration(f.syncOption.LocalFileModifiedCheckIntervalSec) * time.Second)
					}
					if fi, fe := os.Stat(file.Path); fe == nil {
						if fi.ModTime().Unix() > file.UpdateTimeUnix() {
							logger.Verboseln("本地文件已被修改，等下一轮扫描最新的再上传: ", file.Path)
							continue
						}
					}
				}

				syncItem := &SyncFileItem{
					Action:            SyncFileActionUpload,
					Status:            SyncFileStatusCreate,
					LocalFile:         file,
					PanFile:           nil,
					StatusUpdateTime:  "",
					PanFolderPath:     f.task.PanFolderPath,
					LocalFolderPath:   f.task.LocalFolderPath,
					DriveId:           f.task.DriveId,
					DownloadBlockSize: f.syncOption.FileDownloadBlockSize,
					UploadBlockSize:   f.syncOption.FileUploadBlockSize,
				}
				if file.IsFolder() {
					// 创建云盘文件夹，这样就可以同步空文件夹
					f.createPanFolder(file)
				} else {
					// 文件，增加到上传队列
					fileActionTask := &FileActionTask{
						syncItem: syncItem,
					}
					f.addToSyncDb(fileActionTask)
				}
			}
		}
	}

	// 文件共同交集部分，需要处理文件是否有修改，需要重新上传、下载
	for idx, _ := range localFilesNeedToCheck {
		localFile := localFilesNeedToCheck[idx]
		panFile := panFilesNeedToCheck[idx]

		// 跳过文件夹
		if localFile.IsFolder() {
			continue
		}

		// 本地文件和云盘文件SHA1不一样
		// 不同模式同步策略不一样
		if f.task.Mode == UploadOnly {

			// 不再这里计算SHA1，待到上传的时候再计算
			//if localFile.Sha1Hash == "" {
			//	// 计算本地文件SHA1
			//	if localFile.FileSize == 0 {
			//		localFile.Sha1Hash = aliyunpan.DefaultZeroSizeFileContentHash
			//	} else {
			//		fileSum := localfile.NewLocalFileEntity(localFile.Path)
			//		err := fileSum.OpenPath()
			//		if err != nil {
			//			logger.Verbosef("文件不可读, 错误信息: %s, 跳过...\n", err)
			//			continue
			//		}
			//		fileSum.Sum(localfile.CHECKSUM_SHA1) // block operation
			//		localFile.Sha1Hash = fileSum.SHA1
			//		fileSum.Close()
			//	}
			//
			//	// save sha1 to local DB
			//	f.task.localFileDb.Update(localFile)
			//}

			// 校验SHA1是否相同
			if strings.ToLower(panFile.Sha1Hash) == strings.ToLower(localFile.Sha1Hash) {
				// do nothing
				logger.Verboseln("file is the same, no need to upload file: ", localFile.Path)
				continue
			}
			uploadLocalFile := &FileActionTask{
				syncItem: &SyncFileItem{
					Action:            SyncFileActionUpload,
					Status:            SyncFileStatusCreate,
					LocalFile:         localFile,
					PanFile:           nil,
					StatusUpdateTime:  "",
					PanFolderPath:     f.task.PanFolderPath,
					LocalFolderPath:   f.task.LocalFolderPath,
					DriveId:           f.task.DriveId,
					DownloadBlockSize: f.syncOption.FileDownloadBlockSize,
					UploadBlockSize:   f.syncOption.FileUploadBlockSize,
				},
			}
			f.addToSyncDb(uploadLocalFile)
		} else if f.task.Mode == DownloadOnly {
			// 校验SHA1是否相同
			if strings.ToLower(panFile.Sha1Hash) == strings.ToLower(localFile.Sha1Hash) {
				// do nothing
				logger.Verboseln("file is the same, no need to download file: ", localFile.Path)
				continue
			}
			downloadPanFile := &FileActionTask{
				syncItem: &SyncFileItem{
					Action:            SyncFileActionDownload,
					Status:            SyncFileStatusCreate,
					LocalFile:         nil,
					PanFile:           panFile,
					StatusUpdateTime:  "",
					PanFolderPath:     f.task.PanFolderPath,
					LocalFolderPath:   f.task.LocalFolderPath,
					DriveId:           f.task.DriveId,
					DownloadBlockSize: f.syncOption.FileDownloadBlockSize,
					UploadBlockSize:   f.syncOption.FileUploadBlockSize,
				},
			}
			f.addToSyncDb(downloadPanFile)
		} else if f.task.Mode == SyncTwoWay {
			// TODO: no support yet
			logger.Verboseln("not support sync mode")
		}
	}
}

// 创建本地文件夹
func (f *FileActionTaskManager) createLocalFolder(panFileItem *PanFileItem) error {
	panPath := panFileItem.Path
	panPath = strings.ReplaceAll(panPath, "\\", "/")
	panRootPath := strings.ReplaceAll(f.task.PanFolderPath, "\\", "/")
	relativePath := strings.TrimPrefix(panPath, panRootPath)
	localFilePath := path.Join(path.Clean(f.task.LocalFolderPath), relativePath)

	// 创建文件夹
	var er error
	if b, e := utils.PathExists(localFilePath); e == nil && !b {
		f.localCreateMutex.Lock()
		er = os.MkdirAll(localFilePath, 0755)
		f.localCreateMutex.Unlock()
		time.Sleep(200 * time.Millisecond)
	}
	return er
}

// createPanFolder 创建云盘文件夹
func (f *FileActionTaskManager) createPanFolder(localFileItem *LocalFileItem) error {
	localPath := localFileItem.Path
	localPath = strings.ReplaceAll(localPath, "\\", "/")
	localRootPath := strings.ReplaceAll(f.task.LocalFolderPath, "\\", "/")

	relativePath := strings.TrimPrefix(localPath, localRootPath)
	panDirPath := path.Join(path.Clean(f.task.PanFolderPath), relativePath)

	// 创建文件夹
	logger.Verbosef("创建云盘文件夹: %s\n", panDirPath)
	f.panCreateMutex.Lock()
	_, apierr1 := f.panUser.PanClient().OpenapiPanClient().MkdirByFullPath(f.task.DriveId, panDirPath)
	f.panCreateMutex.Unlock()
	if apierr1 == nil {
		logger.Verbosef("创建云盘文件夹成功: %s\n", panDirPath)
		return nil
	} else {
		return apierr1
	}
}

func (f *FileActionTaskManager) addToSyncDb(fileTask *FileActionTask) {
	f.mutex.Lock()
	defer f.mutex.Unlock()

	// check sync db
	if itemInDb, e := f.task.syncFileDb.Get(fileTask.syncItem.Id()); e == nil && itemInDb != nil {
		if itemInDb.Status == SyncFileStatusCreate || itemInDb.Status == SyncFileStatusDownloading || itemInDb.Status == SyncFileStatusUploading {
			return
		}
		if itemInDb.Status == SyncFileStatusSuccess {
			if (time.Now().Unix() - itemInDb.StatusUpdateTimeUnix()) < TimeSecondsOfOneMinute {
				// 少于1分钟，不同步，减少同步频次
				return
			}
		}
		if itemInDb.Status == SyncFileStatusIllegal {
			if (time.Now().Unix() - itemInDb.StatusUpdateTimeUnix()) < TimeSecondsOf60Minute {
				// 非法文件，少于60分钟，不同步，减少同步频次
				return
			}
		}
		if itemInDb.Status == SyncFileStatusNotExisted {
			if itemInDb.Action == SyncFileActionDownload {
				if itemInDb.PanFile.UpdatedAt == fileTask.syncItem.PanFile.UpdatedAt {
					return
				}
			} else if itemInDb.Action == SyncFileActionUpload {
				if itemInDb.LocalFile.UpdatedAt == fileTask.syncItem.LocalFile.UpdatedAt {
					return
				}
			}
		}
	}

	// 进入任务队列
	f.task.syncFileDb.Add(fileTask.syncItem)
}

func (f *FileActionTaskManager) getFromSyncDb(act SyncFileAction) *FileActionTask {
	f.mutex.Lock()
	defer f.mutex.Unlock()

	if act == SyncFileActionDownload {
		// 未完成下载的先执行
		if files, e := f.task.syncFileDb.GetFileList(SyncFileStatusDownloading); e == nil {
			for _, file := range files {
				if !f.fileInProcessQueue.Contains(file) {
					return &FileActionTask{
						localFileDb:            f.task.localFileDb,
						panFileDb:              f.task.panFileDb,
						syncFileDb:             f.task.syncFileDb,
						panClient:              f.task.panClient,
						syncItem:               file,
						maxDownloadRate:        f.syncOption.MaxDownloadRate,
						maxUploadRate:          f.syncOption.MaxUploadRate,
						localFolderCreateMutex: f.localCreateMutex,
						panFolderCreateMutex:   f.panCreateMutex,
						fileRecorder:           f.syncOption.FileRecorder,
					}
				}
			}
		}
	} else if act == SyncFileActionUpload {
		// 未完成上传的先执行
		if files, e := f.task.syncFileDb.GetFileList(SyncFileStatusUploading); e == nil {
			for _, file := range files {
				if !f.fileInProcessQueue.Contains(file) {
					return &FileActionTask{
						localFileDb:            f.task.localFileDb,
						panFileDb:              f.task.panFileDb,
						syncFileDb:             f.task.syncFileDb,
						panClient:              f.task.panClient,
						syncItem:               file,
						maxDownloadRate:        f.syncOption.MaxDownloadRate,
						maxUploadRate:          f.syncOption.MaxUploadRate,
						localFolderCreateMutex: f.localCreateMutex,
						panFolderCreateMutex:   f.panCreateMutex,
						fileRecorder:           f.syncOption.FileRecorder,
					}
				}
			}
		}
	}

	// 未执行的新文件
	if files, e := f.task.syncFileDb.GetFileList(SyncFileStatusCreate); e == nil {
		if len(files) > 0 {
			for _, file := range files {
				if file.Action == act && !f.fileInProcessQueue.Contains(file) {
					return &FileActionTask{
						localFileDb:            f.task.localFileDb,
						panFileDb:              f.task.panFileDb,
						syncFileDb:             f.task.syncFileDb,
						panClient:              f.task.panClient,
						syncItem:               file,
						maxDownloadRate:        f.syncOption.MaxDownloadRate,
						maxUploadRate:          f.syncOption.MaxUploadRate,
						localFolderCreateMutex: f.localCreateMutex,
						panFolderCreateMutex:   f.panCreateMutex,
						fileRecorder:           f.syncOption.FileRecorder,
					}
				}
			}
		}
	}
	return nil
}

// cleanSyncDbRecords 清楚同步数据库无用数据
func (f *FileActionTaskManager) cleanSyncDbRecords(ctx context.Context) {
	// TODO: failed / success / illegal
}

// fileActionTaskExecutor 异步执行文件上传、下载操作
func (f *FileActionTaskManager) fileActionTaskExecutor(ctx context.Context) {
	f.wg.AddDelta()
	defer f.wg.Done()

	downloadWaitGroup := waitgroup.NewWaitGroup(f.syncOption.FileDownloadParallel)
	uploadWaitGroup := waitgroup.NewWaitGroup(f.syncOption.FileUploadParallel)

	for {
		select {
		case <-ctx.Done():
			// cancel routine & done
			logger.Verboseln("file executor routine done")
			downloadWaitGroup.Wait()
			uploadWaitGroup.Wait()
			return
		default:
			actionIsEmptyOfThisTerm := true
			// do upload
			uploadItem := f.getFromSyncDb(SyncFileActionUpload)
			if uploadItem != nil {
				actionIsEmptyOfThisTerm = false
				if uploadWaitGroup.Parallel() < f.syncOption.FileUploadParallel {
					uploadWaitGroup.AddDelta()
					f.fileInProcessQueue.PushUnique(uploadItem.syncItem)
					go func() {
						if e := uploadItem.DoAction(ctx); e == nil {
							// success
							f.fileInProcessQueue.Remove(uploadItem.syncItem)
							f.doPluginCallback(uploadItem.syncItem, "success")
						} else {
							// retry?
							f.fileInProcessQueue.Remove(uploadItem.syncItem)
							f.doPluginCallback(uploadItem.syncItem, "fail")
						}
						uploadWaitGroup.Done()
					}()
				}
			}

			// do download
			downloadItem := f.getFromSyncDb(SyncFileActionDownload)
			if downloadItem != nil {
				actionIsEmptyOfThisTerm = false
				if downloadWaitGroup.Parallel() < f.syncOption.FileDownloadParallel {
					downloadWaitGroup.AddDelta()
					f.fileInProcessQueue.PushUnique(downloadItem.syncItem)
					go func() {
						if e := downloadItem.DoAction(ctx); e == nil {
							// success
							f.fileInProcessQueue.Remove(downloadItem.syncItem)
							f.doPluginCallback(downloadItem.syncItem, "success")
						} else {
							// retry?
							f.fileInProcessQueue.Remove(downloadItem.syncItem)
							f.doPluginCallback(downloadItem.syncItem, "fail")
						}
						downloadWaitGroup.Done()
					}()
				}
			}

			// check action list is empty or not
			if actionIsEmptyOfThisTerm {
				// 文件执行队列是空的
				// 文件扫描进程也结束
				// 完成了一次扫描-执行的循环，可以退出了
				if f.task.IsScanLoopDone() {
					if uploadWaitGroup.Parallel() == 0 && downloadWaitGroup.Parallel() == 0 { // 如果也没有进行中的异步任务
						f.setExecuteLoopFlag(true)
						logger.Verboseln("file execute task is finish, exit normally")
						prompt := ""
						if f.task.Mode == UploadOnly {
							prompt = "完成全部文件的同步上传，等待下一次扫描"
						} else if f.task.Mode == UploadOnly {
							prompt = "完成全部文件的同步下载，等待下一次扫描"
						} else {
							prompt = "完成全部文件的同步，等待下一次扫描"
						}
						PromptPrintln(prompt)
						return
					}
				}
			}

			// delay for next term
			time.Sleep(5 * time.Second)
		}
	}
}

func (f *FileActionTaskManager) doPluginCallback(syncFile *SyncFileItem, actionResult string) bool {
	// 插件回调
	var pluginParam *plugins.SyncFileFinishParams
	if syncFile.Action == SyncFileActionUpload ||
		syncFile.Action == SyncFileActionCreatePanFolder ||
		syncFile.Action == SyncFileActionDeletePan {
		file := syncFile.LocalFile
		pluginParam = &plugins.SyncFileFinishParams{
			Action:        string(syncFile.Action),
			ActionResult:  actionResult,
			DriveId:       syncFile.DriveId,
			FileName:      file.FileName,
			FilePath:      syncFile.getPanFileFullPath(),
			FileSha1:      file.Sha1Hash,
			FileSize:      file.FileSize,
			FileType:      file.FileType,
			FileUpdatedAt: file.UpdatedAt,
		}
	} else if syncFile.Action == SyncFileActionDownload ||
		syncFile.Action == SyncFileActionCreateLocalFolder ||
		syncFile.Action == SyncFileActionDeleteLocal {
		file := syncFile.PanFile
		pluginParam = &plugins.SyncFileFinishParams{
			Action:        string(syncFile.Action),
			ActionResult:  actionResult,
			DriveId:       syncFile.DriveId,
			FileName:      file.FileName,
			FilePath:      syncFile.getLocalFileFullPath(),
			FileSha1:      file.Sha1Hash,
			FileSize:      file.FileSize,
			FileType:      file.FileType,
			FileUpdatedAt: file.UpdatedAt,
		}
	} else {
		return false
	}

	f.pluginMutex.Lock()
	defer f.pluginMutex.Unlock()
	if er := f.plugin.SyncFileFinishCallback(plugins.GetContext(f.panUser), pluginParam); er == nil {
		return true
	}
	return false
}

// getRelativePath 获取文件的相对路径
func (l *localFileSet) getRelativePath(localPath string) string {
	localPath = strings.ReplaceAll(localPath, "\\", "/")
	localRootPath := strings.ReplaceAll(l.localFolderPath, "\\", "/")
	relativePath := strings.TrimPrefix(localPath, localRootPath)
	if strings.HasPrefix(relativePath, "/") {
		relativePath = strings.TrimPrefix(relativePath, "/")
	}
	return path.Clean(relativePath)
}

// Intersection 交集
func (l *localFileSet) Intersection(other *panFileSet) (LocalFileList, PanFileList) {
	localFilePathSet := mapset.NewThreadUnsafeSet()
	relativePathLocalMap := map[string]*LocalFileItem{}
	for _, item := range l.items {
		rp := l.getRelativePath(item.Path)
		relativePathLocalMap[rp] = item
		localFilePathSet.Add(rp)
	}

	localFileList := LocalFileList{}
	panFileList := PanFileList{}
	for _, item := range other.items {
		rp := other.getRelativePath(item.Path)
		if localFilePathSet.Contains(rp) {
			localFileList = append(localFileList, relativePathLocalMap[rp])
			panFileList = append(panFileList, item)
		}
	}
	return localFileList, panFileList
}

// Difference 差集
func (l *localFileSet) Difference(other *panFileSet) LocalFileList {
	panFilePathSet := mapset.NewThreadUnsafeSet()
	for _, item := range other.items {
		rp := other.getRelativePath(item.Path)
		panFilePathSet.Add(rp)
	}

	localFileList := LocalFileList{}
	for _, item := range l.items {
		rp := l.getRelativePath(item.Path)
		if !panFilePathSet.Contains(rp) {
			localFileList = append(localFileList, item)
		}
	}
	return localFileList
}

// getRelativePath 获取文件的相对路径
func (p *panFileSet) getRelativePath(panPath string) string {
	panPath = strings.ReplaceAll(panPath, "\\", "/")
	panRootPath := strings.ReplaceAll(p.panFolderPath, "\\", "/")
	relativePath := strings.TrimPrefix(panPath, panRootPath)
	if strings.HasPrefix(relativePath, "/") {
		relativePath = strings.TrimPrefix(relativePath, "/")
	}
	return path.Clean(relativePath)
}

// Intersection 交集
func (p *panFileSet) Intersection(other *localFileSet) (PanFileList, LocalFileList) {
	localFilePathSet := mapset.NewThreadUnsafeSet()
	relativePathLocalMap := map[string]*LocalFileItem{}
	for _, item := range other.items {
		rp := other.getRelativePath(item.Path)
		relativePathLocalMap[rp] = item
		localFilePathSet.Add(rp)
	}

	localFileList := LocalFileList{}
	panFileList := PanFileList{}
	for _, item := range p.items {
		rp := p.getRelativePath(item.Path)
		if localFilePathSet.Contains(rp) {
			localFileList = append(localFileList, relativePathLocalMap[rp])
			panFileList = append(panFileList, item)
		}
	}
	return panFileList, localFileList
}

// Difference 差集
func (p *panFileSet) Difference(other *localFileSet) PanFileList {
	localFilePathSet := mapset.NewThreadUnsafeSet()
	for _, item := range other.items {
		rp := other.getRelativePath(item.Path)
		localFilePathSet.Add(rp)
	}

	panFileList := PanFileList{}
	for _, item := range p.items {
		rp := p.getRelativePath(item.Path)
		if !localFilePathSet.Contains(rp) {
			panFileList = append(panFileList, item)
		}
	}
	return panFileList
}
