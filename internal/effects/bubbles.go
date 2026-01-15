package effects

import (
	"math"

	"tte-go/internal/config"
	"tte-go/internal/engine"
)

type Bubbles struct {
	base  *BaseEffect
	phase float64
}

func NewBubbles(input string, cfg config.TerminalConfig) engine.Effect {
	base := NewBaseEffect(input, cfg)
	for _, character := range base.Characters {
		character.Visible = true
	}
	return &Bubbles{base: base}
}

func (b *Bubbles) Next() (string, bool) {
	for _, character := range b.base.Characters {
		offset := int(math.Sin(b.phase+float64(character.InputCoord.Row)) * 2)
		character.Coord.Row = character.InputCoord.Row + offset
	}
	b.phase += 0.3
	return engine.RenderFrame(b.base.Canvas, b.base.Characters), true
}
