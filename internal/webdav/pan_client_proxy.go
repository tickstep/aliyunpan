package webdav

import (
	"fmt"
	"github.com/tickstep/aliyunpan-api/aliyunpan"
	"github.com/tickstep/aliyunpan-api/aliyunpan/apierror"
	"github.com/tickstep/aliyunpan/internal/config"
	"github.com/tickstep/library-go/expires"
	"github.com/tickstep/library-go/expires/cachemap"
	"os"
	"path"
	"strings"
	"time"
)

type PanClientProxy struct {
	PanUser *config.PanUser
	PanDriveId string

	filePathCacheMap          cachemap.CacheOpMap
	fileDirectoryListCacheMap cachemap.CacheOpMap
}

// CACHE_EXPIRED_MINUTE  缓存过期分钟
const CACHE_EXPIRED_MINUTE = 60

// DeleteCache 删除含有 dirs 的缓存
func (p *PanClientProxy) deleteFilesDirectoriesListCache(dirs []string) {
	cache := p.fileDirectoryListCacheMap.LazyInitCachePoolOp(p.PanDriveId)
	for _, v := range dirs {
		key := strings.TrimSuffix(v, "/")
		_, ok := cache.Load(key)
		if ok {
			cache.Delete(key)
		}
	}
}

// DeleteOneCache 删除缓存
func (p *PanClientProxy) deleteOneFilesDirectoriesListCache(dirPath string) {
	dirPath = strings.TrimSuffix(dirPath, "/")
	ps := []string{dirPath}
	p.deleteFilesDirectoriesListCache(ps)
}

// cacheFilesDirectoriesList 缓存文件夹下面的所有文件列表
func (p *PanClientProxy) cacheFilesDirectoriesList(pathStr string) (fdl aliyunpan.FileList, apiError *apierror.ApiError) {
	pathStr = strings.TrimSuffix(pathStr, "/")
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

// cacheFilePath 缓存文件绝对路径到网盘文件信息
func (p *PanClientProxy) cacheFilePath(pathStr string) (fe *aliyunpan.FileEntity, apiError *apierror.ApiError) {
	pathStr = strings.TrimSuffix(pathStr, "/")
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
	pathStr := strings.TrimSuffix(fe.Path, "/")
	p.filePathCacheMap.CacheOperation(p.PanDriveId, pathStr, func() expires.DataExpires {
		return expires.NewDataExpires(fe, CACHE_EXPIRED_MINUTE*time.Minute)
	})
}

func (p *PanClientProxy) cacheFilePathEntityList(fdl aliyunpan.FileList) {
	for _,entity := range fdl {
		pathStr := strings.TrimSuffix(entity.Path, "/")
		p.filePathCacheMap.CacheOperation(p.PanDriveId, pathStr, func() expires.DataExpires {
			return expires.NewDataExpires(entity, CACHE_EXPIRED_MINUTE*time.Minute)
		})
	}
}


func (p *PanClientProxy) FileInfoByPath(pathStr string) (fileInfo *aliyunpan.FileEntity, error *apierror.ApiError) {
	return p.cacheFilePath(pathStr)
}

func (p *PanClientProxy) FileListGetAll(pathStr string) (aliyunpan.FileList, *apierror.ApiError)  {
	return p.cacheFilesDirectoriesList(pathStr)
}

func (p *PanClientProxy) Mkdir(pathStr string, perm os.FileMode) error {
	if pathStr == "" {
		return fmt.Errorf("unknown error")
	}
	pathStr = strings.ReplaceAll(pathStr, "\\", "/")
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

func (p *PanClientProxy) Rename(oldpath, newpath string) error {
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