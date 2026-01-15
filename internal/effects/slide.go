package effects

import (
	"tte-go/internal/config"
	"tte-go/internal/engine"
)

type Slide struct {
	base *BaseEffect
}

func NewSlide(input string, cfg config.TerminalConfig) engine.Effect {
	base := NewBaseEffect(input, cfg)
	startCol := -base.Canvas.Width
	for _, character := range base.Characters {
		character.Coord.Col = startCol
		character.Visible = true
	}
	return &Slide{base: base}
}

func (s *Slide) Next() (string, bool) {
	active := false
	for _, character := range s.base.Characters {
		target := character.InputCoord.Col
		if character.Coord.Col < target {
			character.Coord.Col++
			active = true
		}
	}
	frame := engine.RenderFrame(s.base.Canvas, s.base.Characters)
	return frame, active
}
