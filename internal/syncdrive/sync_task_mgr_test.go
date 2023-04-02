package syncdrive

import (
	"fmt"
	"github.com/tickstep/aliyunpan-api/aliyunpan"
	"testing"
	"time"
)

func TestStart(t *testing.T) {
	refreshToken := "1640cc2d4ea...6b8ccb4d6242161a7"
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

	manager := NewSyncTaskManager(
		nil,
		user.FileDriveId,
		panClient,
		"D:\\smb\\feny\\goprojects\\dev\\sync_drive",
		SyncOption{},
	)

	manager.Start(nil, StepSyncFile)
	time.Sleep(30 * time.Minute)
	manager.Stop(StepSyncFile)
}
