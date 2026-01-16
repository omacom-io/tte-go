package effects

import (
	"tte-go/internal/config"
	"tte-go/internal/engine"
	"tte-go/internal/utils"
)

type Spray struct {
	base  *BaseEffect
	index int
}

func NewSpray(input string, cfg config.TerminalConfig) engine.Effect {
	base := NewBaseEffect(input, cfg)
	for _, character := range base.Characters {
		character.Visible = false
	}
	return &Spray{base: base}
}

func (s *Spray) Next() (string, bool) {
	if s.index >= len(s.base.Characters) {
		return "", false
	}
	count := utils.RandIntn(5) + 1
	for i := 0; i < count && s.index < len(s.base.Characters); i++ {
		s.base.Characters[s.index].Visible = true
		s.index++
	}
	frame := engine.RenderFrame(s.base.Canvas, s.base.Characters)
	return frame, s.index < len(s.base.Characters)+1
}

func (s *Spray) CanvasHeight() int {
	return s.base.CanvasHeight()
}
