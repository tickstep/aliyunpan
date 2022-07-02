package config

import (
	"github.com/tickstep/aliyunpan-api/aliyunpan"
	"github.com/tickstep/aliyunpan-api/aliyunpan/apierror"
	"github.com/tickstep/library-go/expires"
	"path"
	"time"
)

// DeleteCache 删除含有 dirs 的缓存
func (pu *PanUser) DeleteCache(dirs []string) {
	cache := pu.cacheOpMap.LazyInitCachePoolOp(pu.ActiveDriveId)
	for _, v := range dirs {
		key := v + "_" + "OrderByName"
		_, ok := cache.Load(key)
		if ok {
			cache.Delete(key)
		}
	}
}

// DeleteOneCache 删除缓存
func (pu *PanUser) DeleteOneCache(dirPath string) {
	ps := []string{dirPath}
	pu.DeleteCache(ps)
}

// CacheFilesDirectoriesList 缓存获取
func (pu *PanUser) CacheFilesDirectoriesList(pathStr string) (fdl aliyunpan.FileList, apiError *apierror.ApiError) {
	data := pu.cacheOpMap.CacheOperation(pu.ActiveDriveId, pathStr+"_OrderByName", func() expires.DataExpires {
		var fi *aliyunpan.FileEntity
		fi, apiError = pu.panClient.FileInfoByPath(pu.ActiveDriveId, pathStr)
		if apiError != nil {
			return nil
		}
		fileListParam := &aliyunpan.FileListParam{
			DriveId:      pu.ActiveDriveId,
			ParentFileId: fi.FileId,
		}
		fdl, apiError = pu.panClient.FileListGetAll(fileListParam, 100)
		if apiError != nil {
			return nil
		}
		// construct full path
		for _, f := range fdl {
			f.Path = path.Join(pathStr, f.FileName)
		}
		return expires.NewDataExpires(fdl, 10*time.Minute)
	})
	if apiError != nil {
		return
	}
	return data.Data().(aliyunpan.FileList), nil
}
