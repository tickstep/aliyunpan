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

type (
	// ByLeftDesc 根据剩余下载量倒序排序
	ByLeftDesc struct {
		WorkerList
	}
)

// Len 返回长度
func (wl WorkerList) Len() int {
	return len(wl)
}

// Swap 交换
func (wl WorkerList) Swap(i, j int) {
	wl[i], wl[j] = wl[j], wl[i]
}

// Less 实现倒序
func (wl ByLeftDesc) Less(i, j int) bool {
	return wl.WorkerList[i].wrange.Len() > wl.WorkerList[j].wrange.Len()
}
