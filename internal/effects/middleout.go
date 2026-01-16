package effects

import (
	"tte-go/internal/config"
	"tte-go/internal/engine"
	"tte-go/internal/utils"
)

type MiddleOutConfig struct {
	StartingColor       utils.Color
	ExpandDirection     string
	CenterMovementSpeed float64
	FullMovementSpeed   float64
	CenterEasing        utils.EasingFunction
	FullEasing          utils.EasingFunction
	FinalGradientStops  []utils.Color
	FinalGradientSteps  []int
	FinalGradientDir    utils.GradientDirection
}

type MiddleOut struct {
	base             *BaseEffect
	config           MiddleOutConfig
	activeCharacters map[*engine.EffectCharacter]struct{}
	finalColors      map[*engine.EffectCharacter]utils.Color
	phase            string
}

func NewMiddleOut(input string, cfg config.TerminalConfig) engine.Effect {
	base := NewBaseEffect(input, cfg)
	effect := &MiddleOut{
		base:             base,
		config:           defaultMiddleOutConfig(),
		activeCharacters: map[*engine.EffectCharacter]struct{}{},
		finalColors:      map[*engine.EffectCharacter]utils.Color{},
		phase:            "center",
	}
	effect.build()
	return effect
}

func defaultMiddleOutConfig() MiddleOutConfig {
	return MiddleOutConfig{
		StartingColor:       utils.Color{R: 255, G: 255, B: 255},
		ExpandDirection:     "vertical",
		CenterMovementSpeed: 0.6,
		FullMovementSpeed:   0.6,
		CenterEasing:        utils.InOutSine,
		FullEasing:          utils.InOutSine,
		FinalGradientStops:  mustColors("8A008A", "00D1FF", "FFFFFF"),
		FinalGradientSteps:  []int{12},
		FinalGradientDir:    utils.GradientVertical,
	}
}

func (m *MiddleOut) build() {
	finalGradient, _ := utils.NewGradient(m.config.FinalGradientStops, m.config.FinalGradientSteps, false)
	finalMapping, _ := finalGradient.BuildCoordinateColorMapping(
		m.base.Canvas.TextBottom,
		m.base.Canvas.TextTop,
		m.base.Canvas.TextLeft,
		m.base.Canvas.TextRight,
		m.config.FinalGradientDir,
	)
	for _, character := range m.base.Terminal.GetCharacters(true, false, false, false, engine.TopToBottomLeftToRight) {
		m.finalColors[character] = finalMapping[character.InputCoord]
	}

	for _, character := range m.base.Terminal.GetCharacters(true, false, false, false, engine.TopToBottomLeftToRight) {
		character.Motion.SetCoordinate(m.base.Canvas.Center)
		centerCoord := utils.Coord{Row: m.base.Canvas.Center.Row, Col: character.InputCoord.Col}
		if m.config.ExpandDirection == "horizontal" {
			centerCoord = utils.Coord{Row: character.InputCoord.Row, Col: m.base.Canvas.Center.Col}
		}
		centerPath, _ := character.Motion.NewPath(m.config.CenterMovementSpeed, m.config.CenterEasing, nil, 0, false, "")
		centerPath.AddWaypoint(centerCoord)
		fullPath, _ := character.Motion.NewPath(m.config.FullMovementSpeed, m.config.FullEasing, nil, 0, false, "full")
		fullPath.AddWaypoint(character.InputCoord)

		fullScene := character.Animation.NewScene("full")
		gradient, _ := utils.NewGradient([]utils.Color{m.config.StartingColor, m.finalColors[character]}, []int{10}, false)
		_ = fullScene.ApplyGradientToSymbols([]string{character.Symbol}, 6, gradient, nil)

		visual := character.Visual()
		visual.Symbol = character.Symbol
		visual.Colors = &utils.ColorPair{FG: &m.config.StartingColor}
		character.SetVisual(visual)

		m.base.Terminal.SetCharacterVisibility(character, true)
		character.Motion.ActivatePath(centerPath)
		m.activeCharacters[character] = struct{}{}
	}
}

func (m *MiddleOut) Next() (string, bool) {
	if m.phase == "center" && len(m.activeCharacters) == 0 {
		m.phase = "full"
		m.activeCharacters = map[*engine.EffectCharacter]struct{}{}
		for _, character := range m.base.Terminal.GetCharacters(true, false, false, false, engine.TopToBottomLeftToRight) {
			if path, err := character.Motion.QueryPath("full"); err == nil {
				character.Motion.ActivatePath(path)
			}
			character.Animation.ActivateScene("full")
			m.activeCharacters[character] = struct{}{}
		}
	}

	if len(m.activeCharacters) == 0 {
		return "", false
	}
	for character := range m.activeCharacters {
		character.Tick()
		if !character.IsActive() {
			delete(m.activeCharacters, character)
		}
	}
	return m.base.Terminal.GetFormattedOutputString(), true
}
