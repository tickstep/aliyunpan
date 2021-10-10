package cmder

import (
	"fmt"
	"github.com/tickstep/aliyunpan-api/aliyunpan"
	"github.com/tickstep/aliyunpan-api/aliyunpan/apierror"
	"github.com/tickstep/aliyunpan/cmder/cmdliner"
	"github.com/tickstep/aliyunpan/internal/config"
	"github.com/tickstep/library-go/logger"
	"github.com/urfave/cli"
	"sync"
)

var (
	appInstance *cli.App

	saveConfigMutex *sync.Mutex = new(sync.Mutex)

	ReloadConfigFunc = func(c *cli.Context) error {
		err := config.Config.Reload()
		if err != nil {
			fmt.Printf("重载配置错误: %s\n", err)
		}
		return nil
	}

	SaveConfigFunc = func(c *cli.Context) error {
		saveConfigMutex.Lock()
		defer saveConfigMutex.Unlock()
		err := config.Config.Save()
		if err != nil {
			fmt.Printf("保存配置错误: %s\n", err)
		}
		return nil
	}
)

func SetApp(app *cli.App) {
	appInstance = app
}

func App() *cli.App {
	return appInstance
}

func DoLoginHelper(refreshToken string) (refreshTokenStr string, webToken aliyunpan.WebLoginToken, error error) {
	line := cmdliner.NewLiner()
	defer line.Close()

	if refreshToken == "" {
		refreshToken, error = line.State.Prompt("请输入RefreshToken, 回车键提交 > ")
		if error != nil {
			return
		}
	}

	// app login
	atoken, apperr := aliyunpan.GetAccessTokenFromRefreshToken(refreshToken)
	if apperr != nil {
		if apperr.Code == apierror.ApiCodeTokenExpiredCode || apperr.Code == apierror.ApiCodeRefreshTokenExpiredCode {
			fmt.Println("Token过期，需要重新登录")
		} else {
			fmt.Println("Token登录失败：", apperr)
		}
		return "", webToken, fmt.Errorf("登录失败")
	}
	refreshTokenStr = refreshToken
	return refreshTokenStr, *atoken, nil
}

func TryLogin() *config.PanUser {
	// can do automatically login?
	for _, u := range config.Config.UserList {
		if u.UserId == config.Config.ActiveUID {
			// login
			_, webToken, err := DoLoginHelper(u.RefreshToken)
			if err != nil {
				logger.Verboseln("automatically login error")
				break
			}
			// success
			u.WebToken = webToken

			// save
			SaveConfigFunc(nil)
			// reload
			ReloadConfigFunc(nil)
			return config.Config.ActiveUser()
		}
	}
	return nil
}