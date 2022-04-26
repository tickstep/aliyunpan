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
	"github.com/urfave/cli"
	"os"
	"strconv"
)

func CmdAlbum() cli.Command {
	return cli.Command{
		Name:      "album",
		Aliases:   []string{"abm"},
		Usage:     "相簿",
		UsageText: cmder.App().Name + " album",
		Category:  "阿里云盘",
		Before:    cmder.ReloadConfigFunc,
		Action: func(c *cli.Context) error {
			cli.ShowCommandHelp(c, c.Command.Name)
			return nil
		},

		Subcommands: []cli.Command{
			{
				Name:      "list",
				Aliases:   []string{"ls"},
				Usage:     "展示相簿列表",
				UsageText: cmder.App().Name + " album list",
				Action: func(c *cli.Context) error {
					RunAlbumList()
					return nil
				},
				Flags: []cli.Flag{},
			},
		},
	}
}

func RunAlbumList() {
	activeUser := GetActiveUser()
	records, err := activeUser.PanClient().AlbumListGetAll(&aliyunpan.AlbumListParam{})
	if err != nil {
		fmt.Printf("获取相簿列表失败: %s\n", err)
		return
	}

	tb := cmdtable.NewTable(os.Stdout)
	tb.SetHeader([]string{"#", "ALBUM_ID", "名称", "文件数量", "创建日期", "修改日期"})
	tb.SetColumnAlignment([]int{tablewriter.ALIGN_DEFAULT, tablewriter.ALIGN_DEFAULT, tablewriter.ALIGN_LEFT, tablewriter.ALIGN_CENTER, tablewriter.ALIGN_DEFAULT, tablewriter.ALIGN_DEFAULT})
	for k, record := range records {
		tb.Append([]string{strconv.Itoa(k), record.AlbumId, record.Name, strconv.Itoa(record.FileCount),
			record.CreatedAtStr(), record.UpdatedAtStr()})
	}
	tb.Render()
}
