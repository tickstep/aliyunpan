package command_local

import (
	"github.com/tickstep/aliyunpan/internal/config"
	"os"
	"path"
)

// getLocalHomeDir 获取本地用户主目录
func getLocalHomeDir() string {
	// 默认为用户主页目录
	if hd, e := os.UserHomeDir(); e == nil {
		return hd
	}
	return ""
}

// localPathJoin 拼接本地路径
func localPathJoin(p string) string {
	if path.IsAbs(p) {
		return p
	}
	wd := config.Config.LocalWorkdir
	if wd == "" {
		wd = getLocalHomeDir()
	}
	return path.Join(wd, p)
}
