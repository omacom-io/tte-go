package effects

import (
	"tte-go/internal/config"
	"tte-go/internal/engine"
	"tte-go/internal/utils"
)

type RandomSequence struct {
	base  *BaseEffect
	order []*engine.EffectCharacter
	index int
}

func NewRandomSequence(input string, cfg config.TerminalConfig) engine.Effect {
	base := NewBaseEffect(input, cfg)
	order := append([]*engine.EffectCharacter{}, base.Characters...)
	for _, character := range order {
		character.Visible = false
	}
	utils.Shuffle(len(order), func(i, j int) { order[i], order[j] = order[j], order[i] })
	return &RandomSequence{base: base, order: order}
}

func (r *RandomSequence) Next() (string, bool) {
	if r.index >= len(r.order) {
		return "", false
	}
	r.order[r.index].Visible = true
	r.index++
	return engine.RenderFrame(r.base.Canvas, r.base.Characters), r.index < len(r.order)+1
}
