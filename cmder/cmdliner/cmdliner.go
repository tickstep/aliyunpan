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
	"github.com/peterh/liner"
)

// CmdLiner 封装 *liner.State, 提供更简便的操作
type CmdLiner struct {
	State   *liner.State
	History *LineHistory

	tmode liner.ModeApplier
	lmode liner.ModeApplier

	paused bool
}

// NewLiner 返回 *CmdLiner, 默认设置允许 Ctrl+C 结束
func NewLiner() *CmdLiner {
	pl := &CmdLiner{}
	pl.tmode, _ = liner.TerminalMode()

	line := liner.NewLiner()
	pl.lmode, _ = liner.TerminalMode()

	line.SetMultiLineMode(true)
	line.SetCtrlCAborts(true)

	pl.State = line

	return pl
}

// Pause 暂停服务
func (pl *CmdLiner) Pause() error {
	if pl.paused {
		panic("CmdLiner already paused")
	}

	pl.paused = true
	pl.DoWriteHistory()

	return pl.tmode.ApplyMode()
}

// Resume 恢复服务
func (pl *CmdLiner) Resume() error {
	if !pl.paused {
		panic("CmdLiner is not paused")
	}

	pl.paused = false

	return pl.lmode.ApplyMode()
}

// Close 关闭服务
func (pl *CmdLiner) Close() (err error) {
	err = pl.State.Close()
	if err != nil {
		return err
	}

	if pl.History != nil && pl.History.historyFile != nil {
		return pl.History.historyFile.Close()
	}

	return nil
}
