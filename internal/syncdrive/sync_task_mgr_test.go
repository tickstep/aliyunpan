package syncdrive

import (
	"fmt"
	"github.com/tickstep/aliyunpan-api/aliyunpan"
	"testing"
	"time"
)

func TestStart(t *testing.T) {
	refreshToken := "927e2a3da30646d8a787e7f11c0fdf1f"
	webToken, err := aliyunpan.GetAccessTokenFromRefreshToken(refreshToken)
	if err != nil {
		fmt.Println("get acccess token error")
		return
	}

	// pan client
	panClient := aliyunpan.NewPanClient(*webToken, aliyunpan.AppLoginToken{})
	user, _ := panClient.GetUserInfo()

	manager := NewSyncTaskManager(
		user.FileDriveId,
		panClient,
		"/Volumes/Downloads/dev/sync_drive",
	)

	manager.Start()
	time.Sleep(1 * time.Minute)
	manager.Stop()
}
