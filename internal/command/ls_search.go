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
	"github.com/tickstep/aliyunpan/internal/config"
	"github.com/tickstep/library-go/converter"
	"github.com/tickstep/library-go/text"
	"github.com/urfave/cli"
	"os"
	"strconv"
)

type (
	// LsOptions 列目录可选项
	LsOptions struct {
		Total bool
	}

	// SearchOptions 搜索可选项
	SearchOptions struct {
		Total   bool
		Recurse bool
	}
)

const (
	opLs int = iota
	opSearch
)

func CmdLs() cli.Command {
	return cli.Command{
		Name:      "ls",
		Aliases:   []string{"l", "ll"},
		Usage:     "列出目录",
		UsageText: cmder.App().Name + " ls <目录>",
		Description: `
	列出当前工作目录内的文件和目录, 或指定目录内的文件和目录

	示例:

	列出 我的资源 内的文件和目录
	aliyunpan ls 我的资源

	绝对路径
	aliyunpan ls /我的资源

	详细列出 我的资源 内的文件和目录
	aliyunpan ll /我的资源
`,
		Category: "阿里云盘",
		Before:   cmder.ReloadConfigFunc,
		Action: func(c *cli.Context) error {
			if config.Config.ActiveUser() == nil {
				fmt.Println("未登录账号")
				return nil
			}

			RunLs(parseDriveId(c), c.Args().Get(0), &LsOptions{
				Total: c.Bool("l") || c.Parent().Args().Get(0) == "ll",
			})

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

func RunLs(driveId, targetPath string, lsOptions *LsOptions)  {
	activeUser := config.Config.ActiveUser()
	targetPath = activeUser.PathJoin(driveId, targetPath)
	if targetPath[len(targetPath) - 1] == '/' {
		targetPath = text.Substr(targetPath, 0, len(targetPath) - 1)
	}

	targetPathInfo, err := activeUser.PanClient().FileInfoByPath(driveId, targetPath)
	if err != nil {
		fmt.Println(err)
		return
	}

	fileList := aliyunpan.FileList{}
	fileListParam := &aliyunpan.FileListParam{}
	fileListParam.ParentFileId = targetPathInfo.FileId
	fileListParam.DriveId = driveId
	if targetPathInfo.IsFolder() {
		fileResult, err := activeUser.PanClient().FileListGetAll(fileListParam)
		if err != nil {
			fmt.Println(err)
			return
		}
		fileList = fileResult
	} else {
		fileList = append(fileList, targetPathInfo)
	}
	renderTable(opLs, lsOptions.Total, targetPath, fileList)
}


func renderTable(op int, isTotal bool, path string, files aliyunpan.FileList) {
	tb := cmdtable.NewTable(os.Stdout)
	var (
		fN, dN   int64
		showPath string
	)

	switch op {
	case opLs:
		showPath = "文件(目录)"
	case opSearch:
		showPath = "路径"
	}

	if isTotal {
		tb.SetHeader([]string{"#", "file_id", "文件大小", "文件SHA1", "文件大小(原始)", "创建日期", "修改日期", showPath})
		tb.SetColumnAlignment([]int{tablewriter.ALIGN_DEFAULT, tablewriter.ALIGN_RIGHT, tablewriter.ALIGN_LEFT, tablewriter.ALIGN_LEFT, tablewriter.ALIGN_LEFT, tablewriter.ALIGN_LEFT, tablewriter.ALIGN_LEFT, tablewriter.ALIGN_LEFT})
		for k, file := range files {
			if file.IsFolder() {
				tb.Append([]string{strconv.Itoa(k), file.FileId, "-", "-", "-", file.CreatedAt, file.UpdatedAt, file.FileName + aliyunpan.PathSeparator})
				continue
			}

			switch op {
			case opLs:
				tb.Append([]string{strconv.Itoa(k), file.FileId, converter.ConvertFileSize(file.FileSize, 2), file.ContentHash, strconv.FormatInt(file.FileSize, 10), file.CreatedAt, file.UpdatedAt, file.FileName})
			case opSearch:
				tb.Append([]string{strconv.Itoa(k), file.FileId, converter.ConvertFileSize(file.FileSize, 2), file.ContentHash, strconv.FormatInt(file.FileSize, 10), file.CreatedAt, file.UpdatedAt, file.Path})
			}
		}
		fN, dN = files.Count()
		tb.Append([]string{"", "", "总: " + converter.ConvertFileSize(files.TotalSize(), 2), "", "", "", fmt.Sprintf("文件总数: %d, 目录总数: %d", fN, dN)})
	} else {
		tb.SetHeader([]string{"#", "文件大小", "修改日期", showPath})
		tb.SetColumnAlignment([]int{tablewriter.ALIGN_DEFAULT, tablewriter.ALIGN_RIGHT, tablewriter.ALIGN_LEFT, tablewriter.ALIGN_LEFT})
		for k, file := range files {
			if file.IsFolder() {
				tb.Append([]string{strconv.Itoa(k), "-", file.UpdatedAt, file.FileName + aliyunpan.PathSeparator})
				continue
			}

			switch op {
			case opLs:
				tb.Append([]string{strconv.Itoa(k), converter.ConvertFileSize(file.FileSize, 2), file.UpdatedAt, file.FileName})
			case opSearch:
				tb.Append([]string{strconv.Itoa(k), converter.ConvertFileSize(file.FileSize, 2), file.UpdatedAt, file.Path})
			}
		}
		fN, dN = files.Count()
		tb.Append([]string{"", "总: " + converter.ConvertFileSize(files.TotalSize(), 2), "", fmt.Sprintf("文件总数: %d, 目录总数: %d", fN, dN)})
	}

	tb.Render()

	if fN+dN >= 60 {
		fmt.Printf("\n当前目录: %s\n", path)
	}

	fmt.Printf("----\n")
}
