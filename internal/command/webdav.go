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
	"github.com/tickstep/aliyunpan/cmder"
	"github.com/tickstep/aliyunpan/internal/config"
	"github.com/tickstep/aliyunpan/internal/webdav"
	"github.com/tickstep/library-go/logger"
	"github.com/urfave/cli"
	"strings"
	"sync/atomic"
	"time"
)

func CmdWebdav() cli.Command {
	return cli.Command{
		Name:        "webdav",
		Usage:       "在线网盘服务",
		Description: "webdav在线网盘服务",
		Category:    "阿里云盘",
		Before:      ReloadConfigFunc,
		After:       SaveConfigFunc,
		Action: func(c *cli.Context) error {
			fmt.Print(`
本文命令可以让阿里云盘变身为webdav协议的文件服务器。这样你可以把阿里云盘挂载为Windows、Linux、Mac系统的磁盘，可以通过NAS系统做文件管理或文件同步等等。
当把阿里云盘作为webdav文件服务器进行使用的时候，上传文件是不支持秒传的，所以当你挂载为网络磁盘使用的时候，不建议在webdav挂载目录中上传、下载过大的文件，不然体验会非常差。
建议作为文档，图片等小文件的同步网盘。

请输入以下命令查看如何启动
aliyunpan webdav start -h

`)
			cli.ShowCommandHelp(c, c.Command.Name)
			return nil
		},
		Subcommands: []cli.Command{
			{
				Name:      "start",
				Usage:     "启动webdav在线网盘服务",
				UsageText: cmder.App().Name + " webdav start [arguments...]",
				Description: `
启动webdav服务，让阿里云盘变身为webdav协议的文件服务器。这样你可以把阿里云盘挂载为Windows、Linux、Mac系统的磁盘，可以通过NAS系统做文件管理或文件同步等等。
当把阿里云盘作为webdav文件服务器进行使用的时候，上传文件是不支持秒传的，所以当你挂载为网络磁盘使用的时候，不建议在webdav挂载目录中上传、下载过大的文件，不然体验会非常差。
建议作为文档，图片等小文件的同步网盘。

	例子:
	1. 查看帮助
	aliyunpan webdav start -h

	2. 使用默认配置启动webdav服务
	aliyunpan webdav start

	3. 启动webdav服务，并配置IP为127.0.0.1，端口为23077，登录用户名为admin，登录密码为admin123，模式为读写，文件网盘目录 /webdav_folder 作为服务的根目录
	aliyunpan webdav start -ip "127.0.0.1" -port 23077 -webdav_user "admin" -webdav_password "admin123" -webdav_mode "rw" -pan_drive "File" -pan_dir_path "/webdav_folder"

`,
				Action: func(c *cli.Context) error {
					if config.Config.ActiveUser() == nil {
						fmt.Println("未登录账号，请先登录")
						return nil
					}
					activeUser := GetActiveUser()

					// pan token expired checker
					continueFlag := int32(0)
					atomic.StoreInt32(&continueFlag, 0)
					defer func() {
						atomic.StoreInt32(&continueFlag, 1)
					}()
					go func(flag *int32) {
						for atomic.LoadInt32(flag) == 0 {
							// token刷新
							time.Sleep(time.Duration(1) * time.Minute)
							//time.Sleep(time.Duration(5) * time.Second)
							if RefreshTokenInNeed(activeUser) {
								logger.Verboseln("reload new access token for webdav")
							}
						}
					}(&continueFlag)

					webdavServ := &webdav.WebdavConfig{
						PanDriveId:      "",
						PanUserId:       "",
						PanUser:         nil,
						UploadChunkSize: c.Int("bs") * 1024,
						TransferUrlType: config.Config.TransferUrlType,
						Address:         "0.0.0.0",
						Port:            23077,
						Prefix:          "/",
						Users: []webdav.WebdavUser{{
							Username: "admin",
							Password: "admin",
							Scope:    "/",
							Mode:     "rw",
						}},
					}

					// pan user
					panUserId := activeUser.UserId
					webdavServ.PanUserId = panUserId
					webdavServ.PanUser = activeUser

					// address
					ip := "0.0.0.0"
					if c.IsSet("ip") {
						ip = c.String("ip")
					}
					webdavServ.Address = ip

					// port
					port := 23077
					if c.IsSet("port") {
						port = c.Int("port")
					}
					webdavServ.Port = port

					// binding pan drive
					panDriveName := "File"
					panDriveNameStr := "文件"
					if c.IsSet("pan_drive") {
						panDriveName = c.String("pan_drive")
					}
					if strings.ToLower(panDriveName) == "album" {
						webdavServ.PanDriveId = activeUser.DriveList.GetAlbumDriveId()
						panDriveNameStr = "相册"
					} else {
						webdavServ.PanDriveId = activeUser.DriveList.GetFileDriveId()
						panDriveNameStr = "文件"
					}

					// binding pan dir path
					panDirPath := "/"
					if c.IsSet("pan_dir_path") {
						panDirPath = c.String("pan_dir_path")
					}
					webdavServ.Users[0].Scope = panDirPath

					webdavUserName := "admin"
					if c.IsSet("webdav_user") {
						webdavUserName = c.String("webdav_user")
					}
					webdavServ.Users[0].Username = webdavUserName

					webdavPassword := "admin"
					if c.IsSet("webdav_password") {
						webdavPassword = c.String("webdav_password")
					}
					webdavServ.Users[0].Password = webdavPassword

					webdavMode := "rw"
					if c.IsSet("webdav_mode") {
						webdavMode = c.String("webdav_mode")
					}
					webdavServ.Users[0].Mode = webdavMode
					if webdavServ.Users[0].Mode != "rw" && webdavServ.Users[0].Mode != "ro" {
						webdavServ.Users[0].Mode = "rw"
					}

					err := config.Config.Save()
					if err != nil {
						fmt.Println(err)
						return err
					}

					modeStr := "读写"
					if webdavServ.Users[0].Mode == "rw" {
						modeStr = "读写"
					} else if webdavServ.Users[0].Mode == "ro" {
						modeStr = "只读"
					}

					fmt.Println("----------------------------------------")
					fmt.Printf("webdav网盘信息：\n链接：http://localhost:%d\n用户名：%s\n密码：%s\n网盘模式：%s\n网盘服务类型：%s\n网盘服务目录：%s\n",
						webdavServ.Port, webdavServ.Users[0].Username, webdavServ.Users[0].Password, modeStr, panDriveNameStr, webdavServ.Users[0].Scope)
					fmt.Println("----------------------------------------")
					fmt.Println("webdav在线网盘服务运行中...")
					webdavServ.StartServer()
					return nil
				},
				Flags: []cli.Flag{
					cli.StringFlag{
						Name:  "ip",
						Usage: "绑定的本地IP，多网卡的环境下建议指定绑定的IP。默认为0.0.0.0代表绑定全部网卡",
					},
					cli.IntFlag{
						Name:  "port",
						Usage: "绑定的本地端口，默认为：23077",
						Value: 23077,
					},
					cli.StringFlag{
						Name:  "webdav_user",
						Usage: "Webdav登录用户名，默认为：admin",
					},
					cli.StringFlag{
						Name:  "webdav_password",
						Usage: "Webdav登录密码，默认为：admin",
					},
					cli.StringFlag{
						Name:  "webdav_mode",
						Usage: "Webdav模式，包括：rw-读写，ro-只读，默认为：rw",
						Value: "rw",
					},
					cli.StringFlag{
						Name:  "pan_drive",
						Usage: "Webdav绑定的网盘类型。File-文件 Album-相册。默认为文件网盘",
						Value: "File",
					},
					cli.StringFlag{
						Name:  "pan_dir_path",
						Usage: "Webdav绑定的网盘文件夹路径，默认为：/",
					},
					cli.IntFlag{
						Name:  "bs",
						Usage: "block size，上传分片大小，单位KB。推荐值：1024 ~ 10240",
						Value: 10240,
					},
				},
			},
		},
	}
}
