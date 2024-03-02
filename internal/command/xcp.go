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
	"path"
	"strconv"
)

func CmdXcp() cli.Command {
	return cli.Command{
		Name:  "xcp",
		Usage: "备份盘和资源库之间转存文件",
		UsageText: `
	aliyunpan xcp <文件/目录1> <文件/目录2> <文件/目录3> ... <目标盘目录>`,
		Description: `
	注意: 拷贝多个文件和目录时, 请确保每一个文件和目录都存在, 否则拷贝操作会失败。

	示例:

	当前程序工作在备份盘下，将备份盘 /1.mp4 文件转存复制到 资源库下的 /来自备份盘/video 目录下
	aliyunpan xcp /1.mp4 /来自备份盘/video

	当前程序工作在资源库下，将资源库 /1.mp4 文件转存复制到 备份盘下的 /来自资源库/video 目录下
	aliyunpan xcp /1.mp4 /来自资源库/video

	当前程序工作在备份盘下，将 /我的资源 目录下所有的.mp4文件 复制到 /来自备份盘/video 目录下面，使用通配符匹配
	aliyunpan xcp /我的资源/*.mp4 /来自备份盘/video
`,
		Category: "阿里云盘",
		Before:   ReloadConfigFunc,
		Action: func(c *cli.Context) error {
			if c.NArg() <= 0 {
				cli.ShowCommandHelp(c, c.Command.Name)
				return nil
			}
			if config.Config.ActiveUser() == nil {
				fmt.Println("未登录账号")
				return nil
			}
			if config.Config.ActiveUser().PanClient().WebapiPanClient() == nil {
				fmt.Println("WEB客户端未登录，请登录后再使用")
				return nil
			}
			srcDriveId := parseDriveId(c)
			dstDriveId := ""
			driveList := config.Config.ActiveUser().DriveList
			if driveList.GetFileDriveId() == srcDriveId {
				dstDriveId = driveList.GetResourceDriveId()
			} else if driveList.GetResourceDriveId() == srcDriveId {
				dstDriveId = driveList.GetFileDriveId()
			} else {
				fmt.Println("不支持该操作")
				return nil
			}
			RunXCopy(srcDriveId, dstDriveId, c.Args()...)
			return nil
		},
		Flags: []cli.Flag{
			cli.StringFlag{
				Name:  "driveId",
				Usage: "源网盘ID",
				Value: "",
			},
		},
	}
}

// RunXCopy 执行复制文件/目录
func RunXCopy(srcDriveId, dstDriveId string, paths ...string) {
	activeUser := GetActiveUser()
	cacheCleanPaths := []string{}
	opFileList, targetFile, _, err := getCrossCopyFileInfo(srcDriveId, dstDriveId, paths...)
	if err != nil {
		fmt.Println(err)
		return
	}
	if targetFile == nil {
		fmt.Println("目标目录不存在")
		return
	}
	if opFileList == nil || len(opFileList) == 0 {
		fmt.Println("没有有效的文件可复制")
		return
	}
	cacheCleanPaths = append(cacheCleanPaths, targetFile.Path)

	failedCopyFiles := []*aliyunpan.FileEntity{}
	copyFileParamList := []string{}
	fileId2FileEntity := map[string]*aliyunpan.FileEntity{}
	for _, mfi := range opFileList {
		fileId2FileEntity[mfi.FileId] = mfi
		copyFileParamList = append(copyFileParamList, mfi.FileId)
		cacheCleanPaths = append(cacheCleanPaths, path.Dir(mfi.Path))
	}
	fccr, er := activeUser.PanClient().WebapiPanClient().FileCrossDriveCopy(&aliyunpan.FileCrossCopyParam{
		FromDriveId:    srcDriveId,
		FromFileIds:    copyFileParamList,
		ToDriveId:      dstDriveId,
		ToParentFileId: targetFile.FileId,
	})
	for _, rs := range fccr {
		if rs.Status != 201 {
			failedCopyFiles = append(failedCopyFiles, fileId2FileEntity[rs.SourceFileId])
		}
	}

	if len(failedCopyFiles) > 0 {
		fmt.Println("以下文件复制失败：")
		for _, f := range failedCopyFiles {
			fmt.Println(f.FileName)
		}
		fmt.Println("")
	}
	if er == nil {
		pnt := func() {
			tb := cmdtable.NewTable(os.Stdout)
			tb.SetHeader([]string{"#", "文件/目录"})
			for k, rs := range fccr {
				tb.Append([]string{strconv.Itoa(k + 1), fileId2FileEntity[rs.SourceFileId].Path})
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

func getCrossCopyFileInfo(srcDriveId, dstDriveId string, paths ...string) (opFileList []*aliyunpan.FileEntity, targetFile *aliyunpan.FileEntity, failedPaths []string, error error) {
	if len(paths) <= 1 {
		return nil, nil, nil, fmt.Errorf("请指定目标文件夹路径")
	}
	activeUser := GetActiveUser()

	// the last one is the target file path
	targetFilePath := path.Clean(paths[len(paths)-1])
	absolutePath := activeUser.PathJoin(dstDriveId, targetFilePath)
	targetFile, err := activeUser.PanClient().WebapiPanClient().FileInfoByPath(dstDriveId, absolutePath)
	if err != nil || !targetFile.IsFolder() {
		return nil, nil, nil, fmt.Errorf("指定目标文件夹不存在")
	}

	// source file list
	for idx := 0; idx < (len(paths) - 1); idx++ {
		absolutePath = path.Clean(activeUser.PathJoin(srcDriveId, paths[idx]))
		fileList, err1 := matchPathByShellPattern(srcDriveId, absolutePath)
		if err1 != nil {
			failedPaths = append(failedPaths, absolutePath)
			continue
		}
		if fileList == nil || len(fileList) == 0 {
			// 文件不存在
			failedPaths = append(failedPaths, absolutePath)
			continue
		}
		for _, f := range fileList {
			// 移动匹配的文件
			opFileList = append(opFileList, f)
		}
	}
	return
}
