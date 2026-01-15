package effects

import (
	"tte-go/internal/config"
	"tte-go/internal/engine"
	"tte-go/internal/utils"
)

type Swarm struct {
	base *BaseEffect
}

func NewSwarm(input string, cfg config.TerminalConfig) engine.Effect {
	base := NewBaseEffect(input, cfg)
	for _, character := range base.Characters {
		character.Coord.Row = utils.RandIntn(max(1, base.Canvas.Height))
		character.Coord.Col = utils.RandIntn(max(1, base.Canvas.Width))
		character.Visible = true
	}
	return &Swarm{base: base}
}

func (s *Swarm) Next() (string, bool) {
	active := false
	for _, character := range s.base.Characters {
		if character.Coord.Row < character.InputCoord.Row {
			character.Coord.Row++
			active = true
		} else if character.Coord.Row > character.InputCoord.Row {
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
	frame := engine.RenderFrame(s.base.Canvas, s.base.Characters)
	return frame, active
}
