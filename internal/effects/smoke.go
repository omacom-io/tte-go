package effects

import (
	"tte-go/internal/config"
	"tte-go/internal/engine"
	"tte-go/internal/utils"
)

type Smoke struct {
	base    *BaseEffect
	current int
}

func NewSmoke(input string, cfg config.TerminalConfig) engine.Effect {
	base := NewBaseEffect(input, cfg)
	for _, character := range base.Characters {
		character.Visible = false
	}
	return &Smoke{base: base}
}

func (s *Smoke) Next() (string, bool) {
	if s.current > s.base.Canvas.Width {
		return "", false
	}
	for _, character := range s.base.Characters {
		if character.Coord.Col <= s.current {
			character.Visible = utils.RandIntn(3) != 0
		}
	}
	frame := engine.RenderFrame(s.base.Canvas, s.base.Characters)
	s.current++
	return frame, s.current <= s.base.Canvas.Width
}

func (s *Smoke) CanvasHeight() int {
	return s.base.CanvasHeight()
}
