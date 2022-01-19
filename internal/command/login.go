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
	"encoding/json"
	"fmt"
	"github.com/tickstep/aliyunpan-api/aliyunpan"
	"github.com/tickstep/aliyunpan-api/aliyunpan/apierror"
	"github.com/tickstep/aliyunpan/cmder"
	"github.com/tickstep/aliyunpan/internal/config"
	"github.com/tickstep/library-go/logger"
	"github.com/tickstep/library-go/requester"
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

		3.指定自行搭建的web服务，从指定的URL获取Token进行登录
		aliyunpan login --RefreshTokenUrl "http://your.host.com/aliyunpan/token/refresh"

		URL获取的响应体必须是JSON，格式要求如下所示，data内容即为token：
		{
		    "code": "0",
		    "msg": "ok",
		    "data": "88771cd41a111521b4471a552bf633ba"
		}
`,
		Category: "阿里云盘账号",
		Before:   cmder.ReloadConfigFunc, // 每次进行登录动作的时候需要调用刷新配置
		After:    cmder.SaveConfigFunc, // 登录完成需要调用保存配置
		Action: func(c *cli.Context) error {
			// 优先从web服务获取token
			refreshTokenStr := ""
			if c.String("RefreshTokenUrl") != "" {
				refreshTokenUrl := c.String("RefreshTokenUrl")
				ts, e := getRefreshTokenFromWebServer(refreshTokenUrl)
				if e != nil {
					fmt.Println("从web服务获取Token失败")
				} else if ts != "" {
					fmt.Println("成功从web服务获取Token：" + ts)
					refreshTokenStr = ts
				}
			}

			if refreshTokenStr == "" {
				refreshTokenStr = c.String("RefreshToken")
			}

			webToken := aliyunpan.WebLoginToken{}
			refreshToken := ""
			var err error
			refreshToken, webToken, err = RunLogin(refreshTokenStr)
			if err != nil {
				fmt.Println(err)
				return err
			}

			cloudUser, err := config.SetupUserByCookie(&webToken)
			if cloudUser == nil {
				fmt.Println("登录失败: ", err)
				return nil
			}
			cloudUser.RefreshToken = refreshToken
			config.Config.SetActiveUser(cloudUser)
			fmt.Println("阿里云盘登录成功: ", cloudUser.Nickname)
			return nil
		},
		// 命令的附加options参数说明，使用 help login 命令即可查看
		Flags: []cli.Flag{
			// aliyunpan login -RefreshToken=8B12CBBCE89CA8DFC3445985B63B511B5E7EC7...
			cli.StringFlag{
				Name:  "RefreshToken",
				Usage: "使用RefreshToken Cookie来登录帐号",
			},
			cli.StringFlag{
				Name:  "RefreshTokenUrl",
				Usage: "使用自行搭建的web服务获取RefreshToken来进行登录",
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
		Before:      cmder.ReloadConfigFunc,
		After:       cmder.SaveConfigFunc,
		Action: func(c *cli.Context) error {
			if config.Config.NumLogins() == 0 {
				fmt.Println("未设置任何帐号, 不能退出")
				return nil
			}

			var (
				confirm    string
				activeUser = config.Config.ActiveUser()
			)

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

func RunLogin(refreshToken string) (refreshTokenStr string, webToken aliyunpan.WebLoginToken, error error) {
	return cmder.DoLoginHelper(refreshToken)
}


// getRefreshTokenFromWebServer 从自定义的web服务获取token
func getRefreshTokenFromWebServer(url string) (string, error) {
	type tokenResult struct {
		Code string `json:"code"`
		Msg string `json:"msg"`
		Data string `json:"data"`
	}

	if url == "" {
		return "", fmt.Errorf("url is empty")
	}

	logger.Verboseln("do request url: " + url)
	header := map[string]string {
		"accept": "application/json, text/plain, */*",
		"content-type": "application/json;charset=UTF-8",
		"user-agent": "aliyunpan/" + config.AppVersion,
	}
	// request
	client := requester.NewHTTPClient()
	client.SetTimeout(10 * time.Second)
	client.SetKeepAlive(false)
	body, err := client.Fetch("GET", url, nil, header)
	if err != nil {
		logger.Verboseln("get token error ", err)
		return "", err
	}

	// parse result
	r := &tokenResult{}
	if err2 := json.Unmarshal(body, r); err2 != nil {
		logger.Verboseln("parse token info result json error ", err2)
		return "", apierror.NewFailedApiError(err2.Error())
	}
	return r.Data, nil
}
