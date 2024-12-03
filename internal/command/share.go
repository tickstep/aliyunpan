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
	"github.com/tickstep/aliyunpan-api/aliyunpan"
	"github.com/tickstep/aliyunpan-api/aliyunpan/apierror"
	"github.com/tickstep/aliyunpan/cmder"
	"github.com/tickstep/aliyunpan/internal/config"
	"github.com/urfave/cli"
	"path"
	"time"
)

func CmdShare() cli.Command {
	return cli.Command{
		Name:      "share",
		Usage:     "分享文件/目录",
		UsageText: cmder.App().Name + " share",
		Category:  "阿里云盘",
		Before:    ReloadConfigFunc,
		Action: func(c *cli.Context) error {
			cli.ShowCommandHelp(c, c.Command.Name)
			return nil
		},
		Subcommands: []cli.Command{
			{
				Name:      "set",
				Aliases:   []string{"s"},
				Usage:     "设置分享文件/目录",
				UsageText: cmder.App().Name + " share set <文件/目录1> <文件/目录2> ...",
				Description: `
示例:

    创建文件 1.mp4 的分享链接 
	aliyunpan share set -mode 1 1.mp4

    创建 /我的视频/ 目录下所有mp4文件的分享链接，支持通配符
	aliyunpan share set -mode 1 /我的视频/*.mp4

    创建文件 1.mp4 的分享链接，并指定分享密码为2333
	aliyunpan share set -mode 1 -sharePwd 2333 1.mp4

    创建文件 1.mp4 的分享链接，并指定有效期为1天
	aliyunpan share set -mode 1 -time 1 1.mp4

    创建文件 1.mp4 的快传链接
	aliyunpan share set 1.mp4
`,
				Action: func(c *cli.Context) error {
					if c.NArg() < 1 {
						cli.ShowCommandHelp(c, c.Command.Name)
						return nil
					}
					if config.Config.ActiveUser() == nil {
						fmt.Println("未登录账号")
						return nil
					}

					// 有效期
					et := ""
					timeFlag := "0"
					if c.IsSet("time") {
						timeFlag = c.String("time")
					}
					now := time.Now()
					if timeFlag == "1" {
						et = now.Add(time.Duration(1) * time.Hour * 24).Format("2006-01-02 15:04:05")
					} else if timeFlag == "2" {
						et = now.Add(time.Duration(7) * time.Hour * 24).Format("2006-01-02 15:04:05")
					} else {
						et = ""
					}

					// 密码
					sharePwd := ""
					if c.IsSet("sharePwd") {
						sharePwd = c.String("sharePwd")
					}
					if sharePwd == "" {
						sharePwd = RandomStr(4)
					}

					// 模式
					modeFlag := "3"
					if c.IsSet("mode") {
						modeFlag = c.String("mode")
					}
					if modeFlag == "1" || modeFlag == "2" {
						if config.Config.ActiveUser().ActiveDriveId != config.Config.ActiveUser().DriveList.GetResourceDriveId() {
							// 只有资源库才支持私有、公开分享
							fmt.Println("只有资源库才支持分享链接，其他请使用快传链接")
							return nil
						}
					}
					if modeFlag == "1" {
						if sharePwd == "" {
							sharePwd = RandomStr(4)
						}
					} else {
						sharePwd = ""
					}

					RunOpenShareSet(modeFlag, parseDriveId(c), c.Args(), et, sharePwd)
					return nil
				},
				Flags: []cli.Flag{
					cli.StringFlag{
						Name:  "driveId",
						Usage: "网盘ID",
						Value: "",
					},
					cli.StringFlag{
						Name:  "time",
						Usage: "有效期，0-永久，1-1天，2-7天",
						Value: "0",
					},
					cli.StringFlag{
						Name:  "mode",
						Usage: "模式，1-私密分享，2-公开分享，3-快传",
						Value: "3",
					},
					cli.StringFlag{
						Name:  "sharePwd",
						Usage: "自定义私密分享密码，4个字符，没有指定则随机生成",
						Value: "",
					},
				},
			},
		},
	}
}

// RunOpenShareSet 执行分享
func RunOpenShareSet(modeFlag, driveId string, paths []string, expiredTime string, sharePwd string) {
	if len(paths) <= 0 {
		fmt.Println("请指定文件路径")
		return
	}
	activeUser := GetActiveUser()
	panClient := activeUser.PanClient()

	allFileList := []*aliyunpan.FileEntity{}
	for idx := 0; idx < len(paths); idx++ {
		absolutePath := path.Clean(activeUser.PathJoin(driveId, paths[idx]))
		fileList, err1 := matchPathByShellPattern(driveId, absolutePath)
		if err1 != nil {
			fmt.Println("文件不存在: " + absolutePath)
			continue
		}
		if fileList == nil || len(fileList) == 0 {
			// 文件不存在
			fmt.Println("文件不存在: " + absolutePath)
			continue
		}
		// 匹配的文件
		allFileList = append(allFileList, fileList...)
	}

	fidList := []string{}
	for _, f := range allFileList {
		fidList = append(fidList, f.FileId)
	}

	if len(fidList) == 0 {
		fmt.Printf("没有指定有效的文件\n")
		return
	}

	// 创建分享类型
	if modeFlag == "3" {
		// 快传
		r, err1 := panClient.OpenapiPanClient().FastShareLinkCreate(aliyunpan.FastShareCreateParam{
			DriveId:    driveId,
			FileIdList: fidList,
		})
		if err1 != nil || r == nil {
			if err1.Code == apierror.ApiCodeFileShareNotAllowed {
				fmt.Printf("创建快传链接失败: 该文件类型不允许分享\n")
			} else {
				fmt.Printf("创建快传链接失败: %s\n", err1)
			}
			return
		}

		fmt.Printf("创建快传链接成功\n")
		fmt.Printf("链接：%s\n", r.ShareUrl)
	} else {
		// 分享
		r, err1 := panClient.OpenapiPanClient().ShareLinkCreate(aliyunpan.ShareCreateParam{
			DriveId:    driveId,
			SharePwd:   sharePwd,
			Expiration: expiredTime,
			FileIdList: fidList,
		})
		if err1 != nil || r == nil {
			if err1.Code == apierror.ApiCodeFileShareNotAllowed {
				fmt.Printf("创建分享链接失败: 该文件类型不允许分享\n")
			} else {
				fmt.Printf("创建分享链接失败: %s\n", err1)
			}
			return
		}

		fmt.Printf("创建分享链接成功\n")
		if len(sharePwd) > 0 {
			fmt.Printf("链接：%s 提取码：%s\n", r.ShareUrl, r.SharePwd)
		} else {
			fmt.Printf("链接：%s\n", r.ShareUrl)
		}
	}
}
