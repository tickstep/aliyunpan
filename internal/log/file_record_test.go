package log

import (
	"testing"
)

func TestCsvFile(t *testing.T) {
	savePath := "D:\\smb\\feny\\goprojects\\dev\\logs\\file_upload_records.csv"
	recorder := NewFileRecorder(savePath)
	recorder.Append(&FileRecordItem{
		Status:   "成功",
		TimeStr:  "2022-12-19 16:46:36",
		FileSize: 453450,
		FilePath: "D:\\smb\\feny\\goprojects\\dev\\myfile.mp4",
	})
}
