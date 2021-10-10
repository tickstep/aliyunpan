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
package waitgroup

import (
	"fmt"
	"testing"
	"time"
)

func TestWg(t *testing.T) {
	wg := NewWaitGroup(2)
	for i := 0; i < 60; i++ {
		wg.AddDelta()
		go func(i int) {
			fmt.Println(i, wg.Parallel())
			time.Sleep(1e9)
			wg.Done()
		}(i)
	}
	wg.Wait()
}
