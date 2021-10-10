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
	"github.com/tickstep/aliyunpan-api/aliyunpan/apierror"
	"github.com/tickstep/aliyunpan/cmder"
	"github.com/tickstep/aliyunpan/cmder/cmdtable"
	"github.com/tickstep/aliyunpan/internal/config"
	"github.com/tickstep/library-go/logger"
	"github.com/urfave/cli"
	"os"
	"strconv"
	"time"
)

func CmdShare() cli.Command {
	return cli.Command{
		Name:      "share",
		Usage:     "分享文件/目录",
		UsageText: cmder.App().Name + " share",
		Category:  "阿里云盘",
		Before:    cmder.ReloadConfigFunc,
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
	aliyunpan share set 1.mp4

    创建文件 1.mp4 的分享链接，并指定有效期为1天
	aliyunpan share set -time 1 1.mp4
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

					sharePwd := ""
					modeFlag := "1"
					if c.IsSet("mode") {
						modeFlag = c.String("mode")
					}
					if modeFlag == "1" {
						sharePwd = RandomStr(4)
					} else {
						sharePwd = ""
					}
					RunShareSet(parseDriveId(c), c.Args(), et, sharePwd)
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
						Usage: "有效期，1-私密分享，2-公开分享",
						Value: "1",
					},
				},
			},
			{
				Name:      "list",
				Aliases:   []string{"l"},
				Usage:     "列出已分享文件/目录",
				UsageText: cmder.App().Name + " share list",
				Action: func(c *cli.Context) error {
					RunShareList()
					return nil
				},
				Flags: []cli.Flag{
				},
			},
			{
				Name:        "cancel",
				Aliases:     []string{"c"},
				Usage:       "取消分享文件/目录",
				UsageText:   cmder.App().Name + " share cancel <shareid_1> <shareid_2> ...",
				Description: `目前只支持通过分享id (shareid) 来取消分享.`,
				Action: func(c *cli.Context) error {
					if c.NArg() < 1 {
						cli.ShowCommandHelp(c, c.Command.Name)
						return nil
					}
					RunShareCancel(c.Args())
					return nil
				},
			},
			{
				Name:      "mc",
				Aliases:   []string{},
				Usage:     "创建秒传链接",
				UsageText: cmder.App().Name + " share mc <文件/目录1> <文件/目录2> ...",
				Description: `
创建文件秒传链接，秒传链接只能是文件，如果是文件夹则会创建文件夹包含的所有文件的秒传链接。秒传链接可以通过RapidUpload命令或者Import命令进行导入到自己的网盘。
示例:
    创建文件 1.mp4 的秒传链接 
	aliyunpan share mc 1.mp4

    创建文件 1.mp4 的秒传链接，但链接隐藏相对路径
	aliyunpan share mc -hp 1.mp4

    创建文件夹 share_folder 下面所有文件的秒传链接
	aliyunpan share mc share_folder/
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
					hp := false
					if c.IsSet("hp") {
						hp = c.Bool("hp")
					}
					RunShareMc(parseDriveId(c), hp, c.Args())
					return nil
				},
				Flags: []cli.Flag{
					cli.StringFlag{
						Name:  "driveId",
						Usage: "网盘ID",
						Value: "",
					},
					cli.BoolFlag{
						Name:  "hp",
						Usage: "hide path, 隐藏相对目录",
					},
				},
			},
		},
	}
}

// RunShareSet 执行分享
func RunShareSet(driveId string, paths []string, expiredTime string, sharePwd string) {
	panClient := GetActivePanClient()
	fileList, _, err := GetFileInfoByPaths(paths[:len(paths)]...)
	if err != nil {
		fmt.Println(err)
		return
	}

	fidList := []string{}
	for _,f := range fileList {
		fidList = append(fidList, f.FileId)
	}

	if len(fidList) == 0 {
		fmt.Printf("没有指定有效的文件\n")
		return
	}

	r, err1 := panClient.ShareLinkCreate(aliyunpan.ShareCreateParam{
		DriveId: driveId,
		SharePwd: sharePwd,
		Expiration: expiredTime,
		FileIdList: fidList,
	})

	if err1 != nil || r == nil {
		if err1.Code == apierror.ApiCodeFileShareNotAllowed {
			fmt.Printf("创建分享链接失败: 该文件类型不允许分享\n")
		} else {
			fmt.Printf("创建分享链接失败: %s\n", err)
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

// RunShareList 执行列出分享列表
func RunShareList() {
	activeUser := GetActiveUser()
	records, err := activeUser.PanClient().ShareLinkList(activeUser.UserId)
	if err != nil {
		fmt.Printf("获取分享列表失败: %s\n", err)
		return
	}

	tb := cmdtable.NewTable(os.Stdout)
	tb.SetHeader([]string{"#", "ShARE_ID", "分享链接", "提取码", "文件名", "FILE_ID", "过期时间", "状态"})
	now := time.Now()
	for k, record := range records {
		et := "永久有效"
		if len(record.Expiration) > 0 {
			et = record.Expiration
		}
		status := "有效"
		if record.FirstFile == nil {
			status = "已删除"
		} else {
			cz := time.FixedZone("CST", 8*3600)
			if len(record.Expiration) > 0 {
				expiredTime, _ := time.ParseInLocation("2006-01-02 15:04:05", record.Expiration, cz)
				if expiredTime.Unix() < now.Unix() {
					status = "已过期"
				}
			}
		}
		tb.Append([]string{strconv.Itoa(k), record.ShareId, record.ShareUrl, record.SharePwd,
			record.ShareName,
			record.FileIdList[0],
			et,
			status})
	}
	tb.Render()
}

// RunShareCancel 执行取消分享
func RunShareCancel(shareIdList []string) {
	if len(shareIdList) == 0 {
		fmt.Printf("取消分享操作失败, 没有任何 shareid\n")
		return
	}

	activeUser := GetActiveUser()
	r, err := activeUser.PanClient().ShareLinkCancel(shareIdList)
	if err != nil {
		fmt.Printf("取消分享操作失败: %s\n", err)
		return
	}

	if r != nil && len(r) > 0 {
		fmt.Printf("取消分享操作成功\n")
	} else {
		fmt.Printf("取消分享操作失败\n")
	}
}

// 创建秒传链接
func RunShareMc(driveId string, hideRelativePath bool, panPaths []string) {
	activeUser := config.Config.ActiveUser()
	panClient := activeUser.PanClient()

	totalCount := 0
	for _,panPath := range panPaths {
		panPath = activeUser.PathJoin(driveId, panPath)
		panClient.FilesDirectoriesRecurseList(driveId, panPath, func(depth int, _ string, fd *aliyunpan.FileEntity, apiError *apierror.ApiError) bool {
			if apiError != nil {
				logger.Verbosef("%s\n", apiError)
				return true
			}

			// 只需要文件即可
			if !fd.IsFolder() {
				item := newRapidUploadItemFromFileEntity(fd)
				jstr := item.createRapidUploadLink(hideRelativePath)
				if len(jstr) <= 0 {
					logger.Verboseln("create rapid upload link err")
					return false
				}
				// print
				fmt.Println(jstr)
				totalCount += 1
				time.Sleep(time.Duration(100) * time.Millisecond)
			}
			return true
		})
	}
	fmt.Printf("\n秒传链接总数量: %d\n", totalCount)
}