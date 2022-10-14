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
	"github.com/tickstep/library-go/logger"
	"github.com/urfave/cli"
	"os"
	"path"
	"strconv"
)

func CmdMv() cli.Command {
	return cli.Command{
		Name:  "mv",
		Usage: "移动文件/目录",
		UsageText: `移动:
	aliyunpan mv <文件/目录1> <文件/目录2> <文件/目录3> ... <目标目录>`,
		Description: `
	注意: 移动多个文件和目录时, 请确保每一个文件和目录都存在, 否则移动操作会失败。支持通配符匹配移动文件，通配符当前只能匹配文件名，不能匹配文件路径。

	示例:

	将 /我的资源/1.mp4 移动到 根目录 /
	aliyunpan mv /我的资源/1.mp4 /

	将 /我的资源 目录下所有的.png文件 移动到 /我的图片 目录下面，使用通配符匹配
	aliyunpan mv /我的资源/*.png /我的图片
`,
		Category: "阿里云盘",
		Before:   cmder.ReloadConfigFunc,
		Action: func(c *cli.Context) error {
			if c.NArg() <= 1 {
				cli.ShowCommandHelp(c, c.Command.Name)
				return nil
			}
			if config.Config.ActiveUser() == nil {
				fmt.Println("未登录账号")
				return nil
			}
			RunMove(parseDriveId(c), c.Args()...)
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

// RunMove 执行移动文件/目录
func RunMove(driveId string, paths ...string) {
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
		fmt.Println("没有有效的文件可移动")
		return
	}
	cacheCleanPaths = append(cacheCleanPaths, targetFile.Path)

	failedMoveFiles := []*aliyunpan.FileEntity{}
	moveFileParamList := []*aliyunpan.FileMoveParam{}
	fileId2FileEntity := map[string]*aliyunpan.FileEntity{}
	for _, mfi := range opFileList {
		fileId2FileEntity[mfi.FileId] = mfi
		moveFileParamList = append(moveFileParamList,
			&aliyunpan.FileMoveParam{
				DriveId:        driveId,
				FileId:         mfi.FileId,
				ToDriveId:      driveId,
				ToParentFileId: targetFile.FileId,
			})
		cacheCleanPaths = append(cacheCleanPaths, path.Dir(mfi.Path))
	}
	fmr, er := activeUser.PanClient().FileMove(moveFileParamList)

	for _, rs := range fmr {
		if !rs.Success {
			failedMoveFiles = append(failedMoveFiles, fileId2FileEntity[rs.FileId])
		}
	}

	if len(failedMoveFiles) > 0 {
		fmt.Println("以下文件移动失败：")
		for _, f := range failedMoveFiles {
			fmt.Println(f.FileName)
		}
		fmt.Println("")
	}
	if er == nil {
		pnt := func() {
			tb := cmdtable.NewTable(os.Stdout)
			tb.SetHeader([]string{"#", "文件/目录"})
			for k, rs := range fmr {
				tb.Append([]string{strconv.Itoa(k + 1), fileId2FileEntity[rs.FileId].Path})
			}
			tb.Render()
		}
		fmt.Println("操作成功, 以下文件已移动到目标目录: ", targetFile.Path)
		pnt()
		activeUser.DeleteCache(cacheCleanPaths)
	} else {
		fmt.Println("无法移动文件，请稍后重试")
	}
}

func getFileInfo(driveId string, paths ...string) (opFileList []*aliyunpan.FileEntity, targetFile *aliyunpan.FileEntity, failedPaths []string, error error) {
	if len(paths) <= 1 {
		return nil, nil, nil, fmt.Errorf("请指定目标文件夹路径")
	}
	activeUser := GetActiveUser()
	// the last one is the target file path
	targetFilePath := path.Clean(paths[len(paths)-1])
	absolutePath := activeUser.PathJoin(driveId, targetFilePath)
	targetFile, err := activeUser.PanClient().FileInfoByPath(driveId, absolutePath)
	if err != nil || !targetFile.IsFolder() {
		return nil, nil, nil, fmt.Errorf("指定目标文件夹不存在")
	}

	for idx := 0; idx < (len(paths) - 1); idx++ {
		absolutePath = path.Clean(activeUser.PathJoin(driveId, paths[idx]))
		name := path.Base(absolutePath)
		fe, err1 := activeUser.PanClient().FileInfoByPath(driveId, absolutePath)
		if err1 != nil {
			// 匹配的文件不存在，检查是否是通配符匹配
			if isMatchWildcardPattern(name) {
				// 通配符
				parentDir := path.Dir(absolutePath)
				wildcardName := path.Base(absolutePath)
				pf, err2 := activeUser.PanClient().FileInfoByPath(driveId, parentDir)
				if err2 != nil {
					failedPaths = append(failedPaths, absolutePath)
					continue
				}
				fileList, er := activeUser.PanClient().FileListGetAll(&aliyunpan.FileListParam{
					DriveId:      driveId,
					ParentFileId: pf.FileId,
				}, 500)
				if er != nil {
					failedPaths = append(failedPaths, absolutePath)
					continue
				}
				for _, f := range fileList {
					if isIncludeFile(wildcardName, f.FileName) {
						f.Path = parentDir + "/" + f.FileName
						logger.Verboseln("wildcard match move: " + f.Path)
						opFileList = append(opFileList, f)
					}
				}
			} else {
				// 文件不存在
				failedPaths = append(failedPaths, absolutePath)
				continue
			}
		} else {
			// 直接删除匹配的文件
			opFileList = append(opFileList, fe)
		}
	}
	return
}
