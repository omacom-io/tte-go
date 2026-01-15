package engine

import (
	"strings"

	"tte-go/internal/terminal"
)

func RenderFrame(canvas *terminal.Canvas, characters []*EffectCharacter) string {
	if canvas == nil {
		return ""
	}
	grid := make([][]string, canvas.Height)
	layerGrid := make([][]int, canvas.Height)
	for row := 0; row < canvas.Height; row++ {
		grid[row] = make([]string, canvas.Width)
		layerGrid[row] = make([]int, canvas.Width)
		for col := 0; col < canvas.Width; col++ {
			grid[row][col] = " "
			layerGrid[row][col] = -999
		}
	}

	for _, character := range characters {
		if character == nil || !character.Visible {
			continue
		}
		row := character.Coord.Row
		col := character.Coord.Col
		if character.Motion != nil {
			row = character.Motion.CurrentCoord.Row
			col = character.Motion.CurrentCoord.Col
		}
		if row < 1 || col < 1 || row > canvas.Height || col > canvas.Width {
			continue
		}
		rowIndex := row - 1
		colIndex := col - 1
		if character.Layer >= layerGrid[rowIndex][colIndex] {
			visual := character.Visual()
			visual.UseXterm = character.UseXterm
			visual.NoColor = character.NoColor
			switch character.ExistingColorHandling {
			case "always":
				if character.InputColors != nil {
					visual.Colors = character.InputColors
				}
			case "dynamic":
				if visual.Colors == nil && character.InputColors != nil {
					visual.Colors = character.InputColors
				}
			case "ignore":
				// leave colors as-is
			default:
				// leave colors as-is
			}
			grid[rowIndex][colIndex] = visual.Formatted()
			layerGrid[rowIndex][colIndex] = character.Layer
		}
	}

	builder := strings.Builder{}
	for row := canvas.Height - 1; row >= 0; row-- {
		line := strings.Join(grid[row], "")
		line = strings.TrimRight(line, " ")
		builder.WriteString(line)
		if row > 0 {
			builder.WriteString("\n")
		}
	}
	return builder.String()
}
