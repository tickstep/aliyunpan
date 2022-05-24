package syncdrive

import (
	"context"
	"fmt"
	"github.com/tickstep/aliyunpan-api/aliyunpan"
	"github.com/tickstep/aliyunpan/internal/utils"
	"github.com/tickstep/aliyunpan/internal/waitgroup"
	"github.com/tickstep/aliyunpan/library/collection"
	"github.com/tickstep/library-go/logger"
	"io/ioutil"
	"os"
	"path"
	"strings"
	"time"
)

type (
	SyncMode string

	// SyncTask 同步任务
	SyncTask struct {
		// Name 任务名称
		Name string `json:"name"`
		// Id 任务ID
		Id string `json:"id"`
		// DriveId 网盘ID，目前支持文件网盘
		DriveId string `json:"driveId"`
		// LocalFolderPath 本地目录
		LocalFolderPath string `json:"localFolderPath"`
		// PanFolderPath 云盘目录
		PanFolderPath string `json:"panFolderPath"`
		// Mode 同步模式
		Mode SyncMode `json:"mode"`
		// LastSyncTime 上一次同步时间
		LastSyncTime string `json:"lastSyncTime"`

		syncDbFolderPath string
		localFileDb      LocalSyncDb
		panFileDb        PanSyncDb

		wg         *waitgroup.WaitGroup
		ctx        context.Context
		cancelFunc context.CancelFunc

		panClient *aliyunpan.PanClient
	}
)

const (
	// UploadOnly 单向上传
	UploadOnly SyncMode = "upload"
	// DownloadOnly 只下载
	DownloadOnly SyncMode = "download"
	// SyncTwoWay 双向同步
	SyncTwoWay SyncMode = "sync"
)

func (t *SyncTask) NameLabel() string {
	return t.Name + "(" + t.Id + ")"
}

func (t *SyncTask) String() string {
	builder := &strings.Builder{}
	builder.WriteString("任务: " + t.NameLabel() + "\n")
	mode := "双向同步"
	if t.Mode == UploadOnly {
		mode = "只上传"
	}
	if t.Mode == UploadOnly {
		mode = "只下载"
	}
	builder.WriteString("同步模式: " + mode + "\n")
	builder.WriteString("本地目录: " + t.LocalFolderPath + "\n")
	builder.WriteString("云盘目录: " + t.PanFolderPath + "\n")
	return builder.String()
}

// Start 启动同步任务
func (t *SyncTask) Start() error {
	if t.ctx != nil {
		return fmt.Errorf("task have starting")
	}
	t.localFileDb = NewLocalSyncDb(t.localSyncDbFullPath())
	t.panFileDb = newPanSyncDbBolt(t.panSyncDbFullPath())
	if _, e := t.localFileDb.Open(); e != nil {
		return e
	}
	if _, e := t.panFileDb.Open(); e != nil {
		return e
	}

	t.wg = waitgroup.NewWaitGroup(2)

	var cancel context.CancelFunc
	t.ctx, cancel = context.WithCancel(context.Background())
	t.cancelFunc = cancel

	go t.scanLocalFile(t.ctx)
	go t.scanPanFile(t.ctx)
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

	// release resources
	if t.localFileDb != nil {
		t.localFileDb.Close()
	}
	if t.panFileDb != nil {
		t.panFileDb.Close()
	}

	// record the sync time
	t.LastSyncTime = utils.NowTimeStr()
	return nil
}

// panSyncDbFullPath 云盘文件数据库
func (t *SyncTask) panSyncDbFullPath() string {
	dir := path.Join(t.syncDbFolderPath, t.Id)
	if b, _ := utils.PathExists(dir); !b {
		os.MkdirAll(dir, 0600)
	}
	return path.Join(dir, "pan.bolt")
}

// localSyncDbFullPath 本地文件数据库
func (t *SyncTask) localSyncDbFullPath() string {
	dir := path.Join(t.syncDbFolderPath, t.Id)
	if b, _ := utils.PathExists(dir); !b {
		os.MkdirAll(dir, 0600)
	}
	return path.Join(dir, "local.bolt")
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
	}
}

// scanLocalFile 本地文件循环扫描进程
func (t *SyncTask) scanLocalFile(ctx context.Context) {
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
	folderQueue.Push(folderItem{
		fileInfo: rootFolder,
		path:     t.LocalFolderPath,
	})
	t.wg.AddDelta()
	defer t.wg.Done()
	for {
		select {
		case <-ctx.Done():
			// cancel routine & done
			logger.Verboseln("local file routine done")
			return
		default:
			// 采用广度优先遍历(BFS)进行文件遍历
			logger.Verboseln("do scan local file process")
			obj := folderQueue.Pop()
			if obj == nil {
				return
			}
			item := obj.(folderItem)
			// TODO: check to run scan process or to wait

			files, err := ioutil.ReadDir(item.path)
			if err != nil {
				continue
			}
			if len(files) == 0 {
				continue
			}
			localFileAppendList := LocalFileList{}
			for _, file := range files {
				localFile := newLocalFileItem(file, item.path+"/"+file.Name())
				localFileInDb, _ := t.localFileDb.Get(localFile.Path)
				if localFileInDb == nil {
					// append
					localFileAppendList = append(localFileAppendList, localFile)
				} else {
					// update newest info into DB
					localFileInDb.UpdatedAt = localFile.UpdatedAt
					localFileInDb.CreatedAt = localFile.CreatedAt
					localFileInDb.FileSize = localFile.FileSize
					localFileInDb.FileType = localFile.FileType
					t.localFileDb.Update(localFileInDb)
				}

				// for next term scan
				if file.IsDir() {
					folderQueue.Push(folderItem{
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

// scanPanFile 云盘文件循环扫描进程
func (t *SyncTask) scanPanFile(ctx context.Context) {
	// init the root folders info
	pathParts := strings.Split(strings.ReplaceAll(t.PanFolderPath, "\\", "/"), "/")
	fullPath := ""
	for _, p := range pathParts {
		if p == "" {
			continue
		}
		fullPath += "/" + p
		fi, err := t.panClient.FileInfoByPath(t.DriveId, fullPath)
		if err != nil {
			return
		}
		t.panFileDb.Add(NewPanFileItem(fi))
		time.Sleep(200 * time.Millisecond)
	}

	folderQueue := collection.NewFifoQueue()
	rootPanFile, err := t.panClient.FileInfoByPath(t.DriveId, t.PanFolderPath)
	if err != nil {
		return
	}
	folderQueue.Push(rootPanFile)

	t.wg.AddDelta()
	defer t.wg.Done()
	for {
		select {
		case <-ctx.Done():
			// cancel routine & done
			logger.Verboseln("pan file routine done")
			return
		default:
			// 采用广度优先遍历(BFS)进行文件遍历
			logger.Verboseln("do scan pan file process")
			obj := folderQueue.Pop()
			if obj == nil {
				return
			}
			item := obj.(*aliyunpan.FileEntity)
			// TODO: check to decide to sync file info or to await
			files, err := t.panClient.FileListGetAll(&aliyunpan.FileListParam{
				DriveId:      t.DriveId,
				ParentFileId: item.FileId,
			})
			if err != nil {
				// retry next term
				folderQueue.Push(item)
				time.Sleep(10 * time.Second)
				continue
			}
			panFileList := PanFileList{}
			for _, file := range files {
				file.Path = path.Join(item.Path, file.FileName)
				//fmt.Println(utils.ObjectToJsonStr(file, true))
				panFileInDb, _ := t.panFileDb.Get(file.Path)
				if panFileInDb == nil {
					// append
					panFileList = append(panFileList, NewPanFileItem(file))
				} else {
					// update newest info into DB
					panFileInDb.DomainId = file.DomainId
					panFileInDb.FileId = file.FileId
					panFileInDb.FileType = file.FileType
					panFileInDb.Category = file.Category
					panFileInDb.Crc64Hash = file.Crc64Hash
					panFileInDb.Sha1Hash = file.ContentHash
					panFileInDb.FileSize = file.FileSize
					panFileInDb.UpdatedAt = file.UpdatedAt
					panFileInDb.CreatedAt = file.CreatedAt
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
			time.Sleep(2 * time.Second) // 延迟避免触发风控
		}
	}
}
