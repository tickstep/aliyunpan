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
	"github.com/tickstep/aliyunpan-api/aliyunpan"
	"github.com/tickstep/aliyunpan-api/aliyunpan/apierror"
	"github.com/tickstep/aliyunpan/cmder"
	"github.com/tickstep/aliyunpan/internal/config"
	"github.com/tickstep/library-go/logger"
	"github.com/urfave/cli"
	"io/ioutil"
	"log"
	"os"
	"path"
	"path/filepath"
	"strings"
	"time"
)

type (
	dirFileListData struct {
		Dir      *aliyunpan.MkdirResult
		FileList aliyunpan.FileList
	}
)

const (
	DefaultSaveToPanPath = "/aliyunpan"
)

func CmdImport() cli.Command {
	return cli.Command{
		Name:      "import",
		Usage:     "导入文件",
		UsageText: cmder.App().Name + " export <本地元数据文件路径>",
		Description: `
    导入文件中记录的元数据文件到网盘。保存到网盘的文件会使用文件元数据记录的路径位置，如果没有指定云盘目录(saveto)则默认导入到目录 aliyunpan 中。
    导入的文件可以使用 export 命令获得。
    
    导入文件每一行是一个文件元数据，样例如下：
    aliyunpan://file.dmg|752FCCBFB2436A6FFCA3B287831D4FAA5654B07E|7005440|pan_folder

	示例:
    导入文件 /Users/tickstep/Downloads/export_files.txt 存储的所有文件元数据项
    aliyunpan import /Users/tickstep/Downloads/export_files.txt

    导入文件 /Users/tickstep/Downloads/export_files.txt 存储的所有文件元数据项并保存到目录 /my2021 中
    aliyunpan import -saveto=/my2021 /Users/tickstep/Downloads/export_files.txt

    导入文件 /Users/tickstep/Downloads/export_files.txt 存储的所有文件元数据项并保存到网盘根目录 / 中
    aliyunpan import -saveto=/ /Users/tickstep/Downloads/export_files.txt
`,
		Category: "阿里云盘",
		Before:   cmder.ReloadConfigFunc,
		Action: func(c *cli.Context) error {
			if c.NArg() < 1 {
				cli.ShowCommandHelp(c, c.Command.Name)
				return nil
			}

			saveTo := ""
			if c.String("saveto") != "" {
				saveTo = filepath.Clean(c.String("saveto"))
			}

			subArgs := c.Args()
			RunImportFiles(parseDriveId(c), c.Bool("ow"), saveTo, subArgs[0])
			return nil
		},
		Flags: []cli.Flag{
			cli.BoolFlag{
				Name:  "ow",
				Usage: "overwrite, 覆盖已存在的网盘文件",
			},
			cli.StringFlag{
				Name:  "driveId",
				Usage: "网盘ID",
				Value: "",
			},
			cli.StringFlag{
				Name:  "saveto",
				Usage: "将文件保存到指定的目录",
			},
		},
	}
}

func RunImportFiles(driveId string, overwrite bool, panSavePath, localFilePath string) {
	lfi, _ := os.Stat(localFilePath)
	if lfi != nil {
		if lfi.IsDir() {
			fmt.Println("请指定导入文件")
			return
		}
	} else {
		// create file
		fmt.Println("导入文件不存在")
		return
	}

	if panSavePath == "" {
		// use default
		panSavePath = DefaultSaveToPanPath
	}

	fmt.Println("导入的文件会存储到目录：" + panSavePath)

	importFile, err := os.OpenFile(localFilePath, os.O_RDONLY, 0755)
	if err != nil {
		log.Fatal(err)
		return
	}
	defer importFile.Close()

	fileData, err := ioutil.ReadAll(importFile)
	if err != nil {
		fmt.Println("读取文件出错")
		return
	}
	fileText := string(fileData)
	if len(fileText) == 0 {
		fmt.Println("文件为空")
		return
	}
	fileText = strings.TrimSpace(fileText)
	fileLines := strings.Split(fileText, "\n")
	importFileItems := []RapidUploadItem{}
	for _, line := range fileLines {
		line = strings.TrimSpace(line)
		if item, e := newRapidUploadItem(line); e != nil {
			fmt.Println(e)
			continue
		} else {
			item.FilePath = strings.ReplaceAll(path.Join(panSavePath, item.FilePath), "\\", "/")
			importFileItems = append(importFileItems, *item)
		}
	}
	if len(importFileItems) == 0 {
		fmt.Println("没有可以导入的文件项目")
		return
	}

	fmt.Println("正在准备导入...")
	dirMap := prepareMkdir(driveId, importFileItems)

	fmt.Println("正在导入...")
	successImportFiles := []RapidUploadItem{}
	failedImportFiles := []RapidUploadItem{}
	for _, item := range importFileItems {
		fmt.Printf("正在处理导入: %s\n", item.FilePath)
		result, abort := processOneImport(driveId, overwrite, dirMap, item)
		if abort {
			fmt.Println("导入任务终止了")
			break
		}
		if result {
			successImportFiles = append(successImportFiles, item)
		} else {
			failedImportFiles = append(failedImportFiles, item)
		}
		time.Sleep(time.Duration(200) * time.Millisecond)
	}
	if len(failedImportFiles) > 0 {
		fmt.Println("\n以下文件导入失败")
		for _, f := range failedImportFiles {
			fmt.Printf("%s %s\n", f.FileSha1, f.FilePath)
		}
		fmt.Println("")
	}
	fmt.Printf("导入结果, 成功 %d, 失败 %d\n", len(successImportFiles), len(failedImportFiles))
}

func processOneImport(driveId string, isOverwrite bool, dirMap map[string]*dirFileListData, item RapidUploadItem) (result, abort bool) {
	panClient := config.Config.ActiveUser().PanClient()
	panDir, fileName := path.Split(item.FilePath)
	dataItem := dirMap[path.Dir(panDir)]
	if isOverwrite {
		// 标记覆盖旧同名文件
		// 检查同名文件是否存在
		var efi *aliyunpan.FileEntity = nil
		for _, fileItem := range dataItem.FileList {
			if !fileItem.IsFolder() && fileItem.FileName == fileName {
				efi = fileItem
				break
			}
		}
		if efi != nil && efi.FileId != "" {
			// existed, delete it
			fdr, err := panClient.FileDelete([]*aliyunpan.FileBatchActionParam{
				{
					DriveId: driveId,
					FileId:  efi.FileId,
				},
			})
			if err != nil || fdr == nil || !fdr[0].Success {
				fmt.Println("无法删除文件，请稍后重试")
				return false, false
			}
			time.Sleep(time.Duration(500) * time.Millisecond)
			fmt.Println("检测到同名文件，已移动到回收站")
		}
	}

	appCreateUploadFileParam := &aliyunpan.CreateFileUploadParam{
		DriveId:      driveId,
		Name:         fileName,
		Size:         item.FileSize,
		ContentHash:  item.FileSha1,
		ParentFileId: dataItem.Dir.FileId,
	}
	uploadOpEntity, apierr := panClient.CreateUploadFile(appCreateUploadFileParam)
	if apierr != nil {
		fmt.Println("创建秒传任务失败：" + apierr.Error())
		return false, true
	}

	if uploadOpEntity.RapidUpload {
		logger.Verboseln("秒传成功, 保存到网盘路径: ", path.Join(panDir, uploadOpEntity.FileName))
	} else {
		fmt.Println("失败，文件未曾上传，无法秒传")
		return false, false
	}
	return true, false
}

func prepareMkdir(driveId string, importFileItems []RapidUploadItem) map[string]*dirFileListData {
	panClient := config.Config.ActiveUser().PanClient()
	resultMap := map[string]*dirFileListData{}
	for _, item := range importFileItems {
		var apierr *apierror.ApiError
		var rs *aliyunpan.MkdirResult
		panDir := path.Dir(item.FilePath)
		if resultMap[panDir] != nil {
			continue
		}
		if panDir != "/" {
			rs, apierr = panClient.MkdirRecursive(driveId, "", "", 0, strings.Split(path.Clean(panDir), "/"))
			if apierr != nil || rs.FileId == "" {
				logger.Verboseln("创建云盘文件夹失败")
				continue
			}
		} else {
			rs = &aliyunpan.MkdirResult{}
			rs.FileId = aliyunpan.DefaultRootParentFileId
		}
		dataItem := &dirFileListData{}
		dataItem.Dir = rs

		// files
		param := &aliyunpan.FileListParam{}
		param.DriveId = driveId
		param.ParentFileId = rs.FileId
		allFileInfo, err1 := panClient.FileListGetAll(param, 0)
		if err1 != nil {
			logger.Verboseln("获取文件信息出错")
			continue
		}
		dataItem.FileList = allFileInfo

		resultMap[panDir] = dataItem
		time.Sleep(time.Duration(500) * time.Millisecond)
	}
	return resultMap
}
