package effects

import (
	"tte-go/internal/config"
	"tte-go/internal/engine"
	"tte-go/internal/utils"
)

// Waves creates an effect where waves travel across the terminal leaving behind characters.
//
// Behavior matches the Python implementation:
//
// - Build phase:
//   - For every character:
//   - Create a "wave" scene
//   - Append wave_count repetitions of ApplyGradientToSymbols(waveSymbols, waveLength, waveGradient)
//   - Create a "final" scene that transitions from the last wave gradient color to the per-character final color
//   - Register event: wave complete -> activate final scene
//   - Activate wave scene immediately (even though character is hidden initially)
//
// - Next phase:
//   - Reveal one group (column/row/etc) per frame
//   - Tick only revealed/active characters
type Waves struct {
	base *BaseEffect

	// Reveal sequencing
	pendingGroups    [][]*engine.EffectCharacter
	activeCharacters map[*engine.EffectCharacter]struct{}

	// Wave config (defaults mirror Python)
	waveSymbols       []string
	waveGradientStops []utils.Color
	waveGradientSteps []int
	waveCount         int
	waveLength        int
	waveDirection     engine.CharacterGroup

	// Final color config
	finalGradientStops     []utils.Color
	finalGradientSteps     []int
	finalGradientDirection utils.GradientDirection
}

func NewWaves(input string, cfg config.TerminalConfig) engine.Effect {
	base := NewBaseEffect(input, cfg)

	// Defaults mirror Python WavesConfig
	w := &Waves{
		base:             base,
		pendingGroups:    nil,
		activeCharacters: make(map[*engine.EffectCharacter]struct{}),

		waveSymbols:       []string{"▁", "▂", "▃", "▄", "▅", "▆", "▇", "█", "▇", "▆", "▅", "▄", "▃", "▂", "▁"},
		waveGradientStops: mustColors("f0ff65", "ffb102", "31a0d4", "ffb102", "f0ff65"),
		waveGradientSteps: []int{6},
		waveCount:         7,
		waveLength:        2,
		waveDirection:     engine.ColumnLeftToRight,

		finalGradientStops:     mustColors("ffb102", "31a0d4", "f0ff65"),
		finalGradientSteps:     []int{12},
		finalGradientDirection: utils.GradientDiagonal,
	}

	w.build()
	return w
}

func (w *Waves) build() {
	// Build gradients
	waveGradient, _ := utils.NewGradient(w.waveGradientStops, w.waveGradientSteps, false)

	finalGradient, _ := utils.NewGradient(w.finalGradientStops, w.finalGradientSteps, false)
	finalGradientMapping, _ := finalGradient.BuildCoordinateColorMapping(
		w.base.Canvas.TextBottom,
		w.base.Canvas.TextTop,
		w.base.Canvas.TextLeft,
		w.base.Canvas.TextRight,
		w.finalGradientDirection,
	)

	// Python groups only input characters by direction
	w.pendingGroups = w.base.Terminal.GetCharactersGrouped(w.waveDirection, true, false, false, false)

	// Create scenes for all characters up front; activate wave immediately; keep hidden initially
	for _, character := range w.base.Terminal.GetCharacters(true, false, false, false, engine.TopToBottomLeftToRight) {
		charFinalColor := finalGradientMapping[character.InputCoord]

		// Wave scene: append waveCount repetitions inside the same scene
		waveScene := character.Animation.NewScene("wave")
		waveScene.Ease = utils.InOutSine // Use easing for wave-like animation (lingers at peaks)

		for i := 0; i < w.waveCount; i++ {
			_ = waveScene.ApplyGradientToSymbols(
				w.waveSymbols,
				w.waveLength,
				waveGradient,
				nil,
			)
		}

		// Final scene: transition from last wave gradient color -> final mapped color
		finalScene := character.Animation.NewScene("final")

		if waveGradient != nil && len(waveGradient.Spectrum) > 0 {
			start := waveGradient.Spectrum[len(waveGradient.Spectrum)-1]
			transition, _ := utils.NewGradient([]utils.Color{start, charFinalColor}, w.finalGradientSteps, false)
			if transition != nil && len(transition.Spectrum) > 0 {
				for _, step := range transition.Spectrum {
					stepCopy := step
					_ = finalScene.AddFrame(character.Symbol, 10, &utils.ColorPair{FG: &stepCopy})
				}
			} else {
				// Fallback: at least show final color
				finalCopy := charFinalColor
				_ = finalScene.AddFrame(character.Symbol, 1, &utils.ColorPair{FG: &finalCopy})
			}
		} else {
			// Fallback: at least show final color
			finalCopy := charFinalColor
			_ = finalScene.AddFrame(character.Symbol, 1, &utils.ColorPair{FG: &finalCopy})
		}

		// Event: when wave completes -> activate final scene
		_ = character.EventHandler.RegisterEvent(
			engine.EventSceneComplete,
			waveScene,
			engine.ActionActivateScene,
			finalScene,
		)

		// Activate wave immediately (but character remains hidden until revealed via Next()).
		character.Animation.ActivateSceneRef(waveScene)

		// Ensure hidden initially
		w.base.Terminal.SetCharacterVisibility(character, false)
	}
}

func (w *Waves) Next() (string, bool) {
	// End condition: no more groups to reveal and no active characters animating
	if len(w.pendingGroups) == 0 && len(w.activeCharacters) == 0 {
		return w.base.Terminal.GetFormattedOutputString(), false
	}

	// Reveal exactly one group per frame (matches Python __next__)
	if len(w.pendingGroups) > 0 {
		nextGroup := w.pendingGroups[0]
		w.pendingGroups = w.pendingGroups[1:]

		for _, char := range nextGroup {
			w.base.Terminal.SetCharacterVisibility(char, true)
			w.activeCharacters[char] = struct{}{}
		}
	}

	// Tick only revealed/active characters (matches Python BaseEffectIterator.update)
	for char := range w.activeCharacters {
		char.Tick()
		if !char.IsActive() {
			delete(w.activeCharacters, char)
		}
	}

	return w.base.Terminal.GetFormattedOutputString(), true
}

func (w *Waves) CanvasHeight() int {
	return w.base.CanvasHeight()
}

func (w *Waves) CanvasWidth() int {
	return w.base.CanvasWidth()
}
