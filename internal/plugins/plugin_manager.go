package plugins

import (
	"fmt"
	"github.com/tickstep/aliyunpan/internal/config"
	"github.com/tickstep/aliyunpan/internal/global"
	"github.com/tickstep/library-go/logger"
	"io/ioutil"
	"os"
	"path"
	"path/filepath"
	"strings"
)

type (
	PluginManager struct {
		PluginPath string
	}
)

func GetContext(user *config.PanUser) *Context {
	if user == nil {
		return &Context{
			AppName:      "aliyunpan",
			Version:      global.AppVersion,
			UserId:       "",
			Nickname:     "",
			FileDriveId:  "",
			AlbumDriveId: "",
		}
	}
	return &Context{
		AppName:      "aliyunpan",
		Version:      global.AppVersion,
		UserId:       user.UserId,
		Nickname:     user.Nickname,
		FileDriveId:  user.DriveList.GetFileDriveId(),
		AlbumDriveId: user.DriveList.GetFileDriveId(),
	}
}

func NewPluginManager(pluginDir string) *PluginManager {
	return &PluginManager{
		PluginPath: pluginDir,
	}
}

func (p *PluginManager) SetPluginPath(pluginPath string) error {
	if fi, err := os.Stat(pluginPath); err == nil && fi.IsDir() {
		p.PluginPath = filepath.Clean(pluginPath)
	} else {
		return fmt.Errorf("path must be a folder")
	}
	return nil
}

func (p *PluginManager) GetPlugin() (Plugin, error) {
	// js plugins folder
	// only support js plugins right now
	jsPluginPath := path.Clean(p.PluginPath + string(os.PathSeparator) + "js")
	if fi, err := os.Stat(jsPluginPath); err == nil && fi.IsDir() {
		jsPlugin := NewJsPlugin()
		if jsPlugin.Start() != nil {
			logger.Verbosef("初始化JS脚本错误\n")
			return interface{}(NewIdlePlugin()).(Plugin), nil
		}

		jsPluginValid := false
		if files, e := ioutil.ReadDir(jsPluginPath); e == nil {
			for _, f := range files {
				if !f.IsDir() {
					if strings.HasPrefix(strings.ToLower(f.Name()), ".") || strings.HasPrefix(strings.ToLower(f.Name()), "~") {
						continue
					}
					if strings.HasSuffix(strings.ToLower(f.Name()), ".js") {
						// this is a js file
						bytes, re := ioutil.ReadFile(path.Clean(jsPluginPath + string(os.PathSeparator) + f.Name()))
						if re != nil {
							logger.Verbosef("读取JS脚本错误: %s\n", re)
							continue
						}
						var script = string(bytes)
						if jsPlugin.LoadScript(script) == nil {
							jsPluginValid = true
							logger.Verbosef("加载JS脚本成功: %s\n", f.Name())
						}
					}
				}
			}
		}
		if jsPluginValid {
			return interface{}(jsPlugin).(Plugin), nil
		}
	}

	// default idle plugins
	return interface{}(NewIdlePlugin()).(Plugin), nil
}
