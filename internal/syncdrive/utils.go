package syncdrive

import (
	"fmt"
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
	localRoot := path.Clean(localRootPath)
	resolved := path.Join(localRoot, relativePath)
	if resolved != localRoot && !strings.HasPrefix(resolved, localRoot+"/") {
		return localRoot + "/.sync_rejected_traversal"
	}
	return resolved
}

// IsSymlinkFile 是否是软链接文件
func IsSymlinkFile(file fs.FileInfo) bool {
	if file.Mode()&os.ModeSymlink != 0 {
		return true
	}
	return false
}

// PromptPrintln 输出提示消息到控制台
func PromptPrintln(msg string) {
	if LogPrompt {
		//fmt.Println("[" + utils.NowTimeStr() + "] " + msg)
		fmt.Println(msg)
	}
}

func PromptPrint(msg string) {
	if LogPrompt {
		fmt.Print(msg)
	}
}
