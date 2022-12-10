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
	"github.com/tickstep/library-go/logger"
	"github.com/urfave/cli"
	"log"
	"os"
	"path"
	"strconv"
	"time"
)

func CmdExport() cli.Command {
	return cli.Command{
		Name:      "export",
		Usage:     "导出文件/目录元数据",
		UsageText: cmder.App().Name + " export <网盘文件/目录的路径1> <文件/目录2> <文件/目录3> ... <本地保存文件路径>",
		Description: `
	导出指定文件/目录下面的所有文件的元数据信息，并保存到指定的本地文件里面。导出的文件元信息可以使用 import 命令（秒传文件功能）导入到网盘中。
	支持多个文件或目录的导出.

	示例:

	导出 /我的资源/1.mp4 元数据到文件 /Users/tickstep/Downloads/export_files.txt
	aliyunpan export /我的资源/1.mp4 /Users/tickstep/Downloads/export_files.txt

	导出 /我的资源 整个目录 元数据到文件 /Users/tickstep/Downloads/export_files.txt
	aliyunpan export /我的资源 /Users/tickstep/Downloads/export_files.txt

    导出 网盘 整个目录 元数据到文件 /Users/tickstep/Downloads/export_files.txt
	aliyunpan export / /Users/tickstep/Downloads/export_files.txt
`,
		Category: "阿里云盘",
		Before:   ReloadConfigFunc,
		Action: func(c *cli.Context) error {
			if c.NArg() < 2 {
				cli.ShowCommandHelp(c, c.Command.Name)
				return nil
			}

			subArgs := c.Args()
			RunExportFiles(parseDriveId(c), c.Bool("ow"), subArgs[:len(subArgs)-1], subArgs[len(subArgs)-1])
			return nil
		},
		Flags: []cli.Flag{
			cli.BoolFlag{
				Name:  "ow",
				Usage: "overwrite, 覆盖已存在的导出文件",
			},
			cli.StringFlag{
				Name:  "driveId",
				Usage: "网盘ID",
				Value: "",
			},
		},
	}
}

func RunExportFiles(driveId string, overwrite bool, panPaths []string, saveLocalFilePath string) {
	activeUser := config.Config.ActiveUser()
	panClient := activeUser.PanClient()

	lfi, _ := os.Stat(saveLocalFilePath)
	realSaveFilePath := saveLocalFilePath
	if lfi != nil {
		if lfi.IsDir() {
			realSaveFilePath = path.Join(saveLocalFilePath, "export_file_") + strconv.FormatInt(time.Now().Unix(), 10) + ".txt"
		} else {
			if !overwrite {
				fmt.Println("导出文件已存在")
				return
			}
		}
	} else {
		// create file
		localDir := path.Dir(saveLocalFilePath)
		dirFs, _ := os.Stat(localDir)
		if dirFs != nil {
			if !dirFs.IsDir() {
				fmt.Println("指定的保存文件路径不合法")
				return
			}
		} else {
			er := os.MkdirAll(localDir, 0755)
			if er != nil {
				fmt.Println("创建本地文件夹出错")
				return
			}
		}
		realSaveFilePath = saveLocalFilePath
	}

	totalCount := 0
	saveFile, err := os.OpenFile(realSaveFilePath, os.O_RDWR|os.O_CREATE|os.O_TRUNC, 0755)
	if err != nil {
		log.Fatal(err)
		return
	}

	for _, panPath := range panPaths {
		panPath = activeUser.PathJoin(driveId, panPath)
		panClient.FilesDirectoriesRecurseList(driveId, panPath, func(depth int, _ string, fd *aliyunpan.FileEntity, apiError *apierror.ApiError) bool {
			if apiError != nil {
				logger.Verbosef("%s\n", apiError)
				return true
			}

			// 只需要存储文件即可
			if !fd.IsFolder() {
				item := newRapidUploadItemFromFileEntity(fd)
				jstr := item.createRapidUploadLink(false)
				if len(jstr) <= 0 {
					logger.Verboseln("create rapid upload link err")
					return false
				}
				saveFile.WriteString(jstr + "\n")
				totalCount += 1
				time.Sleep(time.Duration(100) * time.Millisecond)
				fmt.Printf("\r导出文件数量: %d", totalCount)
			}
			return true
		})
	}

	// close and save
	if err := saveFile.Close(); err != nil {
		log.Fatal(err)
	}

	fmt.Printf("\r导出文件总数量: %d\n", totalCount)
	fmt.Printf("导出文件保存路径: %s\n", realSaveFilePath)
}
