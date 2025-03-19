package command_local

import (
	"github.com/tickstep/aliyunpan/internal/config"
	"github.com/tickstep/aliyunpan/library/homedir"
	"os"
	"path"
	"path/filepath"
	"strings"
)

// GetLocalHomeDir 获取本地用户主目录
func GetLocalHomeDir() string {
	// 默认为用户主页目录
	if hd, e := os.UserHomeDir(); e == nil {
		return hd
	}
	return ""
}

// LocalPathJoin 拼接本地路径
func LocalPathJoin(p string) string {
	p = path.Clean(strings.ReplaceAll(p, "\\", "/"))
	if filepath.IsAbs(p) {
		return p
	} else if strings.HasPrefix(p, "~") {
		if d, e := homedir.Expand(p); e == nil {
			return d
		}
	}
	wd := config.Config.LocalWorkdir
	if wd == "" {
		wd = GetLocalHomeDir()
	}
	return path.Join(wd, p)
}
