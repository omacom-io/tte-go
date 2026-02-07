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
	config       config.TerminalConfig
	frameDelay   time.Duration
	lastLines    int
	initialized  bool
	termWidth    int
	termHeight   int
	canvasHeight int
	canvasWidth  int
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
	// Note: We intentionally do NOT clear the screen here.
	// The Python library preserves terminal history by only writing
	// to the canvas area. Clearing would erase previous output.
	t.initialized = true
}

func (t *Terminal) PrepareCanvas(height int, width int) {
	t.canvasHeight = height
	t.canvasWidth = width

	// If reuse-canvas is set, restore cursor to saved position and move up
	// to reuse the existing canvas area from a previous effect run
	if t.config.ReuseCanvas {
		fmt.Fprint(os.Stdout, utils.AnsiDecRestoreCursor)
		fmt.Fprint(os.Stdout, utils.AnsiDecSaveCursor)
		fmt.Fprint(os.Stdout, utils.MoveCursorUp(height))
	}

	// Print blank lines filled with spaces to establish canvas space.
	// This matches Python behavior which writes full-width lines to claim
	// the canvas area without erasing terminal history.
	blankLine := strings.Repeat(" ", width)
	for i := 0; i < height; i++ {
		fmt.Fprint(os.Stdout, blankLine+"\n")
	}
	fmt.Fprint(os.Stdout, utils.AnsiDecSaveCursor)
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

	// Restore cursor to saved position, save again, then move up to top of canvas
	fmt.Fprint(os.Stdout, utils.AnsiDecRestoreCursor)
	fmt.Fprint(os.Stdout, utils.AnsiDecSaveCursor)
	if t.canvasHeight > 0 {
		fmt.Fprint(os.Stdout, utils.MoveCursorUp(t.canvasHeight))
	}

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
