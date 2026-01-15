package effects

import (
	"math"
	"tte-go/internal/config"
	"tte-go/internal/engine"
	"tte-go/internal/utils"
)

type Rings struct {
	base  *BaseEffect
	phase float64
}

func NewRings(input string, cfg config.TerminalConfig) engine.Effect {
	base := NewBaseEffect(input, cfg)
	return &Rings{base: base}
}

func (r *Rings) Next() (string, bool) {
	center := utils.Coord{Row: r.base.Canvas.Height / 2, Col: r.base.Canvas.Width / 2}
	for _, character := range r.base.Characters {
		distance := math.Sqrt(math.Pow(float64(character.InputCoord.Row-center.Row), 2) + math.Pow(float64(character.InputCoord.Col-center.Col), 2))
		angle := r.phase + distance/5
		character.Coord.Row = center.Row + int(math.Sin(angle)*distance/2)
		character.Coord.Col = center.Col + int(math.Cos(angle)*distance/2)
		character.Visible = true
	}
	r.phase += 0.2
	return engine.RenderFrame(r.base.Canvas, r.base.Characters), true
}
