package effects

import (
	"tte-go/internal/config"
	"tte-go/internal/engine"
	"tte-go/internal/terminal"
)

type BaseEffect struct {
	Input      string
	Config     config.TerminalConfig
	Canvas     *terminal.Canvas
	Characters []*engine.EffectCharacter
	Terminal   *engine.TerminalState
	Done       bool
}

func NewBaseEffect(input string, cfg config.TerminalConfig) *BaseEffect {
	canvas, chars := engine.ParseInput(input, cfg)
	return &BaseEffect{
		Input:      input,
		Config:     cfg,
		Canvas:     canvas,
		Characters: chars,
		Terminal:   engine.NewTerminalState(canvas, chars),
	}
}

func (e *BaseEffect) Next() (string, bool) {
	if e.Done {
		return "", false
	}
	e.Done = true
	return e.Terminal.GetFormattedOutputString(), true
}
