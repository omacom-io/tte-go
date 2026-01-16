package effects

import (
	"tte-go/internal/config"
	"tte-go/internal/engine"
	"tte-go/internal/utils"
)

type Highlight struct {
	base   *BaseEffect
	cursor int
}

func NewHighlight(input string, cfg config.TerminalConfig) engine.Effect {
	base := NewBaseEffect(input, cfg)
	return &Highlight{base: base}
}

func (h *Highlight) Next() (string, bool) {
	for _, character := range h.base.Characters {
		color := utils.Color{R: 255, G: 255, B: 255}
		if character.Coord.Col == h.cursor {
			color = utils.Color{R: 255, G: 215, B: 0}
		}
		visual := character.Visual()
		visual.Colors = &utils.ColorPair{FG: &color}
		character.SetVisual(visual)
	}
	h.cursor++
	frame := engine.RenderFrame(h.base.Canvas, h.base.Characters)
	return frame, h.cursor <= h.base.Canvas.Width
}

func (h *Highlight) CanvasHeight() int {
	return h.base.CanvasHeight()
}
