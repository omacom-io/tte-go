package effects

import (
	"tte-go/internal/config"
	"tte-go/internal/engine"
)

type Sweep struct {
	base    *BaseEffect
	current int
}

func NewSweep(input string, cfg config.TerminalConfig) engine.Effect {
	base := NewBaseEffect(input, cfg)
	for _, character := range base.Characters {
		character.Visible = false
	}
	return &Sweep{base: base, current: 0}
}

func (s *Sweep) Next() (string, bool) {
	width := s.base.Canvas.Width
	if s.current > width {
		return "", false
	}
	for _, character := range s.base.Characters {
		if character.Coord.Col <= s.current {
			character.Visible = true
		}
	}
	frame := engine.RenderFrame(s.base.Canvas, s.base.Characters)
	s.current++
	return frame, s.current <= width
}
