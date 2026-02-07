package effects

import (
	"tte-go/internal/config"
	"tte-go/internal/engine"
	"tte-go/internal/utils"
)

type ExpandConfig struct {
	MovementSpeed       float64
	ExpandEasing        utils.EasingFunction
	FinalGradientStops  []utils.Color
	FinalGradientSteps  []int
	FinalGradientFrames int
	FinalGradientDir    utils.GradientDirection
}

type Expand struct {
	base             *BaseEffect
	config           ExpandConfig
	activeCharacters map[*engine.EffectCharacter]struct{}
	finalColors      map[*engine.EffectCharacter]utils.Color
}

func NewExpand(input string, cfg config.TerminalConfig) engine.Effect {
	base := NewBaseEffect(input, cfg)
	expand := &Expand{
		base:             base,
		config:           defaultExpandConfig(),
		activeCharacters: map[*engine.EffectCharacter]struct{}{},
		finalColors:      map[*engine.EffectCharacter]utils.Color{},
	}
	expand.build()
	return expand
}

func defaultExpandConfig() ExpandConfig {
	return ExpandConfig{
		MovementSpeed:       0.35,
		ExpandEasing:        utils.InOutQuart,
		FinalGradientStops:  mustColors("8A008A", "00D1FF", "FFFFFF"),
		FinalGradientSteps:  []int{12},
		FinalGradientFrames: 5,
		FinalGradientDir:    utils.GradientVertical,
	}
}

func (e *Expand) build() {
	finalGradient, _ := utils.NewGradient(e.config.FinalGradientStops, e.config.FinalGradientSteps, false)
	finalMapping, _ := finalGradient.BuildCoordinateColorMapping(
		e.base.Canvas.TextBottom,
		e.base.Canvas.TextTop,
		e.base.Canvas.TextLeft,
		e.base.Canvas.TextRight,
		e.config.FinalGradientDir,
	)
	for _, character := range e.base.Terminal.GetCharacters(true, false, false, false, engine.TopToBottomLeftToRight) {
		e.finalColors[character] = finalMapping[character.InputCoord]
	}

	for _, character := range e.base.Terminal.GetCharacters(true, false, false, false, engine.TopToBottomLeftToRight) {
		character.Motion.SetCoordinate(e.base.Canvas.Center)
		path, _ := character.Motion.NewPath(e.config.MovementSpeed, e.config.ExpandEasing, nil, 0, false, "")
		path.AddWaypoint(character.InputCoord)
		e.base.Terminal.SetCharacterVisibility(character, true)
		e.activeCharacters[character] = struct{}{}
		_ = character.EventHandler.RegisterEvent(engine.EventPathActivated, path, engine.ActionSetLayer, 1)
		_ = character.EventHandler.RegisterEvent(engine.EventPathComplete, path, engine.ActionSetLayer, 0)
		character.Motion.ActivatePath(path)
		gradientScene := character.Animation.NewScene("gradient")
		gradient, _ := utils.NewGradient([]utils.Color{finalGradient.Spectrum[0], e.finalColors[character]}, []int{10}, false)
		_ = gradientScene.ApplyGradientToSymbols([]string{character.Symbol}, e.config.FinalGradientFrames, gradient, nil)
		character.Animation.ActivateScene("gradient")
	}
}

func (e *Expand) Next() (string, bool) {
	if len(e.activeCharacters) == 0 {
		return "", false
	}
	for character := range e.activeCharacters {
		character.Tick()
		if !character.IsActive() {
			delete(e.activeCharacters, character)
		}
	}
	return e.base.Terminal.GetFormattedOutputString(), true
}

func (e *Expand) CanvasHeight() int {
	return e.base.CanvasHeight()
}

func (e *Expand) CanvasWidth() int {
	return e.base.CanvasWidth()
}
