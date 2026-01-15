package effects

import (
	"sort"

	"tte-go/internal/config"
	"tte-go/internal/engine"
)

type Print struct {
	base  *BaseEffect
	order []*engine.EffectCharacter
	index int
}

func NewPrint(input string, cfg config.TerminalConfig) engine.Effect {
	base := NewBaseEffect(input, cfg)
	order := append([]*engine.EffectCharacter{}, base.Characters...)
	sort.Slice(order, func(i, j int) bool {
		if order[i].Coord.Row == order[j].Coord.Row {
			return order[i].Coord.Col < order[j].Coord.Col
		}
		return order[i].Coord.Row < order[j].Coord.Row
	})
	for _, character := range order {
		character.Visible = false
	}
	return &Print{base: base, order: order}
}

func (p *Print) Next() (string, bool) {
	if p.index >= len(p.order) {
		return "", false
	}
	p.order[p.index].Visible = true
	p.index++
	return engine.RenderFrame(p.base.Canvas, p.base.Characters), p.index < len(p.order)+1
}
