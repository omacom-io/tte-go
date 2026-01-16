package effects

import (
	"tte-go/internal/config"
	"tte-go/internal/engine"
	"tte-go/internal/utils"
)

type Fireworks struct {
	base   *BaseEffect
	frames int
}

func NewFireworks(input string, cfg config.TerminalConfig) engine.Effect {
	base := NewBaseEffect(input, cfg)
	for _, character := range base.Characters {
		character.Coord = utils.Coord{Row: base.Canvas.Height - 1, Col: utils.RandIntn(max(1, base.Canvas.Width))}
		character.Visible = true
	}
	return &Fireworks{base: base}
}

func (f *Fireworks) Next() (string, bool) {
	active := false
	for _, character := range f.base.Characters {
		if character.Coord.Row > character.InputCoord.Row {
			character.Coord.Row--
			active = true
		}
		if character.Coord.Col < character.InputCoord.Col {
			character.Coord.Col++
			active = true
		} else if character.Coord.Col > character.InputCoord.Col {
			character.Coord.Col--
			active = true
		}
	}
	f.frames++
	frame := engine.RenderFrame(f.base.Canvas, f.base.Characters)
	return frame, active
}

func (f *Fireworks) CanvasHeight() int {
	return f.base.CanvasHeight()
}
