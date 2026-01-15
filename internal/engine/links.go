package engine

func (c *EffectCharacter) Link(other *EffectCharacter, bidirectional bool) {
	if other == nil {
		return
	}
	c.Links[other] = struct{}{}
	if bidirectional {
		other.Link(c, false)
	}
}
