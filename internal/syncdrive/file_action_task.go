package syncdrive

import (
	"context"
	"fmt"
	"github.com/tickstep/aliyunpan-api/aliyunpan"
	"github.com/tickstep/aliyunpan-api/aliyunpan/apierror"
	"github.com/tickstep/aliyunpan/internal/file/downloader"
	"github.com/tickstep/aliyunpan/internal/utils"
	"github.com/tickstep/aliyunpan/library/requester/transfer"
	"github.com/tickstep/library-go/logger"
	"github.com/tickstep/library-go/requester"
	"os"
	"path"
	"strings"
	"sync"
	"time"
)

type (
	FileAction     string
	FileActionTask struct {
		localFileDb LocalSyncDb
		panFileDb   PanSyncDb
		syncFileDb  SyncFileDb

		panClient *aliyunpan.PanClient
		blockSize int64

		syncItem *SyncFileItem
	}
)

func (f *FileActionTask) HashCode() string {
	postfix := ""
	if f.syncItem.Action == SyncFileActionDownload {
		postfix = strings.ReplaceAll(f.syncItem.PanFile.Path, "\\", "/")
	} else if f.syncItem.Action == SyncFileActionUpload {
		postfix = strings.ReplaceAll(f.syncItem.LocalFile.Path, "\\", "/")
	}
	return string(f.syncItem.Action) + postfix
}

func (f *FileActionTask) DoAction(ctx context.Context) error {
	fmt.Println("\nfile action task")
	fmt.Println(f.syncItem)
	if f.syncItem.Action == SyncFileActionUpload {
		if e := f.uploadFile(); e != nil {
			return e
		}
	}
	if f.syncItem.Action == SyncFileActionDownload {
		if e := f.downloadFile(); e != nil {
			// TODO: retry / cleanup downloading file
			return e
		} else {
			// download success, post operation
			if b, er := utils.PathExists(f.syncItem.getLocalFileFullPath()); er == nil && b {
				// file existed
				// remove old local file
				logger.Verbosef("delete local old file")
				os.Remove(f.syncItem.getLocalFileFullPath())
				time.Sleep(200 * time.Millisecond)
			}

			// rename downloading file into target name file
			os.Rename(f.syncItem.getLocalFileDownloadingFullPath(), f.syncItem.getLocalFileFullPath())
			time.Sleep(200 * time.Millisecond)

			// change modify time of local file
			if err := os.Chtimes(f.syncItem.getLocalFileFullPath(), f.syncItem.PanFile.UpdateTime(), f.syncItem.PanFile.UpdateTime()); err != nil {
				logger.Verbosef(err.Error())
			}
			time.Sleep(200 * time.Millisecond)

			// save local file info into db
			if file, er := os.Stat(f.syncItem.getLocalFileFullPath()); er == nil {
				f.localFileDb.Add(&LocalFileItem{
					FileName:      file.Name(),
					FileSize:      file.Size(),
					FileType:      "file",
					CreatedAt:     file.ModTime().Format("2006-01-02 15:04:05"),
					UpdatedAt:     file.ModTime().Format("2006-01-02 15:04:05"),
					FileExtension: path.Ext(file.Name()),
					Sha1Hash:      f.syncItem.PanFile.Sha1Hash,
					Path:          f.syncItem.getLocalFileFullPath(),
				})
			}
		}
	}
	return nil
}

func (f *FileActionTask) downloadFile() error {
	// check local file existed or not
	//if b, e := utils.PathExists(f.syncItem.getLocalFileFullPath()); e == nil && b {
	//	// file existed
	//	logger.Verbosef("delete local old file")
	//	os.Remove(f.syncItem.getLocalFileFullPath())
	//	time.Sleep(200 * time.Millisecond)
	//}

	durl, apierr := f.panClient.GetFileDownloadUrl(&aliyunpan.GetFileDownloadUrlParam{
		DriveId: f.syncItem.PanFile.DriveId,
		FileId:  f.syncItem.PanFile.FileId,
	})
	time.Sleep(time.Duration(200) * time.Millisecond)
	if apierr != nil {
		if apierr.Code == apierror.ApiCodeFileNotFoundCode {
			f.syncItem.Status = SyncFileStatusNotExisted
			f.syncItem.StatusUpdateTime = utils.NowTimeStr()
			f.syncFileDb.Update(f.syncItem)
			return fmt.Errorf("文件不存在")
		}
		logger.Verbosef("ERROR: get download url error: %s\n", f.syncItem.PanFile.FileId)
		return apierr
	}
	if durl == nil || durl.Url == "" {
		logger.Verbosef("无法获取有效的下载链接: %+v\n", durl)
		f.syncItem.Status = SyncFileStatusFailed
		f.syncItem.StatusUpdateTime = utils.NowTimeStr()
		f.syncFileDb.Update(f.syncItem)
		return fmt.Errorf("无法获取有效的下载链接")
	}
	if durl.Url == aliyunpan.IllegalDownloadUrl {
		logger.Verbosef("无法获取有效的下载链接: %+v\n", durl)
		f.syncItem.Status = SyncFileStatusIllegal
		f.syncItem.StatusUpdateTime = utils.NowTimeStr()
		f.syncFileDb.Update(f.syncItem)
		return fmt.Errorf("文件非法，无法下载")
	}
	localDir := path.Dir(f.syncItem.getLocalFileFullPath())
	if b, e := utils.PathExists(localDir); e == nil && !b {
		os.MkdirAll(localDir, 0600)
		time.Sleep(200 * time.Millisecond)
	}
	writer, file, err := downloader.NewDownloaderWriterByFilename(f.syncItem.getLocalFileDownloadingFullPath(), os.O_CREATE|os.O_WRONLY, 0666)
	if err != nil {
		return fmt.Errorf("%s, %s", "初始化下载发生错误", err)
	}
	defer file.Close()
	if f.syncItem.PanFile.FileSize == 0 {
		// zero file
		f.syncItem.Status = SyncFileStatusSuccess
		f.syncItem.StatusUpdateTime = utils.NowTimeStr()
		f.syncFileDb.Update(f.syncItem)
		return nil
	}

	worker := downloader.NewWorker(0, f.syncItem.PanFile.DriveId, f.syncItem.PanFile.FileId, durl.Url, writer, nil)

	client := requester.NewHTTPClient()
	client.SetKeepAlive(true)
	client.SetTimeout(10 * time.Minute)
	worker.SetClient(client)
	worker.SetPanClient(f.panClient)

	writeMu := &sync.Mutex{}
	worker.SetWriteMutex(writeMu)
	worker.SetTotalSize(f.syncItem.PanFile.FileSize)
	worker.SetAcceptRange("bytes")
	if f.syncItem.DownloadRange == nil {
		f.syncItem.DownloadRange = &transfer.Range{
			Begin: 0,
			End:   f.blockSize,
		}
	}
	worker.SetRange(f.syncItem.DownloadRange) // 分片

	// update status
	f.syncItem.Status = SyncFileStatusDownloading
	f.syncItem.StatusUpdateTime = utils.NowTimeStr()
	f.syncFileDb.Update(f.syncItem)

	for {
		if f.syncItem.DownloadRange.End > f.syncItem.PanFile.FileSize {
			f.syncItem.DownloadRange.End = f.syncItem.PanFile.FileSize
		}
		worker.SetRange(f.syncItem.DownloadRange) // 分片

		// 下载分片
		// TODO: 分片重试策略
		worker.Execute()

		if worker.GetStatus().StatusCode() == downloader.StatusCodeSuccessed {
			if f.syncItem.DownloadRange.End == f.syncItem.PanFile.FileSize {
				// finished
				f.syncItem.Status = SyncFileStatusSuccess
				f.syncItem.StatusUpdateTime = utils.NowTimeStr()
				f.syncFileDb.Update(f.syncItem)
				break
			}

			// 下一个分片
			f.syncItem.DownloadRange.Begin = f.syncItem.DownloadRange.End
			f.syncItem.DownloadRange.End += f.blockSize

			// 存储状态
			f.syncFileDb.Update(f.syncItem)
		}
	}

	return nil
}

func (f *FileActionTask) uploadFile() error {
	return nil
}
