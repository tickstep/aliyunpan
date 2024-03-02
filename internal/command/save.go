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
	"path"
	"strings"

	"github.com/tickstep/aliyunpan-api/aliyunpan"
	"github.com/tickstep/aliyunpan/internal/config"
	"github.com/urfave/cli"
)

func CmdSave() cli.Command {
	return cli.Command{
		Name:  "save",
		Usage: "保存分享文件/目录",
		UsageText: `
	aliyunpan save <分享链接> (<提取码>) <目标目录>`,
		Description: `
	注意: 保存大量文件时, 命令完成后可能还需要额外等待一段时间; 分享的根目录下如果包含大量文件或文件夹, 可能存在不稳定的情况

	示例:

	将 公开分享 保存到 根目录 /
	aliyunpan save ABCD1234wxyz /
	aliyunpan save https://www.alipan.com/s/ABCD1234wxyz /

	将 私密分享 保存到 指定目录 /资源分享
	aliyunpan save ABCD1234wxyz akd1 /资源分享
	aliyunpan save https://www.alipan.com/s/ABCD1234wxyz akd1 /资源分享
	`,
		Category: "阿里云盘",
		Before:   ReloadConfigFunc,
		Action: func(c *cli.Context) error {
			if c.NArg() <= 1 || c.NArg() > 3 {
				cli.ShowCommandHelp(c, c.Command.Name)
				return nil
			}
			if config.Config.ActiveUser() == nil {
				fmt.Println("未登录账号")
				return nil
			}
			if config.Config.ActiveUser().PanClient().WebapiPanClient() == nil {
				fmt.Println("WEB客户端未登录，请登录后再使用该命令")
				return nil
			}
			RunSave(parseDriveId(c), c.Args()...)
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

// RunSave 保存分享的文件
func RunSave(driveId string, args ...string) {
	activeUser := GetActiveUser()

	targetFilePath := path.Clean(args[len(args)-1])
	absolutePath := activeUser.PathJoin(driveId, targetFilePath)
	targetFile, err := activeUser.PanClient().WebapiPanClient().FileInfoByPath(driveId, absolutePath)
	if err != nil || !targetFile.IsFolder() {
		fmt.Println("指定目标文件夹不存在")
		return
	}
	fmt.Println("保存文件至：", targetFilePath)

	shareID := args[0]
	if i := strings.Index(shareID, "alipan.com/s/"); i > 0 {
		shareID = shareID[i+13:]
	}
	sharePwd := ""
	if len(args) == 3 {
		sharePwd = args[1]
	}

	token, err := activeUser.PanClient().WebapiPanClient().GetShareToken(shareID, sharePwd)
	if err != nil {
		fmt.Println("读取分享链接失败：", err)
		return
	}

	list, err := activeUser.PanClient().WebapiPanClient().GetListByShare(token.ShareToken, shareID, "")
	if err != nil {
		fmt.Println("读取分享文件列表失败：", err)
		return
	}
	for list.NextMarker != "" {
		list2, err := activeUser.PanClient().WebapiPanClient().GetListByShare(token.ShareToken, shareID, "")
		if err != nil {
			fmt.Println("读取分享文件列表失败：", err)
			return
		}
		list.Items = append(list.Items, list2.Items...)
		list.NextMarker = list2.NextMarker
	}

	var params []*aliyunpan.FileSaveParam
	files := make(map[string]*aliyunpan.ListByShareItem)
	for _, item := range list.Items {
		if item.FileExtension != "" {
			fmt.Println(" ", item.Name)
		} else {
			fmt.Println(" ", item.Name+"/")
		}
		files[item.FileID] = item

		params = append(params, &aliyunpan.FileSaveParam{
			ShareID:        shareID,
			FileId:         item.FileID,
			AutoRename:     true,
			ToDriveId:      driveId,
			ToParentFileId: targetFile.FileId,
		})
	}
	fmt.Println()

	result, err := activeUser.PanClient().WebapiPanClient().FileCopy(token.ShareToken, params)
	if err != nil {
		fmt.Println("保存分享文件失败：", err)
		return
	}
	var ids []string
	var failedSaveFileIds []string
	tasks := make(map[string]string)
	for _, item := range result {
		if item.AsyncTaskId == "" {
			if item.Status != 201 {
				failedSaveFileIds = append(failedSaveFileIds, item.FileId)
			}
		} else {
			tasks[item.AsyncTaskId] = item.FileId
			ids = append(ids, item.AsyncTaskId)
		}
	}
	if ids != nil {
		result2, err := activeUser.PanClient().WebapiPanClient().AsyncTaskGet(token.ShareToken, ids)
		if err != nil {
			fmt.Println("读取保存结果失败：", err)
		}

		for _, item := range result2 {
			if !item.Success {
				failedSaveFileIds = append(failedSaveFileIds, tasks[item.AsyncTaskId])
			}
		}
	}

	if failedSaveFileIds != nil {
		fmt.Println("以下文件保存失败：")
		for _, id := range failedSaveFileIds {
			if v, ok := files[id]; ok {
				fmt.Println(v.Name)
			} else {
				fmt.Println(id)
			}
		}
		fmt.Println("")
	}
	fmt.Println("操作成功, 分享文件已保存到目标目录: ", targetFile.Path)
}
