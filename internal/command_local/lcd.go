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
	"github.com/tickstep/aliyunpan/internal/command"
	"github.com/tickstep/aliyunpan/internal/config"
	"github.com/urfave/cli"
	"os"
	"runtime"
	"strings"
)

// CmdLocalCd 切换本地工作目录
func CmdLocalCd() cli.Command {
	return cli.Command{
		Name:     "lcd",
		Category: "本地命令",
		Usage:    "切换本地工作目录",
		Description: `
	aliyunpan lcd <本地目录, 绝对路径或相对路径>

	示例:

	切换 /我的资源 工作目录:
	aliyunpan lcd /我的资源

	切换上级目录:
	aliyunpan lcd ..

	切换根目录:
	aliyunpan lcd /
`,
		Before: command.ReloadConfigFunc,
		After:  command.SaveConfigFunc,
		Action: func(c *cli.Context) error {
			if c.NArg() == 0 {
				cli.ShowCommandHelp(c, c.Command.Name)
				return nil
			}
			RunChangeLocalDirectory(c.Args().Get(0))
			return nil
		},
		Flags: []cli.Flag{},
	}
}

// CmdLocalPwd 当前本地工作目录
func CmdLocalPwd() cli.Command {
	return cli.Command{
		Name:     "lpwd",
		Usage:    "输出本地工作目录",
		Category: "本地命令",
		Before:   command.ReloadConfigFunc,
		Action: func(c *cli.Context) error {
			lwd := config.Config.LocalWorkdir
			if lwd == "" {
				// 默认为用户主页目录
				lwd = GetLocalHomeDir()
				config.Config.LocalWorkdir = lwd
			}
			if runtime.GOOS == "windows" {
				lwd = strings.ReplaceAll(lwd, "/", "\\")
			} else {
				// Unix-like system, so just assume Unix
				lwd = strings.ReplaceAll(lwd, "\\", "/")
			}
			fmt.Println("本地工作目录: " + lwd)
			return nil
		},
	}
}

func RunChangeLocalDirectory(targetPath string) {
	targetPath = LocalPathJoin(targetPath)

	// 获取目标路径文件信息
	localFileInfo, er := os.Stat(targetPath)
	if er != nil {
		fmt.Println("目录路径不存在")
		return
	}
	if !localFileInfo.IsDir() {
		fmt.Printf("错误: %s 不是一个目录 (文件夹)\n", targetPath)
		return
	}
	config.Config.LocalWorkdir = strings.ReplaceAll(targetPath, "\\", "/")
	fmt.Printf("改变本地工作目录: %s\n", config.Config.LocalWorkdir)
}
