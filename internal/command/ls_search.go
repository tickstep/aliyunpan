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

			var (
				orderBy   aliyunpan.FileOrderBy        = aliyunpan.FileOrderByUpdatedAt
				orderSort aliyunpan.FileOrderDirection = aliyunpan.FileOrderDirectionDesc
			)

			switch {
			case c.IsSet("asc"):
				orderSort = aliyunpan.FileOrderDirectionAsc
			case c.IsSet("desc"):
				orderSort = aliyunpan.FileOrderDirectionDesc
			default:
				orderSort = aliyunpan.FileOrderDirectionDesc
			}

			switch {
			case c.IsSet("time"):
				orderBy = aliyunpan.FileOrderByUpdatedAt
			case c.IsSet("name"):
				orderBy = aliyunpan.FileOrderByName
			case c.IsSet("size"):
				orderBy = aliyunpan.FileOrderBySize
			default:
				orderBy = aliyunpan.FileOrderByUpdatedAt
			}

			RunLs(parseDriveId(c), c.Args().Get(0), &LsOptions{
				Total: c.Bool("l") || c.Parent().Args().Get(0) == "ll",
			}, orderBy, orderSort)

			return nil
		},
		Flags: []cli.Flag{
			cli.StringFlag{
				Name:  "driveId",
				Usage: "网盘ID",
				Value: "",
			},
			cli.BoolFlag{
				Name:  "asc",
				Usage: "升序排序",
			},
			cli.BoolFlag{
				Name:  "desc",
				Usage: "降序排序",
			},
			cli.BoolFlag{
				Name:  "time",
				Usage: "根据修改时间排序",
			},
			cli.BoolFlag{
				Name:  "name",
				Usage: "根据文件名排序",
			},
			cli.BoolFlag{
				Name:  "size",
				Usage: "根据大小排序",
			},
		},
	}
}

func RunLs(driveId, targetPath string, lsOptions *LsOptions,
	orderBy aliyunpan.FileOrderBy, orderDirection aliyunpan.FileOrderDirection) {
	activeUser := config.Config.ActiveUser()
	targetPath = activeUser.PathJoin(driveId, targetPath)
	if targetPath[len(targetPath)-1] == '/' {
		targetPath = text.Substr(targetPath, 0, len(targetPath)-1)
	}

	//targetPathInfo, err := activeUser.PanClient().FileInfoByPath(driveId, targetPath)
	//if err != nil {
	//	fmt.Println(err)
	//	return
	//}

	files, err := matchPathByShellPattern(driveId, targetPath)
	if err != nil {
		fmt.Println(err)
		return
	}
	var targetPathInfo *aliyunpan.FileEntity
	if len(files) == 1 {
		targetPathInfo = files[0]
	} else {
		for _, f := range files {
			if f.IsFolder() {
				targetPathInfo = f
				break
			}
		}
	}
	if targetPathInfo == nil {
		fmt.Println("目录路径不存在")
		return
	}

	fileList := aliyunpan.FileList{}
	fileListParam := &aliyunpan.FileListParam{}
	fileListParam.ParentFileId = targetPathInfo.FileId
	fileListParam.DriveId = driveId
	fileListParam.OrderBy = orderBy
	fileListParam.OrderDirection = orderDirection
	if targetPathInfo.IsFolder() {
		fileResult, err := activeUser.PanClient().FileListGetAll(fileListParam, 0)
		if err != nil {
			fmt.Println(err)
			return
		}
		fileList = fileResult
	} else {
		fileList = append(fileList, targetPathInfo)
	}
	renderTable(opLs, lsOptions.Total, targetPathInfo.Path, fileList)
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
				tb.Append([]string{strconv.Itoa(k + 1), file.FileId, "-", "-", "-", file.CreatedAt, file.UpdatedAt, file.FileName + aliyunpan.PathSeparator})
				continue
			}

			switch op {
			case opLs:
				tb.Append([]string{strconv.Itoa(k + 1), file.FileId, converter.ConvertFileSize(file.FileSize, 2), file.ContentHash, strconv.FormatInt(file.FileSize, 10), file.CreatedAt, file.UpdatedAt, file.FileName})
			case opSearch:
				tb.Append([]string{strconv.Itoa(k + 1), file.FileId, converter.ConvertFileSize(file.FileSize, 2), file.ContentHash, strconv.FormatInt(file.FileSize, 10), file.CreatedAt, file.UpdatedAt, file.Path})
			}
		}
		fN, dN = files.Count()
		tb.Append([]string{"", "", "总: " + converter.ConvertFileSize(files.TotalSize(), 2), "", "", "", fmt.Sprintf("文件总数: %d, 目录总数: %d", fN, dN)})
	} else {
		tb.SetHeader([]string{"#", "文件大小", "修改日期", showPath})
		tb.SetColumnAlignment([]int{tablewriter.ALIGN_DEFAULT, tablewriter.ALIGN_RIGHT, tablewriter.ALIGN_LEFT, tablewriter.ALIGN_LEFT})
		for k, file := range files {
			if file.IsFolder() {
				tb.Append([]string{strconv.Itoa(k + 1), "-", file.UpdatedAt, file.FileName + aliyunpan.PathSeparator})
				continue
			}

			switch op {
			case opLs:
				tb.Append([]string{strconv.Itoa(k + 1), converter.ConvertFileSize(file.FileSize, 2), file.UpdatedAt, file.FileName})
			case opSearch:
				tb.Append([]string{strconv.Itoa(k + 1), converter.ConvertFileSize(file.FileSize, 2), file.UpdatedAt, file.Path})
			}
		}
		fN, dN = files.Count()
		tb.Append([]string{"", "总: " + converter.ConvertFileSize(files.TotalSize(), 2), "", fmt.Sprintf("文件总数: %d, 目录总数: %d", fN, dN)})
	}
	fmt.Printf("\n当前目录: %s\n", path)
	fmt.Printf("----\n")
	tb.Render()
	fmt.Printf("----\n")
}
