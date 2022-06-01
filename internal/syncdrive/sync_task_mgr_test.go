package syncdrive

import (
	"fmt"
	"github.com/tickstep/aliyunpan-api/aliyunpan"
	"testing"
	"time"
)

func TestStart(t *testing.T) {
	refreshToken := "ac1010f63...9585338b533bd4ab"
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
	time.Sleep(5 * time.Minute)
	manager.Stop()
}
