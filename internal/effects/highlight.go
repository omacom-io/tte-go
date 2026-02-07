package effects

import (
	"math"

	"tte-go/internal/config"
	"tte-go/internal/engine"
	"tte-go/internal/utils"
)

// Highlight runs a specular highlight across the text.
type Highlight struct {
	base *BaseEffect

	characterGroups  [][]*engine.EffectCharacter
	activeCharacters map[*engine.EffectCharacter]struct{}
	currentStep      int
	totalSteps       int
	prevGroupIndex   int

	// Config
	highlightBrightness    float64
	highlightWidth         int
	finalGradientStops     []utils.Color
	finalGradientSteps     []int
	finalGradientDirection utils.GradientDirection
}

func NewHighlight(input string, cfg config.TerminalConfig) engine.Effect {
	base := NewBaseEffect(input, cfg)

	// Config defaults
	finalGradientStops := mustColors("8A008A", "00D1FF", "FFFFFF")
	finalGradientSteps := []int{12}
	finalGradientDirection := utils.GradientVertical

	h := &Highlight{
		base:                   base,
		activeCharacters:       make(map[*engine.EffectCharacter]struct{}),
		currentStep:            0,
		totalSteps:             100,
		prevGroupIndex:         0,
		highlightBrightness:    1.75,
		highlightWidth:         8,
		finalGradientStops:     finalGradientStops,
		finalGradientSteps:     finalGradientSteps,
		finalGradientDirection: finalGradientDirection,
	}

	h.build()
	return h
}

func (h *Highlight) build() {
	// Build final gradient mapping
	finalGradient, _ := utils.NewGradient(h.finalGradientStops, h.finalGradientSteps, false)
	finalGradientMapping, _ := finalGradient.BuildCoordinateColorMapping(
		h.base.Canvas.TextBottom,
		h.base.Canvas.TextTop,
		h.base.Canvas.TextLeft,
		h.base.Canvas.TextRight,
		h.finalGradientDirection,
	)

	// Get characters grouped diagonally (bottom-left to top-right)
	h.characterGroups = h.base.Terminal.GetCharactersGrouped(engine.DiagonalBottomLeftToTopRight, true, false, false, false)

	for _, character := range h.base.Terminal.GetCharacters(true, false, false, false, engine.TopToBottomLeftToRight) {
		baseColor := finalGradientMapping[character.InputCoord]

		// Calculate highlight color (brightened version of base)
		highlightColor := utils.AdjustBrightness(baseColor, h.highlightBrightness)

		// Create highlight gradient: base -> highlight -> highlight -> base
		// with steps (3, highlightWidth, 3)
		highlightGradient, _ := utils.NewGradient(
			[]utils.Color{baseColor, highlightColor, highlightColor, baseColor},
			[]int{3, h.highlightWidth, 3},
			false,
		)

		// Set initial appearance with base color
		baseColorCopy := baseColor
		character.SetVisual(engine.CharacterVisual{
			Symbol: character.Symbol,
			Colors: &utils.ColorPair{FG: &baseColorCopy},
		})

		// Create highlight scene
		highlightScene := character.Animation.NewScene("highlight")
		for _, color := range highlightGradient.Spectrum {
			colorCopy := color
			_ = highlightScene.AddFrame(character.Symbol, 2, &utils.ColorPair{FG: &colorCopy})
		}

		h.base.Terminal.SetCharacterVisibility(character, true)
	}
}

// inOutCirc easing function
func inOutCirc(t float64) float64 {
	if t < 0.5 {
		return (1 - math.Sqrt(1-math.Pow(2*t, 2))) / 2
	}
	return (math.Sqrt(1-math.Pow(-2*t+2, 2)) + 1) / 2
}

func (h *Highlight) Next() (string, bool) {
	if len(h.activeCharacters) > 0 || h.currentStep < h.totalSteps {
		// Calculate eased progress
		progress := float64(h.currentStep) / float64(h.totalSteps)
		easedProgress := inOutCirc(progress)

		// Calculate which group index we should be at
		groupIndex := int(easedProgress * float64(len(h.characterGroups)))
		if groupIndex > len(h.characterGroups) {
			groupIndex = len(h.characterGroups)
		}

		// Activate any new groups
		for i := h.prevGroupIndex; i < groupIndex; i++ {
			for _, character := range h.characterGroups[i] {
				character.Animation.ActivateScene("highlight")
				h.activeCharacters[character] = struct{}{}
			}
		}
		h.prevGroupIndex = groupIndex

		h.currentStep++

		// Tick all active characters
		for char := range h.activeCharacters {
			char.Tick()
			if !char.IsActive() {
				delete(h.activeCharacters, char)
			}
		}

		return h.base.Terminal.GetFormattedOutputString(), true
	}

	return h.base.Terminal.GetFormattedOutputString(), false
}

func (h *Highlight) CanvasHeight() int {
	return h.base.CanvasHeight()
}

func (h *Highlight) CanvasWidth() int {
	return h.base.CanvasWidth()
}
