package syncdrive

import (
	"fmt"
	"github.com/tickstep/aliyunpan-api/aliyunpan"
	"github.com/tickstep/aliyunpan-api/aliyunpan/apierror"
	"github.com/tickstep/aliyunpan/internal/utils"
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
	refreshToken := "d77420e4daa...9d384d7c44508"
	webToken, err := aliyunpan.GetAccessTokenFromRefreshToken(refreshToken)
	if err != nil {
		fmt.Println("get acccess token error")
		return
	}

	// pan client
	panClient := aliyunpan.NewPanClient(*webToken, aliyunpan.AppLoginToken{}, aliyunpan.AppConfig{
		AppId:     "25dzX3vbYqktVxyX",
		DeviceId:  "E75459EXhOTkI5ZI6S3qDHA3",
		UserId:    "",
		Nonce:     0,
		PublicKey: "",
	}, aliyunpan.SessionConfig{
		DeviceName: "Chrome浏览器",
		ModelName:  "Windows网页版",
	})

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

func TestLocalSyncDbAddFileList(t *testing.T) {
	b := NewLocalSyncDb("D:\\smb\\feny\\goprojects\\dev\\tmp.db")
	b.Open()
	defer b.Close()

	fl := LocalFileList{}
	fl = append(fl, &LocalFileItem{
		FileName:      "aliyunpan_command_history.txt",
		FileSize:      1542,
		FileType:      "file",
		CreatedAt:     "2020-12-12 12:51:12",
		UpdatedAt:     "2020-12-12 12:51:12",
		FileExtension: "",
		Sha1Hash:      "",
		Path:          "D:\\smb\\feny\\goprojects\\dev\\aliyunpan_command_history.txt",
	})
	fl = append(fl, &LocalFileItem{
		FileName:      "f1.txt",
		FileSize:      1542,
		FileType:      "file",
		CreatedAt:     "2020-12-12 12:51:12",
		UpdatedAt:     "2020-12-12 12:51:12",
		FileExtension: "",
		Sha1Hash:      "",
		Path:          "D:\\smb\\feny\\goprojects\\dev\\f1.txt",
	})
	fl = append(fl, &LocalFileItem{
		FileName:      "f2.txt",
		FileSize:      1542,
		FileType:      "file",
		CreatedAt:     "2020-12-12 12:51:12",
		UpdatedAt:     "2020-12-12 12:51:12",
		FileExtension: "",
		Sha1Hash:      "",
		Path:          "D:\\smb\\feny\\goprojects\\dev\\f2.txt",
	})
	fl = append(fl, &LocalFileItem{
		FileName:      "f1.txt",
		FileSize:      1542,
		FileType:      "file",
		CreatedAt:     "2020-12-12 12:51:12",
		UpdatedAt:     "2020-12-12 12:51:12",
		FileExtension: "",
		Sha1Hash:      "",
		Path:          "D:\\smb\\feny\\goprojects\\dev\\fo\\f1.txt",
	})
	fmt.Println(b.AddFileList(fl))
}

func TestLocalGet(t *testing.T) {
	b := NewLocalSyncDb("/Volumes/Downloads/dev/sync_drive/840f28af799747848c0b3155e0bdfeab/local.bolt")
	b.Open()
	defer b.Close()
	v, _ := b.Get("/Volumes/Downloads/dev/upload/未命名文件夹/[HAIDAN.VIDEO].绣春刀.2014.mp4.torrent")
	fmt.Println(utils.ObjectToJsonStr(v, true))
}

func TestLocalGetFileList(t *testing.T) {
	b := NewLocalSyncDb("D:\\smb\\feny\\goprojects\\dev\\local.db")
	b.Open()
	defer b.Close()

	fmt.Println(b.GetFileList("D:/smb/feny/goprojects/dl/a761171495"))
}

func TestSyncDbAdd(t *testing.T) {
	b := NewSyncFileDb("D:\\smb\\feny\\goprojects\\dev\\sync_drive\\sync.db")
	b.Open()
	defer b.Close()

	b.Add(&SyncFileItem{
		Action:    SyncFileActionDownload,
		Status:    SyncFileStatusCreate,
		LocalFile: nil,
		PanFile: &PanFileItem{
			Category:      "others",
			Crc64Hash:     "16173291050517323365",
			CreatedAt:     "2021-07-09 23:17:52",
			DomainId:      "bj29",
			DriveId:       "19519221",
			FileExtension: "apk",
			FileId:        "60e868a0a005315fe2b149b4ade47b2df8bdccee",
			FileName:      "10000996@yunpan-release.apk",
			FileSize:      61029089,
			FileType:      "file",
			ParentFileId:  "60f29e3ac420ae8c08e645db8b80b881e2a3633c",
			Path:          "/sync_drive/10000996@yunpan-release.apk",
			Sha1Hash:      "05A7597967C3C947D84564A7D55309DEA56ED985",
			UpdatedAt:     "2021-07-21 17:05:49",
			UploadId:      "rapid-9141d56d-21c3-4fcc-b41d-5df7bf47540f",
		},
		StatusUpdateTime: "",
	})
	b.Add(&SyncFileItem{
		Action:    SyncFileActionDownload,
		Status:    SyncFileStatusCreate,
		LocalFile: nil,
		PanFile: &PanFileItem{
			Category:      "others",
			Crc64Hash:     "16173291050517323365",
			CreatedAt:     "2021-07-09 23:17:52",
			DomainId:      "bj29",
			DriveId:       "19519221",
			FileExtension: "apk",
			FileId:        "60e868a0a005315fe2b149b4ade47b2df8bdccee",
			FileName:      "10000996@yunpan-release-1.apk",
			FileSize:      61029089,
			FileType:      "file",
			ParentFileId:  "60f29e3ac420ae8c08e645db8b80b881e2a3633c",
			Path:          "/sync_drive/10000996@yunpan-release-1.apk",
			Sha1Hash:      "05A7597967C3C947D84564A7D55309DEA56ED985",
			UpdatedAt:     "2021-07-21 17:05:49",
			UploadId:      "rapid-9141d56d-21c3-4fcc-b41d-5df7bf47540f",
		},
		StatusUpdateTime: "",
	})
	b.Add(&SyncFileItem{
		Action: SyncFileActionUpload,
		Status: SyncFileStatusCreate,
		LocalFile: &LocalFileItem{
			FileName:      "f1.txt",
			FileSize:      1542,
			FileType:      "file",
			CreatedAt:     "2020-12-12 12:51:12",
			UpdatedAt:     "2020-12-12 12:51:12",
			FileExtension: ".txt",
			Sha1Hash:      "",
			Path:          "D:\\smb\\feny\\goprojects\\dev\\fo\\f1.txt",
		},
		PanFile:          nil,
		StatusUpdateTime: "",
	})
	//b.AddUnique(&SyncFileItem{
	//	Action:    SyncFileActionDownload,
	//	Status:    SyncFileStatusCreate,
	//	LocalFile: nil,
	//	PanFile: &PanFileItem{
	//		Category:      "others",
	//		Crc64Hash:     "16173291050517323365",
	//		CreatedAt:     "2021-07-09 23:17:52",
	//		DomainId:      "bj29",
	//		DriveId:       "19519221",
	//		FileExtension: "apk",
	//		FileId:        "60e868a0a005315fe2b149b4ade47b2df8bdccee",
	//		FileName:      "10000996@yunpan-release-1.apk",
	//		FileSize:      61029089,
	//		FileType:      "file",
	//		ParentFileId:  "60f29e3ac420ae8c08e645db8b80b881e2a3633c",
	//		Path:          "/sync_drive/10000996@yunpan-release-1.apk",
	//		Sha1Hash:      "05A7597967C3C947D84564A7D55309DEA56ED985",
	//		UpdatedAt:     "2021-07-21 17:05:49",
	//		UploadId:      "rapid-9141d56d-21c3-4fcc-b41d-5df7bf47540f",
	//	},
	//	StatusUpdateTime: "",
	//})
	//b.AddUnique(&SyncFileItem{
	//	Action:    SyncFileActionDownload,
	//	Status:    SyncFileStatusCreate,
	//	LocalFile: nil,
	//	PanFile: &PanFileItem{
	//		Category:      "others",
	//		Crc64Hash:     "16173291050517323365",
	//		CreatedAt:     "2021-07-09 23:17:52",
	//		DomainId:      "bj29",
	//		DriveId:       "19519221",
	//		FileExtension: "apk",
	//		FileId:        "60e868a0a005315fe2b149b4ade47b2df8bdccee",
	//		FileName:      "10000996@yunpan-release-2.apk",
	//		FileSize:      61029089,
	//		FileType:      "file",
	//		ParentFileId:  "60f29e3ac420ae8c08e645db8b80b881e2a3633c",
	//		Path:          "/sync_drive/10000996@yunpan-release-2.apk",
	//		Sha1Hash:      "05A7597967C3C947D84564A7D55309DEA56ED985",
	//		UpdatedAt:     "2021-07-21 17:05:49",
	//		UploadId:      "rapid-9141d56d-21c3-4fcc-b41d-5df7bf47540f",
	//	},
	//	StatusUpdateTime: "",
	//})
}

func TestSyncDbClear(t *testing.T) {
	b := NewSyncFileDb("D:\\smb\\feny\\goprojects\\dev\\sync.bolt")
	b.Open()
	defer b.Close()

	files, _ := b.GetFileList(SyncFileStatusSuccess)
	for _, file := range files {
		b.Delete(file.Id())
	}
}
