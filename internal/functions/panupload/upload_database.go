// Copyright (c) 2020 tickstep & chenall
//
// Licensed under the Apache License, Version 2.0 (the "License");
// you may not use this file except in compliance with the License.
// You may obtain a copy of the License at
//
//	http://www.apache.org/licenses/LICENSE-2.0
//
// Unless required by applicable law or agreed to in writing, software
// distributed under the License is distributed on an "AS IS" BASIS,
// WITHOUT WARRANTIES OR CONDITIONS OF ANY KIND, either express or implied.
// See the License for the specific language governing permissions and
// limitations under the License.
package panupload

import (
	"errors"
	"github.com/tickstep/library-go/logger"
	"io"
	"os"
	"path/filepath"
	"strings"
	"time"

	"github.com/tickstep/aliyunpan/internal/config"
	"github.com/tickstep/aliyunpan/internal/file/uploader"
	"github.com/tickstep/aliyunpan/internal/localfile"
	"github.com/tickstep/library-go/converter"
	"github.com/tickstep/library-go/jsonhelper"
)

type (
	// Uploading 未完成上传的信息
	Uploading struct {
		*localfile.LocalFileMeta
		State *uploader.InstanceState `json:"state"`
	}

	// UploadingDatabase 未完成上传的数据库
	UploadingDatabase struct {
		UploadingList []*Uploading `json:"upload_state"`
		Timestamp     int64        `json:"timestamp"`

		dataFile *os.File
	}
)

// NewUploadingDatabase 初始化未完成上传的数据库, 从库中读取内容
func NewUploadingDatabase() (ud *UploadingDatabase, err error) {
	file, err := os.OpenFile(filepath.Join(config.GetConfigDir(), UploadingFileName), os.O_CREATE|os.O_RDWR, 0777)
	if err != nil {
		// 打开文件错误，一般是文件权限问题
		return nil, err
	}

	ud = &UploadingDatabase{
		dataFile: file,
	}
	info, err := file.Stat()
	if err != nil {
		return nil, err
	}

	if info.Size() <= 0 {
		return ud, nil
	}

	err = jsonhelper.UnmarshalData(file, ud)
	if err != nil {
		// 上传数据库文件内容解析错误
		// 尝试从备份的文件读取数据
		bakFile, err1 := os.OpenFile(filepath.Join(config.GetConfigDir(), UploadingBackupFileName), os.O_CREATE|os.O_RDONLY, 0777)
		if err1 != nil {
			return nil, err
		}
		err2 := jsonhelper.UnmarshalData(bakFile, ud)
		if err2 != nil {
			return nil, err
		}
		bakFile.Close()
		// 旧的备份文件可以正常使用，复制备份的数据文件到当前数据文件中
		ud.copyFile(filepath.Join(config.GetConfigDir(), UploadingFileName), filepath.Join(config.GetConfigDir(), UploadingBackupFileName))
		return ud, nil
	}

	return ud, nil
}

// Save 保存内容
func (ud *UploadingDatabase) Save() error {
	if ud.dataFile == nil {
		return errors.New("dataFile is nil")
	}

	ud.Timestamp = time.Now().Unix()

	var (
		builder = &strings.Builder{}
		err     = jsonhelper.MarshalData(builder, ud)
	)
	if err != nil {
		panic(err)
	}

	// 备份旧的数据库文件
	// 因为下面有文件内容清空、写入新内容的操作。有小概率出现文件保存没有完成程序就退出的问题，这会导致数据库内容丢失。所以这里必须备份一下旧文件
	err1 := ud.copyFile(filepath.Join(config.GetConfigDir(), UploadingBackupFileName), filepath.Join(config.GetConfigDir(), UploadingFileName))
	if err1 != nil {
		logger.Verboseln("备份上传数据库文件出错： {}", err1)
	} else {
		logger.Verboseln("成功备份旧的上传数据库文件")
	}

	logger.Verboseln("保存最新上传数据库内容")
	// 清空文件旧内容
	err = ud.dataFile.Truncate(int64(builder.Len()))
	if err != nil {
		return err
	}

	// 写入新内容
	str := builder.String()
	_, err = ud.dataFile.WriteAt(converter.ToBytes(str), 0)
	if err != nil {
		return err
	}

	return nil
}

// copyFile 复制文件
func (ud *UploadingDatabase) copyFile(dstFilePath string, srcFilePath string) error {
	os.Remove(dstFilePath)
	dstFile, err1 := os.OpenFile(dstFilePath, os.O_CREATE|os.O_WRONLY, 0777)
	if err1 != nil {
		return err1
	}
	srcFile, err2 := os.OpenFile(srcFilePath, os.O_CREATE|os.O_RDONLY, 0777)
	if err2 != nil {
		return err2
	}
	_, err3 := io.Copy(dstFile, srcFile)
	if err3 != nil {
		return err3
	}
	dstFile.Close()
	srcFile.Close()
	return nil
}

// UpdateUploading 更新正在上传
func (ud *UploadingDatabase) UpdateUploading(meta *localfile.LocalFileMeta, state *uploader.InstanceState) {
	if meta == nil {
		return
	}

	meta.CompleteAbsPath()
	for k, uploading := range ud.UploadingList {
		if uploading.LocalFileMeta == nil {
			continue
		}
		if uploading.LocalFileMeta.EqualLengthMD5(meta) || uploading.LocalFileMeta.Path.LogicPath == meta.Path.LogicPath {
			ud.UploadingList[k].State = state
			return
		}
	}

	ud.UploadingList = append(ud.UploadingList, &Uploading{
		LocalFileMeta: meta,
		State:         state,
	})
}

func (ud *UploadingDatabase) deleteIndex(k int) {
	ud.UploadingList = append(ud.UploadingList[:k], ud.UploadingList[k+1:]...)
}

// Delete 删除
func (ud *UploadingDatabase) Delete(meta *localfile.LocalFileMeta) bool {
	if meta == nil {
		return false
	}

	meta.CompleteAbsPath()
	for k, uploading := range ud.UploadingList {
		if uploading.LocalFileMeta == nil {
			continue
		}
		if uploading.LocalFileMeta.EqualLengthMD5(meta) || uploading.LocalFileMeta.Path == meta.Path {
			ud.deleteIndex(k)
			return true
		}
	}
	return false
}

// Search 搜索
func (ud *UploadingDatabase) Search(meta *localfile.LocalFileMeta) *uploader.InstanceState {
	if meta == nil {
		return nil
	}

	meta.CompleteAbsPath()
	ud.clearModTimeChange()
	for _, uploading := range ud.UploadingList {
		if uploading.LocalFileMeta == nil {
			continue
		}
		// 优选通过SHA1进行匹配，如果匹配失败则通过文件路径进行匹配
		// 因为部分上传文件是直接上传的，没有计算SHA1
		if uploading.LocalFileMeta.EqualLengthSHA1(meta) ||
			uploading.LocalFileMeta.Path.LogicPath == meta.Path.LogicPath {
			// 文件大小或者修改日期不一致，代表本地文件已经更改了，保存的旧的上传文件信息已经无用
			// 移除旧的信息，客户端需要重新上传该文件
			if meta.Length != uploading.LocalFileMeta.Length ||
				meta.ModTime != uploading.LocalFileMeta.ModTime {
				logger.Verboseln("本地文件已经被修改，上传数据记录需要移除，文件需要重新从0开始上传： {}", meta.Path.LogicPath)
				ud.Delete(meta)
				return nil
			}

			// 从上传数据库补全信息并返回
			meta.SHA1 = uploading.LocalFileMeta.SHA1
			meta.ParentFolderId = uploading.LocalFileMeta.ParentFolderId
			meta.UploadOpEntity = uploading.LocalFileMeta.UploadOpEntity
			return uploading.State
		}
	}
	return nil
}

func (ud *UploadingDatabase) clearModTimeChange() {
	for i := 0; i < len(ud.UploadingList); i++ {
		uploading := ud.UploadingList[i]
		if uploading.LocalFileMeta == nil {
			continue
		}

		if uploading.ModTime == -1 { // 忽略
			continue
		}

		info, err := os.Stat(uploading.LocalFileMeta.Path.RealPath)
		if err != nil {
			ud.deleteIndex(i)
			i--
			cmdUploadVerbose.Warnf("clear invalid file path: %s, err: %s\n", uploading.LocalFileMeta.Path, err)
			continue
		}

		if uploading.LocalFileMeta.ModTime != info.ModTime().Unix() {
			ud.deleteIndex(i)
			i--
			cmdUploadVerbose.Infof("clear modified file path: %s\n", uploading.LocalFileMeta.Path)
			continue
		}
	}
}

// Close 关闭数据库
func (ud *UploadingDatabase) Close() error {
	return ud.dataFile.Close()
}
