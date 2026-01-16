package effects

import (
	"tte-go/internal/config"
	"tte-go/internal/engine"
	"tte-go/internal/utils"
)

type RandomSequenceConfig struct {
	StartingColor       utils.Color
	Speed               float64
	FinalGradientStops  []utils.Color
	FinalGradientSteps  []int
	FinalGradientFrames int
	FinalGradientDir    utils.GradientDirection
}

type RandomSequence struct {
	base              *BaseEffect
	config            RandomSequenceConfig
	pending           []*engine.EffectCharacter
	activeCharacters  map[*engine.EffectCharacter]struct{}
	charactersPerTick int
}

func NewRandomSequence(input string, cfg config.TerminalConfig) engine.Effect {
	base := NewBaseEffect(input, cfg)
	effect := &RandomSequence{
		base:             base,
		config:           defaultRandomSequenceConfig(),
		activeCharacters: map[*engine.EffectCharacter]struct{}{},
	}
	effect.build()
	return effect
}

func defaultRandomSequenceConfig() RandomSequenceConfig {
	return RandomSequenceConfig{
		StartingColor:       utils.Color{R: 0, G: 0, B: 0},
		Speed:               0.007,
		FinalGradientStops:  mustColors("8A008A", "00D1FF", "FFFFFF"),
		FinalGradientSteps:  []int{12},
		FinalGradientFrames: 8,
		FinalGradientDir:    utils.GradientVertical,
	}
}

func (r *RandomSequence) build() {
	finalGradient, _ := utils.NewGradient(r.config.FinalGradientStops, r.config.FinalGradientSteps, false)
	finalMapping, _ := finalGradient.BuildCoordinateColorMapping(
		r.base.Canvas.TextBottom,
		r.base.Canvas.TextTop,
		r.base.Canvas.TextLeft,
		r.base.Canvas.TextRight,
		r.config.FinalGradientDir,
	)

	r.pending = make([]*engine.EffectCharacter, 0, len(r.base.Terminal.InputChars))
	for _, character := range r.base.Terminal.GetCharacters(true, false, false, false, engine.TopToBottomLeftToRight) {
		r.base.Terminal.SetCharacterVisibility(character, false)
		finalColor := finalMapping[character.InputCoord]
		gradient, _ := utils.NewGradient([]utils.Color{r.config.StartingColor, finalColor}, []int{7}, false)
		scene := character.Animation.NewScene("gradient")
		_ = scene.ApplyGradientToSymbols([]string{character.Symbol}, r.config.FinalGradientFrames, gradient, nil)
		character.Animation.ActivateScene("gradient")
		r.pending = append(r.pending, character)
	}
	utils.Shuffle(len(r.pending), func(i, j int) { r.pending[i], r.pending[j] = r.pending[j], r.pending[i] })
	perTick := int(r.config.Speed * float64(len(r.base.Terminal.InputChars)))
	if perTick < 1 {
		perTick = 1
	}
	r.charactersPerTick = perTick
}

func (r *RandomSequence) Next() (string, bool) {
	if len(r.pending) > 0 || len(r.activeCharacters) > 0 {
		for i := 0; i < r.charactersPerTick; i++ {
			if len(r.pending) == 0 {
				break
			}
			next := r.pending[len(r.pending)-1]
			r.pending = r.pending[:len(r.pending)-1]
			r.base.Terminal.SetCharacterVisibility(next, true)
			r.activeCharacters[next] = struct{}{}
		}
		for character := range r.activeCharacters {
			character.Tick()
			if !character.IsActive() {
				delete(r.activeCharacters, character)
			}
		}
		return r.base.Terminal.GetFormattedOutputString(), true
	}
	return "", false
}

func (r *RandomSequence) CanvasHeight() int {
	return r.base.CanvasHeight()
}
