package syncdrive

import (
	"testing"
	"time"
)

func TestSyncTask(t *testing.T) {
	task := SyncTask{
		Id:              "840f28af799747848c0b3155e0bdfeab",
		DriveId:         "",
		LocalFolderPath: "/Volumes/Downloads/dev/upload",
		PanFolderPath:   "/sync_drive",
		Mode:            "sync",
		LastSyncTime:    "",

		syncDbFolderPath: "/Volumes/Downloads/dev/sync_drive",
	}
	task.Start()
	//go func() {
	//	time.Sleep(10 * time.Second)
	//	task.Stop()
	//}()
	time.Sleep(10 * time.Second)
	task.Stop()
}
