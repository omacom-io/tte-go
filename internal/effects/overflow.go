package effects

import (
	"tte-go/internal/config"
	"tte-go/internal/engine"
)

type Overflow struct {
	base   *BaseEffect
	offset int
}

func NewOverflow(input string, cfg config.TerminalConfig) engine.Effect {
	base := NewBaseEffect(input, cfg)
	return &Overflow{base: base}
}

func (o *Overflow) Next() (string, bool) {
	for _, character := range o.base.Characters {
		character.Coord.Row = character.InputCoord.Row + o.offset
		character.Visible = true
	}
	o.offset--
	if o.offset < -o.base.Canvas.Height {
		for _, character := range o.base.Characters {
			character.Coord = character.InputCoord
		}
		return engine.RenderFrame(o.base.Canvas, o.base.Characters), false
	}
	return engine.RenderFrame(o.base.Canvas, o.base.Characters), true
}

func (o *Overflow) CanvasHeight() int {
	return o.base.CanvasHeight()
}
