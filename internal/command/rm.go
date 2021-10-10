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
	"path"
	"strconv"
)

func CmdRm() cli.Command {
	return cli.Command{
		Name:      "rm",
		Usage:     "删除文件/目录",
		UsageText: cmder.App().Name + " rm <文件/目录的路径1> <文件/目录2> <文件/目录3> ...",
		Description: `
	注意: 删除多个文件和目录时, 请确保每一个文件和目录都存在, 否则删除操作会失败.
	被删除的文件或目录可在网盘文件回收站找回.

	示例:

	删除 /我的资源/1.mp4
	aliyunpan rm /我的资源/1.mp4

	删除 /我的资源/1.mp4 和 /我的资源/2.mp4
	aliyunpan rm /我的资源/1.mp4 /我的资源/2.mp4

	删除 /我的资源 整个目录 !!
	aliyunpan rm /我的资源
`,
		Category: "阿里云盘",
		Before:   cmder.ReloadConfigFunc,
		Action: func(c *cli.Context) error {
			if c.NArg() == 0 {
				cli.ShowCommandHelp(c, c.Command.Name)
				return nil
			}
			if config.Config.ActiveUser() == nil {
				fmt.Println("未登录账号")
				return nil
			}
			RunRemove(parseDriveId(c), c.Args()...)
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

// RunRemove 执行 批量删除文件/目录
func RunRemove(driveId string, paths ...string) {
	activeUser := GetActiveUser()

	failedRmPaths := make([]string, 0, len(paths))
	delFileInfos := []*aliyunpan.FileBatchActionParam{}
	fileId2FileEntity := map[string]*aliyunpan.FileEntity{}

	for _, p := range paths {
		absolutePath := path.Clean(activeUser.PathJoin(driveId, p))
		fe, err := activeUser.PanClient().FileInfoByPath(driveId, absolutePath)
		if err != nil {
			failedRmPaths = append(failedRmPaths, absolutePath)
			continue
		}
		fe.Path = absolutePath
		delFileInfos = append(delFileInfos, &aliyunpan.FileBatchActionParam{
			DriveId:driveId,
			FileId:fe.FileId,
		})
		fileId2FileEntity[fe.FileId] = fe
	}

	// delete
	successDelFileEntity := []*aliyunpan.FileEntity{}
	fdr, err := activeUser.PanClient().FileDelete(delFileInfos)
	if fdr != nil {
		for _,item := range fdr {
			if !item.Success {
				failedRmPaths = append(failedRmPaths, fileId2FileEntity[item.FileId].Path)
			} else {
				successDelFileEntity = append(successDelFileEntity, fileId2FileEntity[item.FileId])
			}
		}
	}

	pnt := func() {
		tb := cmdtable.NewTable(os.Stdout)
		tb.SetHeader([]string{"#", "文件/目录"})
		for k := range successDelFileEntity {
			tb.Append([]string{strconv.Itoa(k), successDelFileEntity[k].Path})
		}
		tb.Render()
	}
	if len(successDelFileEntity) > 0 {
		fmt.Println("操作成功, 以下文件/目录已删除, 可在云盘文件回收站找回: ")
		pnt()
	}

	if len(successDelFileEntity) == 0 && err != nil {
		fmt.Println("无法删除文件，请稍后重试")
		return
	}
}
