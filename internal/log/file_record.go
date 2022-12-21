package log

import (
	"encoding/csv"
	"fmt"
	"github.com/tickstep/aliyunpan/internal/utils"
	"github.com/tickstep/library-go/converter"
	"github.com/tickstep/library-go/logger"
	"os"
	"path/filepath"
	"sync"
)

type (
	FileRecordItem struct {
		Status   string `json:"status"`
		TimeStr  string `json:"timeStr"`
		FileSize int64  `json:"fileSize"`
		FilePath string `json:"filePath"`
	}

	FileRecorder struct {
		Path   string `json:"path"`
		locker *sync.Mutex
	}
)

// NewFileRecorder 创建文件记录器
func NewFileRecorder(filePath string) *FileRecorder {
	return &FileRecorder{
		Path:   filePath,
		locker: &sync.Mutex{},
	}
}

// Append 增加数据记录
func (f *FileRecorder) Append(item *FileRecordItem) error {
	f.locker.Lock()
	defer f.locker.Unlock()
	savePath := f.Path
	folder := filepath.Dir(savePath)
	if b, err := utils.PathExists(folder); err == nil && !b {
		os.MkdirAll(folder, 0755)
	}

	var fp *os.File
	var write *csv.Writer
	if b, err := utils.PathExists(savePath); err == nil && b {
		file, err1 := os.OpenFile(savePath, os.O_APPEND, 0755)
		if err1 != nil {
			logger.Verbosef("打开文件["+savePath+"]失败,%v", err1)
			return err1
		}
		fp = file
		write = csv.NewWriter(fp) //创建一个新的写入文件流
	} else {
		file, err1 := os.OpenFile(savePath, os.O_RDWR|os.O_CREATE|os.O_TRUNC, 0755) // 创建文件句柄
		if err1 != nil {
			logger.Verbosef("创建文件["+savePath+"]失败,%v", err1)
			return err1
		}
		fp = file
		fp.WriteString("\xEF\xBB\xBF") // 写入UTF-8 BOM
		write = csv.NewWriter(fp)      //创建一个新的写入文件流
		write.Write([]string{"状态", "时间", "文件大小", "文件路径"})
	}
	if fp == nil || write == nil {
		return fmt.Errorf("open recorder file error")
	}
	defer fp.Close()

	data := []string{item.Status, item.TimeStr, converter.ConvertFileSize(item.FileSize, 2), item.FilePath}
	write.Write(data)
	write.Flush()
	return nil
}
