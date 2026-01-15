package terminal

import (
	"os"
	"strconv"
	"syscall"
	"unsafe"
)

type winsize struct {
	rows uint16
	cols uint16
	x    uint16
	y    uint16
}

func getTerminalSize() (int, int) {
	if cols, rows, ok := ioctlSize(os.Stdout.Fd()); ok {
		return cols, rows
	}
	if cols, rows, ok := ioctlSize(os.Stdin.Fd()); ok {
		return cols, rows
	}
	if cols, rows, ok := ioctlSize(os.Stderr.Fd()); ok {
		return cols, rows
	}
	if cols, ok := parseEnvInt("COLUMNS"); ok {
		if rows, okRows := parseEnvInt("LINES"); okRows {
			return cols, rows
		}
	}
	return 0, 0
}

func ioctlSize(fd uintptr) (int, int, bool) {
	ws := &winsize{}
	_, _, err := syscall.Syscall(syscall.SYS_IOCTL, fd, uintptr(syscall.TIOCGWINSZ), uintptr(unsafe.Pointer(ws)))
	if err == 0 && ws.cols > 0 && ws.rows > 0 {
		return int(ws.cols), int(ws.rows), true
	}
	return 0, 0, false
}

func parseEnvInt(key string) (int, bool) {
	value := os.Getenv(key)
	if value == "" {
		return 0, false
	}
	parsed, err := strconv.Atoi(value)
	if err != nil {
		return 0, false
	}
	return parsed, true
}
