package effects

import (
	"tte-go/internal/config"
	"tte-go/internal/engine"
)

type OrbittingVolley struct {
	base *BaseEffect
	step int
}

func NewOrbittingVolley(input string, cfg config.TerminalConfig) engine.Effect {
	base := NewBaseEffect(input, cfg)
	for _, character := range base.Characters {
		character.Visible = false
	}
	return &OrbittingVolley{base: base}
}

func (o *OrbittingVolley) Next() (string, bool) {
	for _, character := range o.base.Characters {
		if character.Coord.Row <= o.step || character.Coord.Col <= o.step {
			character.Visible = true
		}
	}
	o.step++
	frame := engine.RenderFrame(o.base.Canvas, o.base.Characters)
	return frame, o.step <= o.base.Canvas.Width
}
