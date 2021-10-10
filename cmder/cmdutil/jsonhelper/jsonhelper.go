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
package jsonhelper

import (
	"github.com/json-iterator/go"
	"io"
)

// UnmarshalData 将 r 中的 json 格式的数据, 解析到 data
func UnmarshalData(r io.Reader, data interface{}) error {
	d := jsoniter.NewDecoder(r)
	return d.Decode(data)
}

// MarshalData 将 data, 生成 json 格式的数据, 写入 w 中
func MarshalData(w io.Writer, data interface{}) error {
	e := jsoniter.NewEncoder(w)
	return e.Encode(data)
}
