package command_local

import (
	"github.com/tickstep/aliyunpan/internal/config"
	"github.com/tickstep/aliyunpan/library/homedir"
	"os"
	"path"
	"strings"
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
	p = strings.ReplaceAll(p, "\\", "/")
	if path.IsAbs(p) {
		return p
	} else if strings.HasPrefix(p, "~") {
		if d, e := homedir.Expand(p); e == nil {
			return d
		}
	}
	wd := config.Config.LocalWorkdir
	if wd == "" {
		wd = getLocalHomeDir()
	}
	return path.Join(wd, p)
}
