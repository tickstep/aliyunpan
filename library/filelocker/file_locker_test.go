package filelocker

import (
	"fmt"
	"log"
	"testing"
	"time"
)

func TestFlocker(t *testing.T) {
	// lock file first time - success
	locker := NewFileLocker("D:\\smb\\feny\\goprojects\\dev\\aliyunpan")
	e := LockFile(locker, 0755, true, 5*time.Second)
	fmt.Println(e)

	// lock file again - fail
	//time.Sleep(5 * time.Second)
	//e = flock(locker, 0755, true, 5*time.Second)
	//fmt.Println(e)

	// Unlock the file.
	if err := UnlockFile(locker); err != nil {
		log.Printf("funlock error: %s", err)
	}
}
