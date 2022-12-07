// Copyright (c) 2020 tickstep.
//
// Licensed under the Apache License, Version 2.0 (the "License");
// you may not use this file except in compliance with the License.
// You may obtain a copy of the License at
//
//     http://www.apache.org/licenses/LICENSE-2.0
//
// Unless required by applicable law or agreed to in writing, software
// distributed under the License is distributed on an "AS IS" BASIS,
// WITHOUT WARRANTIES OR CONDITIONS OF ANY KIND, either express or implied.
// See the License for the specific language governing permissions and
// limitations under the License.
package command

import (
	"fmt"
	"github.com/tickstep/aliyunpan-api/aliyunpan"
	"github.com/tickstep/aliyunpan/cmder"
	"github.com/tickstep/aliyunpan/internal/config"
	"github.com/tickstep/aliyunpan/internal/plugins"
	"github.com/tickstep/aliyunpan/internal/utils"
	"github.com/tickstep/library-go/logger"
	"github.com/urfave/cli"
)

func CmdToken() cli.Command {
	return cli.Command{
		Name:      "token",
		Usage:     "Token相关操作",
		UsageText: cmder.App().Name + " token",
		Category:  "阿里云盘账号",
		Before:    cmder.ReloadConfigFunc,
		Action: func(c *cli.Context) error {
			cli.ShowCommandHelp(c, c.Command.Name)
			return nil
		},
		Subcommands: []cli.Command{
			{
				Name:      "update",
				Usage:     "更新Token",
				UsageText: cmder.App().Name + " token update",
				Description: `
示例:

    更新当前登录用户的Token
	aliyunpan token update -mode 1

    更新全部登录用户的Token
	aliyunpan token update -mode 2
`,
				Action: func(c *cli.Context) error {
					if config.Config.ActiveUser() == nil {
						fmt.Println("未登录账号，无需刷新Token")
						return nil
					}

					modeFlag := "1"
					if c.IsSet("mode") {
						modeFlag = c.String("mode")
					}
					RunTokenUpdate(modeFlag)
					return nil
				},
				Flags: []cli.Flag{
					cli.StringFlag{
						Name:  "mode",
						Usage: "模式，1-登录用户，2-全部用户",
						Value: "1",
					},
				},
			},
		},
	}
}

// RunTokenUpdate 执行Token更新
func RunTokenUpdate(modeFlag string) {
	cmder.ReloadConfigFunc(nil)

	// 获取当前插件
	pluginManger := plugins.NewPluginManager(config.GetPluginDir())
	plugin, _ := pluginManger.GetPlugin()
	params := &plugins.UserTokenRefreshFinishParams{
		Result:    "success",
		Message:   "",
		OldToken:  "",
		NewToken:  "",
		UpdatedAt: utils.NowTimeStr(),
	}

	userList := config.Config.UserList
	if userList == nil || len(userList) == 0 {
		fmt.Printf("没有登录用户，无需刷新Token\n")
		return
	}
	for _, user := range userList {
		params.Result = "success"

		if modeFlag == "1" {
			if user.UserId != config.Config.ActiveUID {
				continue
			}
		}
		newToken, e := aliyunpan.GetAccessTokenFromRefreshToken(user.RefreshToken)
		if e != nil {
			params.Result = "fail"
			params.Message = e.Error()
			fmt.Printf("无法为%s用户获取新的RefreshToken，可能需要重新登录\n", user.Nickname)
			continue
		}
		if newToken != nil && newToken.RefreshToken != "" {
			params.OldToken = user.RefreshToken
			params.NewToken = newToken.RefreshToken

			user.RefreshToken = newToken.RefreshToken
			fmt.Printf("成功刷新%s用户的RefreshToken\n", user.Nickname)
		} else {
			params.Result = "fail"
		}

		// plugin callback
		if er := plugin.UserTokenRefreshFinishCallback(plugins.GetContext(user), params); er != nil {
			logger.Verbosef("UserTokenRefreshFinishCallback error: " + er.Error())
		}
	}
	cmder.SaveConfigFunc(nil)
}
