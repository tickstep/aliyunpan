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
	"github.com/tickstep/aliyunpan/cmder/cmdtable"
	"github.com/tickstep/aliyunpan/internal/config"
	"github.com/urfave/cli"
	"os"
	"strconv"
)

func CmdCp() cli.Command {
	return cli.Command{
		Name:  "cp",
		Usage: "复制文件/目录",
		UsageText: `
	aliyunpan cp <文件/目录1> <文件/目录2> <文件/目录3> ... <目标目录>`,
		Description: `
	同网盘内复制文件或者目录。支持通配符匹配复制文件，通配符当前只能匹配文件名，不能匹配文件路径。

	示例:

	将 /我的资源/1.mp4 复制到 根目录 /
	aliyunpan cp /我的资源/1.mp4 /

	将 /我的资源 目录下所有的.png文件 复制到 /我的图片 目录下面，使用通配符匹配
	aliyunpan cp /我的资源/*.png /我的图片
`,
		Category: "阿里云盘",
		Before:   ReloadConfigFunc,
		Action: func(c *cli.Context) error {
			if c.NArg() <= 1 {
				cli.ShowCommandHelp(c, c.Command.Name)
				return nil
			}
			if config.Config.ActiveUser() == nil {
				fmt.Println("未登录账号")
				return nil
			}
			RunCopy(parseDriveId(c), c.Args()...)
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

// RunCopy 执行复制文件/目录
func RunCopy(driveId string, paths ...string) {
	activeUser := GetActiveUser()
	cacheCleanPaths := []string{}
	opFileList, targetFile, _, err := getFileInfo(driveId, paths...)
	if err != nil {
		fmt.Println(err)
		return
	}
	if targetFile == nil {
		fmt.Println("目标文件不存在")
		return
	}
	if opFileList == nil || len(opFileList) == 0 {
		fmt.Println("没有有效的文件可复制")
		return
	}
	cacheCleanPaths = append(cacheCleanPaths, targetFile.Path)

	failedCopyFiles := []*aliyunpan.FileEntity{}
	successCopyFiles := []*aliyunpan.FileEntity{}
	fileId2FileEntity := map[string]*aliyunpan.FileEntity{}
	for _, mfi := range opFileList {
		fileId2FileEntity[mfi.FileId] = mfi
		_, er := activeUser.PanClient().OpenapiPanClient().FileCopy(&aliyunpan.FileCopyParam{
			DriveId:        driveId,
			FileId:         mfi.FileId,
			ToParentFileId: targetFile.FileId,
		})
		if er == nil {
			successCopyFiles = append(successCopyFiles, mfi)
		} else {
			failedCopyFiles = append(failedCopyFiles, mfi)
		}
	}

	if len(failedCopyFiles) > 0 {
		fmt.Println("以下文件复制失败：")
		for _, f := range failedCopyFiles {
			fmt.Println(f.FileName)
		}
		fmt.Println("")
	}
	if len(successCopyFiles) > 0 {
		pnt := func() {
			tb := cmdtable.NewTable(os.Stdout)
			tb.SetHeader([]string{"#", "文件/目录"})
			for k, rs := range successCopyFiles {
				tb.Append([]string{strconv.Itoa(k + 1), fileId2FileEntity[rs.FileId].Path})
			}
			tb.Render()
		}
		fmt.Println("操作成功, 以下文件已复制到目标目录: ", targetFile.Path)
		pnt()
		activeUser.DeleteCache(cacheCleanPaths)
	} else {
		fmt.Println("无法复制文件，请稍后重试")
	}
}
