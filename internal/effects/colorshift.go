package effects

import (
	"math"

	"tte-go/internal/config"
	"tte-go/internal/engine"
	"tte-go/internal/terminal"
	"tte-go/internal/utils"
)

// ColorShift displays a gradient that shifts colors across the terminal.
// Uses animation scenes with event-based cycle tracking like Python.
type ColorShift struct {
	base             *BaseEffect
	activeCharacters map[*engine.EffectCharacter]struct{}
	loopTracker      map[*engine.EffectCharacter]int

	// Config
	cycles            int
	skipFinalGradient bool
}

func NewColorShift(input string, cfg config.TerminalConfig) engine.Effect {
	base := NewBaseEffect(input, cfg)

	// Configuration (rainbow colors)
	gradientStops := mustColors("#e81416", "#ffa500", "#faeb36", "#79c314", "#487de7", "#4b369d", "#70369d")
	gradientSteps := []int{12}
	gradientFrames := 2
	cycles := 3
	travelDirection := utils.GradientRadial
	reverseTravel := false
	noTravel := false
	noLoop := false
	skipFinalGradient := false

	// Build main gradient (looped unless noLoop)
	gradient, _ := utils.NewGradient(gradientStops, gradientSteps, !noLoop)

	// Build final gradient mapping
	finalGradientStops := mustColors("#e81416", "#ffa500", "#faeb36", "#79c314", "#487de7", "#4b369d", "#70369d")
	finalGradient, _ := utils.NewGradient(finalGradientStops, []int{12}, false)
	finalMapping, _ := finalGradient.BuildCoordinateColorMapping(
		base.Canvas.TextBottom, base.Canvas.TextTop,
		base.Canvas.TextLeft, base.Canvas.TextRight,
		utils.GradientVertical,
	)

	c := &ColorShift{
		base:              base,
		activeCharacters:  make(map[*engine.EffectCharacter]struct{}),
		loopTracker:       make(map[*engine.EffectCharacter]int),
		cycles:            cycles,
		skipFinalGradient: skipFinalGradient,
	}

	// Build scenes for each character
	for _, character := range base.Terminal.GetCharacters(true, false, false, false, engine.TopToBottomLeftToRight) {
		base.Terminal.SetCharacterVisibility(character, true)

		// Calculate shifted colors for this character based on position
		var colors []utils.Color
		if noTravel {
			colors = gradient.Spectrum
		} else {
			directionIndex := getDirectionIndex(character, base.Canvas, travelDirection)
			shiftDistance := int(float64(len(gradient.Spectrum)) * directionIndex)
			if reverseTravel {
				shiftDistance = -shiftDistance
			}
			// Shift the spectrum
			colors = shiftSpectrum(gradient.Spectrum, shiftDistance)
		}

		// Create gradient scene with shifted colors
		gradientScene := character.Animation.NewScene("gradient")
		for _, color := range colors {
			colorCopy := color
			_ = gradientScene.AddFrame(character.Symbol, gradientFrames, &utils.ColorPair{FG: &colorCopy})
		}

		// Create final gradient scene (transition from last color to final mapped color)
		finalScene := character.Animation.NewScene("final_gradient")
		finalColor := finalMapping[character.InputCoord]
		if len(colors) > 0 {
			lastColor := colors[len(colors)-1]
			transitionGradient, _ := utils.NewGradient([]utils.Color{lastColor, finalColor}, []int{8}, false)
			if transitionGradient != nil {
				for _, color := range transitionGradient.Spectrum {
					colorCopy := color
					_ = finalScene.AddFrame(character.Symbol, gradientFrames, &utils.ColorPair{FG: &colorCopy})
				}
			}
		} else {
			_ = finalScene.AddFrame(character.Symbol, gradientFrames, &utils.ColorPair{FG: &finalColor})
		}

		// Register event: when gradient scene completes, call loop tracker
		_ = character.EventHandler.RegisterEvent(
			engine.EventSceneComplete,
			gradientScene,
			engine.ActionCallback,
			func(char *engine.EffectCharacter) {
				c.loopTracker[char]++
				if c.cycles == 0 || c.loopTracker[char] < c.cycles {
					// Restart gradient scene
					char.Animation.ActivateScene("gradient")
				} else if !c.skipFinalGradient {
					// Activate final gradient scene
					char.Animation.ActivateScene("final_gradient")
				}
				// If skipFinalGradient and cycles complete, scene just ends
			},
		)

		// Activate gradient scene
		character.Animation.ActivateScene("gradient")
		c.activeCharacters[character] = struct{}{}
	}

	return c
}

func getDirectionIndex(char *engine.EffectCharacter, canvas *terminal.Canvas, direction utils.GradientDirection) float64 {
	switch direction {
	case utils.GradientHorizontal:
		if canvas.Right == 0 {
			return 0
		}
		return float64(char.InputCoord.Col) / float64(canvas.Right)
	case utils.GradientVertical:
		if canvas.Top == 0 {
			return 0
		}
		return float64(char.InputCoord.Row) / float64(canvas.Top)
	case utils.GradientDiagonal:
		denom := canvas.Right + canvas.Top
		if denom == 0 {
			return 0
		}
		return float64(char.InputCoord.Row+char.InputCoord.Col) / float64(denom)
	case utils.GradientRadial:
		// Normalized distance from center
		centerRow := float64(canvas.TextBottom+canvas.TextTop) / 2
		centerCol := float64(canvas.TextLeft+canvas.TextRight) / 2
		row := float64(char.InputCoord.Row)
		col := float64(char.InputCoord.Col)

		dx := col - centerCol
		dy := row - centerRow

		maxDx := float64(canvas.TextRight-canvas.TextLeft) / 2
		maxDy := float64(canvas.TextTop-canvas.TextBottom) / 2
		maxDist := math.Sqrt(maxDx*maxDx + maxDy*maxDy)
		if maxDist == 0 {
			return 0
		}

		dist := math.Sqrt(dx*dx + dy*dy)
		return dist / maxDist
	}
	return 0
}

func shiftSpectrum(spectrum []utils.Color, shift int) []utils.Color {
	n := len(spectrum)
	if n == 0 {
		return spectrum
	}
	// Normalize shift to positive
	shift = shift % n
	if shift < 0 {
		shift += n
	}
	// Python: spectrum[shift:] + spectrum[:shift]
	result := make([]utils.Color, n)
	copy(result, spectrum[shift:])
	copy(result[n-shift:], spectrum[:shift])
	return result
}

func (c *ColorShift) Next() (string, bool) {
	if len(c.activeCharacters) == 0 {
		return c.base.Terminal.GetFormattedOutputString(), false
	}

	// Tick all active characters
	for char := range c.activeCharacters {
		char.Tick()
		if !char.IsActive() {
			delete(c.activeCharacters, char)
		}
	}

	return c.base.Terminal.GetFormattedOutputString(), true
}

func (c *ColorShift) CanvasHeight() int {
	return c.base.CanvasHeight()
}

func (c *ColorShift) CanvasWidth() int {
	return c.base.CanvasWidth()
}
