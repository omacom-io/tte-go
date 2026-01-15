package effects

import "fmt"

type Placeholder struct {
	name  string
	input string
	sent  bool
}

func NewPlaceholder(name, input string) *Placeholder {
	return &Placeholder{name: name, input: input}
}

func (p *Placeholder) Next() (string, bool) {
	if p.sent {
		return "", false
	}
	p.sent = true
	return fmt.Sprintf("[effect: %s]\n%s", p.name, p.input), true
}
