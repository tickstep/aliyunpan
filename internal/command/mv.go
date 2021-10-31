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
	"path"
)

func CmdMv() cli.Command {
	return cli.Command{
		Name:  "mv",
		Usage: "移动文件/目录",
		UsageText: `移动:
	aliyunpan mv <文件/目录1> <文件/目录2> <文件/目录3> ... <目标目录>`,
		Description: `
	注意: 移动多个文件和目录时, 请确保每一个文件和目录都存在, 否则移动操作会失败.

	示例:

	将 /我的资源/1.mp4 移动到 根目录 /
	aliyunpan mv /我的资源/1.mp4 /
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
	if err !=  nil {
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
	for _,mfi := range opFileList {
		fileId2FileEntity[mfi.FileId] = mfi
		moveFileParamList = append(moveFileParamList,
			&aliyunpan.FileMoveParam{
				DriveId: driveId,
				FileId: mfi.FileId,
				ToDriveId: driveId,
				ToParentFileId: targetFile.FileId,
			})
		cacheCleanPaths = append(cacheCleanPaths, path.Dir(mfi.Path))
	}
	fmr,er := activeUser.PanClient().FileMove(moveFileParamList)

	for _,rs := range fmr {
		if !rs.Success {
			failedMoveFiles = append(failedMoveFiles, fileId2FileEntity[rs.FileId])
		}
	}

	if len(failedMoveFiles) > 0 {
		fmt.Println("以下文件移动失败：")
		for _,f := range failedMoveFiles {
			fmt.Println(f.FileName)
		}
		fmt.Println("")
	}
	if er == nil {
		fmt.Println("操作成功, 已移动文件到目标目录: ", targetFile.Path)
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

	opFileList, failedPaths, error = GetAppFileInfoByPaths(driveId, paths[:len(paths)-1]...)
	return
}
