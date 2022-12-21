//go:build !windows && !plan9 && !solaris
// +build !windows,!plan9,!solaris

package filelocker

import (
	"os"
	"syscall"
	"time"
)

// LockFile acquires an advisory lock on a file descriptor.
func LockFile(locker *FileLocker, mode os.FileMode, exclusive bool, timeout time.Duration) error {
	f, err := os.OpenFile(locker.LockFilePath, os.O_CREATE, mode)
	if err != nil {
		return err
	}
	locker.lockFile = f

	var t time.Time
	for {
		// If we're beyond our timeout then return an error.
		// This can only occur after we've attempted a flock once.
		if t.IsZero() {
			t = time.Now()
		} else if timeout > 0 && time.Since(t) > timeout {
			return ErrTimeout
		}
		flag := syscall.LOCK_SH
		if exclusive {
			flag = syscall.LOCK_EX
		}

		// Otherwise attempt to obtain an exclusive lock.
		err := syscall.Flock(int(locker.lockFile.Fd()), flag|syscall.LOCK_NB)
		if err == nil {
			return nil
		} else if err != syscall.EWOULDBLOCK {
			return err
		}

		// Wait for a bit and try again.
		time.Sleep(50 * time.Millisecond)
	}
}

// UnlockFile releases an advisory lock on a file descriptor.
func UnlockFile(locker *FileLocker) error {
	return syscall.Flock(int(locker.lockFile.Fd()), syscall.LOCK_UN)
}
