package filelocker

import (
	"os"
)

const (
	lockExt = ".lock"
)

type (
	FileLocker struct {
		FilePath     string
		LockFilePath string
		lockFile     *os.File
	}
)

func NewFileLocker(path string) *FileLocker {
	return &FileLocker{
		FilePath:     path,
		LockFilePath: path + lockExt,
		lockFile:     nil,
	}
}
