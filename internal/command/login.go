// Copyright (c) 2020 tickstep.
//
// Licensed under the Apache License, Version 2.0 (the "License");
// you may not use this file except in compliance with the License.
// You may obtain a copy of the License at
//
//	http://www.apache.org/licenses/LICENSE-2.0
//
// Unless required by applicable law or agreed to in writing, software
// distributed under the License is distributed on an "AS IS" BASIS,
// WITHOUT WARRANTIES OR CONDITIONS OF ANY KIND, either express or implied.
// See the License for the specific language governing permissions and
// limitations under the License.
package command

import (
	"fmt"
	"github.com/tickstep/aliyunpan/cmder/cmdliner"
	"github.com/tickstep/aliyunpan/internal/config"
	"github.com/tickstep/aliyunpan/internal/functions/panlogin"
	"github.com/tickstep/aliyunpan/internal/global"
	_ "github.com/tickstep/library-go/requester"
	"github.com/urfave/cli"
	"strings"
)

func CmdLogin() cli.Command {
	return cli.Command{
		Name:  "login",
		Usage: "登录阿里云盘账号",
		Description: `
	示例:
		1.常规登录，按提示一步一步来即可
		aliyunpan login

`,
		Category: "阿里云盘账号",
		Before:   ReloadConfigFunc, // 每次进行登录动作的时候需要调用刷新配置
		After:    SaveConfigFunc,   // 登录完成需要调用保存配置
		Action: func(c *cli.Context) error {
			ticketId := ""
			openToken := &config.PanClientToken{}
			webToken := &config.PanClientToken{}
			var err error
			ticketId, openToken, webToken, err = RunLogin()
			if err != nil {
				fmt.Println(err)
				return err
			}

			cloudUser, err := config.SetupUserByCookie(openToken, webToken,
				ticketId, "",
				config.Config.DeviceId, config.Config.DeviceName,
				config.Config.ClientId, config.Config.ClientSecret)
			if cloudUser == nil {
				fmt.Println("登录失败: ", err)
				return nil
			}
			cloudUser.TicketId = ticketId
			config.Config.SetActiveUser(cloudUser)
			fmt.Println("阿里云盘登录成功: ", cloudUser.Nickname)
			return nil
		},
		// 命令的附加options参数说明，使用 help panlogin 命令即可查看
		Flags: []cli.Flag{},
	}
}

func CmdLogout() cli.Command {
	return cli.Command{
		Name:        "logout",
		Usage:       "退出阿里帐号",
		Description: "退出当前登录的帐号",
		Category:    "阿里云盘账号",
		Before:      ReloadConfigFunc,
		After:       SaveConfigFunc,
		Action: func(c *cli.Context) error {
			if config.Config.NumLogins() == 0 {
				fmt.Println("未设置任何帐号, 不能退出")
				return nil
			}

			var (
				confirm    string
				activeUser = config.Config.ActiveUser()
			)
			if activeUser == nil {
				return nil
			}

			if !c.Bool("y") {
				fmt.Printf("确认退出当前帐号: %s ? (y/n) > ", activeUser.Nickname)
				_, err := fmt.Scanln(&confirm)
				if err != nil || (confirm != "y" && confirm != "Y") {
					return err
				}
			}

			// 云端注销登录
			//if _, e := activeUser.PanClient().WebapiPanClient().DeviceLogout(); e != nil {
			//	fmt.Printf("登出设备失败，请手动在网页端进行登出: %s\n", e.String())
			//}

			// 删除用户信息
			deletedUser, err := config.Config.DeleteUser(activeUser.UserId)
			if err != nil {
				fmt.Printf("退出用户 %s, 失败, 错误: %s\n", activeUser.Nickname, err)
			}

			fmt.Printf("退出用户成功: %s\n", deletedUser.Nickname)
			return nil
		},
		Flags: []cli.Flag{
			cli.BoolFlag{
				Name:  "y",
				Usage: "确认退出帐号",
			},
		},
	}
}

func RunLogin() (ticketId string, openapiToken, webapiToken *config.PanClientToken, error error) {
	h := panlogin.NewLoginHelper(config.DefaultTokenServiceWebHost)

	// web login request
	qrCodeUrlResult, err := h.GetQRCodeLoginUrl("")
	if err != nil {
		fmt.Println("登录出错：", err)
		return "", nil, nil, err
	}
	ticketId = qrCodeUrlResult.TokenId
	loginUrl := &strings.Builder{}
	if global.IsSupportNoneOpenApiCommands {
		// 兼容以前的版本
		fmt.Fprintf(loginUrl, "https://openapi.alipan.com/oauth/authorize?client_id=%s&redirect_uri=https%%3A%%2F%%2Fapi.tickstep.com%%2Fauth%%2Ftickstep%%2Faliyunpan%%2Ftoken%%2Fopenapi%%2F%s%%2Fauth&scope=user:base,file:all:read,file:all:write,file:share:write,album:shared:read",
			config.Config.ClientId, ticketId)
	} else {
		fmt.Fprintf(loginUrl, "https://openapi.alipan.com/oauth/authorize?client_id=%s&redirect_uri=https%%3A%%2F%%2Fapi.tickstep.com%%2Fauth%%2Ftickstep%%2Faliyunpan%%2Ftoken%%2Fopenapi%%2F%s%%2Fauth2&scope=user:base,file:all:read,file:all:write,file:share:write,album:shared:read",
			config.Config.ClientId, ticketId)
	}
	fmt.Printf("请在浏览器打开以下链接进行登录，链接有效时间为5分钟。\n注意：你需要进行一次授权一次扫码的两次登录。\n%s\n\n", loginUrl)

	// handler waiting
	line := cmdliner.NewLiner()
	defer line.Close()
	line.State.Prompt("请在浏览器里面完成扫码登录，然后再按Enter键继续...")

	// get login token
	comToken, er := h.GetLoginToken(ticketId)
	if er != nil {
		return ticketId, nil, nil, fmt.Errorf("登录失败，请稍后尝试重新登录")
	}

	return ticketId,
		&config.PanClientToken{
			AccessToken: comToken.Openapi.AccessToken,
			Expired:     comToken.Openapi.Expired,
		},
		&config.PanClientToken{
			AccessToken: comToken.Webapi.AccessToken,
			Expired:     comToken.Webapi.Expired,
		},
		nil
}
