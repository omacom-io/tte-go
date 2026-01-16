package effects

import (
	"math"
	"tte-go/internal/config"
	"tte-go/internal/engine"
)

type BouncyBalls struct {
	base  *BaseEffect
	phase float64
}

func NewBouncyBalls(input string, cfg config.TerminalConfig) engine.Effect {
	base := NewBaseEffect(input, cfg)
	for _, character := range base.Characters {
		character.Visible = true
	}
	return &BouncyBalls{base: base}
}

func (b *BouncyBalls) Next() (string, bool) {
	height := float64(b.base.Canvas.Height - 1)
	for _, character := range b.base.Characters {
		bounce := math.Abs(math.Sin(b.phase + float64(character.InputCoord.Col)/3))
		character.Coord.Row = int(height - bounce*height)
	}
	b.phase += 0.2
	frame := engine.RenderFrame(b.base.Canvas, b.base.Characters)
	return frame, true
}

func (b *BouncyBalls) CanvasHeight() int {
	return b.base.CanvasHeight()
}
