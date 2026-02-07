package effects

import (
	"tte-go/internal/config"
	"tte-go/internal/engine"
	"tte-go/internal/utils"
)

// Overflow creates an effect where input text overflows and scrolls the terminal
// in a random order until eventually appearing ordered.
type Overflow struct {
	base *BaseEffect

	pendingRows []*overflowRow
	activeRows  []*overflowRow
	delay       int

	overflowGradient []utils.Color

	// Config
	overflowGradientStops  []utils.Color
	overflowCyclesRange    [2]int
	overflowSpeed          int
	finalGradientStops     []utils.Color
	finalGradientSteps     []int
	finalGradientDirection utils.GradientDirection
}

type overflowRow struct {
	characters []*engine.EffectCharacter
	final      bool
}

func newOverflowRow(chars []*engine.EffectCharacter, final bool) *overflowRow {
	return &overflowRow{characters: chars, final: final}
}

func (r *overflowRow) moveUp() {
	for _, char := range r.characters {
		currentRow := char.Motion.CurrentCoord.Row
		newCoord := utils.Coord{Col: char.Motion.CurrentCoord.Col, Row: currentRow + 1}
		char.Motion.SetCoordinate(newCoord)
		char.Coord = newCoord
	}
}

func (r *overflowRow) setup() {
	for _, char := range r.characters {
		newCoord := utils.Coord{Col: char.InputCoord.Col, Row: 0}
		char.Motion.SetCoordinate(newCoord)
		char.Coord = newCoord
	}
}

func (r *overflowRow) setColor(color utils.Color) {
	for _, char := range r.characters {
		colorCopy := color
		char.SetVisual(engine.CharacterVisual{
			Symbol: char.Symbol,
			Colors: &utils.ColorPair{FG: &colorCopy},
		})
	}
}

func NewOverflow(input string, cfg config.TerminalConfig) engine.Effect {
	base := NewBaseEffect(input, cfg)

	// Config defaults
	overflowGradientStops := mustColors("f2ebc0", "8dbfb3", "f2ebc0")
	finalGradientStops := mustColors("8A008A", "00D1FF", "FFFFFF")
	finalGradientSteps := []int{12}
	finalGradientDirection := utils.GradientVertical

	o := &Overflow{
		base:                   base,
		pendingRows:            make([]*overflowRow, 0),
		activeRows:             make([]*overflowRow, 0),
		delay:                  0,
		overflowGradientStops:  overflowGradientStops,
		overflowCyclesRange:    [2]int{2, 4},
		overflowSpeed:          3,
		finalGradientStops:     finalGradientStops,
		finalGradientSteps:     finalGradientSteps,
		finalGradientDirection: finalGradientDirection,
	}

	o.build()
	return o
}

func (o *Overflow) build() {
	// Build final gradient mapping
	finalGradient, _ := utils.NewGradient(o.finalGradientStops, o.finalGradientSteps, false)
	finalGradientMapping, _ := finalGradient.BuildCoordinateColorMapping(
		o.base.Canvas.TextBottom,
		o.base.Canvas.TextTop,
		o.base.Canvas.TextLeft,
		o.base.Canvas.TextRight,
		o.finalGradientDirection,
	)

	// Build overflow gradient based on canvas height
	steps := max(o.base.Canvas.Top/max(1, len(o.overflowGradientStops)-1), 1)
	overflowGradient, _ := utils.NewGradient(o.overflowGradientStops, []int{steps}, false)
	o.overflowGradient = overflowGradient.Spectrum

	// Hide all characters initially - they'll be revealed as they scroll in
	for _, char := range o.base.Terminal.GetCharacters(true, true, true, false, engine.TopToBottomLeftToRight) {
		o.base.Terminal.SetCharacterVisibility(char, false)
	}

	// Get rows
	rows := o.base.Terminal.GetCharactersGrouped(engine.RowTopToBottom, true, true, true, false)

	// Add shuffled rows for overflow cycles
	numCycles := o.overflowCyclesRange[0] + utils.RandIntn(o.overflowCyclesRange[1]-o.overflowCyclesRange[0]+1)
	for i := 0; i < numCycles; i++ {
		// Shuffle rows
		shuffled := make([][]*engine.EffectCharacter, len(rows))
		copy(shuffled, rows)
		for j := len(shuffled) - 1; j > 0; j-- {
			k := utils.RandIntn(j + 1)
			shuffled[j], shuffled[k] = shuffled[k], shuffled[j]
		}

		// Create copied characters for each row (start hidden)
		for _, row := range shuffled {
			copiedChars := make([]*engine.EffectCharacter, 0, len(row))
			for _, char := range row {
				charCopy := engine.NewEffectCharacter(char.Symbol, char.InputCoord)
				charCopy.IsAddedCharacter = true
				o.base.Terminal.AddCharacter(charCopy)
				o.base.Terminal.SetCharacterVisibility(charCopy, false) // Hide after adding
				copiedChars = append(copiedChars, charCopy)
			}
			o.pendingRows = append(o.pendingRows, newOverflowRow(copiedChars, false))
		}
	}

	// Add final rows in correct order
	for _, row := range rows {
		finalRow := newOverflowRow(row, true)
		for _, char := range row {
			charFinalColor := finalGradientMapping[char.InputCoord]
			colorCopy := charFinalColor
			char.SetVisual(engine.CharacterVisual{
				Symbol: char.Symbol,
				Colors: &utils.ColorPair{FG: &colorCopy},
			})
		}
		o.pendingRows = append(o.pendingRows, finalRow)
	}
}

func (o *Overflow) Next() (string, bool) {
	if len(o.pendingRows) > 0 {
		if o.delay == 0 {
			numToProcess := 1 + utils.RandIntn(o.overflowSpeed)
			for i := 0; i < numToProcess && len(o.pendingRows) > 0; i++ {
				// Move all active rows up
				for _, row := range o.activeRows {
					row.moveUp()
					if !row.final && len(row.characters) > 0 {
						// Set color based on row position
						rowY := row.characters[0].Motion.CurrentCoord.Row
						colorIdx := min(rowY, len(o.overflowGradient)-1)
						if colorIdx >= 0 && colorIdx < len(o.overflowGradient) {
							row.setColor(o.overflowGradient[colorIdx])
						}
					}
				}

				// Add next row
				nextRow := o.pendingRows[0]
				o.pendingRows = o.pendingRows[1:]
				nextRow.setup()
				nextRow.moveUp()
				if !nextRow.final && len(o.overflowGradient) > 0 {
					nextRow.setColor(o.overflowGradient[0])
				}
				for _, char := range nextRow.characters {
					o.base.Terminal.SetCharacterVisibility(char, true)
				}
				o.activeRows = append(o.activeRows, nextRow)
			}
			o.delay = utils.RandIntn(4)
		} else {
			o.delay--
		}

		// Remove rows that have scrolled off the top (runs every frame)
		remaining := make([]*overflowRow, 0)
		for _, row := range o.activeRows {
			if len(row.characters) > 0 && row.characters[0].Motion.CurrentCoord.Row <= o.base.Canvas.Top {
				remaining = append(remaining, row)
			} else {
				// Hide characters that scrolled off
				for _, char := range row.characters {
					o.base.Terminal.SetCharacterVisibility(char, false)
				}
			}
		}
		o.activeRows = remaining

		return o.base.Terminal.GetFormattedOutputString(), true
	}

	return o.base.Terminal.GetFormattedOutputString(), false
}

func (o *Overflow) CanvasHeight() int {
	return o.base.CanvasHeight()
}

func (o *Overflow) CanvasWidth() int {
	return o.base.CanvasWidth()
}
