package syncdrive

import (
	"fmt"
	"github.com/tickstep/aliyunpan-api/aliyunpan"
	"testing"
	"time"
)

func TestSyncTask(t *testing.T) {
	refreshToken := "84c6499b7...9a2fe4f6733c3afd"
	webToken, err := aliyunpan.GetAccessTokenFromRefreshToken(refreshToken)
	if err != nil {
		fmt.Println("get acccess token error")
		return
	}

	// pan client
	panClient := aliyunpan.NewPanClient(*webToken, aliyunpan.AppLoginToken{})
	user, _ := panClient.GetUserInfo()
	task := SyncTask{
		Id:              "5b2d7c10-e927-4e72-8f9d-5abb3bb04814",
		DriveId:         user.FileDriveId,
		LocalFolderPath: "D:\\smb\\feny\\goprojects\\dev\\NS游戏备份",
		PanFolderPath:   "/sync_drive",
		Mode:            "sync",
		LastSyncTime:    "",

		syncDbFolderPath: "D:\\smb\\feny\\goprojects\\dev\\sync_drive",
		panClient:        panClient,
	}
	task.Start()
	//go func() {
	//	time.Sleep(10 * time.Second)
	//	task.Stop()
	//}()
	time.Sleep(60 * time.Second)
	task.Stop()
}
