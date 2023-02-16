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
	"github.com/tickstep/aliyunpan/cmder/cmdliner"
	"github.com/tickstep/aliyunpan/internal/config"
	"github.com/tickstep/aliyunpan/internal/functions/panlogin"
	"github.com/tickstep/library-go/logger"
	_ "github.com/tickstep/library-go/requester"
	"github.com/urfave/cli"
	"time"
)

func CmdLogin() cli.Command {
	return cli.Command{
		Name:  "login",
		Usage: "登录阿里云盘账号",
		Description: `
	示例:
		1.常规登录，按提示一步一步来即可
		aliyunpan login

		2.直接指定RefreshToken进行登录
		aliyunpan login -RefreshToken=8B12CBBCE89CA8DFC3445985B63B511B5E7EC7...

		3.使用二维码方式进行登录
		aliyunpan login -QrCode
`,
		Category: "阿里云盘账号",
		Before:   ReloadConfigFunc, // 每次进行登录动作的时候需要调用刷新配置
		After:    SaveConfigFunc,   // 登录完成需要调用保存配置
		Action: func(c *cli.Context) error {
			refreshTokenStr := ""
			if refreshTokenStr == "" {
				refreshTokenStr = c.String("RefreshToken")
			}
			useQrCode := c.Bool("QrCode")

			tokenId := ""
			webToken := aliyunpan.WebLoginToken{}
			refreshToken := ""
			var err error
			tokenId, refreshToken, webToken, err = RunLogin(useQrCode, refreshTokenStr)
			if err != nil {
				fmt.Println(err)
				return err
			}

			cloudUser, err := config.SetupUserByCookie(&webToken, config.Config.DeviceId, config.Config.DeviceName)
			if cloudUser == nil {
				fmt.Println("登录失败: ", err)
				return nil
			}
			cloudUser.RefreshToken = refreshToken
			cloudUser.TokenId = tokenId
			config.Config.SetActiveUser(cloudUser)
			fmt.Println("阿里云盘登录成功: ", cloudUser.Nickname)
			return nil
		},
		// 命令的附加options参数说明，使用 help panlogin 命令即可查看
		Flags: []cli.Flag{
			// aliyunpan panlogin -RefreshToken=8B12CBBCE89CA8DFC3445985B63B511B5E7EC7...
			cli.StringFlag{
				Name:  "RefreshToken",
				Usage: "使用RefreshToken Cookie来登录帐号",
			},
			cli.BoolFlag{
				Name:  "QrCode",
				Usage: "使用二维码登录",
			},
		},
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

func RunLogin(useQrCodeLogin bool, refreshToken string) (tokenId, refreshTokenStr string, webToken aliyunpan.WebLoginToken, error error) {
	if useQrCodeLogin {
		h := panlogin.NewLoginHelper(config.DefaultTokenServiceWebHost)
		qrCodeUrlResult, err := h.GetQRCodeLoginUrl("")
		if err != nil {
			fmt.Println("二维码登录错误：", err)
			return "", "", aliyunpan.WebLoginToken{}, err
		}
		fmt.Printf("请在浏览器打开以下链接进行扫码登录，链接有效时间为5分钟\n%s\n\n", qrCodeUrlResult.TokenUrl+"&deviceId="+config.Config.DeviceId)

		// handler waiting
		line := cmdliner.NewLiner()
		var qrCodeLoginResult *panlogin.QRCodeLoginResult
		queryResult := true
		defer line.Close()

		go func() {
			for queryResult {
				time.Sleep(3 * time.Second)
				qr, er := h.GetQRCodeLoginResult(qrCodeUrlResult.TokenId)
				if er != nil {
					continue
				}
				logger.Verboseln(qr)
				if qr.QrCodeStatus == "CONFIRMED" {
					// login successfully
					qrCodeLoginResult = qr
					break
				} else if qr.QrCodeStatus == "EXPIRED" {
					break
				}
			}
		}()

		line.State.Prompt("请在浏览器里面完成扫码登录，然后再按Enter键继续...")
		if qrCodeLoginResult == nil {
			queryResult = false
			return "", "", aliyunpan.WebLoginToken{}, fmt.Errorf("二维码登录失败")
		}

		tokenStr, er := h.ParseSecureRefreshToken("", qrCodeLoginResult.SecureRefreshToken)
		if er != nil {
			fmt.Println("解析Token错误：", er)
			return "", "", aliyunpan.WebLoginToken{}, er
		}
		refreshToken = tokenStr
		tokenId = qrCodeUrlResult.TokenId
	}

	refreshTokenStr, webToken, error = DoLoginHelper(refreshToken)
	return
}
