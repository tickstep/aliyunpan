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
package downloader

import (
	"errors"
	"github.com/json-iterator/go"
	"github.com/tickstep/library-go/cachepool"
	"github.com/tickstep/library-go/crypto"
	"github.com/tickstep/library-go/logger"
	"github.com/tickstep/aliyunpan/library/requester/transfer"
	"os"
	"sync"
)

type (
	//InstanceState 状态, 断点续传信息
	InstanceState struct {
		saveFile *os.File
		format   InstanceStateStorageFormat
		ii       *transfer.DownloadInstanceInfoExport
		mu       sync.Mutex
	}

	// InstanceStateStorageFormat 断点续传储存类型
	InstanceStateStorageFormat int
)

const (
	// InstanceStateStorageFormatJSON json 格式
	InstanceStateStorageFormatJSON = iota
	// InstanceStateStorageFormatProto3 protobuf 格式
	InstanceStateStorageFormatProto3
)

//NewInstanceState 初始化InstanceState
func NewInstanceState(saveFile *os.File, format InstanceStateStorageFormat) *InstanceState {
	return &InstanceState{
		saveFile: saveFile,
		format:   format,
	}
}

func (is *InstanceState) checkSaveFile() bool {
	return is.saveFile != nil
}

func (is *InstanceState) getSaveFileContents() []byte {
	if !is.checkSaveFile() {
		return nil
	}

	finfo, err := is.saveFile.Stat()
	if err != nil {
		panic(err)
	}

	size := finfo.Size()
	if size > 0xffffffff {
		panic("savePath too large")
	}
	intSize := int(size)

	buf := cachepool.RawMallocByteSlice(intSize)

	n, _ := is.saveFile.ReadAt(buf, 0)
	return crypto.Base64Decode(buf[:n])
}

//Get 获取断点续传信息
func (is *InstanceState) Get() (eii *transfer.DownloadInstanceInfo) {
	if !is.checkSaveFile() {
		return nil
	}

	is.mu.Lock()
	defer is.mu.Unlock()

	contents := is.getSaveFileContents()
	if len(contents) <= 0 {
		return
	}

	is.ii = &transfer.DownloadInstanceInfoExport{}
	var err error
	err = jsoniter.Unmarshal(contents, is.ii)

	if err != nil {
		logger.Verbosef("DEBUG: InstanceInfo unmarshal error: %s\n", err)
		return
	}

	eii = is.ii.GetInstanceInfo()
	return
}

//Put 提交断点续传信息
func (is *InstanceState) Put(eii *transfer.DownloadInstanceInfo) {
	if !is.checkSaveFile() {
		return
	}

	is.mu.Lock()
	defer is.mu.Unlock()

	if is.ii == nil {
		is.ii = &transfer.DownloadInstanceInfoExport{}
	}
	is.ii.SetInstanceInfo(eii)
	var (
		data []byte
		err  error
	)
	data, err = jsoniter.Marshal(is.ii)
	if err != nil {
		panic(err)
	}

	err = is.saveFile.Truncate(int64(len(data)))
	if err != nil {
		logger.Verbosef("DEBUG: truncate file error: %s\n", err)
	}

	_, err = is.saveFile.WriteAt(crypto.Base64Encode(data), 0)
	if err != nil {
		logger.Verbosef("DEBUG: write instance state error: %s\n", err)
	}
}

//Close 关闭
func (is *InstanceState) Close() error {
	if !is.checkSaveFile() {
		return nil
	}

	return is.saveFile.Close()
}

func (der *Downloader) initInstanceState(format InstanceStateStorageFormat) (err error) {
	if der.instanceState != nil {
		return errors.New("already initInstanceState")
	}

	var saveFile *os.File
	if der.config.InstanceStatePath != "" {
		saveFile, err = os.OpenFile(der.config.InstanceStatePath, os.O_RDWR|os.O_CREATE, 0777)
		if err != nil {
			return err
		}
	}

	der.instanceState = NewInstanceState(saveFile, format)
	return nil
}

func (der *Downloader) removeInstanceState() error {
	der.instanceState.Close()
	if der.config.InstanceStatePath != "" {
		return os.Remove(der.config.InstanceStatePath)
	}
	return nil
}
