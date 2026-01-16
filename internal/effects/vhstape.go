package effects

import (
	"tte-go/internal/config"
	"tte-go/internal/engine"
	"tte-go/internal/utils"
)

type VHSTape struct {
	base *BaseEffect
}

func NewVHSTape(input string, cfg config.TerminalConfig) engine.Effect {
	base := NewBaseEffect(input, cfg)
	return &VHSTape{base: base}
}

func (v *VHSTape) Next() (string, bool) {
	for _, character := range v.base.Characters {
		jitter := utils.RandIntn(3) - 1
		character.Coord.Col = character.InputCoord.Col + jitter
		character.Visible = true
	}
	frame := engine.RenderFrame(v.base.Canvas, v.base.Characters)
	return frame, true
}

func (v *VHSTape) CanvasHeight() int {
	return v.base.CanvasHeight()
}
