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
	"github.com/tickstep/library-go/expires"
	"sync"
	"time"
)

// ResetController 网络连接控制器
type ResetController struct {
	mu          sync.Mutex
	currentTime time.Time
	maxResetNum int
	resetEntity map[expires.Expires]struct{}
}

// NewResetController 初始化*ResetController
func NewResetController(maxResetNum int) *ResetController {
	return &ResetController{
		currentTime: time.Now(),
		maxResetNum: maxResetNum,
		resetEntity: map[expires.Expires]struct{}{},
	}
}

func (rc *ResetController) update() {
	for k := range rc.resetEntity {
		if k.IsExpires() {
			delete(rc.resetEntity, k)
		}
	}
}

// AddResetNum 增加连接
func (rc *ResetController) AddResetNum() {
	rc.mu.Lock()
	defer rc.mu.Unlock()
	rc.update()
	rc.resetEntity[expires.NewExpires(9*time.Second)] = struct{}{}
}

// CanReset 是否可以建立连接
func (rc *ResetController) CanReset() bool {
	rc.mu.Lock()
	defer rc.mu.Unlock()
	rc.update()
	return len(rc.resetEntity) < rc.maxResetNum
}
