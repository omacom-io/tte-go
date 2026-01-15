package effects

import (
	"tte-go/internal/config"
	"tte-go/internal/engine"
)

type Slice struct {
	base *BaseEffect
}

func NewSlice(input string, cfg config.TerminalConfig) engine.Effect {
	base := NewBaseEffect(input, cfg)
	mid := base.Canvas.Width / 2
	for _, character := range base.Characters {
		if character.InputCoord.Col <= mid {
			character.Coord.Col = character.InputCoord.Col - base.Canvas.Width
		} else {
			character.Coord.Col = character.InputCoord.Col + base.Canvas.Width
		}
		character.Visible = true
	}
	return &Slice{base: base}
}

func (s *Slice) Next() (string, bool) {
	active := false
	for _, character := range s.base.Characters {
		if character.Coord.Col < character.InputCoord.Col {
			character.Coord.Col++
			active = true
		} else if character.Coord.Col > character.InputCoord.Col {
			character.Coord.Col--
			active = true
		}
	}
	frame := engine.RenderFrame(s.base.Canvas, s.base.Characters)
	return frame, active
}
