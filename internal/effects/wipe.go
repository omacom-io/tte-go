package effects

import (
	"tte-go/internal/config"
	"tte-go/internal/engine"
)

type Wipe struct {
	base    *BaseEffect
	current int
}

func NewWipe(input string, cfg config.TerminalConfig) engine.Effect {
	base := NewBaseEffect(input, cfg)
	for _, character := range base.Characters {
		character.Visible = false
	}
	return &Wipe{base: base, current: 0}
}

func (w *Wipe) Next() (string, bool) {
	width := w.base.Canvas.Width
	if w.current > width {
		return "", false
	}
	for _, character := range w.base.Characters {
		if character.Coord.Col <= w.current {
			character.Visible = true
		}
	}
	frame := engine.RenderFrame(w.base.Canvas, w.base.Characters)
	w.current++
	return frame, w.current <= width
}
