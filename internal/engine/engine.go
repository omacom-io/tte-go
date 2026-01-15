package engine

import "tte-go/internal/terminal"

type Engine struct {
	terminal *terminal.Terminal
}

func New(term *terminal.Terminal) *Engine {
	return &Engine{terminal: term}
}

func (e *Engine) Run(effect Effect) error {
	e.terminal.Prepare()
	defer e.terminal.Restore("")

	frame, ok := effect.Next()
	for ok {
		nextFrame, nextOk := effect.Next()
		e.terminal.PrintFrame(frame, !nextOk)
		frame = nextFrame
		ok = nextOk
	}
	return nil
}
