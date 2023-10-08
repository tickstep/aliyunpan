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
	"github.com/olekukonko/tablewriter"
	"github.com/tickstep/aliyunpan-api/aliyunpan"
	"github.com/tickstep/aliyunpan/cmder/cmdtable"
	"github.com/tickstep/aliyunpan/internal/config"
	"github.com/urfave/cli"
	"strconv"
	"strings"
)

func CmdDrive() cli.Command {
	return cli.Command{
		Name:  "drive",
		Usage: "切换网盘（备份盘/资源库/相册）",
		Description: `
	切换已登录账号的阿里云盘的工作网盘（备份盘/资源库/相册）
	如果运行该条命令没有提供参数, 程序将会列出所有的网盘列表, 供选择切换.

	示例:
	aliyunpan drive
	aliyunpan drive <driveId>
`,
		Category: "阿里云盘账号",
		Before:   ReloadConfigFunc,
		After:    SaveConfigFunc,
		Action: func(c *cli.Context) error {
			inputData := c.Args().Get(0)
			targetDriveId := strings.TrimSpace(inputData)
			RunSwitchDriveList(targetDriveId)
			return nil
		},
	}
}

func RunSwitchDriveList(targetDriveId string) {
	currentDriveId := config.Config.ActiveUser().ActiveDriveId
	var activeDriveInfo *config.DriveInfo = nil
	driveList, renderStr := getDriveOptionList()

	if driveList == nil || len(driveList) == 0 {
		fmt.Println("切换网盘失败")
		return
	}

	if targetDriveId == "" {
		// show option list
		fmt.Println(renderStr)

		// 提示输入 index
		var index string
		fmt.Printf("输入要切换的网盘 # 值 > ")
		_, err := fmt.Scanln(&index)
		if err != nil {
			return
		}

		if n, err1 := strconv.Atoi(index); err1 == nil && (n-1) >= 0 && (n-1) < len(driveList) {
			activeDriveInfo = driveList[n-1]
		} else {
			fmt.Printf("切换网盘失败, 请检查 # 值是否正确\n")
			return
		}
	} else {
		// 直接切换
		for _, driveInfo := range driveList {
			if driveInfo.DriveId == targetDriveId {
				activeDriveInfo = driveInfo
				break
			}
		}
	}

	if activeDriveInfo == nil {
		fmt.Printf("切换网盘失败\n")
		return
	}

	config.Config.ActiveUser().ActiveDriveId = activeDriveInfo.DriveId
	activeUser := config.Config.ActiveUser()
	if currentDriveId != config.Config.ActiveUser().ActiveDriveId {
		// clear the drive work path
		if activeUser.IsFileDriveActive() {
			if activeUser.Workdir == "" {
				config.Config.ActiveUser().Workdir = "/"
				config.Config.ActiveUser().WorkdirFileEntity = *aliyunpan.NewFileEntityForRootDir()
			}
		} else if activeUser.IsResourceDriveActive() {
			if activeUser.ResourceWorkdir == "" {
				config.Config.ActiveUser().ResourceWorkdir = "/"
				config.Config.ActiveUser().ResourceWorkdirFileEntity = *aliyunpan.NewFileEntityForRootDir()
			}
		} else if activeUser.IsAlbumDriveActive() {
			if activeUser.AlbumWorkdir == "" {
				config.Config.ActiveUser().AlbumWorkdir = "/"
				config.Config.ActiveUser().AlbumWorkdirFileEntity = *aliyunpan.NewFileEntityForRootDir()
			}
		}
	}
	fmt.Printf("切换到网盘：%s\n", activeDriveInfo.DriveName)
}

func getDriveOptionList() (config.DriveInfoList, string) {
	activeUser := config.Config.ActiveUser()

	driveList := activeUser.DriveList
	builder := &strings.Builder{}
	tb := cmdtable.NewTable(builder)
	tb.SetColumnAlignment([]int{tablewriter.ALIGN_DEFAULT, tablewriter.ALIGN_RIGHT, tablewriter.ALIGN_CENTER})
	tb.SetHeader([]string{"#", "drive_id", "网盘名称"})

	for k, info := range driveList {
		tb.Append([]string{strconv.Itoa(k + 1), info.DriveId, info.DriveName})
	}
	tb.Render()
	return driveList, builder.String()
}
