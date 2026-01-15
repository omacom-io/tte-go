package engine

import (
	"regexp"
	"strconv"
	"strings"

	"tte-go/internal/config"
	"tte-go/internal/terminal"
	"tte-go/internal/utils"
)

var ansiPattern = regexp.MustCompile("\x1b\\[[0-9;?]*[A-Za-z]")

func ParseInput(input string, cfg config.TerminalConfig) (*terminal.Canvas, []*EffectCharacter) {
	normalized := terminal.NormalizeInput(input, cfg.TabWidth)
	lines := strings.Split(normalized, "\n")
	trimmed := make([]string, 0, len(lines))
	for _, line := range lines {
		trimmed = append(trimmed, strings.TrimRight(line, " \t\r"))
	}
	for len(trimmed) > 0 && trimmed[len(trimmed)-1] == "" {
		trimmed = trimmed[:len(trimmed)-1]
	}
	normalized = strings.Join(trimmed, "\n")
	lines = trimmed
	canvas := terminal.NewCanvas(normalized, cfg)
	characters := make([]*EffectCharacter, 0)

	currentColors := &utils.ColorPair{}
	inputHeight := len(lines)
	for row, line := range lines {
		col := 0
		for len(line) > 0 {
			if loc := ansiPattern.FindStringIndex(line); loc != nil && loc[0] == 0 {
				sequence := line[:loc[1]]
				line = line[loc[1]:]
				applyAnsiColor(sequence, currentColors)
				continue
			}
			r, size := nextRune(line)
			line = line[size:]
			coord := utils.Coord{Row: inputHeight - row, Col: col + 1}
			if string(r) != " " {
				char := NewEffectCharacter(string(r), coord)
				char.InputColors = cloneColors(currentColors)
				char.ExistingColorHandling = cfg.ExistingColorHandling
				char.UseXterm = cfg.XtermColors
				char.NoColor = cfg.NoColor
				if char.Animation != nil {
					char.Animation.UseXterm = cfg.XtermColors
					char.Animation.NoColor = cfg.NoColor
					char.Animation.InputColors = char.InputColors
				}
				char.SetVisual(CharacterVisual{
					Symbol:   string(r),
					Colors:   char.InputColors,
					UseXterm: cfg.XtermColors,
					NoColor:  cfg.NoColor,
				})
				characters = append(characters, char)
			}
			col++
		}
	}
	characters = ApplyAnchor(canvas, characters, cfg.AnchorText)
	BuildNeighbors(characters)
	return canvas, characters
}

func nextRune(s string) (rune, int) {
	if s == "" {
		return 0, 0
	}
	r := []rune(s)
	if len(r) == 0 {
		return 0, 0
	}
	return r[0], len(string(r[0]))
}

func applyAnsiColor(sequence string, colors *utils.ColorPair) {
	parsed, err := utils.ParseAnsiColorSequence(sequence)
	if err != nil {
		return
	}
	if strings.Contains(sequence, "38;") {
		if color, err := parseColor(parsed); err == nil {
			colors.FG = color
		}
	}
	if strings.Contains(sequence, "48;") {
		if color, err := parseColor(parsed); err == nil {
			colors.BG = color
		}
	}
}

func parseColor(value string) (*utils.Color, error) {
	if len(value) <= 3 {
		code, err := strconv.Atoi(value)
		if err != nil {
			return nil, err
		}
		hex := utils.XtermToHex(code)
		return utils.ParseHexColor(hex)
	}
	return utils.ParseHexColor(value)
}

func cloneColors(colors *utils.ColorPair) *utils.ColorPair {
	if colors == nil {
		return nil
	}
	clone := &utils.ColorPair{}
	if colors.FG != nil {
		fg := *colors.FG
		clone.FG = &fg
	}
	if colors.BG != nil {
		bg := *colors.BG
		clone.BG = &bg
	}
	return clone
}
