package effects

import (
	"tte-go/internal/config"
	"tte-go/internal/engine"
	"tte-go/internal/utils"
)

type Thunderstorm struct {
	base   *BaseEffect
	frames int
}

func NewThunderstorm(input string, cfg config.TerminalConfig) engine.Effect {
	base := NewBaseEffect(input, cfg)
	return &Thunderstorm{base: base}
}

func (t *Thunderstorm) Next() (string, bool) {
	if t.frames > 60 {
		for _, character := range t.base.Characters {
			character.Visible = true
		}
		return engine.RenderFrame(t.base.Canvas, t.base.Characters), false
	}
	for _, character := range t.base.Characters {
		character.Visible = utils.RandIntn(4) != 0
	}
	t.frames++
	frame := engine.RenderFrame(t.base.Canvas, t.base.Characters)
	return frame, true
}

func (t *Thunderstorm) CanvasHeight() int {
	return t.base.CanvasHeight()
}
