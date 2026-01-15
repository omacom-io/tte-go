package effects

import (
	"tte-go/internal/config"
	"tte-go/internal/engine"
	"tte-go/internal/utils"
)

type Expand struct {
	base *BaseEffect
}

func NewExpand(input string, cfg config.TerminalConfig) engine.Effect {
	base := NewBaseEffect(input, cfg)
	center := utils.Coord{Row: base.Canvas.Height / 2, Col: base.Canvas.Width / 2}
	for _, character := range base.Characters {
		character.Coord = center
		character.Visible = true
	}
	return &Expand{base: base}
}

func (e *Expand) Next() (string, bool) {
	active := false
	for _, character := range e.base.Characters {
		if character.Coord.Row < character.InputCoord.Row {
			character.Coord.Row++
			active = true
		} else if character.Coord.Row > character.InputCoord.Row {
			character.Coord.Row--
			active = true
		}
		if character.Coord.Col < character.InputCoord.Col {
			character.Coord.Col++
			active = true
		} else if character.Coord.Col > character.InputCoord.Col {
			character.Coord.Col--
			active = true
		}
	}
	frame := engine.RenderFrame(e.base.Canvas, e.base.Characters)
	return frame, active
}
