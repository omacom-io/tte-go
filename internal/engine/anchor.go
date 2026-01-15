package engine

import (
	"tte-go/internal/terminal"
	"tte-go/internal/utils"
)

func ApplyAnchor(canvas *terminal.Canvas, characters []*EffectCharacter, anchor string) []*EffectCharacter {
	if len(characters) == 0 {
		return characters
	}
	inputWidth := 0
	inputHeight := 0
	for _, character := range characters {
		if character.InputCoord.Col > inputWidth {
			inputWidth = character.InputCoord.Col
		}
		if character.InputCoord.Row > inputHeight {
			inputHeight = character.InputCoord.Row
		}
	}

	columnDelta := 0
	rowDelta := 0
	if inputWidth != canvas.Width {
		switch anchor {
		case "s", "n", "c":
			columnDelta = canvas.Center.Col - (inputWidth / 2)
		case "se", "e", "ne":
			columnDelta = canvas.Right - inputWidth
		case "sw", "w", "nw":
			columnDelta = canvas.Left - 1
		}
	}
	if inputHeight != canvas.Height {
		switch anchor {
		case "w", "e", "c":
			rowDelta = canvas.Center.Row - (inputHeight / 2)
		case "nw", "n", "ne":
			rowDelta = canvas.Top - inputHeight
		case "sw", "s", "se":
			rowDelta = canvas.Bottom - 1
		}
	}

	anchored := make([]*EffectCharacter, 0, len(characters))
	for _, character := range characters {
		coord := utils.Coord{Row: character.InputCoord.Row + rowDelta, Col: character.InputCoord.Col + columnDelta}
		character.InputCoord = coord
		character.Coord = coord
		if character.Motion != nil {
			character.Motion.CurrentCoord = coord
		}
		if canvas.Left <= coord.Col && coord.Col <= canvas.Right && canvas.Bottom <= coord.Row && coord.Row <= canvas.Top {
			anchored = append(anchored, character)
		}
	}
	if len(anchored) > 0 {
		canvas.TextLeft = minCol(anchored)
		canvas.TextRight = maxCol(anchored)
		canvas.TextBottom = minRow(anchored)
		canvas.TextTop = maxRow(anchored)
		canvas.TextWidth = maxInt(1, canvas.TextRight-canvas.TextLeft+1)
		canvas.TextHeight = maxInt(1, canvas.TextTop-canvas.TextBottom+1)
		canvas.TextCenter = utils.Coord{Row: canvas.TextBottom + ((canvas.TextTop - canvas.TextBottom) / 2), Col: canvas.TextLeft + ((canvas.TextRight - canvas.TextLeft) / 2)}
	}
	return anchored
}

func minCol(chars []*EffectCharacter) int {
	min := chars[0].InputCoord.Col
	for _, character := range chars {
		if character.InputCoord.Col < min {
			min = character.InputCoord.Col
		}
	}
	return min
}

func maxCol(chars []*EffectCharacter) int {
	max := chars[0].InputCoord.Col
	for _, character := range chars {
		if character.InputCoord.Col > max {
			max = character.InputCoord.Col
		}
	}
	return max
}

func minRow(chars []*EffectCharacter) int {
	min := chars[0].InputCoord.Row
	for _, character := range chars {
		if character.InputCoord.Row < min {
			min = character.InputCoord.Row
		}
	}
	return min
}

func maxRow(chars []*EffectCharacter) int {
	max := chars[0].InputCoord.Row
	for _, character := range chars {
		if character.InputCoord.Row > max {
			max = character.InputCoord.Row
		}
	}
	return max
}

func maxInt(a, b int) int {
	if a > b {
		return a
	}
	return b
}
