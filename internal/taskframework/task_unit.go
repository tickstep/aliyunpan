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

import "time"

type (
	TaskUnit interface {
		SetTaskInfo(info *TaskInfo)
		// 执行任务
		Run() (result *TaskUnitRunResult)
		// 重试任务执行的方法
		// 当达到最大重试次数, 执行失败
		OnRetry(lastRunResult *TaskUnitRunResult)
		// 每次执行成功执行的方法
		OnSuccess(lastRunResult *TaskUnitRunResult)
		// 每次执行失败执行的方法
		OnFailed(lastRunResult *TaskUnitRunResult)
		// 每次执行结束执行的方法, 不管成功失败
		OnComplete(lastRunResult *TaskUnitRunResult)
		// 重试等待的时间
		RetryWait() time.Duration
	}

	// 任务单元执行结果
	TaskUnitRunResult struct {
		Succeed       bool        // 是否执行成功
		NeedRetry     bool        // 是否需要重试

		// 以下是额外的信息
		Err           error       // 错误信息
		ResultCode    int         // 结果代码
		ResultMessage string      // 结果描述
		Extra         interface{} // 额外的信息
	}
)

var (
	// TaskUnitRunResultSuccess 任务执行成功
	TaskUnitRunResultSuccess = &TaskUnitRunResult{}
)
