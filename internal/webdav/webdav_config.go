package webdav

import (
	"github.com/tickstep/aliyunpan/internal/config"
	"github.com/tickstep/library-go/logger"
	"golang.org/x/net/webdav"
	"log"
	"net"
	"net/http"
	"strconv"
	"strings"
)

type WebdavUser struct {
	Username string `json:"username"`
	Password string `json:"password"`
	Scope    string `json:"scope"`
}

type WebdavConfig struct {
	// 指定Webdav使用哪个账号的云盘资源
	PanUserId string `json:"panUserId"`
	PanDriveId string `json:"panDriveId"`
	PanUser     *config.PanUser `json:"-"`

	Address string `json:"address"`
	Port       int `json:"port"`
	Prefix       string  `json:"prefix"`
	Users []WebdavUser `json:"users"`
}

func (w *WebdavConfig) StartServer() {
	users := map[string]*User{}
	for _,u := range w.Users {
		fileItem,e := w.PanUser.PanClient().FileInfoByPath(w.PanDriveId, u.Scope)
		if e != nil {
			logger.Verboseln("scope not existed, shutting server")
			return
		}
		wdfi := NewWebDavFileInfo(fileItem)
		if wdfi.fullPath != "/" && strings.Index(wdfi.fullPath, "/") != 0 {
			wdfi.fullPath = "/" + wdfi.fullPath
		}
		users[u.Username] = &User{
			Username: u.Username,
			Password: u.Password,
			Scope:    u.Scope,
			Modify:   true,
			Rules:    nil,
			Handler:  &webdav.Handler{
				Prefix: w.Prefix,
				FileSystem: WebDavDir{
					Dir:     webdav.Dir(u.Scope),
					NoSniff: false,
					panClientProxy: &PanClientProxy{
						PanUser:    w.PanUser,
						PanDriveId: w.PanDriveId,
					},
					fileInfo: wdfi,
				},
				LockSystem: webdav.NewMemLS(),
			},
		}
	}
	cfg := &Config{
		Auth:      true,
		NoSniff:   false,
		Cors:      CorsCfg{
			Enabled:     false,
			Credentials: false,
		},
		Users:     users,
		LogFormat: "",
	}

	listener, err := net.Listen("tcp", w.Address + ":" + strconv.Itoa(w.Port))
	if err != nil {
		log.Fatal(err)
	}
	if err := http.Serve(listener, cfg); err != nil {
		logger.Verboseln("shutting server", err)
	}
}

