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
package taskframework_test

import (
	"fmt"
	"github.com/tickstep/aliyunpan/internal/taskframework"
	"testing"
	"time"
)

type (
	TestUnit struct {
		retry    bool
		taskInfo *taskframework.TaskInfo
	}
)

func (tu *TestUnit) SetTaskInfo(taskInfo *taskframework.TaskInfo) {
	tu.taskInfo = taskInfo
}

func (tu *TestUnit) OnFailed(lastRunResult *taskframework.TaskUnitRunResult) {
	fmt.Printf("[%s] error: %s, failed\n", tu.taskInfo.Id(), lastRunResult.Err)
}

func (tu *TestUnit) OnSuccess(lastRunResult *taskframework.TaskUnitRunResult) {
	fmt.Printf("[%s] success\n", tu.taskInfo.Id())
}

func (tu *TestUnit) OnComplete(lastRunResult *taskframework.TaskUnitRunResult) {
	fmt.Printf("[%s] complete\n", tu.taskInfo.Id())
}

func (tu *TestUnit) Run() (result *taskframework.TaskUnitRunResult) {
	fmt.Printf("[%s] running...\n", tu.taskInfo.Id())
	return &taskframework.TaskUnitRunResult{
		//Succeed:   true,
		NeedRetry: true,
	}
}

func (tu *TestUnit) OnCancel(lastRunResult *taskframework.TaskUnitRunResult) {

}

func (tu *TestUnit) OnRetry(lastRunResult *taskframework.TaskUnitRunResult) {
	fmt.Printf("[%s] prepare retry, times [%d/%d]...\n", tu.taskInfo.Id(), tu.taskInfo.Retry(), tu.taskInfo.MaxRetry())
}

func (tu *TestUnit) RetryWait() time.Duration {
	return 1 * time.Second
}

func TestTaskExecutor(t *testing.T) {
	te := taskframework.NewTaskExecutor()
	te.SetParallel(2)
	for i := 0; i < 3; i++ {
		tu := TestUnit{
			retry: false,
		}
		te.Append(&tu, 2)
	}
	te.Execute()
}
