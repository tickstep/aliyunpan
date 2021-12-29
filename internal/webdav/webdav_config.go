package webdav

import (
	"github.com/tickstep/library-go/logger"
	"golang.org/x/net/webdav"
	"log"
	"net"
	"net/http"
	"strconv"
)

type WebdavUser struct {
	Username string `json:"username"`
	Password string `json:"password"`
	Scope    string `json:"scope"`
}

type WebdavConfig struct {
	// 指定Webdav使用哪个账号的云盘资源
	PanUserId string `json:"panUserId"`

	Address string `json:"address"`
	Port       int `json:"port"`
	Prefix       string  `json:"prefix"`
	Users []WebdavUser `json:"users"`
}

func (w *WebdavConfig) StartServer() {
	users := map[string]*User{}
	for _,u := range w.Users {
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

