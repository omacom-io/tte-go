package effects

import (
	"tte-go/internal/config"
	"tte-go/internal/engine"
	"tte-go/internal/utils"
)

type Blackhole struct {
	base  *BaseEffect
	phase int
}

func NewBlackhole(input string, cfg config.TerminalConfig) engine.Effect {
	base := NewBaseEffect(input, cfg)
	return &Blackhole{base: base}
}

func (b *Blackhole) Next() (string, bool) {
	center := utils.Coord{Row: b.base.Canvas.Height / 2, Col: b.base.Canvas.Width / 2}
	done := true
	for _, character := range b.base.Characters {
		target := character.InputCoord
		if b.phase < 10 {
			target = center
		}
		if character.Coord.Row < target.Row {
			character.Coord.Row++
			done = false
		} else if character.Coord.Row > target.Row {
			character.Coord.Row--
			done = false
		}
		if character.Coord.Col < target.Col {
			character.Coord.Col++
			done = false
		} else if character.Coord.Col > target.Col {
			character.Coord.Col--
			done = false
		}
	}
	b.phase++
	frame := engine.RenderFrame(b.base.Canvas, b.base.Characters)
	return frame, !done
}

func (b *Blackhole) CanvasHeight() int {
	return b.base.CanvasHeight()
}
