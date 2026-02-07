package effects

import (
	"tte-go/internal/config"
	"tte-go/internal/engine"
	"tte-go/internal/utils"
)

// BouncyBalls creates an effect where characters fall from the top of the canvas
// as bouncy balls before settling into place.
type BouncyBalls struct {
	base *BaseEffect

	pendingChars     []*engine.EffectCharacter
	groupByRow       map[int][]*engine.EffectCharacter
	activeCharacters map[*engine.EffectCharacter]struct{}
	ballDelay        int

	// Config
	ballColors     []utils.Color
	ballSymbols    []string
	ballDelayMax   int
	movementSpeed  float64
	movementEasing utils.EasingFunction

	finalGradientStops     []utils.Color
	finalGradientSteps     []int
	finalGradientDirection utils.GradientDirection
}

func NewBouncyBalls(input string, cfg config.TerminalConfig) engine.Effect {
	base := NewBaseEffect(input, cfg)

	// Config defaults
	ballColors := mustColors("d1f4a5", "96e2a4", "5acda9")
	ballSymbols := []string{"*", "o", "O", "0", "."}
	ballDelayMax := 4
	movementSpeed := 0.45
	movementEasing := utils.OutBounce

	finalGradientStops := mustColors("f8ffae", "43c6ac")
	finalGradientSteps := []int{12}
	finalGradientDirection := utils.GradientDiagonal

	b := &BouncyBalls{
		base:                   base,
		pendingChars:           make([]*engine.EffectCharacter, 0),
		groupByRow:             make(map[int][]*engine.EffectCharacter),
		activeCharacters:       make(map[*engine.EffectCharacter]struct{}),
		ballDelay:              0,
		ballColors:             ballColors,
		ballSymbols:            ballSymbols,
		ballDelayMax:           ballDelayMax,
		movementSpeed:          movementSpeed,
		movementEasing:         movementEasing,
		finalGradientStops:     finalGradientStops,
		finalGradientSteps:     finalGradientSteps,
		finalGradientDirection: finalGradientDirection,
	}

	b.build()
	return b
}

func (b *BouncyBalls) build() {
	// Build final gradient mapping
	finalGradient, _ := utils.NewGradient(b.finalGradientStops, b.finalGradientSteps, false)
	finalGradientMapping, _ := finalGradient.BuildCoordinateColorMapping(
		b.base.Canvas.TextBottom,
		b.base.Canvas.TextTop,
		b.base.Canvas.TextLeft,
		b.base.Canvas.TextRight,
		b.finalGradientDirection,
	)

	for _, character := range b.base.Terminal.GetCharacters(true, false, false, false, engine.TopToBottomLeftToRight) {
		// Get the final color for this character based on its position
		charFinalColor := finalGradientMapping[character.InputCoord]

		// Random ball color and symbol
		color := b.ballColors[utils.RandIntn(len(b.ballColors))]
		symbol := b.ballSymbols[utils.RandIntn(len(b.ballSymbols))]

		// Ball scene - show the bouncing ball symbol with ball color
		ballScene := character.Animation.NewScene("ball")
		colorCopy := color
		_ = ballScene.AddFrame(symbol, 1, &utils.ColorPair{FG: &colorCopy})

		// Final scene - gradient from ball color to final color, then show original symbol
		finalScene := character.Animation.NewScene("final")
		charFinalGradient, _ := utils.NewGradient([]utils.Color{color, charFinalColor}, []int{10}, false)
		_ = finalScene.ApplyGradientToSymbols([]string{character.Symbol}, 6, charFinalGradient, nil)

		// Position character above the canvas
		// Random height between 1.0 and 1.5 times the canvas top
		startRow := int(float64(b.base.Canvas.Top) * (1.0 + utils.RandFloat64()*0.5))
		character.Motion.SetCoordinate(utils.Coord{Col: character.InputCoord.Col, Row: startRow})

		// Create path to final position with bounce easing
		inputCoordPath, _ := character.Motion.NewPath(b.movementSpeed, b.movementEasing, nil, 0, false, "input_coord")
		inputCoordPath.AddWaypoint(character.InputCoord)
		character.Motion.ActivatePath(inputCoordPath)
		character.Animation.ActivateScene("ball")

		// Register event: when path completes, activate final scene
		_ = character.EventHandler.RegisterEvent(
			engine.EventPathComplete,
			inputCoordPath,
			engine.ActionActivateScene,
			finalScene,
		)

		b.pendingChars = append(b.pendingChars, character)
	}

	// Group characters by row (for row-by-row release)
	for _, character := range b.pendingChars {
		row := character.InputCoord.Row
		b.groupByRow[row] = append(b.groupByRow[row], character)
	}
	b.pendingChars = b.pendingChars[:0] // Clear pending chars
}

func (b *BouncyBalls) Next() (string, bool) {
	// Check if we have more work to do
	if len(b.groupByRow) == 0 && len(b.activeCharacters) == 0 && len(b.pendingChars) == 0 {
		return "", false
	}

	// If pending chars is empty and we have more rows, pop the minimum row
	if len(b.pendingChars) == 0 && len(b.groupByRow) > 0 {
		minRow := b.findMinRow()
		b.pendingChars = append(b.pendingChars, b.groupByRow[minRow]...)
		delete(b.groupByRow, minRow)
	}

	// Release characters from pending
	if len(b.pendingChars) > 0 {
		if b.ballDelay == 0 {
			// Release 2-6 random characters
			numToRelease := 2 + utils.RandIntn(5) // 2-6
			for i := 0; i < numToRelease && len(b.pendingChars) > 0; i++ {
				idx := utils.RandIntn(len(b.pendingChars))
				character := b.pendingChars[idx]
				// Remove from pending
				b.pendingChars[idx] = b.pendingChars[len(b.pendingChars)-1]
				b.pendingChars = b.pendingChars[:len(b.pendingChars)-1]

				b.base.Terminal.SetCharacterVisibility(character, true)
				b.activeCharacters[character] = struct{}{}
			}
			b.ballDelay = b.ballDelayMax
		} else {
			b.ballDelay--
		}
	}

	// Update all active characters
	for character := range b.activeCharacters {
		character.Tick()
		if !character.IsActive() {
			delete(b.activeCharacters, character)
		}
	}

	frame := b.base.Terminal.GetFormattedOutputString()
	return frame, true
}

func (b *BouncyBalls) findMinRow() int {
	minRow := int(^uint(0) >> 1) // MaxInt
	for row := range b.groupByRow {
		if row < minRow {
			minRow = row
		}
	}
	return minRow
}

func (b *BouncyBalls) CanvasHeight() int {
	return b.base.CanvasHeight()
}

func (b *BouncyBalls) CanvasWidth() int {
	return b.base.CanvasWidth()
}
