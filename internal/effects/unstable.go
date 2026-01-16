package effects

import (
	"tte-go/internal/config"
	"tte-go/internal/engine"
	"tte-go/internal/utils"
)

type Unstable struct {
	base   *BaseEffect
	frames int
}

func NewUnstable(input string, cfg config.TerminalConfig) engine.Effect {
	base := NewBaseEffect(input, cfg)
	return &Unstable{base: base}
}

func (u *Unstable) Next() (string, bool) {
	if u.frames > 30 {
		for _, character := range u.base.Characters {
			character.Coord = character.InputCoord
			character.Visible = true
		}
		return engine.RenderFrame(u.base.Canvas, u.base.Characters), false
	}
	for _, character := range u.base.Characters {
		character.Coord.Row = utils.RandIntn(max(1, u.base.Canvas.Height))
		character.Coord.Col = utils.RandIntn(max(1, u.base.Canvas.Width))
		character.Visible = true
	}
	u.frames++
	return engine.RenderFrame(u.base.Canvas, u.base.Characters), true
}

func (u *Unstable) CanvasHeight() int {
	return u.base.CanvasHeight()
}
