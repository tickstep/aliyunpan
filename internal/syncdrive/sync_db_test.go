package syncdrive

import (
	"fmt"
	"github.com/tickstep/aliyunpan-api/aliyunpan"
	"github.com/tickstep/aliyunpan-api/aliyunpan/apierror"
	"github.com/tickstep/library-go/logger"
	"io/ioutil"
	"os"
	"path"
	"path/filepath"
	"strings"
	"testing"
	"time"
)

func TestPanSyncDb(t *testing.T) {
	// get access token
	refreshToken := "c2b11bfc07...f090dc07ea59"
	webToken, err := aliyunpan.GetAccessTokenFromRefreshToken(refreshToken)
	if err != nil {
		fmt.Println("get acccess token error")
		return
	}

	// pan client
	panClient := aliyunpan.NewPanClient(*webToken, aliyunpan.AppLoginToken{})

	// get user info
	ui, err := panClient.GetUserInfo()
	if err != nil {
		fmt.Println("get user info error")
		return
	}
	fmt.Println("当前登录用户：" + ui.Nickname)

	b := NewPanSyncDb("D:\\smb\\feny\\goprojects\\dev\\pan.db")
	b.Open()
	defer b.Close()
	// do some file operation
	panClient.FilesDirectoriesRecurseList(ui.FileDriveId, "/Parallels Desktop", func(depth int, _ string, fd *aliyunpan.FileEntity, apiError *apierror.ApiError) bool {
		if apiError != nil {
			logger.Verbosef("%s\n", apiError)
			return true
		}
		fmt.Println("add file：" + fd.String())
		b.Add(NewPanFileItem(fd))
		time.Sleep(2 * time.Second)
		return true
	})
}

func TestGet(t *testing.T) {
	b := NewPanSyncDb("D:\\smb\\feny\\goprojects\\dev\\pan.db")
	b.Open()
	defer b.Close()

	fmt.Println(b.Get("/Parallels Desktop/v17/部分电脑安装v17可能有问题，请退回v16版本.txt"))
}

func TestGetFileList(t *testing.T) {
	b := NewPanSyncDb("D:\\smb\\feny\\goprojects\\dev\\pan.db")
	b.Open()
	defer b.Close()

	fmt.Println(b.GetFileList("/Parallels Desktop/v17"))
}

func WalkAllFile(dirPath string, walkFn filepath.WalkFunc) error {
	dirPath = strings.ReplaceAll(dirPath, "\\", "/")
	info, err := os.Lstat(dirPath)
	if err != nil {
		err = walkFn(dirPath, nil, err)
	} else {
		err = walkAllFile(dirPath, info, walkFn)
	}
	return err
}

func walkAllFile(dirPath string, info os.FileInfo, walkFn filepath.WalkFunc) error {
	if !info.IsDir() {
		return walkFn(dirPath, info, nil)
	}

	files, err := ioutil.ReadDir(dirPath)
	if err != nil {
		return walkFn(dirPath, nil, err)
	}
	for _, fi := range files {
		subFilePath := dirPath + "/" + fi.Name()
		err = walkFn(subFilePath, fi, err)
		if err != nil {
			return err
		}
		if fi.IsDir() {
			err = walkAllFile(subFilePath, fi, walkFn)
			if err != nil {
				return err
			}
		}
	}
	return nil
}

func TestLocalSyncDb(t *testing.T) {
	b := NewLocalSyncDb("D:\\smb\\feny\\goprojects\\dev\\local.db")
	b.Open()
	defer b.Close()

	var walkFunc filepath.WalkFunc
	walkFunc = func(file string, fi os.FileInfo, err error) error {
		fmt.Println(file)
		fileType := "file"
		if fi.IsDir() {
			fileType = "folder"
		}
		item := &LocalFileItem{
			FileName:      path.Base(file),
			FileSize:      fi.Size(),
			FileType:      fileType,
			CreatedAt:     fi.ModTime().Format("2006-01-02 15:04:05"),
			UpdatedAt:     fi.ModTime().Format("2006-01-02 15:04:05"),
			FileExtension: "",
			Sha1Hash:      "",
			Path:          file,
		}
		if _, e := b.Add(item); e != nil {
			fmt.Println(e)
		}
		return nil
	}
	WalkAllFile("D:\\smb\\feny\\goprojects\\dl\\a761171495", walkFunc)
}

func TestLocalGet(t *testing.T) {
	b := NewLocalSyncDb("D:\\smb\\feny\\goprojects\\dev\\local.db")
	b.Open()
	defer b.Close()

	fmt.Println(b.Get("D:\\smb\\feny\\goprojects\\dl\\a761171495\\1.jpg"))
}

func TestLocalGetFileList(t *testing.T) {
	b := NewLocalSyncDb("D:\\smb\\feny\\goprojects\\dev\\local.db")
	b.Open()
	defer b.Close()

	fmt.Println(b.GetFileList("D:/smb/feny/goprojects/dl/a761171495"))
}
