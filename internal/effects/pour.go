package effects

import (
	"tte-go/internal/config"
	"tte-go/internal/engine"
)

type Pour struct {
	base *BaseEffect
}

func NewPour(input string, cfg config.TerminalConfig) engine.Effect {
	base := NewBaseEffect(input, cfg)
	for _, character := range base.Characters {
		character.Coord.Row = 0
		character.Visible = true
	}
	return &Pour{base: base}
}

func (p *Pour) Next() (string, bool) {
	active := false
	for _, character := range p.base.Characters {
		if character.Coord.Row < character.InputCoord.Row {
			character.Coord.Row++
			active = true
		}
	}
	return engine.RenderFrame(p.base.Canvas, p.base.Characters), active
}

func (p *Pour) CanvasHeight() int {
	return p.base.CanvasHeight()
}
