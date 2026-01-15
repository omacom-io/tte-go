package effects

import (
	"math"

	"tte-go/internal/config"
	"tte-go/internal/engine"
)

type Waves struct {
	base  *BaseEffect
	phase float64
}

func NewWaves(input string, cfg config.TerminalConfig) engine.Effect {
	base := NewBaseEffect(input, cfg)
	return &Waves{base: base}
}

func (w *Waves) Next() (string, bool) {
	for _, character := range w.base.Characters {
		offset := int(math.Round(math.Sin(float64(character.InputCoord.Col)/3+w.phase) * 1.5))
		character.Coord.Row = character.InputCoord.Row + offset
		character.Visible = true
	}
	w.phase += 0.3
	frame := engine.RenderFrame(w.base.Canvas, w.base.Characters)
	return frame, true
}
