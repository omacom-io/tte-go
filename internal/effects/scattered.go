package effects

import (
	"tte-go/internal/config"
	"tte-go/internal/engine"
	"tte-go/internal/utils"
)

type ScatteredConfig struct {
	MovementSpeed       float64
	MovementEasing      utils.EasingFunction
	FinalGradientStops  []utils.Color
	FinalGradientSteps  []int
	FinalGradientFrames int
	FinalGradientDir    utils.GradientDirection
}

type Scattered struct {
	base              *BaseEffect
	config            ScatteredConfig
	activeCharacters  map[*engine.EffectCharacter]struct{}
	finalColors       map[*engine.EffectCharacter]utils.Color
	initialHoldFrames int
}

func NewScattered(input string, cfg config.TerminalConfig) engine.Effect {
	base := NewBaseEffect(input, cfg)
	effect := &Scattered{
		base:             base,
		config:           defaultScatteredConfig(),
		activeCharacters: map[*engine.EffectCharacter]struct{}{},
		finalColors:      map[*engine.EffectCharacter]utils.Color{},
	}
	effect.build()
	return effect
}

func defaultScatteredConfig() ScatteredConfig {
	return ScatteredConfig{
		MovementSpeed:       0.5,
		MovementEasing:      utils.InOutBack,
		FinalGradientStops:  mustColors("ff9048", "ab9dff", "bdffea"),
		FinalGradientSteps:  []int{12},
		FinalGradientFrames: 9,
		FinalGradientDir:    utils.GradientVertical,
	}
}

func (s *Scattered) build() {
	finalGradient, _ := utils.NewGradient(s.config.FinalGradientStops, s.config.FinalGradientSteps, false)
	finalMapping, _ := finalGradient.BuildCoordinateColorMapping(
		s.base.Canvas.TextBottom,
		s.base.Canvas.TextTop,
		s.base.Canvas.TextLeft,
		s.base.Canvas.TextRight,
		s.config.FinalGradientDir,
	)
	ordered := s.base.Terminal.GetCharacters(true, false, false, false, engine.TopToBottomLeftToRight)
	for _, character := range ordered {
		s.finalColors[character] = finalMapping[character.InputCoord]
	}

	for _, character := range ordered {
		if s.base.Canvas.Right < 2 || s.base.Canvas.Top < 2 {
			character.Motion.SetCoordinate(utils.Coord{Row: 1, Col: 1})
		} else {
			character.Motion.SetCoordinate(utils.Coord{
				Row: utils.RandIntn(max(1, s.base.Canvas.Height)) + 1,
				Col: utils.RandIntn(max(1, s.base.Canvas.Width)) + 1,
			})
		}
		path, _ := character.Motion.NewPath(s.config.MovementSpeed, s.config.MovementEasing, nil, 0, false, "")
		path.AddWaypoint(character.InputCoord)
		_ = character.EventHandler.RegisterEvent(engine.EventPathActivated, path, engine.ActionSetLayer, 1)
		_ = character.EventHandler.RegisterEvent(engine.EventPathComplete, path, engine.ActionSetLayer, 0)
		character.Motion.ActivatePath(path)
		s.base.Terminal.SetCharacterVisibility(character, true)
		gradientScene := character.Animation.NewScene("gradient")
		gradient, _ := utils.NewGradient(
			[]utils.Color{finalGradient.Spectrum[0], s.finalColors[character]},
			[]int{10},
			false,
		)
		_ = gradientScene.ApplyGradientToSymbols([]string{character.Symbol}, s.config.FinalGradientFrames, gradient, nil)
		character.Animation.ActivateScene("gradient")
		s.activeCharacters[character] = struct{}{}
	}
	s.initialHoldFrames = 25
}

func (s *Scattered) Next() (string, bool) {
	if len(s.activeCharacters) == 0 {
		return "", false
	}
	if s.initialHoldFrames > 0 {
		s.initialHoldFrames--
		return s.base.Terminal.GetFormattedOutputString(), true
	}
	for character := range s.activeCharacters {
		character.Tick()
		if !character.IsActive() {
			delete(s.activeCharacters, character)
		}
	}
	return s.base.Terminal.GetFormattedOutputString(), true
}

func (s *Scattered) CanvasHeight() int {
	return s.base.CanvasHeight()
}

func (s *Scattered) CanvasWidth() int {
	return s.base.CanvasWidth()
}
