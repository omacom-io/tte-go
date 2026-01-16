package effects

import (
	"tte-go/internal/config"
	"tte-go/internal/engine"
	"tte-go/internal/utils"
)

type Matrix struct {
	base   *BaseEffect
	speeds []int
}

func NewMatrix(input string, cfg config.TerminalConfig) engine.Effect {
	base := NewBaseEffect(input, cfg)
	speeds := make([]int, len(base.Characters))
	for i, character := range base.Characters {
		speeds[i] = utils.RandIntn(3) + 1
		character.Coord.Row = utils.RandIntn(max(1, base.Canvas.Height)) * -1
		character.Visible = true
	}
	return &Matrix{base: base, speeds: speeds}
}

func (m *Matrix) Next() (string, bool) {
	for i, character := range m.base.Characters {
		character.Coord.Row += m.speeds[i]
		if character.Coord.Row > m.base.Canvas.Height {
			character.Coord.Row = -utils.RandIntn(m.base.Canvas.Height)
		}
	}
	frame := engine.RenderFrame(m.base.Canvas, m.base.Characters)
	return frame, true
}

func (m *Matrix) CanvasHeight() int {
	return m.base.CanvasHeight()
}
