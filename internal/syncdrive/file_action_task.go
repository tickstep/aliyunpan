package syncdrive

import (
	"context"
	"fmt"
	"github.com/tickstep/aliyunpan-api/aliyunpan"
	"github.com/tickstep/aliyunpan-api/aliyunpan/apierror"
	"github.com/tickstep/aliyunpan/internal/file/downloader"
	"github.com/tickstep/aliyunpan/internal/file/uploader"
	"github.com/tickstep/aliyunpan/internal/functions/panupload"
	"github.com/tickstep/aliyunpan/internal/localfile"
	"github.com/tickstep/aliyunpan/internal/utils"
	"github.com/tickstep/aliyunpan/library/requester/transfer"
	"github.com/tickstep/library-go/logger"
	"github.com/tickstep/library-go/requester"
	"github.com/tickstep/library-go/requester/rio"
	"os"
	"path"
	"path/filepath"
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

		panFolderCreateMutex *sync.Mutex
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
	logger.Verboseln("file action task")
	logger.Verboseln(f.syncItem)
	if f.syncItem.Action == SyncFileActionUpload {
		if e := f.uploadFile(ctx); e != nil {
			// TODO: retry / cleanup downloading file
			return e
		} else {
			// upload success, post operation
			// save local file info into db
			if file, er := f.panClient.FileInfoByPath(f.syncItem.DriveId, f.syncItem.getPanFileFullPath()); er == nil {
				f.panFileDb.Add(NewPanFileItem(file))
			}
		}
	}

	if f.syncItem.Action == SyncFileActionDownload {
		if e := f.downloadFile(ctx); e != nil {
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

func (f *FileActionTask) downloadFile(ctx context.Context) error {
	// check local file existed or not
	if b, e := utils.PathExists(f.syncItem.getLocalFileFullPath()); e == nil && b {
		// file existed
		logger.Verbosef("delete local old file")
		os.Remove(f.syncItem.getLocalFileFullPath())
		time.Sleep(200 * time.Millisecond)
	}

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
		select {
		case <-ctx.Done():
			// cancel routine & done
			logger.Verboseln("file download routine done")
			return nil
		default:
			logger.Verboseln("do file download process")
			if f.syncItem.DownloadRange.End > f.syncItem.PanFile.FileSize {
				f.syncItem.DownloadRange.End = f.syncItem.PanFile.FileSize
			}
			worker.SetRange(f.syncItem.DownloadRange) // 分片

			// 下载分片
			// TODO: 下载失败，分片重试策略
			worker.Execute()

			if worker.GetStatus().StatusCode() == downloader.StatusCodeSuccessed {
				if f.syncItem.DownloadRange.End == f.syncItem.PanFile.FileSize {
					// finished
					f.syncItem.Status = SyncFileStatusSuccess
					f.syncItem.StatusUpdateTime = utils.NowTimeStr()
					f.syncFileDb.Update(f.syncItem)
					return nil
				}

				// 下一个分片
				f.syncItem.DownloadRange.Begin = f.syncItem.DownloadRange.End
				f.syncItem.DownloadRange.End += f.blockSize

				// 存储状态
				f.syncFileDb.Update(f.syncItem)
			}
		}
	}
}

func (f *FileActionTask) uploadFile(ctx context.Context) error {
	localFile := localfile.NewLocalFileEntity(f.syncItem.LocalFile.Path)
	err := localFile.OpenPath()
	if err != nil {
		logger.Verbosef("文件不可读 %s, 错误信息: %s\n", localFile.Path, err)
		return err
	}
	defer localFile.Close() // 关闭文件

	// 网盘目标文件路径
	targetPanFilePath := f.syncItem.getPanFileFullPath()

	if f.syncItem.UploadEntity == nil {
		// 计算文件SHA1
		sha1Str := ""

		if f.syncItem.LocalFile.Sha1Hash != "" {
			sha1Str = f.syncItem.LocalFile.Sha1Hash
		} else {
			logger.Verbosef("正在计算文件SHA1: %s\n", localFile.Path)
			localFile.Sum(localfile.CHECKSUM_SHA1)
			sha1Str = localFile.SHA1
			if localFile.Length == 0 {
				sha1Str = aliyunpan.DefaultZeroSizeFileContentHash
			}
			f.syncItem.LocalFile.Sha1Hash = sha1Str
			f.syncFileDb.Update(f.syncItem)
		}

		// 检查同名文件是否存在
		efi, apierr := f.panClient.FileInfoByPath(f.syncItem.DriveId, targetPanFilePath)
		if apierr != nil && apierr.Code != apierror.ApiCodeFileNotFoundCode {
			return apierr
		}
		if efi != nil && efi.FileId != "" {
			if strings.ToUpper(efi.ContentHash) == strings.ToUpper(sha1Str) {
				logger.Verbosef("检测到同名文件，文件内容完全一致，无需重复上传: %s\n", targetPanFilePath)
				f.syncItem.Status = SyncFileStatusSuccess
				f.syncItem.StatusUpdateTime = utils.NowTimeStr()
				f.syncFileDb.Update(f.syncItem)
				return nil
			}
			// existed, delete it
			var fileDeleteResult []*aliyunpan.FileBatchActionResult
			var err *apierror.ApiError
			fileDeleteResult, err = f.panClient.FileDelete([]*aliyunpan.FileBatchActionParam{{DriveId: efi.DriveId, FileId: efi.FileId}})
			if err != nil || len(fileDeleteResult) == 0 {
				return err
			}
			time.Sleep(time.Duration(500) * time.Millisecond)
			logger.Verbosef("检测到同名文件，已移动到回收站: %s\n", targetPanFilePath)
		}

		// 创建文件夹
		panDirPath := path.Dir(targetPanFilePath)
		panDirFileId := ""
		if panDirItem, er := f.panFileDb.Get(panDirPath); er == nil {
			if panDirItem != nil && panDirItem.IsFolder() {
				panDirFileId = panDirItem.FileId
			}
		} else {
			logger.Verbosef("创建云盘文件夹: %s\n", panDirPath)
			f.panFolderCreateMutex.Lock()
			rs, apierr1 := f.panClient.Mkdir(f.syncItem.DriveId, "root", panDirPath)
			f.panFolderCreateMutex.Unlock()
			if apierr1 != nil || rs.FileId == "" {
				return apierr1
			}
			panDirFileId = rs.FileId
			logger.Verbosef("创建云盘文件夹成功: %s\n", panDirPath)

			// save into DB
			if panDirFile, e := f.panClient.FileInfoById(f.syncItem.DriveId, panDirFileId); e == nil {
				panDirFile.Path = panDirPath
				f.panFileDb.Add(NewPanFileItem(panDirFile))
			}
		}

		// 计算proof code
		proofCode := ""
		localFileEntity, _ := os.Open(localFile.Path)
		localFileInfo, _ := localFileEntity.Stat()
		proofCode = aliyunpan.CalcProofCode(f.panClient.GetAccessToken(), rio.NewFileReaderAtLen64(localFileEntity), localFileInfo.Size())
		//localFile.Close()

		// 创建上传任务
		appCreateUploadFileParam := &aliyunpan.CreateFileUploadParam{
			DriveId:         f.syncItem.DriveId,
			Name:            filepath.Base(targetPanFilePath),
			Size:            localFile.Length,
			ContentHash:     sha1Str,
			ContentHashName: "sha1",
			CheckNameMode:   "auto_rename",
			ParentFileId:    panDirFileId,
			BlockSize:       f.syncItem.UploadBlockSize,
			ProofCode:       proofCode,
			ProofVersion:    "v1",
		}
		if uploadOpEntity, err := f.panClient.CreateUploadFile(appCreateUploadFileParam); err != nil {
			logger.Verbosef("创建云盘上传任务失败: %s\n", panDirPath)
			return err
		} else {
			f.syncItem.UploadEntity = uploadOpEntity
			// 存储状态
			f.syncFileDb.Update(f.syncItem)
		}

		// 秒传
		if f.syncItem.UploadEntity.RapidUpload {
			logger.Verbosef("秒传成功, 保存到网盘路径: %s\n", targetPanFilePath)
			f.syncItem.Status = SyncFileStatusSuccess
			f.syncItem.StatusUpdateTime = utils.NowTimeStr()
			f.syncFileDb.Update(f.syncItem)
			return nil
		}
	} else {
		// 检测链接是否过期
		// check url expired or not
		uploadUrl := f.syncItem.UploadEntity.PartInfoList[f.syncItem.UploadPartSeq].UploadURL
		if f.syncItem.UseInternalUrl {
			uploadUrl = f.syncItem.UploadEntity.PartInfoList[f.syncItem.UploadPartSeq].InternalUploadURL
		}
		if panupload.IsUrlExpired(uploadUrl) {
			// get renew upload url
			logger.Verbosef("链接过期，获取新的上传链接: %s\n", targetPanFilePath)
			infoList := make([]aliyunpan.FileUploadPartInfoParam, 0)
			for _, item := range f.syncItem.UploadEntity.PartInfoList {
				infoList = append(infoList, aliyunpan.FileUploadPartInfoParam{
					PartNumber: item.PartNumber,
				})
			}
			refreshUploadParam := &aliyunpan.GetUploadUrlParam{
				DriveId:      f.syncItem.UploadEntity.DriveId,
				FileId:       f.syncItem.UploadEntity.FileId,
				PartInfoList: infoList,
				UploadId:     f.syncItem.UploadEntity.UploadId,
			}
			newUploadInfo, err1 := f.panClient.GetUploadUrl(refreshUploadParam)
			if err1 != nil {
				return err1
			}
			f.syncItem.UploadEntity.PartInfoList = newUploadInfo.PartInfoList
			f.syncFileDb.Update(f.syncItem)
		}
	}

	// 创建分片上传器
	// 阿里云盘默认就是分片上传，每一个分片对应一个part_info
	// 但是不支持分片同时上传，必须单线程，并且按照顺序从1开始一个一个上传
	worker := panupload.NewPanUpload(f.panClient, f.syncItem.getPanFileFullPath(), f.syncItem.DriveId, f.syncItem.UploadEntity, f.syncItem.UseInternalUrl)

	// 上传客户端
	uploadClient := requester.NewHTTPClient()
	uploadClient.SetTimeout(0)
	uploadClient.SetKeepAlive(true)

	if f.syncItem.UploadRange == nil {
		f.syncItem.UploadRange = &transfer.Range{
			Begin: 0,
			End:   f.syncItem.UploadBlockSize,
		}
	}

	worker.Precreate()
	for {
		select {
		case <-ctx.Done():
			// cancel routine & done
			logger.Verboseln("file upload routine done")
			return nil
		default:
			logger.Verboseln("do file upload process")
			if f.syncItem.UploadRange.End > f.syncItem.LocalFile.FileSize {
				f.syncItem.UploadRange.End = f.syncItem.LocalFile.FileSize
			}
			fileReader := uploader.NewBufioSplitUnit(rio.NewFileReaderAtLen64(localFile.GetFile()), *f.syncItem.UploadRange, nil, nil, nil)

			if uploadDone, terr := worker.UploadFile(ctx, f.syncItem.UploadPartSeq, f.syncItem.UploadRange.Begin, f.syncItem.UploadRange.End, fileReader, uploadClient); terr == nil {
				if uploadDone {
					// 上传成功
					if f.syncItem.UploadRange.End == f.syncItem.LocalFile.FileSize {
						// commit
						worker.CommitFile()

						// finished
						f.syncItem.Status = SyncFileStatusSuccess
						f.syncItem.StatusUpdateTime = utils.NowTimeStr()
						f.syncFileDb.Update(f.syncItem)
						return nil
					}

					// 下一个分片
					f.syncItem.UploadPartSeq += 1
					f.syncItem.UploadRange.Begin = f.syncItem.UploadRange.End
					f.syncItem.UploadRange.End += f.syncItem.UploadBlockSize

					// 存储状态
					f.syncFileDb.Update(f.syncItem)
				} else {
					// TODO: 上传失败，重试策略
					logger.Verboseln("upload file part error")
				}
			} else {
				// error
				logger.Verboseln("error: ", terr)
			}
		}
	}
}
