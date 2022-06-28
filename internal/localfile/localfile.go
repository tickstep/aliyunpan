// Copyright (c) 2020 tickstep.
//
// Licensed under the Apache License, Version 2.0 (the "License");
// you may not use this file except in compliance with the License.
// You may obtain a copy of the License at
//
//     http://www.apache.org/licenses/LICENSE-2.0
//
// Unless required by applicable law or agreed to in writing, software
// distributed under the License is distributed on an "AS IS" BASIS,
// WITHOUT WARRANTIES OR CONDITIONS OF ANY KIND, either express or implied.
// See the License for the specific language governing permissions and
// limitations under the License.
package localfile

import (
	"crypto/md5"
	"crypto/sha1"
	"encoding/hex"
	"github.com/tickstep/aliyunpan-api/aliyunpan"
	"hash/crc32"
	"io"
	"os"
	"strings"

	"github.com/tickstep/library-go/cachepool"
	"github.com/tickstep/library-go/converter"
)

const (
	// DefaultBufSize 默认的bufSize
	DefaultBufSize = int(256 * converter.KB)
)

const (
	// CHECKSUM_MD5 获取文件的 md5 值
	CHECKSUM_MD5 int = 1 << iota

	// CHECKSUM_CRC32 获取文件的 crc32 值
	CHECKSUM_CRC32

	// CHECKSUM_SHA1 获取文件的 sha1 值
	CHECKSUM_SHA1
)

type (
	// LocalFileMeta 本地文件元信息
	LocalFileMeta struct {
		Path    SymlinkFile `json:"path,omitempty"`   // 本地路径
		Length  int64       `json:"length,omitempty"` // 文件大小
		MD5     string      `json:"md5,omitempty"`    // 文件的 md5
		CRC32   uint32      `json:"crc32,omitempty"`  // 文件的 crc32
		SHA1    string      `json:"sha1,omitempty"`   // 文件的 sha1
		ModTime int64       `json:"modtime"`          // 修改日期

		// 网盘上传参数
		UploadOpEntity *aliyunpan.CreateFileUploadResult `json:"uploadOpEntity"`

		// ParentFolderId 存储云盘的目录ID
		ParentFolderId string `json:"parent_folder_id,omitempty"`
	}

	// LocalFileEntity 校验本地文件
	LocalFileEntity struct {
		LocalFileMeta
		bufSize int
		buf     []byte
		file    *os.File // 文件
	}
)

func NewLocalSymlinkFileEntity(file SymlinkFile) *LocalFileEntity {
	return NewLocalFileEntityWithBufSize(file, DefaultBufSize)
}

func NewLocalFileEntity(localPath string) *LocalFileEntity {
	return NewLocalFileEntityWithBufSize(NewSymlinkFile(localPath), DefaultBufSize)
}

func NewLocalFileEntityWithBufSize(file SymlinkFile, bufSize int) *LocalFileEntity {
	return &LocalFileEntity{
		LocalFileMeta: LocalFileMeta{
			Path: file,
		},
		bufSize: bufSize,
	}
}

// OpenPath 检查文件状态并获取文件的大小 (Length)
func (lfc *LocalFileEntity) OpenPath() error {
	if lfc.file != nil {
		lfc.file.Close()
	}

	var err error
	lfc.file, err = os.Open(lfc.Path.RealPath)
	if err != nil {
		return err
	}

	info, err := lfc.file.Stat()
	if err != nil {
		return err
	}

	lfc.Length = info.Size()
	lfc.ModTime = info.ModTime().Unix()
	return nil
}

// GetFile 获取文件
func (lfc *LocalFileEntity) GetFile() *os.File {
	return lfc.file
}

// Close 关闭文件
func (lfc *LocalFileEntity) Close() error {
	if lfc.file == nil {
		return ErrFileIsNil
	}

	return lfc.file.Close()
}

func (lfc *LocalFileEntity) initBuf() {
	if lfc.buf == nil {
		lfc.buf = cachepool.RawMallocByteSlice(lfc.bufSize)
	}
}

func (lfc *LocalFileEntity) writeChecksum(data []byte, wus ...*ChecksumWriteUnit) (err error) {
	doneCount := 0
	for _, wu := range wus {
		_, err := wu.Write(data)
		switch err {
		case ErrChecksumWriteStop:
			doneCount++
			continue
		case nil:
		default:
			return err
		}
	}
	if doneCount == len(wus) {
		return ErrChecksumWriteAllStop
	}
	return nil
}

func (lfc *LocalFileEntity) repeatRead(wus ...*ChecksumWriteUnit) (err error) {
	if lfc.file == nil {
		return ErrFileIsNil
	}

	lfc.initBuf()

	defer func() {
		_, err = lfc.file.Seek(0, os.SEEK_SET) // 恢复文件指针
		if err != nil {
			return
		}
	}()

	// 读文件
	var (
		n int
	)
read:
	for {
		n, err = lfc.file.Read(lfc.buf)
		switch err {
		case io.EOF:
			err = lfc.writeChecksum(lfc.buf[:n], wus...)
			break read
		case nil:
			err = lfc.writeChecksum(lfc.buf[:n], wus...)
		default:
			return
		}
	}
	switch err {
	case ErrChecksumWriteAllStop: // 全部结束
		err = nil
	}
	return
}

func (lfc *LocalFileEntity) createChecksumWriteUnit(cw ChecksumWriter, isAll bool, getSumFunc func(sum interface{})) (wu *ChecksumWriteUnit, deferFunc func(err error)) {
	wu = &ChecksumWriteUnit{
		ChecksumWriter: cw,
		End:            lfc.LocalFileMeta.Length,
		OnlySliceSum:   !isAll,
	}

	return wu, func(err error) {
		if err != nil {
			return
		}
		getSumFunc(wu.Sum)
	}
}

// Sum 计算文件摘要值
func (lfc *LocalFileEntity) Sum(checkSumFlag int) (err error) {
	lfc.fix()
	wus := make([]*ChecksumWriteUnit, 0, 2)
	if (checkSumFlag & (CHECKSUM_MD5)) != 0 {
		md5w := md5.New()
		wu, d := lfc.createChecksumWriteUnit(
			NewHashChecksumWriter(md5w),
			(checkSumFlag&CHECKSUM_MD5) != 0,
			func(sum interface{}) {
				if sum != nil {
					lfc.MD5 = hex.EncodeToString(sum.([]byte))
				}

				// zero size file
				if lfc.Length == 0 {
					lfc.MD5 = aliyunpan.DefaultZeroSizeFileContentHash
				}
			},
		)

		wus = append(wus, wu)
		defer d(err)
	}
	if (checkSumFlag & CHECKSUM_CRC32) != 0 {
		crc32w := crc32.NewIEEE()
		wu, d := lfc.createChecksumWriteUnit(
			NewHash32ChecksumWriter(crc32w),
			true,
			func(sum interface{}) {
				if sum != nil {
					lfc.CRC32 = sum.(uint32)
				}
			},
		)

		wus = append(wus, wu)
		defer d(err)
	}
	if (checkSumFlag & (CHECKSUM_SHA1)) != 0 {
		sha1w := sha1.New()
		wu, d := lfc.createChecksumWriteUnit(
			NewHashChecksumWriter(sha1w),
			(checkSumFlag&CHECKSUM_SHA1) != 0,
			func(sum interface{}) {
				if sum != nil {
					lfc.SHA1 = strings.ToUpper(hex.EncodeToString(sum.([]byte)))
				}

				// zero size file
				if lfc.Length == 0 {
					lfc.SHA1 = aliyunpan.DefaultZeroSizeFileContentHash
				}
			},
		)

		wus = append(wus, wu)
		defer d(err)
	}

	err = lfc.repeatRead(wus...)
	return
}

func (lfc *LocalFileEntity) fix() {
	if lfc.bufSize < DefaultBufSize {
		lfc.bufSize = DefaultBufSize
	}
}
