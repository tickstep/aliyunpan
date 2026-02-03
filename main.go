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
	"github.com/olekukonko/tablewriter"
	"github.com/peterh/liner"
	"github.com/tickstep/aliyunpan/cmder"
	"github.com/tickstep/aliyunpan/cmder/cmdliner"
	"github.com/tickstep/aliyunpan/cmder/cmdliner/args"
	"github.com/tickstep/aliyunpan/cmder/cmdtable"
	"github.com/tickstep/aliyunpan/cmder/cmdutil"
	"github.com/tickstep/aliyunpan/cmder/cmdutil/escaper"
	"github.com/tickstep/aliyunpan/internal/command"
	"github.com/tickstep/aliyunpan/internal/command_local"
	"github.com/tickstep/aliyunpan/internal/config"
	"github.com/tickstep/aliyunpan/internal/global"
	"github.com/tickstep/aliyunpan/internal/panupdate"
	"github.com/tickstep/aliyunpan/internal/utils"
	"github.com/tickstep/aliyunpan/library/homedir"
	"github.com/tickstep/library-go/converter"
	"github.com/tickstep/library-go/logger"
	"github.com/urfave/cli"
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
)

const (
	// NameShortDisplayNum 文件名缩略显示长度
	NameShortDisplayNum = 16
)

var (
	// Version 版本号
	Version = "v0.3.7"

	// 命令历史文件
	historyFilePath = filepath.Join(config.GetConfigDir(), "aliyunpan_command_history.txt")

	// 是否是交互命令行形态
	isCli bool
)

func init() {
	b, ok := os.LookupEnv("ALIYUNPAN_NONE_OPENAPI")
	if ok && b == "1" {
		global.IsSupportNoneOpenApiCommands = true
	}

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
	// 尝试登录
	activeUser := command.TryLogin()
	if activeUser != nil && activeUser.UserId != "" {
		// 刷新过期Token并保存到配置文件
		command.RefreshWebTokenInNeed(activeUser)
		command.RefreshOpenTokenInNeed(activeUser)
		command.SaveConfigFunc(nil)
	}
}

func main() {
	defer config.Config.Close()

	// check & relogin
	checkLoginExpiredAndRelogin()

	// check token expired task
	command.AutomaticallyRefreshTokenTask() // Token刷新进程，不管是CLI命令行模式，还是直接命令模式，本刷新任务都会执行

	app := cli.NewApp()
	cmder.SetApp(app)

	app.Name = "aliyunpan"
	app.Version = Version
	app.Author = "tickstep/aliyunpan: https://github.com/tickstep/aliyunpan"
	app.Copyright = "(c) 2021-2025 tickstep."
	app.Usage = "阿里云盘客户端 for " + runtime.GOOS + "/" + runtime.GOARCH
	app.Description = `aliyunpan 是一款阿里云盘命令行客户端工具, 为操作阿里云盘, 提供实用功能。
	支持同步备份功能，支持备份本地文件到云盘，备份云盘文件到本地。
	具体功能, 参见 COMMANDS 列表。

	支持设置环境变量 ALIYUNPAN_CONFIG_DIR 更改配置文件存储路径：
	export ALIYUNPAN_CONFIG_DIR=/etc/aliyunpan/config
	
	支持XDG目录规范：
	默认XDG配置目录：$XDG_CONFIG_HOME/aliyunpan
	默认XDG下载目录：$XDG_DOWNLOAD_DIR
	
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
			// 命令补全处理器
			// line: 输入的命令行完整字符，包括命令名称、空格、（完整或部分）路径字符
			// s: 返回的满足符合条件的提示列表，命令名称、或者文件完整路径

			var (
				lineArgs = args.Parse(line)
				numArgs  = len(lineArgs)
				// 支持TAB补全文件路径的命令
				acceptCompleteFilePanCommands = []string{ // 云盘命令
					"cd", "cp", "xcp", "download", "ls", "mkdir", "mv", "rename", "rm", "upload", "tree",
				}
				acceptCompleteFileLocalCommands = []string{ // 本地命令
					"lcd", "lls",
				}
				closed = false
			)
			// 检查是否是命令输入的结尾
			if strings.LastIndex(line, " ") == len(line)-1 {
				closed = true
			}

			// 开始补全命令名称
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

			// 开始补全文件路径
			thisCmd := app.Command(lineArgs[0])
			if thisCmd == nil {
				return
			}

			// 检测输入的命令是否支持tab补全路径
			if cmdutil.ContainsString(acceptCompleteFilePanCommands, thisCmd.FullName()) {
				// 云盘文件路径的补全
				if thisCmd.FullName() == "ls" || thisCmd.FullName() == "cd" {
					// 只需要补全文件夹路径
					return panFilePathCompleter(line, lineArgs, numArgs, true)
				} else {
					// 文件夹、文件都需要
					return panFilePathCompleter(line, lineArgs, numArgs, false)
				}
			} else if cmdutil.ContainsString(acceptCompleteFileLocalCommands, thisCmd.FullName()) {
				// 本地文件路径的补全
				return localFilePathCompleter(line, lineArgs, numArgs, true)
			}
			return
		})

		fmt.Printf("提示: 方向键上下可切换历史命令.\n")
		fmt.Printf("提示: Ctrl + A / E 跳转命令 首 / 尾.\n")
		fmt.Printf("提示: 输入 help 获取帮助.\n")

		// 刷新配置文件
		command.ReloadConfigFunc(c)

		// check update
		if config.Config.UpdateCheckInfo.LatestVer != "" {
			if utils.ParseVersionNum(config.Config.UpdateCheckInfo.LatestVer) > utils.ParseVersionNum(global.AppVersion) {
				fmt.Printf("\n当前的软件版本为：%s， 现在有新版本 %s 可供更新，强烈推荐进行更新！（可以输入 update 命令进行更新）\n\n",
					global.AppVersion, config.Config.UpdateCheckInfo.LatestVer)
			}
		}
		go func() {
			// 新版本检测异步任务
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

		for { // 命令行交互进程，每一次命令执行都会进入到这个for循环
			var (
				prompt     string
				activeUser = config.Config.ActiveUser()
			)

			if activeUser == nil {
				// 尝试重新登录并获取新的token
				activeUser = command.TryLogin()

				// 自动登录成功
				if activeUser != nil {
					logger.Verboseln("自动重新登录成功")
					// 保存新token到配置文件
					command.SaveConfigFunc(c)
				}
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

			commandLine, err1 := line.State.Prompt(prompt)
			switch err1 {
			case liner.ErrPromptAborted:
				return
			case nil:
				// continue
			default:
				fmt.Println(err1)
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

		// 复制文件/目录 cp
		command.CmdCp(),

		// 移动文件/目录 mv
		command.CmdMv(),

		// 重命名文件 rename
		command.CmdRename(),

		// 同步备份 sync
		command.CmdSync(),

		// 上传文件/目录 upload
		command.CmdUpload(),

		// 下载文件/目录 download
		command.CmdDownload(),

		// 显示和修改程序配置项 config
		command.CmdConfig(),

		// 工具箱 tool
		command.CmdTool(),

		// 分享文件/目录 share
		command.CmdShare(),

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

		// 显示程序环境变量
		{
			Name:  "env",
			Usage: "显示程序环境变量",
			Description: `	ALIYUNPAN_CONFIG_DIR: 配置文件路径
	ALIYUNPAN_DOWNLOAD_DIR: 配置下载路径
	ALIYUNPAN_VERBOSE: 是否启用调试
	XDG_CONFIG_HOME: XDG配置主目录
	XDG_DOWNLOAD_DIR: XDG配置下载目录
`,
			Category: "其他",
			Action: func(c *cli.Context) error {
				envStr := "%s=%s\n"
				envVar, ok := os.LookupEnv(config.EnvVerbose)
				if ok {
					if envVar == "1" {
						fmt.Printf(envStr, config.EnvVerbose, "1")
					} else {
						fmt.Printf(envStr, config.EnvVerbose, "0")
					}
				} else {
					fmt.Printf(envStr, config.EnvVerbose, "0")
				}

				envVar, ok = os.LookupEnv(config.EnvConfigDir)
				if ok {
					fmt.Printf(envStr, config.EnvConfigDir, envVar)
				} else {
					fmt.Printf(envStr, config.EnvConfigDir, config.GetConfigDir())
				}

				envVar, ok = os.LookupEnv(config.EnvDownloadDir)
				if ok {
					fmt.Printf(envStr, config.EnvDownloadDir, envVar)
				} else {
					fmt.Printf(envStr, config.EnvDownloadDir, config.GetDefaultDownloadDir())
				}

				envVar, ok = os.LookupEnv("XDG_CONFIG_HOME")
				if ok {
					fmt.Printf(envStr, "XDG_CONFIG_HOME", envVar)
				} else {
					fmt.Printf(envStr, "XDG_CONFIG_HOME", "")
				}
				envVar, ok = os.LookupEnv("XDG_DOWNLOAD_DIR")
				if ok {
					fmt.Printf(envStr, "XDG_DOWNLOAD_DIR", envVar)
				} else {
					fmt.Printf(envStr, "XDG_DOWNLOAD_DIR", "")
				}
				return nil
			},
		},

		// 当前本地工作目录
		command_local.CmdLocalPwd(),
		// 切换本地工作目录
		command_local.CmdLocalCd(),
		// 展示本地目录文件列表
		command_local.CmdLocalLs(),

		// 调试用 debug
		//{
		//	Name:        "debug",
		//	Aliases:     []string{"dg"},
		//	Usage:       "开发调试用",
		//	Description: "",
		//	Category:    "debug",
		//	Before:      command.ReloadConfigFunc,
		//	Action: func(c *cli.Context) error {
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

	// 隐藏不支持的命令
	if global.IsSupportNoneOpenApiCommands {
		hiddenCommands := []cli.Command{
			// 备份盘和资源库之间拷贝文件 xcp
			command.CmdXcp(),

			// 分享文件/目录 sharew，web端接口的分享功能
			command.CmdShareWeb(),

			// 保存分享文件/目录 save
			command.CmdSave(),

			// 回收站
			command.CmdRecycle(),

			// 相簿
			//command.CmdAlbum(),
		}
		app.Commands = append(app.Commands, hiddenCommands...)
	}
	sort.Sort(cli.FlagsByName(app.Flags))
	sort.Sort(cli.CommandsByName(app.Commands))
	app.Run(os.Args)
}

// panFilePathCompleter 云盘文件路径Tab补全处理器
func panFilePathCompleter(line string, lineArgs []string, numArgs int, onlyNeedFolder bool) (s []string) {
	return filePathCompleterProcessor("pan", line, lineArgs, numArgs, onlyNeedFolder, getPanFileListByTargetPath)
}

// localFilePathCompleter 本地文件路径Tab补全处理器
func localFilePathCompleter(line string, lineArgs []string, numArgs int, onlyNeedFolder bool) (s []string) {
	return filePathCompleterProcessor("local", line, lineArgs, numArgs, onlyNeedFolder, getLocalFileListByTargetPath)
}

// filePathCompleterProcessor 路径自动补充处理器
// 典型测试样例
// lls d:\
// lls ~/
// lls ~\
// lls ~\Doc
// ls ~/
// ls<空格><tab>
// ls app<tab>
// ls /doc<tab>
func filePathCompleterProcessor(targetType string, line string, lineArgs []string, numArgs int, onlyNeedFolder bool, fileListFunc getFileListByTargetPath) (s []string) {
	var (
		runeFunc    = unicode.IsSpace
		cmdRuneFunc = func(r rune) bool {
			switch r {
			case '\'', '"':
				return true
			}
			return unicode.IsSpace(r)
		}
		targetPath string
		// 路径的结尾是否是有前缀，如果有前缀则文件、文件夹名称需要匹配前缀
		endPathHasPrefixStr = false
	)
	if numArgs > 1 {
		targetPath = lineArgs[numArgs-1]
	} else {
		// 没有输入文件路径的时候，默认为当前目录
		targetPath = "./"
		line += targetPath
		lineArgs = append(lineArgs, targetPath)
		numArgs = 2
	}
	switch {
	case targetPath == "." || strings.HasSuffix(targetPath, "/."):
		s = append(s, line+"/")
		return
	case targetPath == ".." || strings.HasSuffix(targetPath, "/.."):
		s = append(s, line+"/")
		return
	}
	// 主目录"~"特殊路径处理
	if strings.HasPrefix(targetPath, "~") {
		if targetType == "local" {
			if p, e := homedir.Expand(targetPath); e == nil {
				targetPath = p
			}
		} else {
			// 云盘的主目录~默认为根目录
			targetPath = strings.ReplaceAll(strings.Replace(targetPath, "~", "/", 1), "//", "/")
		}
	}
	// 完整路径是需要“/”作为结尾，否则默认用户只输入了一部分文件名，输入的部分字符作为文件名前缀进行搜索匹配
	// 如果用户输入：/Users/tickstep/doc，则自动补全的路径需要在/Users/tickstep/目录下搜索并且文件名必须包含doc这个前缀，例如：/Users/tickstep/document/ 和 /Users/tickstep/doc/ 都是匹配项
	// 如果用户输入：/Users/tickstep/，则自动补全的路径需要在/Users/tickstep/目录下搜索即可，例如：/Users/tickstep/photo/ 和 /Users/tickstep/doc/ 都是匹配项
	if strings.HasSuffix(targetPath, "/") {
		endPathHasPrefixStr = false
	} else {
		endPathHasPrefixStr = true
	}

	if endPathHasPrefixStr {
		escaper.EscapeStringsByRuneFunc(lineArgs[:numArgs-1], runeFunc) // 转义
	} else {
		escaper.EscapeStringsByRuneFunc(lineArgs, runeFunc)
	}

	targetFullPath, targetDir, files := fileListFunc(targetPath)
	for _, file := range files {
		if file == nil {
			continue
		}
		if onlyNeedFolder && !file.IsFolder {
			// 过滤文件，只需要文件夹
			continue
		}

		var (
			appendLine string
		)

		// 用户已经输入了部分路径字符，搜索匹配文件路径需要匹配用户输入的路径字符串前缀
		if endPathHasPrefixStr {
			fp := command_local.LocalPathClean(file.Path)
			tp := command_local.LocalPathClean(targetFullPath)
			if !strings.HasPrefix(fp, tp) {
				if command_local.LocalPathBase(targetDir) == command_local.LocalPathBase(targetFullPath) {
					appendLine = strings.Join(append(lineArgs[:numArgs-1], escaper.EscapeByRuneFunc(path.Join(targetPath, file.FileName), cmdRuneFunc)), " ")
					goto handle
				}
				continue
			}
			appendLine = strings.Join(append(lineArgs[:numArgs-1], escaper.EscapeByRuneFunc(command_local.LocalPathClean(path.Join(command_local.LocalPathDir(targetPath), file.FileName)), cmdRuneFunc)), " ")
			goto handle
		}
		// 没有输入任何路径前缀字符，默认都符合条件，全部显示到提示列表
		appendLine = strings.Join(append(lineArgs[:numArgs-1], escaper.EscapeByRuneFunc(command_local.LocalPathClean(path.Join(targetPath, file.FileName)), cmdRuneFunc)), " ")
		goto handle

	handle:
		if file.IsFolder {
			s = append(s, appendLine+"/")
			continue
		}
		s = append(s, appendLine+" ")
		continue
	}

	// 没有命中的路径，返回当前用户输入的内容即可
	if len(s) == 0 {
		s = append(s, line)
	}
	return
}

type fileEntity struct {
	FileName string
	Path     string
	IsFolder bool
}
type getFileListByTargetPath func(targetPath string) (string, string, []*fileEntity)

// getPanFileListByTargetPath 通过路径获取云盘文件列表
func getPanFileListByTargetPath(targetPath string) (string, string, []*fileEntity) {
	var (
		activeUser     = config.Config.ActiveUser()
		targetDir      string
		targetFullPath string
		isAbs          = path.IsAbs(targetPath)
		isDir          = strings.LastIndex(targetPath, "/") == len(targetPath)-1
		r              = []*fileEntity{}
	)
	if activeUser == nil {
		return targetFullPath, targetDir, r
	}

	if isAbs {
		targetDir = path.Dir(targetPath)
		targetFullPath = targetPath
	} else {
		wd := "/"
		if activeUser.IsFileDriveActive() {
			wd = activeUser.Workdir
		} else if activeUser.IsResourceDriveActive() {
			wd = activeUser.ResourceWorkdir
		} else if activeUser.IsAlbumDriveActive() {
			wd = activeUser.AlbumWorkdir
		}
		targetFullPath = path.Join(wd, targetPath)
		targetDir = targetFullPath
		if !isDir {
			targetDir = path.Dir(targetDir)
		}
	}

	files, err := activeUser.CacheFilesDirectoriesList(targetDir)
	if err != nil {
		return targetFullPath, targetDir, r
	}

	for _, file := range files {
		if file == nil {
			continue
		}
		r = append(r, &fileEntity{
			FileName: file.FileName,
			Path:     file.Path,
			IsFolder: file.IsFolder(),
		})
	}
	return targetFullPath, targetDir, r
}

// getLocalFileListByTargetPath 通过路径获取本地文件列表
func getLocalFileListByTargetPath(targetPath string) (string, string, []*fileEntity) {
	targetPath = strings.ReplaceAll(targetPath, "\\", "/")
	var (
		targetDir      string
		targetFullPath string
		isAbs          = filepath.IsAbs(targetPath)
		isDir          = strings.HasSuffix(targetPath, "/")
		r              = []*fileEntity{}
	)

	if isAbs {
		// 绝对路径
		targetDir = targetPath
		targetFullPath = targetPath
		if !isDir {
			targetDir = command_local.LocalPathDir(targetDir)
		}
	} else {
		// 相对路径
		targetFullPath = command_local.LocalPathJoin(targetPath)
		targetDir = targetFullPath
		if !isDir {
			targetDir = command_local.LocalPathDir(targetDir)
		}
	}
	// windows
	if runtime.GOOS == "windows" {
		if strings.HasSuffix(targetDir, ":") {
			targetDir += "/"
		}
	}

	fileEntryList, err := os.ReadDir(targetDir)
	if err != nil {
		return targetFullPath, targetDir, r
	}
	for _, file := range fileEntryList {
		if file == nil {
			continue
		}

		if runtime.GOOS == "windows" { // windows隐藏文件
			if strings.HasPrefix(file.Name(), "$") {
				continue
			}
		} else { // linux / macOS隐藏文件
			if strings.HasPrefix(file.Name(), ".") {
				continue
			}
		}

		r = append(r, &fileEntity{
			FileName: file.Name(),
			Path:     path.Join(targetDir, file.Name()),
			IsFolder: file.IsDir(),
		})
	}
	return targetFullPath, targetDir, r
}
