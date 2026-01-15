package effects

import (
	"tte-go/internal/config"
	"tte-go/internal/engine"
	"tte-go/internal/utils"
)

type BinaryPath struct {
	base   *BaseEffect
	frames int
}

func NewBinaryPath(input string, cfg config.TerminalConfig) engine.Effect {
	base := NewBaseEffect(input, cfg)
	for _, character := range base.Characters {
		character.Visible = true
	}
	return &BinaryPath{base: base}
}

func (b *BinaryPath) Next() (string, bool) {
	if b.frames > 20 {
		for _, character := range b.base.Characters {
			character.SetVisual(engine.CharacterVisual{Symbol: character.Symbol})
		}
		return engine.RenderFrame(b.base.Canvas, b.base.Characters), false
	}
	for _, character := range b.base.Characters {
		if utils.RandIntn(2) == 0 {
			character.SetVisual(engine.CharacterVisual{Symbol: "0"})
		} else {
			character.SetVisual(engine.CharacterVisual{Symbol: "1"})
		}
	}
	b.frames++
	return engine.RenderFrame(b.base.Canvas, b.base.Characters), true
}
