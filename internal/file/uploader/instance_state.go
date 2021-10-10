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
package uploader

import (
	"github.com/tickstep/aliyunpan/library/requester/transfer"
)

type (
	// BlockState 文件区块信息
	BlockState struct {
		ID       int            `json:"id"`
		Range    transfer.Range `json:"range"`
		UploadDone bool `json:"upload_done"`
	}

	// InstanceState 上传断点续传信息
	InstanceState struct {
		BlockList []*BlockState `json:"block_list"`
	}
)

func (muer *MultiUploader) getWorkerListByInstanceState(is *InstanceState) workerList {
	workers := make(workerList, 0, len(is.BlockList))
	for _, blockState := range is.BlockList {
		if !blockState.UploadDone {
			workers = append(workers, &worker{
				id:         blockState.ID,
				partOffset: blockState.Range.Begin,
				splitUnit:  NewBufioSplitUnit(muer.file, blockState.Range, muer.speedsStat, muer.rateLimit),
				uploadDone:   false,
			})
		} else {
			// 已经完成的, 也要加入 (可继续优化)
			workers = append(workers, &worker{
				id:         blockState.ID,
				partOffset: blockState.Range.Begin,
				splitUnit: &fileBlock{
					readRange: blockState.Range,
					readed:    blockState.Range.End - blockState.Range.Begin,
					readerAt:  muer.file,
				},
				uploadDone: true,
			})
		}
	}
	return workers
}
