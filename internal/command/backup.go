// Copyright (c) 2020 tickstep
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
	"github.com/tickstep/aliyunpan/internal/functions/panupload"
	"github.com/tickstep/library-go/logger"
	"github.com/urfave/cli"
	"os"
	"path"
	"path/filepath"
	"strings"
	"sync"
)

func CmdBackup() cli.Command {
	return cli.Command{
		Name: "backup",
		Description: `备份指定 <文件/目录> 到云盘 <目标目录> 中

和上传的功能一样，只是备份多进行了如下操作

1. 增加了数据库，记录已经上传的文件信息。
   目前只记录 文件位置、大小、修改时间、MD5 。
2. 上传前先根据数据库记录判断是否需要重新上传。
3. 强制同名覆盖。

注：只备份(上传)新的文件（同名覆盖），不处理删除操作。

  示例:
    1. 将本地的 C:\Users\Administrator\Video 整个目录备份到网盘 /视频 目录
    注意区别反斜杠 "\" 和 斜杠 "/" !!!
    aliyunpan-go backup C:/Users/Administrator/Video /视频

    2. 将本地的 C:\Users\Administrator\Video 整个目录备份到网盘 /视频 目录，但是排除所有的.jpg文件
    aliyunpan-go backup -exn "\.jpg$" C:/Users/Administrator/Video /视频

    3. 将本地的 C:\Users\Administrator\Video 整个目录备份到网盘 /视频 目录，但是排除所有的.jpg文件和.mp3文件，每一个排除项就是一个exn参数
    aliyunpan-go backup -exn "\.jpg$" -exn "\.mp3$" C:/Users/Administrator/Video /视频

    4. 将本地的 C:\Users\Administrator\Video 整个目录备份到网盘 /视频 目录，但是排除所有的 @eadir 文件夹
    aliyunpan-go backup -exn "^@eadir$" C:/Users/Administrator/Video /视频

  参考：
    以下是典型的排除特定文件或者文件夹的例子，注意：参数值必须是正则表达式。在正则表达式中，^表示匹配开头，$表示匹配结尾。
    1)排除@eadir文件或者文件夹：-exn "^@eadir$"
    2)排除.jpg文件：-exn "\.jpg$"
    3)排除.号开头的文件：-exn "^\."
    4)排除~号开头的文件：-exn "^~"
    5)排除 myfile.txt 文件：-exn "^myfile.txt$"
`,
		Usage:     "备份文件或目录",
		UsageText: "backup <文件/目录路径1> <文件/目录2> <文件/目录3> ... <目标目录>",
		Category:  "阿里云盘",
		Before:    cmder.ReloadConfigFunc,
		Action:    Backup,
		Flags: append(UploadFlags, cli.BoolFlag{
			Name:  "delete",
			Usage: "通过本地数据库记录同步删除网盘文件",
		}, cli.BoolFlag{
			Name:  "sync",
			Usage: "本地同步到网盘（会同步删除网盘文件）",
		}),
	}
}

func OpenSyncDb(path string) (panupload.SyncDb, error) {
	return panupload.OpenSyncDb(path, BackupMetaBucketName)
}

// 删除那些本地不存在而网盘存在的网盘文件 默认使用本地数据库判断，如果 flagSync 为 true 则遍历网盘文件列表进行判断（速度较慢）。
func DelRemoteFileFromDB(driveId string, localDir string, savePath string, flagSync bool) {
	activeUser := config.Config.ActiveUser()
	var db panupload.SyncDb
	var err error

	dbpath := filepath.Join(localDir, BackupMetaDirName)
	db, err = OpenSyncDb(dbpath + string(os.PathSeparator) + "db")
	if err != nil {
		fmt.Println("同步数据库打开失败！", err)
		return
	}
	defer db.Close()

	savePath = path.Join(savePath, filepath.Base(localDir))

	//判断本地文件是否存在，如果存在返回 true 否则删除数据库相关记录和网盘上的文件。
	isLocalFileExist := func(ent *panupload.UploadedFileMeta) (isExists bool) {
		testPath := strings.TrimPrefix(ent.Path, savePath)
		testPath = filepath.Join(localDir, testPath)
		logger.Verboseln("同步删除检测:", testPath, ent.Path)

		//为防止误删，只有当 err 是文件不存在的时候才进行删除处理。
		if fi, err := os.Stat(testPath); err == nil || !os.IsNotExist(err) {
			//使用sync功能时没有传时间参数进来，为方便对比回写数据库需补上时间。
			if fi != nil {
				ent.ModTime = fi.ModTime().Unix()
			}
			return true
		}

		var err *apierror.ApiError

		// 尝试从本地数据库查找
		if ent.ParentId == "" {
			if test := db.Get(path.Dir(ent.Path)); test != nil && test.IsFolder && test.FileId != "" {
				ent.ParentId = test.FileId
			}
		}

		// 从网盘查找
		if ent.FileId == "" || ent.ParentId == "" {
			efi, err := activeUser.PanClient().FileInfoById(driveId, ent.FileId)
			//网盘上不存在这个文件或目录，只需要清理数据库
			if err != nil && err.Code == apierror.ApiCodeFileNotFoundCode {
				db.DelWithPrefix(ent.Path)
				logger.Verboseln("删除数据库记录", ent.Path)
				return
			}
			if efi != nil {
				ent.FileId = efi.FileId
				ent.ParentId = efi.ParentFileId
			}
		}

		if ent.FileId == "" {
			return
		}

		// 本地文件不存在
		// 删除网盘对应文件
		fileDeleteResult, err := activeUser.PanClient().FileDelete([]*aliyunpan.FileBatchActionParam{{DriveId:driveId, FileId:ent.FileId}})
		if err != nil || len(fileDeleteResult) == 0 {
			fmt.Println("删除网盘文件或目录失败", ent.Path, err)
		} else {
			db.DelWithPrefix(ent.Path)
			logger.Verboseln("删除网盘文件和数据库记录", ent.Path)
		}
		return
	}

	// 根据数据库记录删除不存在的文件
	if !flagSync {
		for ent, err := db.First(savePath); err == nil; ent, err = db.Next(savePath) {
			isLocalFileExist(ent)
		}
		return
	}

	parent := db.Get(savePath)
	if parent.FileId == "" {
		efi, err := activeUser.PanClient().FileInfoByPath(driveId, savePath)
		if err != nil {
			return
		}
		parent.FileId = efi.FileId
	}

	var syncFunc func(curPath, parentID string)

	syncFunc = func(curPath, parentID string) {
		param := &aliyunpan.FileListParam{
			DriveId: driveId,
			ParentFileId: parentID,
		}
		fileResult, err := activeUser.PanClient().FileListGetAll(param)
		if err != nil {
			return
		}
		if fileResult == nil || len(fileResult) == 0 {
			return
		}
		for _, fileEntity := range fileResult {
			ufm := &panupload.UploadedFileMeta{
				FileId:   fileEntity.FileId,
				ParentId: fileEntity.ParentFileId,
				Size:     fileEntity.FileSize,
				IsFolder: fileEntity.IsFolder(),
				Path:     path.Join(curPath, fileEntity.FileName),
				SHA1:      strings.ToLower(fileEntity.ContentHash),
			}

			if !isLocalFileExist(ufm) {
				continue
			}

			//如果这是一个目录就直接更新数据库，否则判断原始记录的Hash信息，如果一致才更新。
			if ufm.IsFolder {
				db.Put(ufm.Path, ufm)
				syncFunc(ufm.Path, ufm.FileId)
			} else if test := db.Get(ufm.Path); test.SHA1 == ufm.SHA1 {
				db.Put(ufm.Path, ufm)
			}
		}
	}

	//开启自动清理功能
	db.AutoClean(parent.Path, true)
	db.Put(parent.Path, parent)

	syncFunc(savePath, parent.FileId)
}

func checkPath(localdir string) (string, error) {
	fullPath, err := filepath.Abs(localdir)
	if err != nil {
		fullPath = localdir
	}

	if fi, err := os.Stat(fullPath); err != nil && !fi.IsDir() {
		return fullPath, os.ErrInvalid
	}

	dbpath := filepath.Join(fullPath, BackupMetaDirName)
	//数据库目录判断
	fi, err := os.Stat(dbpath)

	if err != nil {
		if os.IsNotExist(err) {
			err = os.Mkdir(dbpath, 0755)
		}
		if err != nil {
			return fullPath, fmt.Errorf("数据库目录[%s]创建失败，跳过处理: %s", dbpath, err)
		}
	}

	if fi != nil && !fi.IsDir() {
		return fullPath, os.ErrPermission
	}

	return fullPath, nil
}

func Backup(c *cli.Context) error {
	if c.NArg() < 2 {
		cli.ShowCommandHelp(c, c.Command.Name)
		return nil
	}

	subArgs := c.Args()
	localpaths := make([]string, 0)
	flagSync := c.Bool("sync")
	flagDelete := c.Bool("delete")

	opt := &UploadOptions{
		AllParallel:   c.Int("p"),
		Parallel:      1, // 阿里云盘一个文件只支持单线程上传
		MaxRetry:      c.Int("retry"),
		NoRapidUpload: c.Bool("norapid"),
		ShowProgress:  !c.Bool("np"),
		IsOverwrite:   true,
		DriveId:      parseDriveId(c),
		ExcludeNames: c.StringSlice("exn"),
		BlockSize: int64(c.Int("bs") * 1024),
	}

	localCount := c.NArg() - 1
	savePath := GetActiveUser().PathJoin(opt.DriveId, subArgs[localCount])

	wg := sync.WaitGroup{}
	wg.Add(localCount)
	for _, p := range subArgs[:localCount] {
		go func(p string) {
			defer wg.Done()
			fullPath, err := checkPath(p)
			switch err {
			case nil:
				if flagSync || flagDelete {
					DelRemoteFileFromDB(opt.DriveId, fullPath, savePath, flagSync)
				}
			case os.ErrInvalid:
			default:
				return
			}
			localpaths = append(localpaths, fullPath)
		}(p)
	}

	wg.Wait()

	if len(localpaths) == 0 {
		return nil
	}

	RunUpload(localpaths, savePath, opt)
	return nil
}
