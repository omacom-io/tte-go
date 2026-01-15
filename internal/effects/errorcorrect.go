package effects

import (
	"tte-go/internal/config"
	"tte-go/internal/engine"
	"tte-go/internal/utils"
)

type ErrorCorrect struct {
	base   *BaseEffect
	frames int
}

func NewErrorCorrect(input string, cfg config.TerminalConfig) engine.Effect {
	base := NewBaseEffect(input, cfg)
	for _, character := range base.Characters {
		character.Coord.Row = utils.RandIntn(max(1, base.Canvas.Height))
		character.Coord.Col = utils.RandIntn(max(1, base.Canvas.Width))
		character.Visible = true
	}
	return &ErrorCorrect{base: base}
}

func (e *ErrorCorrect) Next() (string, bool) {
	if e.frames > 15 {
		for _, character := range e.base.Characters {
			character.Coord = character.InputCoord
		}
		return engine.RenderFrame(e.base.Canvas, e.base.Characters), false
	}
	e.frames++
	return engine.RenderFrame(e.base.Canvas, e.base.Characters), true
}
