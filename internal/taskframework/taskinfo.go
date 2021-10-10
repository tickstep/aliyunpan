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
package taskframework

type (
	TaskInfo struct {
		id       string
		maxRetry int
		retry    int
	}

	TaskInfoItem struct {
		Info *TaskInfo
		Unit TaskUnit
	}
)

// IsExceedRetry 重试次数达到限制
func (t *TaskInfo) IsExceedRetry() bool {
	return t.retry >= t.maxRetry
}

func (t *TaskInfo) Id() string {
	return t.id
}

func (t *TaskInfo) MaxRetry() int {
	return t.maxRetry
}

func (t *TaskInfo) SetMaxRetry(maxRetry int) {
	t.maxRetry = maxRetry
}

func (t *TaskInfo) Retry() int {
	return t.retry
}
