package terminal

import (
	"regexp"
	"strings"
	"unicode/utf8"

	"tte-go/internal/config"
	"tte-go/internal/utils"
)

var ansiRegex = regexp.MustCompile("\x1b\\[[0-9;?]*[A-Za-z]")

type Canvas struct {
	Width        int
	Height       int
	Left         int
	Right        int
	Bottom       int
	Top          int
	Center       utils.Coord
	TextWidth    int
	TextHeight   int
	AnchorCanvas string
	AnchorText   string
	Offset       utils.Coord
	TextLeft     int
	TextRight    int
	TextBottom   int
	TextTop      int
	TextCenter   utils.Coord
}

func NewCanvas(input string, cfg config.TerminalConfig) *Canvas {
	clean := stripAnsi(expandTabs(input, cfg.TabWidth))
	lines := strings.Split(clean, "\n")
	textHeight := len(lines)
	textWidth := 0
	for _, line := range lines {
		lineWidth := utf8.RuneCountInString(line)
		if lineWidth > textWidth {
			textWidth = lineWidth
		}
	}

	width := resolveDimension(cfg.CanvasWidth, textWidth)
	height := resolveDimension(cfg.CanvasHeight, textHeight)
	canvas := &Canvas{
		Width:        width,
		Height:       height,
		Left:         1,
		Right:        width,
		Bottom:       1,
		Top:          height,
		TextWidth:    textWidth,
		TextHeight:   textHeight,
		AnchorCanvas: cfg.AnchorCanvas,
		AnchorText:   cfg.AnchorText,
	}
	canvas.Center = canvas.computeCenter()
	canvas.Offset = canvas.computeOffset()
	canvas.computeTextBounds()
	return canvas
}

func resolveDimension(value int, fallback int) int {
	if value == 0 {
		return fallback
	}
	if value < 0 {
		return fallback
	}
	return value
}

func (c *Canvas) computeOffset() utils.Coord {
	anchorRow, anchorCol := anchorPosition(c.AnchorCanvas, c.Height, c.Width)
	textAnchorRow, textAnchorCol := anchorPosition(c.AnchorText, c.TextHeight, c.TextWidth)
	row := anchorRow - textAnchorRow
	col := anchorCol - textAnchorCol
	return utils.Coord{Row: row, Col: col}
}

func anchorPosition(anchor string, height int, width int) (int, int) {
	row := 0
	col := 0
	switch anchor {
	case "n":
		row = 0
		col = width / 2
	case "ne":
		row = 0
		col = width - 1
	case "e":
		row = height / 2
		col = width - 1
	case "se":
		row = height - 1
		col = width - 1
	case "s":
		row = height - 1
		col = width / 2
	case "sw":
		row = height - 1
		col = 0
	case "w":
		row = height / 2
		col = 0
	case "nw":
		row = 0
		col = 0
	case "c":
		row = height / 2
		col = width / 2
	default:
		row = height - 1
		col = 0
	}
	return row, col
}

func (c *Canvas) computeCenter() utils.Coord {
	centerRow := max(c.Top/2, c.Bottom)
	if c.Top%2 != 0 && c.Top > 1 {
		centerRow++
	}
	centerCol := max(c.Right/2, c.Left)
	if c.Right%2 != 0 && c.Right > 1 {
		centerCol++
	}
	return utils.Coord{Row: centerRow, Col: centerCol}
}

func (c *Canvas) computeTextBounds() {
	c.TextLeft = c.Offset.Col
	c.TextBottom = c.Offset.Row
	c.TextRight = c.TextLeft + c.TextWidth - 1
	c.TextTop = c.TextBottom + c.TextHeight - 1
	c.TextCenter = utils.Coord{Row: (c.TextBottom + c.TextTop) / 2, Col: (c.TextLeft + c.TextRight) / 2}
}

func NormalizeInput(input string, tabWidth int) string {
	return stripAnsi(expandTabs(input, tabWidth))
}

func stripAnsi(input string) string {
	return ansiRegex.ReplaceAllString(input, "")
}

func expandTabs(input string, tabWidth int) string {
	if tabWidth <= 0 {
		return input
	}
	return strings.ReplaceAll(input, "\t", strings.Repeat(" ", tabWidth))
}
