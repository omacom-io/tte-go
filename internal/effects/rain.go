package effects

import (
	"tte-go/internal/config"
	"tte-go/internal/engine"
)

type Rain struct {
	base *BaseEffect
}

func NewRain(input string, cfg config.TerminalConfig) engine.Effect {
	base := NewBaseEffect(input, cfg)
	for _, character := range base.Characters {
		character.Coord.Row = 0
		character.Visible = true
	}
	return &Rain{base: base}
}

func (r *Rain) Next() (string, bool) {
	active := false
	for _, character := range r.base.Characters {
		targetRow := character.InputCoord.Row
		if character.Coord.Row < targetRow {
			character.Coord.Row++
			active = true
		}
	}
	frame := engine.RenderFrame(r.base.Canvas, r.base.Characters)
	return frame, active
}
