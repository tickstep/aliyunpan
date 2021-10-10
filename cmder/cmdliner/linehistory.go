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
package cmdliner

import (
	"fmt"
	"os"
)

// LineHistory 命令行历史
type LineHistory struct {
	historyFilePath string
	historyFile     *os.File
}

// NewLineHistory 设置历史
func NewLineHistory(filePath string) (lh *LineHistory, err error) {
	lh = &LineHistory{
		historyFilePath: filePath,
	}

	lh.historyFile, err = os.OpenFile(filePath, os.O_RDWR|os.O_CREATE, 0644)
	if err != nil {
		return nil, err
	}

	return lh, nil
}

// DoWriteHistory 执行写入历史
func (pl *CmdLiner) DoWriteHistory() (err error) {
	if pl.History == nil {
		return fmt.Errorf("history not set")
	}

	pl.History.historyFile, err = os.Create(pl.History.historyFilePath)
	if err != nil {
		return fmt.Errorf("写入历史错误, %s", err)
	}

	_, err = pl.State.WriteHistory(pl.History.historyFile)
	if err != nil {
		return fmt.Errorf("写入历史错误: %s", err)
	}

	return nil
}

// ReadHistory 读取历史
func (pl *CmdLiner) ReadHistory() (err error) {
	if pl.History == nil {
		return fmt.Errorf("history not set")
	}

	_, err = pl.State.ReadHistory(pl.History.historyFile)
	return err
}
