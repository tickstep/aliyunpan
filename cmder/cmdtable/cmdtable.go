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
package cmdtable

import (
	"github.com/olekukonko/tablewriter"
	"io"
)

type CmdTable struct {
	*tablewriter.Table
}

// NewTable 预设了一些配置
func NewTable(wt io.Writer) CmdTable {
	tb := tablewriter.NewWriter(wt)
	tb.SetAutoWrapText(false)
	tb.SetBorder(false)
	tb.SetHeaderLine(false)
	tb.SetColumnSeparator("")
	return CmdTable{tb}
}
