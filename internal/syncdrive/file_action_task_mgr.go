package syncdrive

import (
	"context"
	"fmt"
	mapset "github.com/deckarep/golang-set"
	"github.com/tickstep/aliyunpan/internal/localfile"
	"github.com/tickstep/aliyunpan/internal/waitgroup"
	"github.com/tickstep/aliyunpan/library/collection"
	"github.com/tickstep/library-go/logger"
	"path"
	"strings"
	"sync"
	"time"
)

type (
	FileActionTaskList []*FileActionTask

	FileActionTaskManager struct {
		mutex *sync.Mutex

		task       *SyncTask
		wg         *waitgroup.WaitGroup
		ctx        context.Context
		cancelFunc context.CancelFunc
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
		mutex: &sync.Mutex{},
		task:  task,
	}
}

func (f *FileActionTaskManager) Start() error {
	if f.ctx != nil {
		return fmt.Errorf("task have starting")
	}
	f.wg = waitgroup.NewWaitGroup(2)

	var cancel context.CancelFunc
	f.ctx, cancel = context.WithCancel(context.Background())
	f.cancelFunc = cancel

	go f.doFileDiffRoutine(f.ctx)
	go f.fileActionTaskExecutor(f.ctx)
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

// getPanPathFromLocalPath 通过本地文件路径获取网盘文件的对应路径
func (f *FileActionTaskManager) getPanPathFromLocalPath(localPath string) string {
	localPath = strings.ReplaceAll(localPath, "\\", "/")
	localRootPath := strings.ReplaceAll(f.task.LocalFolderPath, "\\", "/")

	relativePath := strings.TrimPrefix(localPath, localRootPath)
	return path.Join(path.Clean(f.task.PanFolderPath), relativePath)
}

// getLocalPathFromPanPath 通过网盘文件路径获取对应的本地文件的对应路径
func (f *FileActionTaskManager) getLocalPathFromPanPath(panPath string) string {
	panPath = strings.ReplaceAll(panPath, "\\", "/")
	panRootPath := strings.ReplaceAll(f.task.PanFolderPath, "\\", "/")

	relativePath := strings.TrimPrefix(panPath, panRootPath)
	return path.Join(path.Clean(f.task.LocalFolderPath), relativePath)
}

// doFileDiffRoutine 对比网盘文件和本地文件信息，差异化上传或者下载文件
func (f *FileActionTaskManager) doFileDiffRoutine(ctx context.Context) {
	localFolderQueue := collection.NewFifoQueue()
	panFolderQueue := collection.NewFifoQueue()

	// init root folder
	if localRootFolder, e := f.task.localFileDb.Get(f.task.LocalFolderPath); e == nil {
		localFolderQueue.Push(localRootFolder)
	} else {
		logger.Verboseln(e)
		return
	}
	if panRootFolder, e := f.task.panFileDb.Get(f.task.PanFolderPath); e == nil {
		panFolderQueue.Push(panRootFolder)
	} else {
		logger.Verboseln(e)
		return
	}

	f.wg.AddDelta()
	defer f.wg.Done()
	for {
		select {
		case <-ctx.Done():
			// cancel routine & done
			logger.Verboseln("file diff routine done")
			return
		default:
			logger.Verboseln("do file diff process")
			localFiles := LocalFileList{}
			panFiles := PanFileList{}
			var err error

			// iterator local folder
			objLocal := localFolderQueue.Pop()
			if objLocal != nil {
				localItem := objLocal.(*LocalFileItem)
				localFiles, err = f.task.localFileDb.GetFileList(localItem.Path)
				if err != nil {
					localFiles = LocalFileList{}
				}
				panFiles, err = f.task.panFileDb.GetFileList(f.getPanPathFromLocalPath(localItem.Path))
				if err != nil {
					panFiles = PanFileList{}
				}
			} else {
				// iterator pan folder
				objPan := panFolderQueue.Pop()
				if objPan != nil {
					panItem := objPan.(*PanFileItem)
					panFiles, err = f.task.panFileDb.GetFileList(panItem.Path)
					if err != nil {
						panFiles = PanFileList{}
					}
					localFiles, err = f.task.localFileDb.GetFileList(f.getLocalPathFromPanPath(panItem.Path))
					if err != nil {
						localFiles = LocalFileList{}
					}
				}
			}

			// empty loop
			if len(panFiles) == 0 && len(localFiles) == 0 {
				time.Sleep(100 * time.Millisecond)
				continue
			}

			localFilesSet := &localFileSet{
				items:           localFiles,
				localFolderPath: f.task.LocalFolderPath,
			}
			panFilesSet := &panFileSet{
				items:         panFiles,
				panFolderPath: f.task.PanFolderPath,
			}
			localFilesNeedToUpload := localFilesSet.Difference(panFilesSet)
			panFilesNeedToDownload := panFilesSet.Difference(localFilesSet)
			localFilesNeedToCheck, panFilesNeedToCheck := localFilesSet.Intersection(panFilesSet)

			// download file from pan drive
			if f.task.Mode != UploadOnly {
				for _, file := range panFilesNeedToDownload {
					if file.IsFolder() {
						panFolderQueue.PushUnique(file)
						continue
					}
					fileActionTask := &FileActionTask{
						syncItem: &SyncFileItem{
							Action:           SyncFileActionDownload,
							Status:           SyncFileStatusCreate,
							LocalFile:        nil,
							PanFile:          file,
							StatusUpdateTime: "",
							PanFolderPath:    f.task.PanFolderPath,
							LocalFolderPath:  f.task.LocalFolderPath,
						},
					}
					f.addToQueue(fileActionTask)
				}
			}

			// upload file to pan drive
			if f.task.Mode != DownloadOnly {
				for _, file := range localFilesNeedToUpload {
					if file.IsFolder() {
						localFolderQueue.PushUnique(file)
						continue
					}
					fileActionTask := &FileActionTask{
						syncItem: &SyncFileItem{
							Action:           SyncFileActionUpload,
							Status:           SyncFileStatusCreate,
							LocalFile:        file,
							PanFile:          nil,
							StatusUpdateTime: "",
							PanFolderPath:    f.task.PanFolderPath,
							LocalFolderPath:  f.task.LocalFolderPath,
						},
					}
					f.addToQueue(fileActionTask)
				}
			}

			// compare file to decide download / upload
			for idx, _ := range localFilesNeedToCheck {
				localFile := localFilesNeedToCheck[idx]
				panFile := panFilesNeedToCheck[idx]
				if localFile.IsFolder() {
					localFolderQueue.PushUnique(localFile)
					continue
				}

				if localFile.Sha1Hash == "" {
					// calc sha1
					fileSum := localfile.NewLocalFileEntity(localFile.Path)
					fileSum.Sum(localfile.CHECKSUM_SHA1) // block operation
					localFile.Sha1Hash = fileSum.SHA1
					fileSum.Close()

					// save sha1
					f.task.localFileDb.Update(localFile)
				}

				if strings.ToLower(panFile.Sha1Hash) == strings.ToLower(localFile.Sha1Hash) {
					// do nothing
					logger.Verboseln("no need to update file: ", localFile.Path)
					continue
				}

				if localFile.UpdateTimeUnix() > panFile.UpdateTimeUnix() { // upload file
					uploadLocalFile := &FileActionTask{
						syncItem: &SyncFileItem{
							Action:           SyncFileActionUpload,
							Status:           SyncFileStatusCreate,
							LocalFile:        localFile,
							PanFile:          nil,
							StatusUpdateTime: "",
							PanFolderPath:    f.task.PanFolderPath,
							LocalFolderPath:  f.task.LocalFolderPath,
						},
					}
					f.addToQueue(uploadLocalFile)
				} else if localFile.UpdateTimeUnix() < panFile.UpdateTimeUnix() { // download file
					downloadPanFile := &FileActionTask{
						syncItem: &SyncFileItem{
							Action:           SyncFileActionDownload,
							Status:           SyncFileStatusCreate,
							LocalFile:        nil,
							PanFile:          panFile,
							StatusUpdateTime: "",
							PanFolderPath:    f.task.PanFolderPath,
							LocalFolderPath:  f.task.LocalFolderPath,
						},
					}
					f.addToQueue(downloadPanFile)
				}
			}

			time.Sleep(100 * time.Millisecond)
		}
	}
}

func (f *FileActionTaskManager) addToQueue(fileTask *FileActionTask) {
	f.mutex.Lock()
	defer f.mutex.Unlock()
	if itemInDb, e := f.task.syncFileDb.Get(fileTask.syncItem.Id()); e == nil && itemInDb != nil {
		if itemInDb.Status == SyncFileStatusCreate || itemInDb.Status == SyncFileStatusDownloading || itemInDb.Status == SyncFileStatusUploading {
			return
		}
		if itemInDb.Status == SyncFileStatusSuccess {
			if (time.Now().Unix() - itemInDb.StatusUpdateTimeUnix()) < TimeSecondsOf5Minute {
				// 少于5分钟，不同步，减少同步频次
				return
			}
		}
	}

	// 进入任务队列
	f.task.syncFileDb.Add(fileTask.syncItem)
}

func (f *FileActionTaskManager) getFromQueue(act SyncFileAction) *FileActionTask {
	f.mutex.Lock()
	defer f.mutex.Unlock()

	if act == SyncFileActionDownload {
		if files, e := f.task.syncFileDb.GetFileList(SyncFileStatusDownloading); e == nil {
			if len(files) > 0 {
				return &FileActionTask{
					localFileDb: f.task.localFileDb,
					panFileDb:   f.task.panFileDb,
					syncFileDb:  f.task.syncFileDb,
					panClient:   f.task.panClient,
					blockSize:   int64(10485760),
					syncItem:    files[0],
				}
			}
		}
	} else if act == SyncFileActionUpload {
		if files, e := f.task.syncFileDb.GetFileList(SyncFileStatusUploading); e == nil {
			if len(files) > 0 {
				return &FileActionTask{
					localFileDb: f.task.localFileDb,
					panFileDb:   f.task.panFileDb,
					syncFileDb:  f.task.syncFileDb,
					panClient:   f.task.panClient,
					blockSize:   int64(10485760),
					syncItem:    files[0],
				}
			}
		}
	}

	if files, e := f.task.syncFileDb.GetFileList(SyncFileStatusCreate); e == nil {
		if len(files) > 0 {
			for _, file := range files {
				if file.Action == act {
					return &FileActionTask{
						localFileDb: f.task.localFileDb,
						panFileDb:   f.task.panFileDb,
						syncFileDb:  f.task.syncFileDb,
						panClient:   f.task.panClient,
						blockSize:   int64(10485760),
						syncItem:    file,
					}
				}
			}
		}
	}

	// TODO: failed file retry?

	return nil
}

// fileActionTaskExecutor 异步执行文件操作
func (f *FileActionTaskManager) fileActionTaskExecutor(ctx context.Context) {
	f.wg.AddDelta()
	defer f.wg.Done()

	for {
		select {
		case <-ctx.Done():
			// cancel routine & done
			logger.Verboseln("file executor routine done")
			return
		default:
			logger.Verboseln("do file executor process")

			// do upload
			uploadItem := f.getFromQueue(SyncFileActionUpload)
			if uploadItem != nil {
				uploadItem.DoAction(ctx)
			}

			// do download
			downloadItem := f.getFromQueue(SyncFileActionDownload)
			if downloadItem != nil {
				downloadItem.DoAction(ctx)
			}

			// delay
			time.Sleep(500 * time.Millisecond)
		}
	}
}

// getRelativePath 获取文件的相对路径
func (l *localFileSet) getRelativePath(localPath string) string {
	localPath = strings.ReplaceAll(localPath, "\\", "/")
	localRootPath := strings.ReplaceAll(l.localFolderPath, "\\", "/")
	relativePath := strings.TrimPrefix(localPath, localRootPath)
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
