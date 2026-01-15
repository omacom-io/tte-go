package effects

import (
	"tte-go/internal/config"
	"tte-go/internal/engine"
	"tte-go/internal/utils"
)

type ColorShift struct {
	base   *BaseEffect
	colors []utils.Color
	offset int
}

func NewColorShift(input string, cfg config.TerminalConfig) engine.Effect {
	base := NewBaseEffect(input, cfg)
	stops := []utils.Color{
		{R: 138, G: 0, B: 138},
		{R: 0, G: 209, B: 255},
		{R: 255, G: 255, B: 255},
	}
	gradient := engine.NewGradient(stops, 24)
	return &ColorShift{base: base, colors: gradient.Colors()}
}

func (c *ColorShift) Next() (string, bool) {
	if len(c.colors) == 0 {
		return "", false
	}
	for _, character := range c.base.Characters {
		index := (character.Coord.Col + c.offset) % len(c.colors)
		color := c.colors[index]
		visual := character.Visual()
		visual.Colors = &utils.ColorPair{FG: &color}
		character.SetVisual(visual)
	}
	frame := engine.RenderFrame(c.base.Canvas, c.base.Characters)
	c.offset++
	return frame, true
}
