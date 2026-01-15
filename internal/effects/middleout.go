package effects

import (
	"math"
	"sort"

	"tte-go/internal/config"
	"tte-go/internal/engine"
)

type MiddleOut struct {
	base  *BaseEffect
	order []*engine.EffectCharacter
	index int
}

func NewMiddleOut(input string, cfg config.TerminalConfig) engine.Effect {
	base := NewBaseEffect(input, cfg)
	centerRow := float64(base.Canvas.Height-1) / 2
	centerCol := float64(base.Canvas.Width-1) / 2
	order := append([]*engine.EffectCharacter{}, base.Characters...)
	for _, character := range order {
		character.Visible = false
	}
	sort.Slice(order, func(i, j int) bool {
		di := distanceToCenter(order[i], centerRow, centerCol)
		dj := distanceToCenter(order[j], centerRow, centerCol)
		if di == dj {
			if order[i].Coord.Row == order[j].Coord.Row {
				return order[i].Coord.Col < order[j].Coord.Col
			}
			return order[i].Coord.Row < order[j].Coord.Row
		}
		return di < dj
	})
	return &MiddleOut{base: base, order: order}
}

func (m *MiddleOut) Next() (string, bool) {
	if m.index >= len(m.order) {
		return "", false
	}
	m.order[m.index].Visible = true
	m.index++
	return engine.RenderFrame(m.base.Canvas, m.base.Characters), m.index < len(m.order)+1
}

func distanceToCenter(character *engine.EffectCharacter, row, col float64) float64 {
	dr := float64(character.Coord.Row) - row
	dc := float64(character.Coord.Col) - col
	return math.Sqrt(dr*dr + dc*dc)
}
