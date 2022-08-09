package syncdrive

import (
	"path"
	"regexp"
	"strings"
)

// FormatFilePath 格式化文件路径
func FormatFilePath(filePath string) string {
	if filePath == "" {
		return ""
	}

	// 是否是windows路径
	matched, _ := regexp.MatchString("^([a-zA-Z]:)", filePath)
	if matched {
		// 去掉卷标签，例如：D:
		filePath = string([]rune(filePath)[2:])
	}
	filePath = strings.ReplaceAll(filePath, "\\", "/")
	return path.Clean(filePath)
}
