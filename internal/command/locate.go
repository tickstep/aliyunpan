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
	"github.com/tickstep/aliyunpan/library/collection"
	"github.com/urfave/cli"
	"io/ioutil"
	"os"
	"strconv"
	"strings"
)

func CmdLocateUrl() cli.Command {
	return cli.Command{
		Name:      "locate",
		Usage:     "获取文件下载链接",
		UsageText: cmder.App().Name + " locate <文件/目录1> <文件/目录2> <文件/目录3> ...",
		Description: `
	获取文件下载直链，支持文件和文件夹。下载链接有效时间为4个小时。
	导出的下载链接可以使用任何支持http下载的工具进行下载，如果是视频文件还可以使用支持在线流播放的视频软件进行在线播放。
	
	注意：由于阿里云盘网页端有防盗链设置所以不能直接使用网页Token登录，你必须使用手机扫码二维码登录(命令：login -QrCode)，否则获取的直链无法正常下载，会提示 403 Forbidden 下载被禁止。

	示例:
	获取 /我的资源/1.mp4 下载直链
	aliyunpan locate /我的资源/1.mp4

	获取 /我的资源 目录下面所有文件的下载直链并保存到本地文件 /Volumes/Downloads/file_url.txt 中
	aliyunpan locate -saveto "/Volumes/Downloads/file_url.txt" /我的资源
`,
		Category: "阿里云盘",
		Before:   cmder.ReloadConfigFunc,
		Action: func(c *cli.Context) error {
			if c.NArg() == 0 {
				cli.ShowCommandHelp(c, c.Command.Name)
				return nil
			}
			saveFilePath := ""
			if c.IsSet("saveto") {
				saveFilePath = c.String("saveto")
			}
			RunLocateUrl(parseDriveId(c), c.Args(), saveFilePath)
			return nil
		},
		Flags: []cli.Flag{
			cli.StringFlag{
				Name:  "driveId",
				Usage: "网盘ID",
				Value: "",
			},
			cli.StringFlag{
				Name:  "saveto",
				Usage: "导出链接成文件并保存到指定的位置",
			},
		},
	}
}

// RunLocateUrl 执行下载网盘内文件
func RunLocateUrl(driveId string, paths []string, saveFilePath string) {
	useInternalUrl := config.Config.TransferUrlType == 2
	activeUser := GetActiveUser()
	activeUser.PanClient().EnableCache()
	activeUser.PanClient().ClearCache()
	defer activeUser.PanClient().DisableCache()

	paths, err := makePathAbsolute(driveId, paths...)
	if err != nil {
		fmt.Println(err)
		return
	}
	fileEntityQueue := collection.NewFifoQueue()
	sb := &strings.Builder{}
	failedList := []string{}
	for _, p := range paths {
		if fileInfo, e := activeUser.PanClient().FileInfoByPath(driveId, p); e == nil {
			fileInfo.Path = p
			fileEntityQueue.Push(fileInfo)
		} else {
			failedList = append(failedList, p)
		}
	}

	for {
		item := fileEntityQueue.Pop()
		if item == nil {
			break
		}
		fileInfo := item.(*aliyunpan.FileEntity)
		if fileInfo.IsFolder() { // 文件夹，获取下面所有文件
			allFilesInFolder, er := activeUser.PanClient().FileListGetAll(&aliyunpan.FileListParam{
				DriveId:      driveId,
				ParentFileId: fileInfo.FileId,
			}, 300)
			if er != nil {
				failedList = append(failedList, fileInfo.Path)
				continue
			}
			for _, f := range allFilesInFolder {
				f.Path = fileInfo.Path + "/" + f.FileName
				if f.IsFolder() {
					// for next term
					fileEntityQueue.Push(f)
					continue
				}
				durl, apierr := activeUser.PanClient().GetFileDownloadUrl(&aliyunpan.GetFileDownloadUrlParam{
					DriveId: driveId,
					FileId:  f.FileId,
				})
				if apierr != nil {
					failedList = append(failedList, f.Path)
					continue
				}
				url := durl.Url
				if useInternalUrl {
					url = durl.InternalUrl
				}
				if saveFilePath != "" {
					fmt.Printf("获取文件下载链接：%s\n", f.Path)
					fmt.Fprintf(sb, "\n文件：%s\n%s\n", f.Path, url)
				} else {
					fmt.Printf("\n文件：%s\n%s\n", f.Path, url)
				}
			}
		} else { // 文件
			durl, apierr := activeUser.PanClient().GetFileDownloadUrl(&aliyunpan.GetFileDownloadUrlParam{
				DriveId: driveId,
				FileId:  fileInfo.FileId,
			})
			if apierr != nil {
				failedList = append(failedList, fileInfo.Path)
				continue
			}
			url := durl.Url
			if useInternalUrl {
				url = durl.InternalUrl
			}
			if saveFilePath != "" {
				fmt.Printf("获取文件下载链接：%s\n", fileInfo.Path)
				fmt.Fprintf(sb, "\n文件：%s\n%s\n", fileInfo.Path, url)
			} else {
				fmt.Printf("\n文件：%s\n%s\n", fileInfo.Path, url)
			}
		}
	}

	if saveFilePath != "" {
		// save file
		if e := ioutil.WriteFile(saveFilePath, []byte(sb.String()), 0777); e == nil {
			fmt.Printf("保存文件成功：%s\n", saveFilePath)
		} else {
			fmt.Printf("保存文件失败：%s\n", saveFilePath)
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
