package effects

import (
	"tte-go/internal/config"
	"tte-go/internal/engine"
	"tte-go/internal/utils"
)

type Crumble struct {
	base   *BaseEffect
	frames int
}

func NewCrumble(input string, cfg config.TerminalConfig) engine.Effect {
	base := NewBaseEffect(input, cfg)
	return &Crumble{base: base}
}

func (c *Crumble) Next() (string, bool) {
	if c.frames > 20 {
		for _, character := range c.base.Characters {
			character.Visible = true
			character.Coord = character.InputCoord
		}
		return engine.RenderFrame(c.base.Canvas, c.base.Characters), false
	}
	for _, character := range c.base.Characters {
		character.Visible = utils.RandIntn(2) == 0
	}
	c.frames++
	return engine.RenderFrame(c.base.Canvas, c.base.Characters), true
}

func (c *Crumble) CanvasHeight() int {
	return c.base.CanvasHeight()
}
