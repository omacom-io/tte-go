package effects

import (
	"tte-go/internal/config"
	"tte-go/internal/engine"
	"tte-go/internal/utils"
)

// Waves creates an effect where waves travel across the terminal leaving behind characters.
type Waves struct {
	base *BaseEffect

	pendingGroups    [][]*engine.EffectCharacter
	activeCharacters map[*engine.EffectCharacter]struct{}
	waveIndex        int
	frameInGroup     int
	wavesRemaining   int

	waveGradientColors []utils.Color
	waveSymbols        []string

	// Config
	waveGradientStops      []utils.Color
	waveGradientSteps      []int
	waveCount              int
	waveLength             int
	waveDirection          engine.CharacterGroup
	finalGradientStops     []utils.Color
	finalGradientSteps     []int
	finalGradientDirection utils.GradientDirection
}

func NewWaves(input string, cfg config.TerminalConfig) engine.Effect {
	base := NewBaseEffect(input, cfg)

	// Config defaults
	waveSymbols := []string{"▁", "▂", "▃", "▄", "▅", "▆", "▇", "█", "▇", "▆", "▅", "▄", "▃", "▂", "▁"}
	waveGradientStops := mustColors("f0ff65", "ffb102", "31a0d4", "ffb102", "f0ff65")
	waveGradientSteps := []int{6}
	finalGradientStops := mustColors("ffb102", "31a0d4", "f0ff65")
	finalGradientSteps := []int{12}
	finalGradientDirection := utils.GradientDiagonal

	w := &Waves{
		base:                   base,
		pendingGroups:          make([][]*engine.EffectCharacter, 0),
		activeCharacters:       make(map[*engine.EffectCharacter]struct{}),
		waveIndex:              0,
		frameInGroup:           0,
		waveSymbols:            waveSymbols,
		waveGradientStops:      waveGradientStops,
		waveGradientSteps:      waveGradientSteps,
		waveCount:              7,
		waveLength:             2,
		waveDirection:          engine.ColumnLeftToRight,
		finalGradientStops:     finalGradientStops,
		finalGradientSteps:     finalGradientSteps,
		finalGradientDirection: finalGradientDirection,
	}

	w.wavesRemaining = w.waveCount

	w.build()
	return w
}

func (w *Waves) build() {
	// Build wave gradient
	waveGradient, _ := utils.NewGradient(w.waveGradientStops, w.waveGradientSteps, false)
	w.waveGradientColors = waveGradient.Spectrum

	// Build final gradient mapping
	finalGradient, _ := utils.NewGradient(w.finalGradientStops, w.finalGradientSteps, false)
	finalGradientMapping, _ := finalGradient.BuildCoordinateColorMapping(
		w.base.Canvas.TextBottom,
		w.base.Canvas.TextTop,
		w.base.Canvas.TextLeft,
		w.base.Canvas.TextRight,
		w.finalGradientDirection,
	)

	// Get character groups based on wave direction
	w.pendingGroups = w.base.Terminal.GetCharactersGrouped(w.waveDirection, true, true, true, false)

	// Set up each character with wave and final scenes
	for _, character := range w.base.Terminal.GetCharacters(true, true, true, false, engine.TopToBottomLeftToRight) {
		charFinalColor := finalGradientMapping[character.InputCoord]

		// Create wave scene with wave symbols and colors
		waveScene := character.Animation.NewScene("wave")
		for i, symbol := range w.waveSymbols {
			colorIdx := i * len(w.waveGradientColors) / len(w.waveSymbols)
			if colorIdx >= len(w.waveGradientColors) {
				colorIdx = len(w.waveGradientColors) - 1
			}
			colorCopy := w.waveGradientColors[colorIdx]
			_ = waveScene.AddFrame(symbol, w.waveLength, &utils.ColorPair{FG: &colorCopy})
		}

		// Create final scene - show character in final color
		finalScene := character.Animation.NewScene("final")
		finalColorCopy := charFinalColor
		_ = finalScene.AddFrame(character.Symbol, 1, &utils.ColorPair{FG: &finalColorCopy})

		// Event: wave complete -> activate final
		character.EventHandler.RegisterEvent(
			engine.EventSceneComplete,
			waveScene,
			engine.ActionActivateScene,
			finalScene,
		)

		// Start invisible
		dark := mustColors("000000")[0]
		character.SetVisual(engine.CharacterVisual{
			Symbol: " ",
			Colors: &utils.ColorPair{FG: &dark},
		})
	}
}

func (w *Waves) Next() (string, bool) {
	// Process waves
	if w.waveIndex < len(w.pendingGroups) || len(w.activeCharacters) > 0 {
		// Activate next group
		if w.waveIndex < len(w.pendingGroups) {
			w.frameInGroup++
			if w.frameInGroup >= w.waveLength {
				w.frameInGroup = 0
				group := w.pendingGroups[w.waveIndex]
				for _, char := range group {
					w.base.Terminal.SetCharacterVisibility(char, true)
					char.Animation.ActivateScene("wave")
					w.activeCharacters[char] = struct{}{}
				}
				w.waveIndex++
			}
		}

		// Tick all active characters
		for char := range w.activeCharacters {
			char.Tick()
			if !char.IsActive() {
				delete(w.activeCharacters, char)
			}
		}

		// Check if we need to restart for next wave
		if w.waveIndex >= len(w.pendingGroups) && len(w.activeCharacters) == 0 {
			w.wavesRemaining--
			if w.wavesRemaining > 0 {
				w.waveIndex = 0
				// Re-enable wave scenes for all characters
				for _, group := range w.pendingGroups {
					for _, char := range group {
						if waveScene := char.Animation.Scenes["wave"]; waveScene != nil {
							waveScene.ResetScene()
						}
					}
				}
			}
		}

		return w.base.Terminal.GetFormattedOutputString(), true
	}

	return w.base.Terminal.GetFormattedOutputString(), false
}

func (w *Waves) CanvasHeight() int {
	return w.base.CanvasHeight()
}
