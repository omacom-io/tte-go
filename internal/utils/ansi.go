package utils

import "fmt"

const (
	AnsiHideCursor = "\x1b[?25l"
	AnsiShowCursor = "\x1b[?25h"
)

func MoveCursorUp(lines int) string {
	if lines <= 0 {
		return ""
	}
	return fmt.Sprintf("\x1b[%dA", lines)
}

func MoveCursorToColumn(column int) string {
	if column <= 0 {
		column = 1
	}
	return fmt.Sprintf("\x1b[%dG", column)
}
