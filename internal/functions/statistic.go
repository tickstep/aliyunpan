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
package functions

import (
	"github.com/tickstep/library-go/expires"
	"sync/atomic"
	"time"
)

type (
	Statistic struct {
		totalSize int64
		startTime time.Time
	}
)

func (s *Statistic) AddTotalSize(size int64) int64 {
	return atomic.AddInt64(&s.totalSize, size)
}

func (s *Statistic) TotalSize() int64 {
	return s.totalSize
}

func (s *Statistic) StartTimer() {
	s.startTime = time.Now()
	expires.StripMono(&s.startTime)
}

func (s *Statistic) Elapsed() time.Duration {
	return time.Now().Sub(s.startTime)
}