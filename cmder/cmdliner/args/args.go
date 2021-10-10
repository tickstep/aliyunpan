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
package args

import (
	"strings"
	"unicode"
)

const (
	CharEscape      = '\\'
	CharSingleQuote = '\''
	CharDoubleQuote = '"'
	CharBackQuote   = '`'
)

// IsQuote 是否为引号
func IsQuote(r rune) bool {
	return r == CharSingleQuote || r == CharDoubleQuote || r == CharBackQuote
}

// Parse 解析line, 忽略括号
func Parse(line string) (lineArgs []string) { // 在函数中定义的返回值变量，会自动赋为 zero-value，即相当于 var lineArgs string[]
	var (
		rl        = []rune(line + " ")
		buf       = strings.Builder{}
		quoteChar rune
		nextChar  rune
		escaped   bool
		in        bool
	)

	var (
		isSpace bool
	)

	for k, r := range rl {
		isSpace = unicode.IsSpace(r)
		if !isSpace && !in {
			in = true
		}

		switch {
		case escaped: // 已转义, 跳过
			escaped = false
			//pass
		case r == CharEscape: // 转义模式
			if k+1+1 < len(rl) { // 不是最后一个字符, 多+1是因为最后一个空格
				nextChar = rl[k+1]
				// 仅支持转义这些字符, 否则原样输出反斜杠
				if unicode.IsSpace(nextChar) || IsQuote(nextChar) || nextChar == CharEscape {
					escaped = true
					continue
				}
			}
			// pass
		case IsQuote(r):
			if quoteChar == 0 { //未引
				quoteChar = r
				continue
			}

			if quoteChar == r { //取消引
				quoteChar = 0
				continue
			}
		case isSpace:
			if !in { // 忽略多余的空格
				continue
			}
			if quoteChar == 0 { // 未在引号内
				lineArgs = append(lineArgs, buf.String())
				buf.Reset()
				in = false
				continue
			}
		}

		buf.WriteRune(r)
	}

	// Go 允许在定义函数时，命名返回值，当然这些变量可以在函数中使用。
	// 在 return 语句中，无需显示的返回这些值，Go 会自动将其返回。当然 return 语句还是必须要写的，否则编译器会报错。
	// 相当于 return lineArgs
	return
}
