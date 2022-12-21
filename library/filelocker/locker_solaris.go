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
		var lock syscall.Flock_t
		lock.Start = 0
		lock.Len = 0
		lock.Pid = 0
		lock.Whence = 0
		lock.Pid = 0
		if exclusive {
			lock.Type = syscall.F_WRLCK
		} else {
			lock.Type = syscall.F_RDLCK
		}
		err := syscall.FcntlFlock(locker.lockFile.Fd(), syscall.F_SETLK, &lock)
		if err == nil {
			return nil
		} else if err != syscall.EAGAIN {
			return err
		}

		// Wait for a bit and try again.
		time.Sleep(50 * time.Millisecond)
	}
}

// UnlockFile releases an advisory lock on a file descriptor.
func UnlockFile(locker *FileLocker) error {
	var lock syscall.Flock_t
	lock.Start = 0
	lock.Len = 0
	lock.Type = syscall.F_UNLCK
	lock.Whence = 0
	return syscall.FcntlFlock(uintptr(locker.lockFile.Fd()), syscall.F_SETLK, &lock)
}
