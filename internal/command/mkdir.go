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
	"github.com/tickstep/aliyunpan/internal/config"
	"github.com/urfave/cli"
)

func CmdMkdir() cli.Command {
	return cli.Command{
		Name:      "mkdir",
		Usage:     "创建目录",
		UsageText: cmder.App().Name + " mkdir <目录>",
		Category:  "阿里云盘",
		Before:    ReloadConfigFunc,
		Action: func(c *cli.Context) error {
			if c.NArg() == 0 {
				cli.ShowCommandHelp(c, c.Command.Name)
				return nil
			}
			if config.Config.ActiveUser() == nil {
				fmt.Println("未登录账号")
				return nil
			}
			RunMkdir(parseDriveId(c), c.Args().Get(0))
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

func RunMkdir(driveId, name string) {
	activeUser := GetActiveUser()
	fullpath := activeUser.PathJoin(driveId, name)
	rs := &aliyunpan.MkdirResult{}
	err := apierror.NewFailedApiError("")
	rs, err = activeUser.PanClient().Mkdir(driveId, "", fullpath)

	if err != nil {
		fmt.Println("创建文件夹失败：" + err.Error())
		return
	}

	if rs.FileId != "" {
		fmt.Println("创建文件夹成功: ", fullpath)

		// cache
		activeUser.DeleteCache(GetAllPathFolderByPath(fullpath))
	} else {
		fmt.Println("创建文件夹失败: ", fullpath)
	}
}
