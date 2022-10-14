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
	"github.com/tickstep/aliyunpan/cmder/cmdtable"
	"github.com/tickstep/aliyunpan/internal/config"
	"github.com/urfave/cli"
	"os"
	"strconv"
)

func CmdLocateUrl() cli.Command {
	return cli.Command{
		Name:      "locate",
		Usage:     "获取文件下载链接",
		UsageText: cmder.App().Name + " locate <文件1> <文件2> <文件3> ...",
		Description: `
	获取文件下载直链，只支持文件，不支持文件夹。下载链接有效时间为4个小时。

	示例:
	获取 /我的资源/1.mp4 下载直链
	aliyunpan locate /我的资源/1.mp4
`,
		Category: "阿里云盘",
		Before:   cmder.ReloadConfigFunc,
		Action: func(c *cli.Context) error {
			if c.NArg() == 0 {
				cli.ShowCommandHelp(c, c.Command.Name)
				return nil
			}
			RunLocateUrl(parseDriveId(c), c.Args())
			return nil
		},
		Flags: []cli.Flag{
			cli.StringFlag{
				Name:  "driveId",
				Usage: "网盘ID",
				Value: "",
			},
		},
	}
}

// RunLocateUrl 执行下载网盘内文件
func RunLocateUrl(driveId string, paths []string) {
	useInternalUrl := config.Config.TransferUrlType == 2
	activeUser := GetActiveUser()
	activeUser.PanClient().EnableCache()
	activeUser.PanClient().ClearCache()
	defer activeUser.PanClient().DisableCache()

	paths, err := matchPathByShellPattern(driveId, paths...)
	if err != nil {
		fmt.Println(err)
		return
	}

	failedList := []string{}
	for k := range paths {
		p := paths[k]
		if fileInfo, e := activeUser.PanClient().FileInfoByPath(driveId, p); e == nil {
			if fileInfo.IsFolder() {
				failedList = append(failedList, p)
				continue
			}
			durl, apierr := activeUser.PanClient().GetFileDownloadUrl(&aliyunpan.GetFileDownloadUrlParam{
				DriveId: driveId,
				FileId:  fileInfo.FileId,
			})
			if apierr != nil {
				failedList = append(failedList, p)
				continue
			}
			url := durl.Url
			if useInternalUrl {
				url = durl.InternalUrl
			}
			fmt.Printf("\n文件：%s\n%s\n", p, url)
		}
	}

	// 输出失败的文件列表
	if len(failedList) > 0 {
		pnt := func() {
			tb := cmdtable.NewTable(os.Stdout)
			tb.SetHeader([]string{"#", "文件/目录"})
			for k, f := range failedList {
				tb.Append([]string{strconv.Itoa(k + 1), f})
			}
			tb.Render()
		}
		fmt.Printf("\n\n以下文件获取直链失败: \n")
		pnt()
	}
}
