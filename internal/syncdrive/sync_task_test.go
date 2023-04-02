package syncdrive

import (
	"fmt"
	"github.com/tickstep/aliyunpan-api/aliyunpan"
	"sync"
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
	task.Start(StepSyncFile)
	//go func() {
	//	time.Sleep(10 * time.Second)
	//	task.Stop()
	//}()
	time.Sleep(60 * time.Second)
	task.Stop()
}

func TestSyncLocker(t *testing.T) {
	locker := &sync.Mutex{}
	locker.Lock()
	fmt.Println("lock")
	//locker.Unlock()
	defer locker.Unlock()
}
