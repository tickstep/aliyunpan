package syncdrive

import (
	"context"
	"errors"
	"fmt"
	"github.com/tickstep/aliyunpan-api/aliyunpan"
	"github.com/tickstep/aliyunpan-api/aliyunpan/apierror"
	"github.com/tickstep/aliyunpan/internal/config"
	"github.com/tickstep/aliyunpan/internal/file/downloader"
	"github.com/tickstep/aliyunpan/internal/file/uploader"
	"github.com/tickstep/aliyunpan/internal/functions/panupload"
	"github.com/tickstep/aliyunpan/internal/localfile"
	"github.com/tickstep/aliyunpan/internal/log"
	"github.com/tickstep/aliyunpan/internal/utils"
	"github.com/tickstep/aliyunpan/library/requester/transfer"
	"github.com/tickstep/library-go/converter"
	"github.com/tickstep/library-go/logger"
	"github.com/tickstep/library-go/requester"
	"github.com/tickstep/library-go/requester/rio"
	"github.com/tickstep/library-go/requester/rio/speeds"
	"math/rand"
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

		panClient *config.PanClient

		syncItem        *SyncFileItem
		maxDownloadRate int64 // 限制最大下载速度
		maxUploadRate   int64 // 限制最大上传速度

		localFolderCreateMutex *sync.Mutex
		panFolderCreateMutex   *sync.Mutex

		// 文件记录器，存储同步文件记录
		fileRecorder *log.FileRecorder
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
	logger.Verboseln("file action task：", utils.ObjectToJsonStr(f.syncItem, false))
	if f.syncItem.Action == SyncFileActionUpload {
		PromptPrintln("上传文件：" + f.syncItem.getLocalFileFullPath())
		if e := f.uploadFile(ctx); e != nil {
			// TODO: retry / cleanup downloading file
			return e
		} else {
			// upload success, post operation
			// save local file info into db
			var actFile *aliyunpan.FileEntity
			if f.syncItem.UploadEntity != nil && f.syncItem.UploadEntity.FileId != "" {
				if file, er := f.panClient.OpenapiPanClient().FileInfoById(f.syncItem.DriveId, f.syncItem.UploadEntity.FileId); er == nil {
					file.Path = f.syncItem.getPanFileFullPath()
					actFile = file
				}
			} else {
				if file, er := f.panClient.OpenapiPanClient().FileInfoByPath(f.syncItem.DriveId, f.syncItem.getPanFileFullPath()); er == nil {
					file.Path = f.syncItem.getPanFileFullPath()
					actFile = file
				}
			}

			if actFile != nil {
				// save file sha1 to local DB
				if file, e := f.localFileDb.Get(f.syncItem.getLocalFileFullPath()); e == nil {
					file.Sha1Hash = actFile.ContentHash
					f.localFileDb.Update(file)
				}

				// recorder file
				f.appendRecord(&log.FileRecordItem{
					Status:   "成功-上传",
					TimeStr:  utils.NowTimeStr(),
					FileSize: actFile.FileSize,
					FilePath: actFile.Path,
				})
			}
		}
	}

	if f.syncItem.Action == SyncFileActionDownload {
		PromptPrintln("下载文件：" + f.syncItem.getPanFileFullPath())
		if e := f.downloadFile(ctx); e != nil {
			// TODO: retry / cleanup downloading file
			return e
		} else {
			// download success, post operation
			if b, er := utils.PathExists(f.syncItem.getLocalFileFullPath()); er == nil && b {
				// file existed
				// remove old local file
				logger.Verbosef("delete local old file")
				if err := os.Remove(f.syncItem.getLocalFileFullPath()); err != nil {
					// error
					logger.Verbosef("移除本地旧文件出错: %s, %s\n", f.syncItem.getLocalFileFullPath(), err)
				}
				time.Sleep(200 * time.Millisecond)
			}

			// rename downloading file into target name file
			if err1 := os.Rename(f.syncItem.getLocalFileDownloadingFullPath(), f.syncItem.getLocalFileFullPath()); err1 != nil {
				logger.Verbosef("重命名下载文件出错: %s, %s\n", f.syncItem.getLocalFileDownloadingFullPath(), err1)
				time.Sleep(200 * time.Millisecond)
				return fmt.Errorf("重命名下载文件出错")
			}
			// success
			f.syncItem.Status = SyncFileStatusSuccess
			f.syncItem.StatusUpdateTime = utils.NowTimeStr()
			f.syncFileDb.Update(f.syncItem)
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
					ScanTimeAt:    utils.NowTimeStr(),
					ScanStatus:    ScanStatusNormal,
				})
			}

			// recorder
			f.appendRecord(&log.FileRecordItem{
				Status:   "成功-下载",
				TimeStr:  utils.NowTimeStr(),
				FileSize: f.syncItem.PanFile.FileSize,
				FilePath: f.syncItem.PanFile.Path,
			})
		}
	}
	return nil
}

func (f *FileActionTask) downloadFile(ctx context.Context) error {
	durl, apierr := f.panClient.OpenapiPanClient().GetFileDownloadUrl(&aliyunpan.GetFileDownloadUrlParam{
		DriveId: f.syncItem.PanFile.DriveId,
		FileId:  f.syncItem.PanFile.FileId,
	})
	time.Sleep(time.Duration(200) * time.Millisecond)
	if apierr != nil {
		if apierr.Code == apierror.ApiCodeFileNotFoundCode || apierr.Code == apierror.ApiCodeForbiddenFileInTheRecycleBin {
			f.syncItem.Status = SyncFileStatusNotExisted
			f.syncItem.StatusUpdateTime = utils.NowTimeStr()
			f.syncFileDb.Update(f.syncItem)
			return fmt.Errorf("文件不存在")
		}
		logger.Verbosef("ERROR: get download url error: %s, %s\n", f.syncItem.PanFile.Path, apierr.Error())
		return apierr
	}
	if durl == nil || durl.Url == "" {
		logger.Verbosef("无法获取有效的下载链接: %+v\n", durl)
		f.syncItem.Status = SyncFileStatusFailed
		f.syncItem.StatusUpdateTime = utils.NowTimeStr()
		f.syncFileDb.Update(f.syncItem)
		return fmt.Errorf("无法获取有效的下载链接")
	}
	if strings.HasPrefix(durl.Url, aliyunpan.IllegalDownloadUrlPrefix) {
		logger.Verbosef("无法获取有效的下载链接: %+v\n", durl)
		f.syncItem.Status = SyncFileStatusIllegal
		f.syncItem.StatusUpdateTime = utils.NowTimeStr()
		f.syncFileDb.Update(f.syncItem)
		return fmt.Errorf("文件非法，无法下载")
	}
	localDir := filepath.Dir(f.syncItem.getLocalFileFullPath())
	if b, e := utils.PathExists(localDir); e == nil && !b {
		f.localFolderCreateMutex.Lock()
		os.MkdirAll(localDir, 0755)
		f.localFolderCreateMutex.Unlock()
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
	if f.syncItem.DownloadRange == nil {
		f.syncItem.DownloadRange = &transfer.Range{
			Begin: 0,
			End:   f.syncItem.DownloadBlockSize,
		}
	}

	downloadUrl := durl.Url
	worker := downloader.NewWorker(0, f.syncItem.PanFile.DriveId, f.syncItem.PanFile.FileId, downloadUrl, writer, nil)

	status := &transfer.DownloadStatus{}
	status.AddDownloaded(f.syncItem.DownloadRange.Begin)
	status.SetTotalSize(f.syncItem.PanFile.FileSize)
	// 限速
	if f.maxDownloadRate > 0 {
		rl := speeds.NewRateLimit(f.maxDownloadRate)
		defer rl.Stop()
		status.SetRateLimit(rl)
	}
	worker.SetDownloadStatus(status)
	completed := make(chan struct{}, 0)
	rand.Seed(time.Now().UnixNano())
	go func() {
		ticker := time.NewTicker(1 * time.Second)
		defer ticker.Stop()
		for {
			select {
			case <-completed:
				return
			case <-ticker.C:
				time.Sleep(time.Duration(rand.Intn(10)*33) * time.Millisecond) // 延迟随机时间
				builder := &strings.Builder{}
				status.UpdateSpeeds()
				downloadedPercentage := fmt.Sprintf("%.2f%%", float64(status.Downloaded())/float64(status.TotalSize())*100)
				fmt.Fprintf(builder, "\r下载到本地:%s ↓ %s/%s(%s) %s/s............",
					f.syncItem.getLocalFileFullPath(),
					converter.ConvertFileSize(status.Downloaded(), 2),
					converter.ConvertFileSize(status.TotalSize(), 2),
					downloadedPercentage,
					converter.ConvertFileSize(status.SpeedsPerSecond(), 2),
				)
				PromptPrint(builder.String())
			}
		}
	}()

	client := requester.NewHTTPClient()
	client.SetKeepAlive(true)
	client.SetTimeout(10 * time.Minute)
	worker.SetClient(client)
	worker.SetPanClient(f.panClient)

	writeMu := &sync.Mutex{}
	worker.SetWriteMutex(writeMu)
	worker.SetTotalSize(f.syncItem.PanFile.FileSize)
	worker.SetAcceptRange("bytes")
	worker.SetRange(f.syncItem.DownloadRange) // 分片

	// update status
	f.syncItem.Status = SyncFileStatusDownloading
	f.syncItem.StatusUpdateTime = utils.NowTimeStr()
	f.syncFileDb.Update(f.syncItem)

	for {
		select {
		case <-ctx.Done():
			// cancel routine & done
			logger.Verboseln("file download routine cancel")
			close(completed)
			return errors.New("file download routine cancel")
		default:
			logger.Verboseln("do file download process")
			if f.syncItem.DownloadRange.End > f.syncItem.PanFile.FileSize {
				f.syncItem.DownloadRange.End = f.syncItem.PanFile.FileSize
			}
			worker.SetRange(f.syncItem.DownloadRange) // 分片

			// 检查上次执行是否有下载已完成
			if f.syncItem.DownloadRange.Begin == f.syncItem.PanFile.FileSize {
				f.syncItem.Status = SyncFileStatusSuccess
				f.syncItem.StatusUpdateTime = utils.NowTimeStr()
				f.syncFileDb.Update(f.syncItem)
				close(completed)
				return nil
			}

			// 下载分片
			// TODO: 下载失败，分片重试策略
			worker.Execute()

			if worker.GetStatus().StatusCode() == downloader.StatusCodeSuccessed {
				if f.syncItem.DownloadRange.End == f.syncItem.PanFile.FileSize {
					// finished
					f.syncItem.Status = SyncFileStatusSuccess
					f.syncItem.StatusUpdateTime = utils.NowTimeStr()
					f.syncFileDb.Update(f.syncItem)
					close(completed)
					PromptPrintln("下载完毕：" + f.syncItem.getLocalFileFullPath())
					return nil
				}

				// 下一个分片
				f.syncItem.DownloadRange.Begin = f.syncItem.DownloadRange.End
				f.syncItem.DownloadRange.End += f.syncItem.DownloadBlockSize

				// 存储状态
				f.syncFileDb.Update(f.syncItem)
			} else if worker.GetStatus().StatusCode() == downloader.StatusCodeDownloadUrlExpired {
				logger.Verboseln("download url expired: ", f.syncItem.PanFile.Path)
				// 下载链接过期，获取新的链接
				newUrl, apierr1 := f.panClient.OpenapiPanClient().GetFileDownloadUrl(&aliyunpan.GetFileDownloadUrlParam{
					DriveId: f.syncItem.PanFile.DriveId,
					FileId:  f.syncItem.PanFile.FileId,
				})
				time.Sleep(time.Duration(3) * time.Second)
				if apierr1 != nil {
					if apierr1.Code == apierror.ApiCodeFileNotFoundCode || apierr1.Code == apierror.ApiCodeForbiddenFileInTheRecycleBin {
						f.syncItem.Status = SyncFileStatusNotExisted
						f.syncItem.StatusUpdateTime = utils.NowTimeStr()
						f.syncFileDb.Update(f.syncItem)
						return fmt.Errorf("文件不存在")
					}
					logger.Verbosef("ERROR: get download url error: %s, %s\n", f.syncItem.PanFile.Path, apierr.Error())
					return apierr
				}
				if newUrl == nil || newUrl.Url == "" {
					logger.Verbosef("无法获取有效的下载链接: %+v\n", durl)
					f.syncItem.Status = SyncFileStatusFailed
					f.syncItem.StatusUpdateTime = utils.NowTimeStr()
					f.syncFileDb.Update(f.syncItem)
					return fmt.Errorf("无法获取有效的下载链接")
				}
				logger.Verboseln("query new download url: ", newUrl.Url)
				worker.SetUrl(newUrl.Url)
			} else if worker.GetStatus().StatusCode() == downloader.StatusCodeDownloadUrlExceedMaxConcurrency {
				logger.Verboseln("download url exceed max concurrency: ", f.syncItem.PanFile.Path)
				// 下载遇到限流了，下一次重试
			}
		}
	}
}

func (f *FileActionTask) uploadFile(ctx context.Context) error {
	if b, e := utils.PathExists(f.syncItem.LocalFile.Path); e == nil {
		if !b {
			// 本地文件不存在，无法上传
			f.syncItem.Status = SyncFileStatusNotExisted
			f.syncItem.StatusUpdateTime = utils.NowTimeStr()
			f.syncFileDb.Update(f.syncItem)
			return nil
		}
	}

	localFile := localfile.NewLocalFileEntity(f.syncItem.LocalFile.Path)
	err := localFile.OpenPath()
	if err != nil {
		logger.Verbosef("文件不可读 %s, 错误信息: %s\n", localFile.Path, err)
		f.syncItem.Status = SyncFileStatusFailed
		f.syncItem.StatusUpdateTime = utils.NowTimeStr()
		f.syncFileDb.Update(f.syncItem)
		return err
	}
	defer localFile.Close() // 关闭文件

	// 网盘目标文件路径
	targetPanFilePath := f.syncItem.getPanFileFullPath()

	if f.syncItem.UploadEntity == nil {
		// 尝试创建文件夹
		panDirPath := filepath.Dir(targetPanFilePath)
		panDirFileId := ""
		logger.Verbosef("检测云盘文件夹: %s\n", panDirPath)
		if dirFile, er2 := f.panClient.OpenapiPanClient().FileInfoByPath(f.syncItem.DriveId, panDirPath); er2 != nil {
			if er2.Code == apierror.ApiCodeFileNotFoundCode {
				logger.Verbosef("创建云盘文件夹: %s\n", panDirPath)
				f.panFolderCreateMutex.Lock()
				rs, apierr1 := f.panClient.OpenapiPanClient().MkdirByFullPath(f.syncItem.DriveId, panDirPath)
				f.panFolderCreateMutex.Unlock()
				if apierr1 != nil || rs.FileId == "" {
					return apierr1
				}
				panDirFileId = rs.FileId
				logger.Verbosef("创建云盘文件夹成功: %s\n", panDirPath)
			} else {
				logger.Verbosef("查询云盘文件夹错误: %s\n", er2.String())
				return er2
			}
		} else {
			if dirFile != nil && dirFile.FileId != "" {
				panDirFileId = dirFile.FileId
			}
		}

		// 计算文件SHA1
		sha1Str := ""
		proofCode := ""
		contentHashName := "sha1"
		if f.syncItem.LocalFile.Sha1Hash != "" {
			sha1Str = f.syncItem.LocalFile.Sha1Hash
		} else {
			// 正常上传流程，检测是否能秒传
			preHashMatch := true
			if f.syncItem.LocalFile.FileSize >= panupload.DefaultCheckPreHashFileSize {
				// 大文件，先计算 PreHash，用于检测是否可能支持秒传
				preHash := panupload.CalcFilePreHash(f.syncItem.LocalFile.Path)
				if len(preHash) > 0 {
					if b, er := f.panClient.OpenapiPanClient().CheckUploadFilePreHash(&aliyunpan.FileUploadCheckPreHashParam{
						DriveId:      f.syncItem.DriveId,
						Name:         f.syncItem.LocalFile.FileName,
						Size:         f.syncItem.LocalFile.FileSize,
						ParentFileId: panDirFileId,
						PreHash:      preHash,
					}); er == nil {
						preHashMatch = b
					}
				}
			}
			if preHashMatch {
				// 再计算完整文件SHA1
				logger.Verbosef("正在计算文件SHA1: %s\n", localFile.Path)
				if localFile.Length == 0 {
					sha1Str = aliyunpan.DefaultZeroSizeFileContentHash
				} else {
					localFile.Sum(localfile.CHECKSUM_SHA1)
					sha1Str = localFile.SHA1
				}
				f.syncItem.LocalFile.Sha1Hash = sha1Str
				f.syncFileDb.Update(f.syncItem)

				// 计算proof code
				localFileEntity, _ := os.Open(localFile.Path.RealPath)
				localFileInfo, _ := localFileEntity.Stat()
				proofCode = aliyunpan.CalcProofCode(f.panClient.OpenapiPanClient().GetAccessToken(), rio.NewFileReaderAtLen64(localFileEntity), localFileInfo.Size())
			} else {
				// 无需计算 sha1，直接上传
				logger.Verboseln("PreHash not match, upload file directly")
				sha1Str = ""
				contentHashName = ""
			}
		}

		// 检查同名文件是否存在
		panFileId := ""
		panFileSha1Str := ""
		efi, apierr := f.panClient.OpenapiPanClient().FileInfoByPath(f.syncItem.DriveId, targetPanFilePath)
		if apierr != nil && apierr.Code != apierror.ApiCodeFileNotFoundCode {
			logger.Verbosef("上传文件错误: %s\n", apierr.String())
			return apierr
		}
		if efi != nil && efi.FileId != "" {
			panFileId = efi.FileId
		}
		if panFileId != "" {
			if strings.ToUpper(panFileSha1Str) == strings.ToUpper(sha1Str) {
				logger.Verbosef("检测到同名文件，文件内容完全一致，无需重复上传: %s\n", targetPanFilePath)
				f.syncItem.Status = SyncFileStatusSuccess
				f.syncItem.StatusUpdateTime = utils.NowTimeStr()
				f.syncFileDb.Update(f.syncItem)
				return nil
			} else {
				// 删除云盘文件
				dp := &aliyunpan.FileBatchActionParam{
					DriveId: f.syncItem.DriveId,
					FileId:  panFileId,
				}
				if _, e := f.panClient.OpenapiPanClient().FileDeleteCompletely(dp); e != nil {
					logger.Verbosef(" 删除云盘旧文件失败: %s\n", targetPanFilePath)
					return e
				}
			}
		}

		// 自动调整BlockSize大小
		newBlockSize := utils.ResizeUploadBlockSize(localFile.Length, f.syncItem.UploadBlockSize)
		if newBlockSize != f.syncItem.UploadBlockSize {
			logger.Verboseln("resize upload block size to: " + converter.ConvertFileSize(newBlockSize, 2))
			f.syncItem.UploadBlockSize = newBlockSize
			// 存储状态
			f.syncFileDb.Update(f.syncItem)
		}

		// 创建上传任务
		appCreateUploadFileParam := &aliyunpan.CreateFileUploadParam{
			DriveId:         f.syncItem.DriveId,
			Name:            filepath.Base(targetPanFilePath),
			Size:            localFile.Length,
			ContentHash:     sha1Str,
			ContentHashName: contentHashName,
			CheckNameMode:   "refuse",
			ParentFileId:    panDirFileId,
			BlockSize:       f.syncItem.UploadBlockSize,
			ProofCode:       proofCode,
			ProofVersion:    "v1",
			LocalCreatedAt:  utils.UnixTime2LocalFormatStr(localFile.ModTime),
			LocalModifiedAt: utils.UnixTime2LocalFormatStr(localFile.ModTime),
		}
		if uploadOpEntity, err := f.panClient.OpenapiPanClient().CreateUploadFile(appCreateUploadFileParam); err != nil {
			logger.Verbosef("创建云盘上传任务失败: %s\n", targetPanFilePath)
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
			PromptPrintln("上传完毕：" + f.syncItem.getPanFileFullPath())
			return nil
		}
	} else {
		if len(f.syncItem.UploadEntity.PartInfoList) == 0 {
			// finished
			f.syncItem.Status = SyncFileStatusSuccess
			f.syncItem.StatusUpdateTime = utils.NowTimeStr()
			f.syncFileDb.Update(f.syncItem)

			PromptPrintln("上传完毕：" + f.syncItem.getPanFileFullPath())
			return nil
		}
		// 检测链接是否过期
		// check url expired or not
		uploadUrl := f.syncItem.UploadEntity.PartInfoList[f.syncItem.UploadPartSeq].UploadURL
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
			newUploadInfo, err1 := f.panClient.OpenapiPanClient().GetUploadUrl(refreshUploadParam)
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
	worker := panupload.NewPanUpload(f.panClient, f.syncItem.getPanFileFullPath(), f.syncItem.DriveId, f.syncItem.UploadEntity)

	// 初始化上传Range
	if f.syncItem.UploadRange == nil {
		f.syncItem.UploadRange = &transfer.Range{
			Begin: 0,
			End:   f.syncItem.UploadBlockSize,
		}
	}

	// 限速配置
	var rateLimit *speeds.RateLimit
	if f.maxUploadRate > 0 {
		rateLimit = speeds.NewRateLimit(f.maxUploadRate)
	}
	// 速度指示器
	speedsStat := &speeds.Speeds{}
	// 进度指示器
	status := &uploader.UploadStatus{}
	status.SetTotalSize(f.syncItem.LocalFile.FileSize)
	completed := make(chan struct{}, 0)
	rand.Seed(time.Now().UnixNano())
	go func() {
		ticker := time.NewTicker(1 * time.Second)
		defer ticker.Stop()
		for {
			select {
			case <-completed:
				return
			case <-ticker.C:
				status.SetUploaded(f.syncItem.UploadRange.Begin)
				time.Sleep(time.Duration(rand.Intn(10)*33) * time.Millisecond) // 延迟随机时间
				builder := &strings.Builder{}
				uploadedPercentage := fmt.Sprintf("%.2f%%", float64(status.Uploaded())/float64(status.TotalSize())*100)
				fmt.Fprintf(builder, "\r上传到网盘:%s ↑ %s/%s(%s) %s/s............",
					f.syncItem.getPanFileFullPath(),
					converter.ConvertFileSize(status.Uploaded(), 2),
					converter.ConvertFileSize(status.TotalSize(), 2),
					uploadedPercentage,
					converter.ConvertFileSize(speedsStat.GetSpeeds(), 2),
				)
				PromptPrint(builder.String())
			}
		}
	}()

	// 上传客户端
	uploadClient := requester.NewHTTPClient()
	uploadClient.SetTimeout(0)
	uploadClient.SetKeepAlive(true)

	// 标记上传状态
	f.syncItem.Status = SyncFileStatusUploading
	f.syncItem.StatusUpdateTime = utils.NowTimeStr()
	f.syncFileDb.Update(f.syncItem)

	worker.Precreate()
	for {
		select {
		case <-ctx.Done():
			// cancel routine & done
			logger.Verboseln("file upload routine cancel")
			close(completed)
			return errors.New("file upload routine cancel")
		default:
			logger.Verboseln("do file upload process")
			if f.syncItem.UploadRange.End > f.syncItem.LocalFile.FileSize {
				f.syncItem.UploadRange.End = f.syncItem.LocalFile.FileSize
			}
			fileReader := uploader.NewBufioSplitUnit(rio.NewFileReaderAtLen64(localFile.GetFile()), *f.syncItem.UploadRange, speedsStat, rateLimit, nil)

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
						close(completed)
						PromptPrintln("上传完毕：" + f.syncItem.getPanFileFullPath())
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

func (f *FileActionTask) appendRecord(item *log.FileRecordItem) error {
	if item == nil {
		return nil
	}
	if config.Config.FileRecordConfig == "1" {
		return f.fileRecorder.Append(item)
	}
	return nil
}
