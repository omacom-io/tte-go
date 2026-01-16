package effects

import (
	"tte-go/internal/config"
	"tte-go/internal/engine"
	"tte-go/internal/utils"
)

type WipeConfig struct {
	WipeDirection       engine.CharacterGroup
	WipeDelay           int
	WipeEase            utils.EasingFunction
	FinalGradientStops  []utils.Color
	FinalGradientSteps  []int
	FinalGradientFrames int
	FinalGradientDir    utils.GradientDirection
}

type Wipe struct {
	base             *BaseEffect
	config           WipeConfig
	groups           [][]*engine.EffectCharacter
	easer            *utils.SequenceEaser[[]*engine.EffectCharacter]
	wipeDelay        int
	activeCharacters map[*engine.EffectCharacter]struct{}
}

func NewWipe(input string, cfg config.TerminalConfig) engine.Effect {
	base := NewBaseEffect(input, cfg)
	wipe := &Wipe{
		base:             base,
		config:           defaultWipeConfig(),
		activeCharacters: map[*engine.EffectCharacter]struct{}{},
	}
	wipe.build()
	return wipe
}

func defaultWipeConfig() WipeConfig {
	return WipeConfig{
		WipeDirection:       engine.DiagonalTopLeftToBottomRight,
		WipeDelay:           0,
		WipeEase:            utils.InOutCirc,
		FinalGradientStops:  mustColors("833ab4", "fd1d1d", "fcb045"),
		FinalGradientSteps:  []int{12},
		FinalGradientFrames: 3,
		FinalGradientDir:    utils.GradientVertical,
	}
}

func (w *Wipe) build() {
	finalGradient, _ := utils.NewGradient(w.config.FinalGradientStops, w.config.FinalGradientSteps, false)
	finalMapping, _ := finalGradient.BuildCoordinateColorMapping(
		w.base.Canvas.TextBottom,
		w.base.Canvas.TextTop,
		w.base.Canvas.TextLeft,
		w.base.Canvas.TextRight,
		w.config.FinalGradientDir,
	)
	w.groups = w.base.Terminal.GetCharactersGrouped(w.config.WipeDirection, true, false, false, false)
	w.easer = utils.NewSequenceEaser(w.groups, w.config.WipeEase)
	for _, group := range w.groups {
		for _, character := range group {
			w.base.Terminal.SetCharacterVisibility(character, false)
			finalColor := finalMapping[character.InputCoord]
			wipeGradient, _ := utils.NewGradient([]utils.Color{finalGradient.Spectrum[0], finalColor}, w.config.FinalGradientSteps, false)
			scene := character.Animation.NewScene("wipe")
			_ = scene.ApplyGradientToSymbols([]string{character.Symbol}, w.config.FinalGradientFrames, wipeGradient, nil)
		}
	}
	w.wipeDelay = w.config.WipeDelay
}

func (w *Wipe) Next() (string, bool) {
	if len(w.groups) == 0 && len(w.activeCharacters) == 0 {
		return "", false
	}
	if w.wipeDelay == 0 {
		if w.easer != nil && !w.easer.IsComplete() {
			w.easer.Step()
			for _, group := range w.easer.Added {
				for _, character := range group {
					character.Animation.ActivateScene("wipe")
					w.base.Terminal.SetCharacterVisibility(character, true)
					w.activeCharacters[character] = struct{}{}
				}
			}
			for _, group := range w.easer.Removed {
				for _, character := range group {
					character.Animation.DeactivateScene()
					if scene := character.Animation.QueryScene("wipe"); scene != nil {
						scene.ResetScene()
					}
					w.base.Terminal.SetCharacterVisibility(character, false)
				}
			}
			w.wipeDelay = w.config.WipeDelay
		}
	} else {
		w.wipeDelay--
	}
	for character := range w.activeCharacters {
		character.Tick()
		if !character.IsActive() {
			character.Animation.DeactivateScene()
			if scene := character.Animation.QueryScene("wipe"); scene != nil {
				scene.ResetScene()
			}
			delete(w.activeCharacters, character)
		}
	}
	if (w.easer == nil || w.easer.IsComplete()) && len(w.activeCharacters) == 0 {
		return "", false
	}
	return w.base.Terminal.GetFormattedOutputString(), true
}

func (w *Wipe) CanvasHeight() int {
	return w.base.CanvasHeight()
}
