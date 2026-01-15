package engine

import (
	"strings"

	"tte-go/internal/utils"
)

type CharacterVisual struct {
	Symbol    string
	Bold      bool
	Dim       bool
	Italic    bool
	Underline bool
	Blink     bool
	Reverse   bool
	Hidden    bool
	Strike    bool
	Colors    *utils.ColorPair
	UseXterm  bool
	NoColor   bool
}

func (v CharacterVisual) Formatted() string {
	builder := strings.Builder{}
	if v.Bold {
		builder.WriteString(utils.AnsiBold)
	}
	if v.Dim {
		builder.WriteString(utils.AnsiDim)
	}
	if v.Italic {
		builder.WriteString(utils.AnsiItalic)
	}
	if v.Underline {
		builder.WriteString(utils.AnsiUnderline)
	}
	if v.Blink {
		builder.WriteString(utils.AnsiBlink)
	}
	if v.Reverse {
		builder.WriteString(utils.AnsiReverse)
	}
	if v.Hidden {
		builder.WriteString(utils.AnsiHidden)
	}
	if v.Strike {
		builder.WriteString(utils.AnsiStrike)
	}
	if v.Colors != nil && !v.NoColor {
		if v.Colors.FG != nil {
			builder.WriteString(colorSequence(*v.Colors.FG, v.UseXterm, true))
		}
		if v.Colors.BG != nil {
			builder.WriteString(colorSequence(*v.Colors.BG, v.UseXterm, false))
		}
	}
	builder.WriteString(v.Symbol)
	if builder.Len() > len(v.Symbol) {
		builder.WriteString(utils.AnsiReset)
	}
	return builder.String()
}

func colorSequence(color utils.Color, useXterm bool, foreground bool) string {
	if useXterm {
		code := utils.NearestXtermColor(color)
		if foreground {
			return utils.FgXterm(code)
		}
		return utils.BgXterm(code)
	}
	if foreground {
		return utils.FgRGB(color.Hex())
	}
	return utils.BgRGB(color.Hex())
}
