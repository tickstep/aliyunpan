package syncdrive

import (
	"path"
	"strings"
)

// GetPanFileFullPathFromLocalPath 获取网盘文件的路径
func GetPanFileFullPathFromLocalPath(localFilePath, localRootPath, panRootPath string) string {
	localFilePath = strings.ReplaceAll(localFilePath, "\\", "/")
	localRootPath = strings.ReplaceAll(localRootPath, "\\", "/")

	relativePath := strings.TrimPrefix(localFilePath, localRootPath)
	return path.Join(path.Clean(panRootPath), relativePath)
}

// GetLocalFileFullPathFromPanPath 获取本地文件的路径
func GetLocalFileFullPathFromPanPath(panFilePath, localRootPath, panRootPath string) string {
	panFilePath = strings.ReplaceAll(panFilePath, "\\", "/")
	panRootPath = strings.ReplaceAll(panRootPath, "\\", "/")

	relativePath := strings.TrimPrefix(panFilePath, panRootPath)
	return path.Join(path.Clean(localRootPath), relativePath)
}
