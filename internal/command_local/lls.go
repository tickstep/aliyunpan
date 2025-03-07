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
package command_local

import (
	"fmt"
	"github.com/olekukonko/tablewriter"
	"github.com/tickstep/aliyunpan-api/aliyunpan"
	"github.com/tickstep/aliyunpan/cmder/cmdtable"
	"github.com/tickstep/aliyunpan/internal/command"
	"github.com/tickstep/library-go/converter"
	"github.com/urfave/cli"
	"log"
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

func CmdLocalLs() cli.Command {
	return cli.Command{
		Name:     "lls",
		Category: "本地命令",
		Usage:    "列出目录",
		Description: `
	列出当前本地工作目录内的文件和目录, 或指定目录内的文件和目录

	示例:

	列出 我的资源 内的文件和目录
	aliyunpan lls 我的资源

	绝对路径
	aliyunpan lls /我的资源

	详细列出 我的资源 内的文件和目录
	aliyunpan lls /我的资源
`,
		Before: command.ReloadConfigFunc,
		Action: func(c *cli.Context) error {
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

			RunLocalLs(c.Args().Get(0), &LsOptions{
				Total: c.Bool("l") || c.Parent().Args().Get(0) == "ll",
			}, orderBy, orderSort)

			return nil
		},
		Flags: []cli.Flag{
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

func RunLocalLs(targetPath string,
	lsOptions *LsOptions,
	orderBy aliyunpan.FileOrderBy, orderDirection aliyunpan.FileOrderDirection) {
	targetPath = localPathJoin(targetPath)

	// 获取目标路径文件信息
	localFileInfo, er := os.Stat(targetPath)
	if er != nil {
		fmt.Println("目录路径不存在")
		return
	}
	if !localFileInfo.IsDir() {
		fmt.Println("指定的路径不是目录")
		return
	}

	fileEntryList, err := os.ReadDir(targetPath)
	if err != nil {
		log.Fatal(err)
	}
	fileInfoList := []os.FileInfo{}
	for _, entry := range fileEntryList {
		if fi, e := entry.Info(); e == nil {
			fileInfoList = append(fileInfoList, fi)
		}
	}
	renderTable(opLs, lsOptions.Total, targetPath, fileInfoList)
}

func renderTable(op int, isTotal bool, path string, files []os.FileInfo) {
	tb := cmdtable.NewTable(os.Stdout)
	var (
		fileCount, dirCount, totalSize int64
	)

	tb.SetHeader([]string{"#", "文件大小", "修改日期", "文件(目录)"})
	tb.SetColumnAlignment([]int{tablewriter.ALIGN_DEFAULT, tablewriter.ALIGN_RIGHT, tablewriter.ALIGN_LEFT, tablewriter.ALIGN_LEFT})
	for k, file := range files {
		if file.IsDir() {
			dirCount += 1
			tb.Append([]string{strconv.Itoa(k + 1), "-", file.ModTime().Format("2006-01-02 15:04:05"), file.Name() + aliyunpan.PathSeparator})
			continue
		}
		fileCount += 1
		totalSize += file.Size()
		tb.Append([]string{strconv.Itoa(k + 1), converter.ConvertFileSize(file.Size(), 2), file.ModTime().Format("2006-01-02 15:04:05"), file.Name()})
	}
	tb.Append([]string{"", "总: " + converter.ConvertFileSize(totalSize, 2), "", fmt.Sprintf("文件总数: %d, 目录总数: %d", fileCount, dirCount)})

	fmt.Printf("\n当前目录: %s\n", path)
	fmt.Printf("----\n")
	tb.Render()
	fmt.Printf("----\n")
}
