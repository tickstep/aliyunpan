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
	"github.com/urfave/cli"
)

func CmdCd() cli.Command {
	return cli.Command{
		Name:     "cd",
		Category: "阿里云盘",
		Usage:    "切换工作目录",
		Description: `
	aliyunpan cd <目录, 绝对路径或相对路径>

	示例:

	切换 /我的资源 工作目录:
	aliyunpan cd /我的资源

	切换 /我的资源 工作目录，使用通配符:
	aliyunpan cd /我的*

	切换上级目录:
	aliyunpan cd ..

	切换根目录:
	aliyunpan cd /
`,
		Before: cmder.ReloadConfigFunc,
		After:  cmder.SaveConfigFunc,
		Action: func(c *cli.Context) error {
			if c.NArg() == 0 {
				cli.ShowCommandHelp(c, c.Command.Name)
				return nil
			}
			if config.Config.ActiveUser() == nil {
				fmt.Println("未登录账号")
				return nil
			}
			RunChangeDirectory(parseDriveId(c), c.Args().Get(0))
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

func CmdPwd() cli.Command {
	return cli.Command{
		Name:      "pwd",
		Usage:     "输出工作目录",
		UsageText: cmder.App().Name + " pwd",
		Category:  "阿里云盘",
		Before:    cmder.ReloadConfigFunc,
		Action: func(c *cli.Context) error {
			if config.Config.ActiveUser() == nil {
				fmt.Println("未登录账号")
				return nil
			}
			activeUser := config.Config.ActiveUser()
			if activeUser.IsFileDriveActive() {
				fmt.Println(activeUser.Workdir)
			} else if activeUser.IsAlbumDriveActive() {
				fmt.Println(activeUser.AlbumWorkdir)
			}
			return nil
		},
	}
}

func RunChangeDirectory(driveId, targetPath string) {
	user := config.Config.ActiveUser()
	targetPath = user.PathJoin(driveId, targetPath)

	//targetPathInfo, err := user.PanClient().FileInfoByPath(driveId, targetPath)
	files, err := matchPathByShellPattern(driveId, targetPath)
	if err != nil {
		fmt.Println(err)
		return
	}

	var targetPathInfo *aliyunpan.FileEntity
	if len(files) == 1 {
		targetPathInfo = files[0]
	} else {
		for _, f := range files {
			if f.IsFolder() {
				targetPathInfo = f
				break
			}
		}
	}
	if targetPathInfo == nil {
		fmt.Println("路径不存在")
		return
	}

	if !targetPathInfo.IsFolder() {
		fmt.Printf("错误: %s 不是一个目录 (文件夹)\n", targetPathInfo.Path)
		return
	}

	if user.IsFileDriveActive() {
		user.Workdir = targetPathInfo.Path
		user.WorkdirFileEntity = *targetPathInfo
	} else if user.IsAlbumDriveActive() {
		user.AlbumWorkdir = targetPathInfo.Path
		user.AlbumWorkdirFileEntity = *targetPathInfo
	}

	fmt.Printf("改变工作目录: %s\n", targetPathInfo.Path)
}
