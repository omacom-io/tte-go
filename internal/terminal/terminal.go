package terminal

import (
	"fmt"
	"os"
	"strings"
	"time"

	"tte-go/internal/config"
	"tte-go/internal/utils"
)

type Terminal struct {
	config      config.TerminalConfig
	frameDelay  time.Duration
	lastLines   int
	initialized bool
	termWidth   int
	termHeight  int
}

func New(cfg config.TerminalConfig) *Terminal {
	delay := time.Duration(0)
	if cfg.FrameRate > 0 {
		delay = time.Second / time.Duration(cfg.FrameRate)
	}
	return &Terminal{config: cfg, frameDelay: delay}
}

func (t *Terminal) Prepare() {
	if t.initialized {
		return
	}
	fmt.Fprint(os.Stdout, utils.AnsiHideCursor)
	fmt.Fprint(os.Stdout, utils.AnsiClearScreen)
	fmt.Fprint(os.Stdout, utils.AnsiHomeCursor)
	t.initialized = true
}

func (t *Terminal) Restore(endSymbol string) {
	if t.config.NoRestoreCursor {
		return
	}
	fmt.Fprint(os.Stdout, utils.AnsiShowCursor)
	if endSymbol != "" {
		fmt.Fprint(os.Stdout, endSymbol)
	}
}

func (t *Terminal) PrintFrame(frame string, isLast bool) {
	if t.frameDelay > 0 {
		time.Sleep(t.frameDelay)
	}

	normalized := normalizeNewlines(frame)
	fmt.Fprint(os.Stdout, utils.AnsiClearScreen)
	fmt.Fprint(os.Stdout, utils.AnsiHomeCursor)
	fmt.Fprint(os.Stdout, normalized)

	lines := countLines(frame)
	t.lastLines = lines

	if isLast {
		if !t.config.NoEOL {
			fmt.Fprint(os.Stdout, "\n")
		}
	}
}

func countLines(frame string) int {
	if frame == "" {
		return 0
	}
	return strings.Count(frame, "\n") + 1
}

func normalizeNewlines(frame string) string {
	return strings.ReplaceAll(frame, "\r\n", "\n")
}
