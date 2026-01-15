package effects

import (
	"sort"

	"tte-go/internal/config"
	"tte-go/internal/engine"
)

type LaserEtch struct {
	base  *BaseEffect
	order []*engine.EffectCharacter
	index int
}

func NewLaserEtch(input string, cfg config.TerminalConfig) engine.Effect {
	base := NewBaseEffect(input, cfg)
	order := append([]*engine.EffectCharacter{}, base.Characters...)
	sort.Slice(order, func(i, j int) bool {
		if order[i].Coord.Col == order[j].Coord.Col {
			return order[i].Coord.Row < order[j].Coord.Row
		}
		return order[i].Coord.Col < order[j].Coord.Col
	})
	for _, character := range order {
		character.Visible = false
	}
	return &LaserEtch{base: base, order: order}
}

func (l *LaserEtch) Next() (string, bool) {
	if l.index >= len(l.order) {
		return "", false
	}
	l.order[l.index].Visible = true
	l.index++
	return engine.RenderFrame(l.base.Canvas, l.base.Characters), l.index < len(l.order)+1
}
