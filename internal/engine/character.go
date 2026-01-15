package engine

import "tte-go/internal/utils"

type EffectCharacter struct {
	Symbol                string
	Coord                 utils.Coord
	InputCoord            utils.Coord
	InputColors           *utils.ColorPair
	ExistingColorHandling string
	UseXterm              bool
	NoColor               bool
	Visible               bool
	Layer                 int
	Animation             *Animation
	Motion                *Motion
	EventHandler          *EventHandler
	Neighbors             map[string]*EffectCharacter
	Links                 map[*EffectCharacter]struct{}
	IsFillCharacter       bool
	IsAddedCharacter      bool
	current               CharacterVisual
}

func NewEffectCharacter(symbol string, coord utils.Coord) *EffectCharacter {
	visual := CharacterVisual{Symbol: symbol}
	anim := NewAnimation(false, false, nil)
	scene := NewScene("default", false, false, false)
	_ = scene.AddFrame(visual.Symbol, 1, nil)
	anim.AddScene(scene)
	anim.ActiveScene = nil
	character := &EffectCharacter{
		Symbol:     symbol,
		Coord:      coord,
		InputCoord: coord,
		Visible:    true,
		Layer:      0,
		Animation:  anim,
		Neighbors:  map[string]*EffectCharacter{},
		Links:      map[*EffectCharacter]struct{}{},
		current:    visual,
	}
	anim.InputColors = character.InputColors
	character.Motion = NewMotion(character)
	character.EventHandler = NewEventHandler(character)
	character.Motion.EventHandler = character.EventHandler
	character.Animation.UseXterm = character.UseXterm
	character.Animation.NoColor = character.NoColor
	character.Animation.InputColors = character.InputColors
	return character
}

func (c *EffectCharacter) Tick() {
	if c.Motion != nil {
		c.Motion.Move()
		c.Coord = c.Motion.CurrentCoord
	}
	if c.Animation == nil {
		return
	}
	visual, ok := c.Animation.Next()
	if ok {
		c.current = visual
	}
}

func (c *EffectCharacter) IsActive() bool {
	activeAnimation := c.Animation != nil && c.Animation.ActiveScene != nil && !c.Animation.ActiveScene.IsComplete()
	activeMotion := c.Motion != nil && !c.Motion.MovementComplete()
	return activeAnimation || activeMotion
}

func (c *EffectCharacter) Visual() CharacterVisual {
	return c.current
}

func (c *EffectCharacter) SetVisual(visual CharacterVisual) {
	c.current = visual
}
