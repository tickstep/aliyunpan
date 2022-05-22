package syncdrive

import (
	"fmt"
	"github.com/tickstep/aliyunpan-api/aliyunpan"
	"testing"
	"time"
)

func TestSyncTask(t *testing.T) {
	refreshToken := "f7bab...0de09517"
	webToken, err := aliyunpan.GetAccessTokenFromRefreshToken(refreshToken)
	if err != nil {
		fmt.Println("get acccess token error")
		return
	}

	// pan client
	panClient := aliyunpan.NewPanClient(*webToken, aliyunpan.AppLoginToken{})
	user, _ := panClient.GetUserInfo()
	task := SyncTask{
		Id:              "840f28af799747848c0b3155e0bdfeab",
		DriveId:         user.FileDriveId,
		LocalFolderPath: "/Volumes/Downloads/dev/upload",
		PanFolderPath:   "/sync_drive",
		Mode:            "sync",
		LastSyncTime:    "",

		syncDbFolderPath: "/Volumes/Downloads/dev/sync_drive",
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
