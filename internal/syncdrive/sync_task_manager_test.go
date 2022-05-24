package syncdrive

import (
	"fmt"
	"github.com/tickstep/aliyunpan-api/aliyunpan"
	"testing"
	"time"
)

func TestStart(t *testing.T) {
	refreshToken := "d710ba220d...b1efbaab0f82d899"
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
		"D:\\smb\\feny\\goprojects\\dev\\sync_drive",
	)

	manager.Start()
	time.Sleep(1 * time.Minute)
	manager.Stop()
}
