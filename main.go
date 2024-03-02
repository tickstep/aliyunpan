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
package main

import (
	"fmt"
	"github.com/tickstep/aliyunpan/internal/global"
	"io/ioutil"
	"os"
	"os/exec"
	"path"
	"path/filepath"
	"runtime"
	"sort"
	"strconv"
	"strings"
	"time"
	"unicode"

	"github.com/olekukonko/tablewriter"
	"github.com/peterh/liner"
	"github.com/tickstep/aliyunpan/cmder"
	"github.com/tickstep/aliyunpan/cmder/cmdliner"
	"github.com/tickstep/aliyunpan/cmder/cmdliner/args"
	"github.com/tickstep/aliyunpan/cmder/cmdtable"
	"github.com/tickstep/aliyunpan/cmder/cmdutil"
	"github.com/tickstep/aliyunpan/cmder/cmdutil/escaper"
	"github.com/tickstep/aliyunpan/internal/command"
	"github.com/tickstep/aliyunpan/internal/config"
	"github.com/tickstep/aliyunpan/internal/panupdate"
	"github.com/tickstep/aliyunpan/internal/utils"
	"github.com/tickstep/library-go/converter"
	"github.com/tickstep/library-go/logger"
	"github.com/urfave/cli"
)

const (
	// NameShortDisplayNum 文件名缩略显示长度
	NameShortDisplayNum = 16
)

var (
	// Version 版本号
	Version = "v0.2.9"

	// 命令历史文件
	historyFilePath = filepath.Join(config.GetConfigDir(), "aliyunpan_command_history.txt")

	// 是否是交互命令行形态
	isCli bool
)

func init() {
	global.AppVersion = Version
	cmdutil.ChWorkDir()

	err := config.Config.Init()
	switch err {
	case nil:
	case config.ErrConfigFileNoPermission, config.ErrConfigContentsParseError:
		fmt.Fprintf(os.Stderr, "FATAL ERROR: config file error: %s\n", err)
		//os.Exit(1)
	default:
		fmt.Printf("WARNING: config init error: %s\n", err)
	}
}

func checkLoginExpiredAndRelogin() {
	command.ReloadConfigFunc(nil)
	activeUser := config.Config.ActiveUser()
	if activeUser == nil || activeUser.UserId == "" {
		// maybe expired, try to login
		command.TryLogin()
	} else {
		// 刷新过期Token并保存到配置文件
		command.RefreshWebTokenInNeed(activeUser, config.Config.DeviceName)
	}
	command.SaveConfigFunc(nil)
}

func main() {
	defer config.Config.Close()

	// check & relogin
	checkLoginExpiredAndRelogin()

	// check token expired task
	go func() {
		for {
			// Token刷新进程，不管是CLI命令行模式，还是直接命令模式，本刷新任务都会执行
			time.Sleep(time.Duration(1) * time.Minute)
			//time.Sleep(time.Duration(5) * time.Second)
			checkLoginExpiredAndRelogin()
		}
	}()

	app := cli.NewApp()
	cmder.SetApp(app)

	app.Name = "aliyunpan"
	app.Version = Version
	app.Author = "tickstep/aliyunpan: https://github.com/tickstep/aliyunpan"
	app.Copyright = "(c) 2021-2024 tickstep."
	app.Usage = "阿里云盘客户端 for " + runtime.GOOS + "/" + runtime.GOARCH
	app.Description = `aliyunpan 是一款阿里云盘命令行客户端工具, 为操作阿里云盘, 提供实用功能。
	支持同步备份功能，支持备份本地文件到云盘，备份云盘文件到本地，双向同步备份。
	具体功能, 参见 COMMANDS 列表。

	支持设置环境变量 ALIYUNPAN_CONFIG_DIR 更改配置文件存储路径：
	export ALIYUNPAN_CONFIG_DIR=/etc/aliyunpan/config

	------------------------------------------------------------------------------
	前往 https://github.com/tickstep/aliyunpan 以获取更多帮助信息!
	前往 https://github.com/tickstep/aliyunpan/releases 以获取程序更新信息!
	------------------------------------------------------------------------------

	交流反馈:
		提交Issue: https://github.com/tickstep/aliyunpan/issues
		联系邮箱: tickstep@outlook.com`

	// 全局options
	app.Flags = []cli.Flag{
		cli.BoolFlag{
			Name:        "verbose",
			Usage:       "启用调试",
			EnvVar:      config.EnvVerbose,
			Destination: &logger.IsVerbose,
		},
	}

	// 进入交互CLI命令行界面
	app.Action = func(c *cli.Context) {
		if c.NArg() != 0 {
			fmt.Printf("未找到命令: %s\n运行命令 %s help 获取帮助\n", c.Args().Get(0), app.Name)
			return
		}

		os.Setenv(config.EnvVerbose, c.String("verbose"))
		isCli = true
		global.IsAppInCliMode = true
		logger.Verbosef("提示: 你已经开启VERBOSE调试日志\n\n")

		var (
			line = cmdliner.NewLiner()
			err  error
		)

		line.History, err = cmdliner.NewLineHistory(historyFilePath)
		if err != nil {
			fmt.Printf("警告: 读取历史命令文件错误, %s\n", err)
		}

		line.ReadHistory()
		defer func() {
			line.DoWriteHistory()
			line.Close()
		}()

		// tab 自动补全命令
		line.State.SetCompleter(func(line string) (s []string) {
			var (
				lineArgs                   = args.Parse(line)
				numArgs                    = len(lineArgs)
				acceptCompleteFileCommands = []string{
					"cd", "cp", "xcp", "download", "ls", "mkdir", "mv", "pwd", "rename", "rm", "share", "save", "upload", "login", "loglist", "logout",
					"clear", "quit", "exit", "quota", "who", "sign", "update", "who", "su", "config",
					"drive", "export", "import", "sync", "tree",
				}
				closed = strings.LastIndex(line, " ") == len(line)-1
			)

			for _, cmd := range app.Commands {
				for _, name := range cmd.Names() {
					if !strings.HasPrefix(name, line) {
						continue
					}

					s = append(s, name+" ")
				}
			}

			switch numArgs {
			case 0:
				return
			case 1:
				if !closed {
					return
				}
			}

			thisCmd := app.Command(lineArgs[0])
			if thisCmd == nil {
				return
			}

			if !cmdutil.ContainsString(acceptCompleteFileCommands, thisCmd.FullName()) {
				return
			}

			var (
				activeUser  = config.Config.ActiveUser()
				runeFunc    = unicode.IsSpace
				cmdRuneFunc = func(r rune) bool {
					switch r {
					case '\'', '"':
						return true
					}
					return unicode.IsSpace(r)
				}
				targetPath string
			)

			if activeUser == nil {
				return
			}

			if !closed {
				targetPath = lineArgs[numArgs-1]
				escaper.EscapeStringsByRuneFunc(lineArgs[:numArgs-1], runeFunc) // 转义
			} else {
				escaper.EscapeStringsByRuneFunc(lineArgs, runeFunc)
			}

			switch {
			case targetPath == "." || strings.HasSuffix(targetPath, "/."):
				s = append(s, line+"/")
				return
			case targetPath == ".." || strings.HasSuffix(targetPath, "/.."):
				s = append(s, line+"/")
				return
			}

			var (
				targetDir string
				isAbs     = path.IsAbs(targetPath)
				isDir     = strings.LastIndex(targetPath, "/") == len(targetPath)-1
			)

			if isAbs {
				targetDir = path.Dir(targetPath)
			} else {
				wd := "/"
				if activeUser.IsFileDriveActive() {
					wd = activeUser.Workdir
				} else if activeUser.IsResourceDriveActive() {
					wd = activeUser.ResourceWorkdir
				} else if activeUser.IsAlbumDriveActive() {
					wd = activeUser.AlbumWorkdir
				}
				targetDir = path.Join(wd, targetPath)
				if !isDir {
					targetDir = path.Dir(targetDir)
				}
			}

			// tab键补全路径
			files, err := activeUser.CacheFilesDirectoriesList(targetDir)
			if err != nil {
				return
			}
			for _, file := range files {
				if file == nil {
					continue
				}

				var (
					appendLine string
				)

				// 已经有的情况
				if !closed {
					if !strings.HasPrefix(file.Path, path.Clean(path.Join(targetDir, path.Base(targetPath)))) {
						if path.Base(targetDir) == path.Base(targetPath) {
							appendLine = strings.Join(append(lineArgs[:numArgs-1], escaper.EscapeByRuneFunc(path.Join(targetPath, file.FileName), cmdRuneFunc)), " ")
							goto handle
						}
						continue
					}
					appendLine = strings.Join(append(lineArgs[:numArgs-1], escaper.EscapeByRuneFunc(path.Clean(path.Join(path.Dir(targetPath), file.FileName)), cmdRuneFunc)), " ")
					goto handle
				}
				// 没有的情况
				appendLine = strings.Join(append(lineArgs, escaper.EscapeByRuneFunc(file.FileName, cmdRuneFunc)), " ")
				goto handle

			handle:
				if file.IsFolder() {
					s = append(s, appendLine+"/")
					continue
				}
				s = append(s, appendLine+" ")
				continue
			}
			return
		})

		fmt.Printf("提示: 方向键上下可切换历史命令.\n")
		fmt.Printf("提示: Ctrl + A / E 跳转命令 首 / 尾.\n")
		fmt.Printf("提示: 输入 help 获取帮助.\n")

		// check update
		command.ReloadConfigFunc(c)
		if config.Config.UpdateCheckInfo.LatestVer != "" {
			if utils.ParseVersionNum(config.Config.UpdateCheckInfo.LatestVer) > utils.ParseVersionNum(global.AppVersion) {
				fmt.Printf("\n当前的软件版本为：%s， 现在有新版本 %s 可供更新，强烈推荐进行更新！（可以输入 update 命令进行更新）\n\n",
					global.AppVersion, config.Config.UpdateCheckInfo.LatestVer)
			}
		}
		go func() {
			latestCheckTime := config.Config.UpdateCheckInfo.CheckTime
			nowTime := time.Now().Unix()
			secsOf12Hour := int64(43200)
			if (nowTime - latestCheckTime) > secsOf12Hour {
				releaseInfo := panupdate.GetLatestReleaseInfo(false)
				if releaseInfo == nil {
					logger.Verboseln("获取版本信息失败!")
					return
				}
				config.Config.UpdateCheckInfo.LatestVer = releaseInfo.TagName
				config.Config.UpdateCheckInfo.CheckTime = nowTime

				// save
				command.SaveConfigFunc(c)
			}
		}()

		for {
			var (
				prompt     string
				activeUser = config.Config.ActiveUser()
			)

			if activeUser == nil {
				activeUser = command.TryLogin()
			}

			if activeUser != nil && activeUser.Nickname != "" {
				// 格式: aliyunpan:<工作目录> <UserName>$
				// 工作目录太长时, 会自动缩略
				wd := "/"
				if activeUser.IsFileDriveActive() {
					wd = activeUser.Workdir
					prompt = app.Name + ":" + converter.ShortDisplay(path.Base(wd), NameShortDisplayNum) + " " + activeUser.Nickname + "(备份盘)$ "
				} else if activeUser.IsResourceDriveActive() {
					wd = activeUser.ResourceWorkdir
					prompt = app.Name + ":" + converter.ShortDisplay(path.Base(wd), NameShortDisplayNum) + " " + activeUser.Nickname + "(资源库)$ "
				} else if activeUser.IsAlbumDriveActive() {
					wd = activeUser.AlbumWorkdir
					prompt = app.Name + ":" + converter.ShortDisplay(path.Base(wd), NameShortDisplayNum) + " " + activeUser.Nickname + "(相册)$ "
				}

			} else {
				// aliyunpan >
				prompt = app.Name + " > "
			}

			commandLine, err := line.State.Prompt(prompt)
			switch err {
			case liner.ErrPromptAborted:
				return
			case nil:
				// continue
			default:
				fmt.Println(err)
				return
			}

			line.State.AppendHistory(commandLine)

			cmdArgs := args.Parse(commandLine)
			if len(cmdArgs) == 0 {
				continue
			}

			s := []string{os.Args[0]}
			s = append(s, cmdArgs...)

			// 恢复原始终端状态
			// 防止运行命令时程序被结束, 终端出现异常
			line.Pause()
			c.App.Run(s)
			line.Resume()
		}
	}

	// 命令配置和对应的处理func
	app.Commands = []cli.Command{
		// 登录账号 login
		command.CmdLogin(),

		// 退出登录帐号 logout
		command.CmdLogout(),

		// 列出帐号列表 loglist
		command.CmdLoglist(),

		// 切换网盘 drive
		command.CmdDrive(),

		// 切换阿里账号 su
		command.CmdSu(),

		// 获取当前帐号 who
		command.CmdWho(),

		// 获取当前帐号空间配额 quota
		command.CmdQuota(),

		// Token操作
		//command.CmdToken(),

		// 切换工作目录 cd
		command.CmdCd(),

		// 输出工作目录 pwd
		command.CmdPwd(),

		// 列出目录 ls
		command.CmdLs(),

		// 显示树形目录 tree
		command.CmdTree(),

		// 创建目录 mkdir
		command.CmdMkdir(),

		// 删除文件/目录 rm
		command.CmdRm(),

		//// 拷贝文件/目录 cp
		//command.CmdCp(),

		// 备份盘和资源库之间拷贝文件 xcp
		command.CmdXcp(),

		// 移动文件/目录 mv
		command.CmdMv(),

		// 重命名文件 rename
		command.CmdRename(),

		// 分享文件/目录 share
		command.CmdShare(),

		// 保存分享文件/目录 save
		command.CmdSave(),

		// 同步备份 sync
		command.CmdSync(),

		// 上传文件/目录 upload
		command.CmdUpload(),

		// 手动秒传
		//command.CmdRapidUpload(),

		// 下载文件/目录 download
		command.CmdDownload(),

		// 获取文件下载链接
		//command.CmdLocateUrl(),

		// 导出文件/目录元数据 export
		//command.CmdExport(),

		// 导入文件 import
		//command.CmdImport(),

		// webdav服务(depressed)
		//command.CmdWebdav(),

		// 回收站
		command.CmdRecycle(),

		// 显示和修改程序配置项 config
		command.CmdConfig(),

		// 工具箱 tool
		command.CmdTool(),

		// 相簿
		command.CmdAlbum(),

		// 显示命令历史
		{
			Name:      "history",
			Aliases:   []string{},
			Usage:     "显示命令历史",
			UsageText: app.Name + " history",
			Description: `显示命令历史

		示例:
		1. 显示最近命令历史
		aliyunpan history

		2. 显示最近10条命令历史
		aliyunpan history -n 10

		3. 显示全部命令历史
		aliyunpan history -n 0
`,
			Category: "其他",
			Action: func(c *cli.Context) error {
				lineCount := 20
				if c.IsSet("n") {
					lineCount = c.Int("n")
				}
				printTable := func(lines []string) {
					tb := cmdtable.NewTable(os.Stdout)
					tb.SetHeader([]string{"序号", "命令"})
					tb.SetHeaderAlignment(tablewriter.ALIGN_LEFT)
					tb.SetColumnAlignment([]int{tablewriter.ALIGN_LEFT, tablewriter.ALIGN_LEFT})
					idx := 1
					for _, line := range lines {
						if line == "" {
							continue
						}
						tb.Append([]string{strconv.Itoa(idx + 1), line})
						idx++
					}
					tb.Render()
				}
				if contents, err := ioutil.ReadFile(historyFilePath); err == nil {
					result := strings.Split(string(contents), "\n")
					if lineCount == 0 {
						printTable(result)
					} else {
						outputLine := make([]string, 0)
						for idx := len(result) - 1; idx >= 0; idx-- {
							line := result[idx]
							if line != "" {
								outputLine = append(outputLine, line)
							}
							if len(outputLine) >= lineCount {
								break
							}
						}
						lines := make([]string, 0)
						for idx := len(outputLine) - 1; idx >= 0; idx-- {
							lines = append(lines, outputLine[idx])
						}
						printTable(lines)
					}
				}
				return nil
			},
			Flags: []cli.Flag{
				cli.IntFlag{
					Name:  "n",
					Usage: "显示最近历史的行数。0-代表全部，默认为20",
					Value: 20,
				},
			},
		},

		// 清空控制台 clear
		{
			Name:        "clear",
			Aliases:     []string{"cls"},
			Usage:       "清空控制台",
			UsageText:   app.Name + " clear",
			Description: "清空控制台屏幕",
			Category:    "其他",
			Action: func(c *cli.Context) error {
				cmdliner.ClearScreen()
				return nil
			},
		},

		// 检测程序更新 update
		{
			Name:     "update",
			Usage:    "检测程序更新",
			Category: "其他",
			Action: func(c *cli.Context) error {
				if c.IsSet("y") {
					if !c.Bool("y") {
						return nil
					}
				}
				panupdate.CheckUpdate(app.Version, c.Bool("y"))
				return nil
			},
			Flags: []cli.Flag{
				cli.BoolFlag{
					Name:  "y",
					Usage: "确认更新",
				},
			},
		},

		// 退出程序 quit
		{
			Name:        "quit",
			Aliases:     []string{"exit"},
			Usage:       "退出程序",
			Description: "退出程序",
			Category:    "其他",
			Action: func(c *cli.Context) error {
				return cli.NewExitError("", 0)
			},
			Hidden:   true,
			HideHelp: true,
		},

		// 执行系统命令
		{
			Name:     "run",
			Usage:    "执行系统命令",
			Category: "其他",
			Action: func(c *cli.Context) error {
				if c.NArg() == 0 {
					cli.ShowCommandHelp(c, c.Command.Name)
					return nil
				}

				cmd := exec.Command(c.Args().First(), c.Args().Tail()...)
				cmd.Stdout = os.Stdout
				cmd.Stdin = os.Stdin
				cmd.Stderr = os.Stderr
				err := cmd.Run()
				if err != nil {
					fmt.Println(err)
				}
				return nil
			},
		},

		// 调试用 debug
		//{
		//	Name:        "debug",
		//	Aliases:     []string{"dg"},
		//	Usage:       "开发调试用",
		//	Description: "",
		//	Category:    "debug",
		//	Before:      cmder.ReloadConfigFunc,
		//	Action: func(c *cli.Context) error {
		//		os.Setenv(config.EnvVerbose, "1")
		//		logger.IsVerbose = true
		//		fmt.Println("显示调试日志", logger.IsVerbose)
		//		return nil
		//	},
		//	Flags: []cli.Flag{
		//		cli.StringFlag{
		//			Name:  "param",
		//			Usage: "参数",
		//		},
		//		cli.BoolFlag{
		//			Name:        "verbose",
		//			Destination: &logger.IsVerbose,
		//			EnvVar:      config.EnvVerbose,
		//			Usage:       "显示调试信息",
		//		},
		//	},
		//},
	}

	sort.Sort(cli.FlagsByName(app.Flags))
	sort.Sort(cli.CommandsByName(app.Commands))
	app.Run(os.Args)
}
