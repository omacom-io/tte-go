package effects

import (
	"tte-go/internal/config"
	"tte-go/internal/engine"
	"tte-go/internal/utils"
)

type Decrypt struct {
	base   *BaseEffect
	frames int
}

func NewDecrypt(input string, cfg config.TerminalConfig) engine.Effect {
	base := NewBaseEffect(input, cfg)
	return &Decrypt{base: base}
}

func (d *Decrypt) Next() (string, bool) {
	if d.frames > 20 {
		for _, character := range d.base.Characters {
			character.SetVisual(engine.CharacterVisual{Symbol: character.Symbol})
			character.Visible = true
		}
		return engine.RenderFrame(d.base.Canvas, d.base.Characters), false
	}
	for _, character := range d.base.Characters {
		character.SetVisual(engine.CharacterVisual{Symbol: randomGlyph()})
		character.Visible = true
	}
	d.frames++
	return engine.RenderFrame(d.base.Canvas, d.base.Characters), true
}

func randomGlyph() string {
	glyphs := []rune("ABCDEFGHIJKLMNOPQRSTUVWXYZ0123456789#$%&*@")
	return string(glyphs[utils.RandIntn(len(glyphs))])
}

func (d *Decrypt) CanvasHeight() int {
	return d.base.CanvasHeight()
}
