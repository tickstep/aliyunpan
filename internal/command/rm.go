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
	被删除的文件或目录可在网盘文件回收站找回。支持通配符匹配删除文件，通配符当前只能匹配文件名，不能匹配文件路径。

	示例:

	删除 /我的资源/1.mp4
	aliyunpan rm /我的资源/1.mp4

	删除 /我的资源/1.mp4 和 /我的资源/2.mp4
	aliyunpan rm /我的资源/1.mp4 /我的资源/2.mp4

	删除 /我的资源 整个目录 !!
	aliyunpan rm /我的资源

	删除 /我的资源 目录下面的所有.zip文件，使用通配符匹配
	aliyunpan rm /我的资源/*.zip
`,
		Category: "阿里云盘",
		Before:   ReloadConfigFunc,
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

	cacheCleanDirs := []string{}
	failedRmPaths := make([]string, 0, len(paths))
	successDelFileEntity := []*aliyunpan.FileEntity{}

	for _, p := range paths {
		absolutePath := path.Clean(activeUser.PathJoin(driveId, p))
		fileList, err1 := matchPathByShellPattern(driveId, absolutePath)
		if err1 != nil {
			failedRmPaths = append(failedRmPaths, absolutePath)
			continue
		}
		if fileList == nil || len(fileList) == 0 {
			// 失败，文件不存在
			failedRmPaths = append(failedRmPaths, absolutePath)
			continue
		}
		for _, f := range fileList {
			// 删除匹配的文件
			fdr, err := activeUser.PanClient().OpenapiPanClient().FileDelete(&aliyunpan.FileBatchActionParam{
				DriveId: driveId,
				FileId:  f.FileId,
			})
			if err != nil || !fdr.Success {
				failedRmPaths = append(failedRmPaths, absolutePath)
			} else {
				successDelFileEntity = append(successDelFileEntity, f)
			}
			cacheCleanDirs = append(cacheCleanDirs, path.Dir(f.Path))
		}
	}

	// output
	if len(failedRmPaths) > 0 {
		fmt.Println("以下文件删除失败：")
		for _, fp := range failedRmPaths {
			fmt.Println(fp)
		}
		fmt.Println("")
	}
	pnt := func() {
		tb := cmdtable.NewTable(os.Stdout)
		tb.SetHeader([]string{"#", "文件/目录"})
		for k := range successDelFileEntity {
			tb.Append([]string{strconv.Itoa(k + 1), successDelFileEntity[k].Path})
		}
		tb.Render()
	}
	if len(successDelFileEntity) > 0 {
		fmt.Println("操作成功, 以下文件/目录已删除, 可在云盘文件回收站找回: ")
		pnt()
		activeUser.DeleteCache(cacheCleanDirs)
	}
}
