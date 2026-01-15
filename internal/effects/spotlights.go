package effects

import (
	"math"

	"tte-go/internal/config"
	"tte-go/internal/engine"
)

type Spotlights struct {
	base *BaseEffect
	step int
}

func NewSpotlights(input string, cfg config.TerminalConfig) engine.Effect {
	base := NewBaseEffect(input, cfg)
	for _, character := range base.Characters {
		character.Visible = false
	}
	return &Spotlights{base: base}
}

func (s *Spotlights) Next() (string, bool) {
	centerCol := float64(s.base.Canvas.Width-1) / 2
	for _, character := range s.base.Characters {
		distance := math.Abs(float64(character.InputCoord.Col) - centerCol)
		character.Visible = distance <= float64(s.step)
	}
	s.step++
	frame := engine.RenderFrame(s.base.Canvas, s.base.Characters)
	return frame, s.step <= s.base.Canvas.Width
}
