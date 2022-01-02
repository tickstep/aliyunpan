package webdav

import (
	"bytes"
	"fmt"
	"github.com/tickstep/aliyunpan-api/aliyunpan"
	"github.com/tickstep/aliyunpan-api/aliyunpan/apierror"
	"github.com/tickstep/aliyunpan/internal/config"
	"github.com/tickstep/library-go/expires"
	"github.com/tickstep/library-go/expires/cachemap"
	"github.com/tickstep/library-go/logger"
	"github.com/tickstep/library-go/requester"
	"net/http"
	"os"
	"path"
	"path/filepath"
	"strconv"
	"strings"
	"sync"
	"time"
)

type FileDownloadStream struct {
	readOffset  int64
	resp *http.Response
	timestamp int64
}

type FileUploadStream struct {
	createFileUploadResult *aliyunpan.CreateFileUploadResult

	filePath string
	fileSize int64
	fileId string
	fileWritePos int64
	fileUploadUrlIndex int

	chunkBuffer []byte
	chunkPos int64
	chunkSize int64

	timestamp int64

	mutex sync.Mutex
}

type PanClientProxy struct {
	PanUser *config.PanUser
	PanDriveId string
	PanTransferUrlType int

	mutex sync.Mutex

	// 网盘文件路径到网盘文件信息实体映射缓存
	filePathCacheMap          cachemap.CacheOpMap

	// 网盘文件夹路径到文件夹下面所有子文件映射缓存
	fileDirectoryListCacheMap cachemap.CacheOpMap

	// 网盘文件ID到文件下载链接映射缓存
	fileIdDownloadUrlCacheMap cachemap.CacheOpMap

	// 网盘文件ID到文件下载数据流映射缓存
	fileIdDownloadStreamCacheMap cachemap.CacheOpMap

	// 网盘文件到文件上传数据流映射缓存
	filePathUploadStreamCacheMap cachemap.CacheOpMap
}

// DefaultChunkSize 默认上传的文件块大小，10MB
const DefaultChunkSize = 10 * 1024 * 1024

// CacheExpiredMinute  缓存过期分钟
const CacheExpiredMinute = 60

// FileDownloadUrlExpiredSeconds 文件下载URL过期时间
const FileDownloadUrlExpiredSeconds = 14400

// FileUploadExpiredMinute 文件上传过期时间
const FileUploadExpiredMinute = 1440 // 24小时

func formatPathStyle(pathStr string) string {
	pathStr = strings.ReplaceAll(pathStr, "\\", "/")
	pathStr = strings.TrimSuffix(pathStr, "/")
	return pathStr
}

// getDownloadFileUrl 获取文件下载URL
func (p *PanClientProxy) getFileDownloadUrl(urlResult *aliyunpan.GetFileDownloadUrlResult) string {
	if urlResult == nil {
		return ""
	}

	if p.PanTransferUrlType == 2 { // 阿里ECS内网链接
		return urlResult.InternalUrl
	}
	return urlResult.Url
}

// getFileUploadUrl 获取文件上传URL
func (p *PanClientProxy) getFileUploadUrl(urlResult aliyunpan.FileUploadPartInfoResult) string {
	if p.PanTransferUrlType == 2 { // 阿里ECS内网链接
		return urlResult.InternalUploadURL
	}
	return urlResult.UploadURL
}

// DeleteCache 删除含有 dirs 的缓存
func (p *PanClientProxy) deleteFilesDirectoriesListCache(dirs []string) {
	cache := p.fileDirectoryListCacheMap.LazyInitCachePoolOp(p.PanDriveId)
	for _, v := range dirs {
		key := formatPathStyle(v)
		_, ok := cache.Load(key)
		if ok {
			cache.Delete(key)
		}
	}
}

// DeleteOneCache 删除缓存
func (p *PanClientProxy) deleteOneFilesDirectoriesListCache(dirPath string) {
	dirPath = formatPathStyle(dirPath)
	ps := []string{dirPath}
	p.deleteFilesDirectoriesListCache(ps)
}

// cacheFilesDirectoriesList 缓存文件夹下面的所有文件列表
func (p *PanClientProxy) cacheFilesDirectoriesList(pathStr string) (fdl aliyunpan.FileList, apiError *apierror.ApiError) {
	pathStr = formatPathStyle(pathStr)
	data := p.fileDirectoryListCacheMap.CacheOperation(p.PanDriveId, pathStr, func() expires.DataExpires {
		fi, er := p.cacheFilePath(pathStr)
		if er != nil {
			return nil
		}
		fileListParam := &aliyunpan.FileListParam{
			DriveId: p.PanDriveId,
			ParentFileId: fi.FileId,
			Limit: 200,
		}
		fdl, apiError = p.PanUser.PanClient().FileListGetAll(fileListParam)
		if apiError != nil {
			return nil
		}
		if len(fdl) == 0{
			// 空目录不缓存
			return nil
		}
		// construct full path
		for _, f := range fdl {
			f.Path = path.Join(pathStr, f.FileName)
		}
		p.cacheFilePathEntityList(fdl)
		return expires.NewDataExpires(fdl, CacheExpiredMinute*time.Minute)
	})
	if apiError != nil {
		return
	}
	if data == nil {
		return aliyunpan.FileList{}, nil
	}
	return data.Data().(aliyunpan.FileList), nil
}

// deleteOneFilePathCache 删除缓存
func (p *PanClientProxy) deleteOneFilePathCache(pathStr string) {
	key := formatPathStyle(pathStr)
	cache := p.filePathCacheMap.LazyInitCachePoolOp(p.PanDriveId)
	_, ok := cache.Load(key)
	if ok {
		cache.Delete(key)
	}
}

// cacheFilePath 缓存文件绝对路径到网盘文件信息
func (p *PanClientProxy) cacheFilePath(pathStr string) (fe *aliyunpan.FileEntity, apiError *apierror.ApiError) {
	pathStr = formatPathStyle(pathStr)
	data := p.filePathCacheMap.CacheOperation(p.PanDriveId, pathStr, func() expires.DataExpires {
		var fi *aliyunpan.FileEntity
		fi, apiError = p.PanUser.PanClient().FileInfoByPath(p.PanDriveId, pathStr)
		if apiError != nil {
			return nil
		}
		return expires.NewDataExpires(fi, CacheExpiredMinute*time.Minute)
	})
	if apiError != nil {
		return nil, apiError
	}
	if data == nil {
		return nil, nil
	}
	return data.Data().(*aliyunpan.FileEntity), nil
}

func (p *PanClientProxy) cacheFilePathEntity(fe *aliyunpan.FileEntity) {
	pathStr := formatPathStyle(fe.Path)
	p.filePathCacheMap.CacheOperation(p.PanDriveId, pathStr, func() expires.DataExpires {
		return expires.NewDataExpires(fe, CacheExpiredMinute*time.Minute)
	})
}

func (p *PanClientProxy) cacheFilePathEntityList(fdl aliyunpan.FileList) {
	for _,entity := range fdl {
		pathStr := formatPathStyle(entity.Path)
		p.filePathCacheMap.CacheOperation(p.PanDriveId, pathStr, func() expires.DataExpires {
			return expires.NewDataExpires(entity, CacheExpiredMinute*time.Minute)
		})
	}
}

// cacheFileDownloadStream 缓存文件下载路径
func (p *PanClientProxy) cacheFileDownloadUrl(sessionId, fileId string) (urlResult *aliyunpan.GetFileDownloadUrlResult, apiError *apierror.ApiError) {
	k := sessionId + "-" + fileId
	data := p.fileIdDownloadUrlCacheMap.CacheOperation(p.PanDriveId, k, func() expires.DataExpires {
		urlResult, err1 := p.PanUser.PanClient().GetFileDownloadUrl(&aliyunpan.GetFileDownloadUrlParam{
			DriveId:   p.PanDriveId,
			FileId:    fileId,
			ExpireSec: FileDownloadUrlExpiredSeconds,
		})
		if err1 != nil {
			return nil
		}
		return expires.NewDataExpires(urlResult, (FileDownloadUrlExpiredSeconds-60)*time.Second)
	})
	if data == nil {
		return nil, nil
	}
	return data.Data().(*aliyunpan.GetFileDownloadUrlResult), nil
}

// deleteOneFileDownloadStreamCache 删除缓存文件下载流缓存
func (p *PanClientProxy) deleteOneFileDownloadStreamCache(sessionId, fileId string) {
	key := sessionId + "-" + fileId
	cache := p.fileIdDownloadStreamCacheMap.LazyInitCachePoolOp(p.PanDriveId)
	_, ok := cache.Load(key)
	if ok {
		cache.Delete(key)
	}
}

// cacheFileDownloadStream 缓存文件下载流
func (p *PanClientProxy) cacheFileDownloadStream(sessionId, fileId string, offset int64) (fds *FileDownloadStream, apiError *apierror.ApiError) {
	k := sessionId + "-" + fileId
	data := p.fileIdDownloadStreamCacheMap.CacheOperation(p.PanDriveId, k, func() expires.DataExpires {
		urlResult, err1 := p.cacheFileDownloadUrl(sessionId, fileId)
		if err1 != nil {
			return nil
		}

		var resp *http.Response
		var err error
		var client = requester.NewHTTPClient()
		// set to no timeout
		client.Timeout = 0
		apierr := p.PanUser.PanClient().DownloadFileData(
			p.getFileDownloadUrl(urlResult),
			aliyunpan.FileDownloadRange{
				Offset: offset,
				End:    0,
			},
			func(httpMethod, fullUrl string, headers map[string]string) (*http.Response, error) {
				resp, err = client.Req(httpMethod, fullUrl, nil, headers)
				if err != nil {
					return nil, err
				}
				return resp, err
			})

		if apierr != nil {
			return nil
		}

		switch resp.StatusCode {
		case 200, 206:
			// do nothing, continue
			break
		case 416: //Requested Range Not Satisfiable
			fallthrough
		case 403: // Forbidden
			fallthrough
		case 406: // Not Acceptable
			return nil
		case 404:
			return nil
		case 429, 509: // Too Many Requests
			return nil
		default:
			return nil
		}

		logger.Verboseln(sessionId + " create new cache for offset = " + strconv.Itoa(int(offset)))
		return expires.NewDataExpires(&FileDownloadStream{
			readOffset: offset,
			resp: resp,
			timestamp:  time.Now().Unix(),
		}, CacheExpiredMinute*time.Minute)
	})

	if data == nil {
		return nil, nil
	}
	return data.Data().(*FileDownloadStream), nil
}

// deleteOneFileUploadStreamCache 删除缓存文件下载流缓存
func (p *PanClientProxy) deleteOneFileUploadStreamCache(userId, pathStr string) {
	pathStr = formatPathStyle(pathStr)
	key := userId + "-" + pathStr
	cache := p.filePathUploadStreamCacheMap.LazyInitCachePoolOp(p.PanDriveId)
	_, ok := cache.Load(key)
	if ok {
		cache.Delete(key)
	}
}

// cacheFileUploadStream 缓存创建的文件上传流
func (p *PanClientProxy) cacheFileUploadStream(userId, pathStr string, fileSize int64, chunkSize int64) (*FileUploadStream, *apierror.ApiError) {
	pathStr = formatPathStyle(pathStr)
	k := userId + "-" + pathStr
	// TODO: add locker for upload file create
	data := p.filePathUploadStreamCacheMap.CacheOperation(p.PanDriveId, k, func() expires.DataExpires {
		// check parent dir is existed or not
		parentFileId := ""
		parentFileEntity, err1 := p.cacheFilePath(path.Dir(pathStr))
		if err1 != nil {
			return nil
		}
		if parentFileEntity == nil {
			// create parent folder
			mkr, err2 := p.mkdir(path.Dir(pathStr), 0)
			if err2 != nil {
				return nil
			}
			parentFileId = mkr.FileId
		} else {
			parentFileId = parentFileEntity.FileId
		}


		// 检查同名文件是否存在
		efi, apierr := p.PanUser.PanClient().FileInfoByPath(p.PanDriveId, pathStr)
		if apierr != nil {
			if apierr.Code == apierror.ApiCodeFileNotFoundCode {
				// file not existed
				logger.Verbosef("%s 没有存在同名文件，直接上传: %s", userId, pathStr)
			} else {
				// TODO: handle error
				return nil
			}
		} else {
			if efi != nil && efi.FileId != "" {
				// existed, delete it
				var fileDeleteResult []*aliyunpan.FileBatchActionResult
				var err *apierror.ApiError
				fileDeleteResult, err = p.PanUser.PanClient().FileDelete([]*aliyunpan.FileBatchActionParam{{DriveId:efi.DriveId, FileId:efi.FileId}})
				if err != nil || len(fileDeleteResult) == 0 {
					logger.Verbosef("%s 同名无法删除文件，请稍后重试: %s", userId, pathStr)
					return nil
				}
				time.Sleep(time.Duration(500) * time.Millisecond)
				logger.Verbosef("%s 检测到同名文件，已移动到回收站: %s", userId, pathStr)

				// clear cache
				p.deleteOneFilePathCache(pathStr)
				p.deleteOneFilesDirectoriesListCache(path.Dir(pathStr))
			}
		}

		// create new upload file
		appCreateUploadFileParam := &aliyunpan.CreateFileUploadParam{
			DriveId:      p.PanDriveId,
			Name:         filepath.Base(pathStr),
			Size:         fileSize,
			ContentHash:  "",
			ContentHashName: "none",
			CheckNameMode: "refuse",
			ParentFileId: parentFileId,
			BlockSize: chunkSize,
			ProofCode: "",
			ProofVersion: "v1",
		}

		uploadOpEntity, apierr := p.PanUser.PanClient().CreateUploadFile(appCreateUploadFileParam)
		if apierr != nil {
			logger.Verbosef("%s 创建上传任务失败: %s", userId, pathStr)
			return nil
		}

		logger.Verbosef("%s create new upload cache for path = %s", userId, pathStr)
		return expires.NewDataExpires(&FileUploadStream{
			createFileUploadResult: uploadOpEntity,
			filePath:               pathStr,
			fileSize:               fileSize,
			fileId:                 uploadOpEntity.FileId,
			fileWritePos:           0,
			fileUploadUrlIndex:     0,
			chunkBuffer:            make([]byte, chunkSize, chunkSize),
			chunkPos:               0,
			chunkSize:              chunkSize,
			timestamp:              time.Now().Unix(),
		}, FileUploadExpiredMinute*time.Minute)
	})

	if data == nil {
		return nil, nil
	}
	return data.Data().(*FileUploadStream), nil
}

// FileInfoByPath 通过文件路径获取网盘文件信息
func (p *PanClientProxy) FileInfoByPath(pathStr string) (fileInfo *aliyunpan.FileEntity, error *apierror.ApiError) {
	return p.cacheFilePath(pathStr)
}

// FileListGetAll 获取文件路径下的所有子文件列表
func (p *PanClientProxy) FileListGetAll(pathStr string) (aliyunpan.FileList, *apierror.ApiError)  {
	return p.cacheFilesDirectoriesList(pathStr)
}

func (p *PanClientProxy) mkdir(pathStr string, perm os.FileMode) (*aliyunpan.MkdirResult, error) {
	pathStr = formatPathStyle(pathStr)
	r,er := p.PanUser.PanClient().MkdirByFullPath(p.PanDriveId, pathStr)
	if er != nil {
		return nil, er
	}

	// invalidate cache
	p.deleteOneFilesDirectoriesListCache(path.Dir(pathStr))

	if r.FileId != "" {
		fe,_ := p.PanUser.PanClient().FileInfoById(p.PanDriveId, r.FileId)
		if fe != nil {
			fe.Path = pathStr
			p.cacheFilePathEntity(fe)
		}
		return r, nil
	}
	return nil, fmt.Errorf("unknown error")
}

// Mkdir 创建目录
func (p *PanClientProxy) Mkdir(pathStr string, perm os.FileMode) error {
	if pathStr == "" {
		return fmt.Errorf("unknown error")
	}
	pathStr = formatPathStyle(pathStr)
	_, er := p.mkdir(pathStr, perm)
	return er
}

// Rename 重命名文件
func (p *PanClientProxy) Rename(oldpath, newpath string) error {
	oldpath = formatPathStyle(oldpath)
	newpath = formatPathStyle(newpath)

	oldFile, er := p.cacheFilePath(oldpath)
	if er != nil {
		return os.ErrNotExist
	}
	_,e := p.PanUser.PanClient().FileRename(p.PanDriveId, oldFile.FileId, path.Base(newpath))
	if e != nil {
		return os.ErrInvalid
	}

	// invalidate parent folder cache
	p.deleteOneFilesDirectoriesListCache(path.Dir(oldpath))

	// add new name cache
	oldFile.Path = newpath
	oldFile.FileName = path.Base(newpath)
	p.cacheFilePathEntity(oldFile)

	return nil
}

// Move 移动文件
func (p *PanClientProxy) Move(oldpath, newpath string) error {
	oldpath = formatPathStyle(oldpath)
	newpath = formatPathStyle(newpath)

	oldFile, er := p.cacheFilePath(oldpath)
	if er != nil {
		return os.ErrNotExist
	}

	newFileParentDir,er := p.cacheFilePath(path.Dir(newpath))
	if er != nil {
		return os.ErrNotExist
	}

	param := aliyunpan.FileMoveParam{
		DriveId:        p.PanDriveId,
		FileId:         oldFile.FileId,
		ToDriveId:      p.PanDriveId,
		ToParentFileId: newFileParentDir.FileId,
	}
	params := []*aliyunpan.FileMoveParam{}
	params = append(params, &param)
	_,e := p.PanUser.PanClient().FileMove(params)
	if e != nil {
		return os.ErrInvalid
	}

	// invalidate parent folder cache
	p.deleteOneFilesDirectoriesListCache(path.Dir(oldpath))
	p.deleteOneFilesDirectoriesListCache(path.Dir(newpath))

	return nil
}

// DownloadFilePart 下载文件指定数据片段
func (p *PanClientProxy) DownloadFilePart(sessionId, fileId string, offset int64, buffer []byte) (int, error) {
	fds, err1 := p.cacheFileDownloadStream(sessionId, fileId, offset)
	if err1 != nil {
		return 0, err1
	}

	if fds.readOffset != offset {
		// delete old one
		if fds.resp != nil {
			fds.resp.Body.Close()
		}
		p.deleteOneFileDownloadStreamCache(sessionId, fileId)
		logger.Verboseln(sessionId + " offset mismatch offset = " + strconv.Itoa(int(offset)) + " cache offset = " + strconv.Itoa(int(fds.readOffset)))

		// create new one
		fds, err1 = p.cacheFileDownloadStream(sessionId, fileId, offset)
		if err1 != nil {
			return 0, err1
		}
	}

	if fds.resp.Close {
		// delete old one
		p.deleteOneFileDownloadStreamCache(sessionId, fileId)
		logger.Verboseln(sessionId + "remote data stream close, stream offset = " + strconv.Itoa(int(fds.readOffset)))

		// create new one
		fds, err1 = p.cacheFileDownloadStream(sessionId, fileId, offset)
		if err1 != nil {
			return 0, err1
		}
	}

	readByteCount, readErr := fds.resp.Body.Read(buffer)
	if readErr != nil {
		if readErr.Error() == "EOF" {
			logger.Verboseln(sessionId + " read EOF last offset = " + strconv.Itoa(int(offset)))
			// end of file
			if fds.resp != nil {
				fds.resp.Body.Close()
			}
			p.deleteOneFileDownloadStreamCache(sessionId, fileId)
		} else {
			// TODO: handler other error
			return 0, readErr
		}
	}
	fds.readOffset += int64(readByteCount)
	return readByteCount, nil
}

// RemoveAll 删除文件
func (p *PanClientProxy) RemoveAll(pathStr string) error {
	fi,er := p.FileInfoByPath(pathStr)
	if er != nil {
		return er
	}
	if fi == nil {
		return nil
	}

	param := &aliyunpan.FileBatchActionParam{
		DriveId: p.PanDriveId,
		FileId:  fi.FileId,
	}
	_, e := p.PanUser.PanClient().FileDelete(append([]*aliyunpan.FileBatchActionParam{}, param))
	if e != nil {
		return e
	}

	// delete cache
	p.deleteOneFilesDirectoriesListCache(path.Dir(pathStr))

	return nil
}

// UploadFilePrepare 创建文件上传
func (p *PanClientProxy) UploadFilePrepare(userId, pathStr string, fileSize int64, chunkSize int64) (*FileUploadStream, error) {
	p.mutex.Lock()
	defer p.mutex.Unlock()

	cs := chunkSize
	if cs == 0 {
		cs = DefaultChunkSize
	}

	// remove old file cache
	oldFus,err := p.UploadFileCache(userId, pathStr)
	if err != nil {
		logger.Verboseln("query upload file cache error: ", err)
	}
	if oldFus != nil {
		// remove old upload stream cache
		oldFus.mutex.Lock()
		p.deleteOneFileUploadStreamCache(userId, pathStr)
		oldFus.mutex.Unlock()
	}

	// create new one
	fus, er := p.cacheFileUploadStream(userId, pathStr, fileSize, cs)
	if er != nil {
		return nil, er
	}
	return fus, nil
}

func (p *PanClientProxy) UploadFileCache(userId, pathStr string) (*FileUploadStream, error) {
	key := userId + "-" + formatPathStyle(pathStr)
	cache := p.filePathUploadStreamCacheMap.LazyInitCachePoolOp(p.PanDriveId)
	v, ok := cache.Load(key)
	if ok {
		return v.Data().(*FileUploadStream), nil
	}
	return nil, fmt.Errorf("upload file not found")
}

func (p *PanClientProxy) needToUploadChunk(fus *FileUploadStream) bool {
	if fus.chunkPos == fus.chunkSize {
		return true
	}

	// maybe the final part
	if fus.fileUploadUrlIndex == (len(fus.createFileUploadResult.PartInfoList)-1) {
		finalPartSize := fus.fileSize % fus.chunkSize
		if finalPartSize == 0 {
			finalPartSize = fus.chunkSize
		}
		if fus.chunkPos == finalPartSize {
			return true
		}
	}
	return false
}

// UploadFilePart 上传文件数据块
func (p *PanClientProxy) UploadFilePart(userId, pathStr string, offset int64, buffer []byte) (int, error) {
	fus, err := p.UploadFileCache(userId, pathStr)
	if err != nil {
		return 0, err
	}
	fus.mutex.Lock()
	defer fus.mutex.Unlock()

	if fus.fileWritePos != offset {
		// error
		return 0, fmt.Errorf("file write offset position mismatch")
	}

	// write buffer to chunk buffer
	uploadCount := 0
	for _,b := range buffer {
		fus.chunkBuffer[fus.chunkPos] = b
		fus.chunkPos += 1
		fus.fileWritePos += 1
		uploadCount += 1

		if p.needToUploadChunk(fus) {
			// upload chunk to drive
			uploadBuffer := fus.chunkBuffer
			if fus.chunkPos < fus.chunkSize {
				uploadBuffer = make([]byte, fus.chunkPos)
				copy(uploadBuffer, fus.chunkBuffer)
			}
			uploadChunk := bytes.NewReader(uploadBuffer)
			if fus.fileUploadUrlIndex >= len(fus.createFileUploadResult.PartInfoList) {
				return uploadCount, fmt.Errorf("upload file uploading status mismatch")
			}
			uploadPartInfo := fus.createFileUploadResult.PartInfoList[fus.fileUploadUrlIndex]
			cd := &aliyunpan.FileUploadChunkData{
				Reader: uploadChunk,
				ChunkSize: uploadChunk.Size(),
			}
			e := p.PanUser.PanClient().UploadDataChunk(p.getFileUploadUrl(uploadPartInfo), cd)
			if e != nil {
				// upload error
				// TODO: handle error, retry upload
				return uploadCount, nil
			}
			fus.fileUploadUrlIndex += 1

			// reset chunk buffer
			fus.chunkPos = 0
		}
	}

	// check file upload completely or not
	if fus.fileSize == fus.fileWritePos {
		// complete file upload
		cufr,err := p.PanUser.PanClient().CompleteUploadFile(&aliyunpan.CompleteUploadFileParam{
			DriveId: p.PanDriveId,
			FileId: fus.fileId,
			UploadId: fus.createFileUploadResult.UploadId,
		})
		logger.Verbosef("%s complete upload file: %+v\n", userId, cufr)

		if err != nil {
			logger.Verbosef("%s complete upload file error: %s\n", userId, err)
			return 0, err
		}

		// remove cache
		p.deleteOneFileUploadStreamCache(userId, pathStr)
		p.deleteOneFilePathCache(pathStr)
		p.deleteOneFilesDirectoriesListCache(path.Dir(pathStr))
	}

	return uploadCount, nil
}