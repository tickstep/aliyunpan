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
	"github.com/olekukonko/tablewriter"
	"github.com/tickstep/aliyunpan-api/aliyunpan"
	"github.com/tickstep/aliyunpan/cmder"
	"github.com/tickstep/aliyunpan/cmder/cmdtable"
	"github.com/tickstep/library-go/converter"
	"github.com/tickstep/library-go/logger"
	"github.com/urfave/cli"
	"os"
	"strconv"
)

func CmdRecycle() cli.Command {
	return cli.Command{
		Name:  "recycle",
		Usage: "回收站",
		Description: `
	回收站操作.

	示例:

	1. 从回收站还原两个文件, 其中的两个文件的 file_id 分别为 1013792297798440 和 643596340463870
	aliyunpan recycle restore 1013792297798440 643596340463870

	2. 从回收站删除两个文件, 其中的两个文件的 file_id 分别为 1013792297798440 和 643596340463870
	aliyunpan recycle delete 1013792297798440 643596340463870

	3. 清空回收站, 程序不会进行二次确认, 谨慎操作!!!
	aliyunpan recycle delete -all
`,
		Category: "阿里云盘",
		Before:   ReloadConfigFunc,
		Action: func(c *cli.Context) error {
			if c.NumFlags() <= 0 || c.NArg() <= 0 {
				cli.ShowCommandHelp(c, c.Command.Name)
			}
			return nil
		},
		Subcommands: []cli.Command{
			{
				Name:      "list",
				Aliases:   []string{"ls", "l"},
				Usage:     "列出回收站文件列表",
				UsageText: cmder.App().Name + " recycle list",
				Action: func(c *cli.Context) error {
					RunRecycleList(parseDriveId(c))
					return nil
				},
				Flags: []cli.Flag{
					cli.StringFlag{
						Name:  "driveId",
						Usage: "网盘ID",
						Value: "",
					},
				},
			},
			{
				Name:        "restore",
				Aliases:     []string{"r"},
				Usage:       "还原回收站文件或目录",
				UsageText:   cmder.App().Name + " recycle restore <file_id 1> <file_id 2> <file_id 3> ...",
				Description: `根据文件/目录的 fs_id, 还原回收站指定的文件或目录`,
				Action: func(c *cli.Context) error {
					if c.NArg() <= 0 {
						cli.ShowCommandHelp(c, c.Command.Name)
						return nil
					}
					RunRecycleRestore(parseDriveId(c), c.Args()...)
					return nil
				},
				Flags: []cli.Flag{
					cli.StringFlag{
						Name:  "driveId",
						Usage: "网盘ID",
						Value: "",
					},
				},
			},
			{
				Name:        "delete",
				Aliases:     []string{"d"},
				Usage:       "删除回收站文件或目录 / 清空回收站",
				UsageText:   cmder.App().Name + " recycle delete [-all] <file_id 1> <file_id 2> <file_id 3> ...",
				Description: `根据文件/目录的 file_id 或 -all 参数, 删除回收站指定的文件或目录或清空回收站`,
				Action: func(c *cli.Context) error {
					if c.Bool("all") {
						// 清空回收站
						RunRecycleClear(parseDriveId(c))
						return nil
					}

					if c.NArg() <= 0 {
						cli.ShowCommandHelp(c, c.Command.Name)
						return nil
					}
					RunRecycleDelete(parseDriveId(c), c.Args()...)
					return nil
				},
				Flags: []cli.Flag{
					cli.BoolFlag{
						Name:  "all",
						Usage: "清空回收站, 程序不会进行二次确认, 谨慎操作!!!",
					},
					cli.StringFlag{
						Name:  "driveId",
						Usage: "网盘ID",
						Value: "",
					},
				},
			},
		},
	}
}

// RunRecycleList 执行列出回收站文件列表
func RunRecycleList(driveId string) {
	panClient := GetActivePanClient()
	fdl, err := panClient.RecycleBinFileListGetAll(&aliyunpan.RecycleBinFileListParam{
		DriveId: driveId,
		Limit:   100,
	})
	if err != nil {
		fmt.Println(err)
		return
	}

	tb := cmdtable.NewTable(os.Stdout)
	tb.SetHeader([]string{"#", "file_id", "文件/目录名", "文件大小", "创建日期", "修改日期"})
	tb.SetColumnAlignment([]int{tablewriter.ALIGN_DEFAULT, tablewriter.ALIGN_RIGHT, tablewriter.ALIGN_RIGHT, tablewriter.ALIGN_LEFT, tablewriter.ALIGN_LEFT})
	for k, file := range fdl {
		fn := file.FileName
		fs := converter.ConvertFileSize(file.FileSize, 2)
		if file.IsFolder() {
			fn = fn + "/"
			fs = "-"
		}
		tb.Append([]string{strconv.Itoa(k + 1), file.FileId, fn, fs, file.CreatedAt, file.UpdatedAt})
	}

	tb.Render()
}

// RunRecycleRestore 执行还原回收站文件或目录
func RunRecycleRestore(driveId string, fidStrList ...string) {
	panClient := GetActivePanClient()
	restoreFileList := []*aliyunpan.FileBatchActionParam{}

	for _, fid := range fidStrList {
		restoreFileList = append(restoreFileList, &aliyunpan.FileBatchActionParam{
			DriveId: driveId,
			FileId:  fid,
		})
	}

	if len(restoreFileList) == 0 {
		fmt.Printf("没有需要还原的文件")
		return
	}

	rbfr, err := panClient.RecycleBinFileRestore(restoreFileList)
	if rbfr != nil && len(rbfr) > 0 {
		fmt.Printf("还原文件成功\n")
		return
	}

	if len(rbfr) == 0 && err != nil {
		fmt.Printf("还原文件失败：%s\n", err)
		return
	}
}

// RunRecycleDelete 执行删除回收站文件或目录
func RunRecycleDelete(driveId string, fidStrList ...string) {
	panClient := GetActivePanClient()
	deleteFileList := []*aliyunpan.FileBatchActionParam{}

	for _, fid := range fidStrList {
		deleteFileList = append(deleteFileList, &aliyunpan.FileBatchActionParam{
			DriveId: driveId,
			FileId:  fid,
		})
	}

	if len(deleteFileList) == 0 {
		fmt.Printf("没有需要删除的文件")
		return
	}

	rbfr, err := panClient.RecycleBinFileDelete(deleteFileList)
	if rbfr != nil && len(rbfr) > 0 {
		fmt.Printf("彻底删除文件成功\n")
		return
	}

	if len(rbfr) == 0 && err != nil {
		fmt.Printf("彻底删除文件失败：%s\n", err)
		return
	}
}

// RunRecycleClear 清空回收站
func RunRecycleClear(driveId string) {
	panClient := GetActivePanClient()

	for {
		// get file list
		fdl, err := panClient.RecycleBinFileListGetAll(&aliyunpan.RecycleBinFileListParam{
			DriveId: driveId,
			Limit:   100,
		})
		if err != nil {
			logger.Verboseln(err)
			break
		}
		if fdl == nil || len(fdl) == 0 {
			break
		}

		// delete
		deleteFileList := []*aliyunpan.FileBatchActionParam{}
		for _, f := range fdl {
			deleteFileList = append(deleteFileList, &aliyunpan.FileBatchActionParam{
				DriveId: driveId,
				FileId:  f.FileId,
			})
		}

		if len(deleteFileList) == 0 {
			logger.Verboseln("没有需要删除的文件")
			break
		}

		rbfr, err := panClient.RecycleBinFileDelete(deleteFileList)
		if rbfr != nil && len(rbfr) > 0 {
			logger.Verboseln("彻底删除文件成功")
		}
	}

	fmt.Printf("清空回收站成功\n")
}
