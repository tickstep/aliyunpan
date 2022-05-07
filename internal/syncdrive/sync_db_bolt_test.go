package syncdrive

import (
	"fmt"
	"github.com/tickstep/aliyunpan-api/aliyunpan"
	"github.com/tickstep/aliyunpan-api/aliyunpan/apierror"
	"github.com/tickstep/library-go/logger"
	"testing"
	"time"
)

func TestPanSyncDb(t *testing.T) {
	// get access token
	refreshToken := "39b6583...b662b2a522"
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
