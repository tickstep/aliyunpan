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
	"github.com/tickstep/aliyunpan/cmder"
	"github.com/tickstep/aliyunpan/internal/config"
	"github.com/urfave/cli"
	"os"
	"strconv"
)

func CmdLoglist() cli.Command {
	return cli.Command{
		Name:        "loglist",
		Usage:       "列出帐号列表",
		Description: "列出所有已登录的阿里账号",
		Category:    "阿里云盘账号",
		Before:      ReloadConfigFunc,
		Action: func(c *cli.Context) error {
			fmt.Println(config.Config.UserList.String())
			return nil
		},
	}
}

func CmdSu() cli.Command {
	return cli.Command{
		Name:  "su",
		Usage: "切换阿里账号",
		Description: `
	切换已登录的阿里账号:
	如果运行该条命令没有提供参数, 程序将会列出所有的帐号, 供选择切换.

	示例:
	aliyunpan su
	aliyunpan su <uid or name>
`,
		Category: "阿里云盘账号",
		Before:   ReloadConfigFunc,
		After:    SaveConfigFunc,
		Action: func(c *cli.Context) error {
			if c.NArg() >= 2 {
				cli.ShowCommandHelp(c, c.Command.Name)
				return nil
			}

			numLogins := config.Config.NumLogins()

			if numLogins == 0 {
				fmt.Printf("未设置任何帐号, 不能切换\n")
				return nil
			}

			var (
				inputData = c.Args().Get(0)
				uid       string
			)

			if c.NArg() == 1 {
				// 直接切换
				uid = inputData
			} else if c.NArg() == 0 {
				// 输出所有帐号供选择切换
				cli.HandleAction(cmder.App().Command("loglist").Action, c)

				// 提示输入 index
				var index string
				fmt.Printf("输入要切换帐号的 # 值 > ")
				_, err := fmt.Scanln(&index)
				if err != nil {
					return nil
				}

				if n, err1 := strconv.Atoi(index); err1 == nil && (n-1) >= 0 && (n-1) < numLogins {
					uid = config.Config.UserList[n-1].UserId
				} else {
					fmt.Printf("切换用户失败, 请检查 # 值是否正确\n")
					return nil
				}
			} else {
				cli.ShowCommandHelp(c, c.Command.Name)
			}

			switchedUser, err := config.Config.SwitchUser(uid, inputData)
			if err != nil {
				fmt.Printf("切换用户失败, %s\n", err)
				return nil
			}

			if switchedUser == nil {
				switchedUser = TryLogin()
			}

			if switchedUser != nil {
				fmt.Printf("切换用户: %s\n", switchedUser.Nickname)
			} else {
				fmt.Printf("切换用户失败\n")
			}

			return nil
		},
	}
}

func CmdWho() cli.Command {
	return cli.Command{
		Name:        "who",
		Usage:       "获取当前帐号",
		Description: "获取当前帐号的信息",
		Category:    "阿里云盘账号",
		Before:      ReloadConfigFunc,
		Action: func(c *cli.Context) error {
			if config.Config.ActiveUser() == nil {
				fmt.Println("未登录账号")
				os.Exit(1)
			}
			activeUser := config.Config.ActiveUser()
			cloudName := activeUser.GetDriveById(activeUser.ActiveDriveId).DriveName
			fmt.Printf("当前帐号UID: %s, 昵称: %s, 用户名: %s, 网盘：%s\n", activeUser.UserId, activeUser.Nickname, activeUser.AccountName, cloudName)
			return nil
		},
	}
}
