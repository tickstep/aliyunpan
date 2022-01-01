package webdav

import (
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
	"strconv"
	"strings"
	"time"
)

type FileDownloadStream struct {
	fileUrl string
	readOffset  int64
	resp *http.Response
	timestamp int64
}

type PanClientProxy struct {
	PanUser *config.PanUser
	PanDriveId string

	// 网盘文件路径到网盘文件信息实体映射缓存
	filePathCacheMap          cachemap.CacheOpMap

	// 网盘文件夹路径到文件夹下面所有子文件映射缓存
	fileDirectoryListCacheMap cachemap.CacheOpMap

	// 网盘文件ID到文件下载链接映射缓存
	fileIdDownloadUrlCacheMap cachemap.CacheOpMap

	// 网盘文件ID到文件下载数据流映射缓存
	fileIdDownloadStreamCacheMap cachemap.CacheOpMap
}

// CACHE_EXPIRED_MINUTE  缓存过期分钟
const CACHE_EXPIRED_MINUTE = 60

// FILE_DOWNLOAD_URL_EXPIRED_MINUTE  文件下载URL过期分钟,
const FILE_DOWNLOAD_URL_EXPIRED_SECONDS = 14400

func formatPathStyle(pathStr string) string {
	pathStr = strings.ReplaceAll(pathStr, "\\", "/")
	pathStr = strings.TrimSuffix(pathStr, "/")
	return pathStr
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
		return expires.NewDataExpires(fdl, CACHE_EXPIRED_MINUTE*time.Minute)
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
		return expires.NewDataExpires(fi, CACHE_EXPIRED_MINUTE*time.Minute)
	})
	if apiError != nil {
		return
	}
	if data == nil {
		return nil, nil
	}
	return data.Data().(*aliyunpan.FileEntity), nil
}

func (p *PanClientProxy) cacheFilePathEntity(fe *aliyunpan.FileEntity) {
	pathStr := formatPathStyle(fe.Path)
	p.filePathCacheMap.CacheOperation(p.PanDriveId, pathStr, func() expires.DataExpires {
		return expires.NewDataExpires(fe, CACHE_EXPIRED_MINUTE*time.Minute)
	})
}

func (p *PanClientProxy) cacheFilePathEntityList(fdl aliyunpan.FileList) {
	for _,entity := range fdl {
		pathStr := formatPathStyle(entity.Path)
		p.filePathCacheMap.CacheOperation(p.PanDriveId, pathStr, func() expires.DataExpires {
			return expires.NewDataExpires(entity, CACHE_EXPIRED_MINUTE*time.Minute)
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
			ExpireSec: FILE_DOWNLOAD_URL_EXPIRED_SECONDS,
		})
		if err1 != nil {
			return nil
		}
		return expires.NewDataExpires(urlResult, (FILE_DOWNLOAD_URL_EXPIRED_SECONDS-60)*time.Second)
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
			urlResult.Url,
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
		}, CACHE_EXPIRED_MINUTE*time.Minute)
	})

	if data == nil {
		return nil, nil
	}
	return data.Data().(*FileDownloadStream), nil
}

// FileInfoByPath 通过文件路径获取网盘文件信息
func (p *PanClientProxy) FileInfoByPath(pathStr string) (fileInfo *aliyunpan.FileEntity, error *apierror.ApiError) {
	return p.cacheFilePath(pathStr)
}

// FileListGetAll 获取文件路径下的所有子文件列表
func (p *PanClientProxy) FileListGetAll(pathStr string) (aliyunpan.FileList, *apierror.ApiError)  {
	return p.cacheFilesDirectoriesList(pathStr)
}

// Mkdir 创建目录
func (p *PanClientProxy) Mkdir(pathStr string, perm os.FileMode) error {
	if pathStr == "" {
		return fmt.Errorf("unknown error")
	}
	pathStr = formatPathStyle(pathStr)
	r,er := p.PanUser.PanClient().MkdirByFullPath(p.PanDriveId, pathStr)
	if er != nil {
		return er
	}
	// invalidate cache
	p.deleteOneFilesDirectoriesListCache(path.Dir(pathStr))

	if r.FileId != "" {
		fe,_ := p.PanUser.PanClient().FileInfoById(p.PanDriveId, r.FileId)
		if fe != nil {
			fe.Path = pathStr
			p.cacheFilePathEntity(fe)
		}
		return nil
	}
	return fmt.Errorf("unknown error")
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