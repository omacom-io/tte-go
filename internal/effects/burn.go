package effects

import (
	"tte-go/internal/config"
	"tte-go/internal/engine"
)

type Burn struct {
	base    *BaseEffect
	current int
}

func NewBurn(input string, cfg config.TerminalConfig) engine.Effect {
	base := NewBaseEffect(input, cfg)
	for _, character := range base.Characters {
		character.Visible = false
	}
	return &Burn{base: base}
}

func (b *Burn) Next() (string, bool) {
	if b.current > b.base.Canvas.Height {
		return "", false
	}
	for _, character := range b.base.Characters {
		if character.Coord.Row <= b.current {
			character.Visible = true
		}
	}
	b.current++
	frame := engine.RenderFrame(b.base.Canvas, b.base.Characters)
	return frame, b.current <= b.base.Canvas.Height
}

func (b *Burn) CanvasHeight() int {
	return b.base.CanvasHeight()
}
