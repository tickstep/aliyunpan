package syncdrive

import (
	"io/fs"
	"os"
	"path"
	"strings"
)

// GetPanFileFullPathFromLocalPath 获取网盘文件的路径
func GetPanFileFullPathFromLocalPath(localFilePath, localRootPath, panRootPath string) string {
	localFilePath = strings.ReplaceAll(localFilePath, "\\", "/")
	localRootPath = strings.ReplaceAll(localRootPath, "\\", "/")

	relativePath := strings.TrimPrefix(localFilePath, localRootPath)
	panPath := path.Join(path.Clean(panRootPath), relativePath)
	return strings.ReplaceAll(panPath, "\\", "/")
}

// GetLocalFileFullPathFromPanPath 获取本地文件的路径
func GetLocalFileFullPathFromPanPath(panFilePath, localRootPath, panRootPath string) string {
	panFilePath = strings.ReplaceAll(panFilePath, "\\", "/")
	panRootPath = strings.ReplaceAll(panRootPath, "\\", "/")

	relativePath := strings.TrimPrefix(panFilePath, panRootPath)
	return path.Join(path.Clean(localRootPath), relativePath)
}

// IsSymlinkFile 是否是软链接文件
func IsSymlinkFile(file fs.FileInfo) bool {
	if file.Mode()&os.ModeSymlink != 0 {
		return true
	}
	return false
}
