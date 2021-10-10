// +build !windows

package cmdliner

import (
	"fmt"
)

// ClearScreen 清空屏幕
func (pl *CmdLiner) ClearScreen() {
	ClearScreen()
}

// ClearScreen 清空屏幕
func ClearScreen() {
	fmt.Print("\x1b[H\x1b[2J")
}
