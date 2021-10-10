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
package uploader_test

import (
	"fmt"
	"github.com/tickstep/library-go/cachepool"
	"github.com/tickstep/library-go/requester/rio"
	"github.com/tickstep/aliyunpan/library/requester/transfer"
	"github.com/tickstep/aliyunpan/internal/file/uploader"
	"io"
	"testing"
)

var (
	blockList = uploader.SplitBlock(10000, 999)
)

func TestSplitBlock(t *testing.T) {
	for k, e := range blockList {
		fmt.Printf("%d %#v\n", k, e)
	}
}

func TestSplitUnitRead(t *testing.T) {
	var size int64 = 65536*2+3432
	buffer := rio.NewBuffer(cachepool.RawMallocByteSlice(int(size)))
	unit := uploader.NewBufioSplitUnit(buffer, transfer.Range{Begin: 2, End: size}, nil, nil)

	buf := cachepool.RawMallocByteSlice(1022)
	for {
		n, err := unit.Read(buf)
		if err != nil {
			if err == io.EOF {
				break
			}
			t.Fatalf("read error: %s\n", err)
		}
		fmt.Printf("n: %d, left: %d\n", n, unit.Left())
	}
}
