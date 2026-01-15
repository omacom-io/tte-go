package engine

import (
	"tte-go/internal/terminal"
)

type TerminalState struct {
	Canvas         *terminal.Canvas
	InputChars     []*EffectCharacter
	InnerFillChars []*EffectCharacter
	OuterFillChars []*EffectCharacter
	AddedChars     []*EffectCharacter
	Visible        map[*EffectCharacter]struct{}
}

func NewTerminalState(canvas *terminal.Canvas, characters []*EffectCharacter) *TerminalState {
	inner, outer := BuildFillCharacters(1, canvas.Width, 1, canvas.Height, characters)
	state := &TerminalState{
		Canvas:         canvas,
		InputChars:     characters,
		InnerFillChars: inner,
		OuterFillChars: outer,
		AddedChars:     []*EffectCharacter{},
		Visible:        map[*EffectCharacter]struct{}{},
	}
	for _, character := range characters {
		character.Visible = false
	}
	for _, character := range inner {
		character.Visible = false
	}
	for _, character := range outer {
		character.Visible = false
	}
	return state
}

func (t *TerminalState) AddCharacter(character *EffectCharacter) {
	if character == nil {
		return
	}
	character.IsAddedCharacter = true
	t.AddedChars = append(t.AddedChars, character)
	t.SetCharacterVisibility(character, true)
}

func (t *TerminalState) SetCharacterVisibility(character *EffectCharacter, visible bool) {
	if character == nil {
		return
	}
	character.Visible = visible
	if visible {
		t.Visible[character] = struct{}{}
	} else {
		delete(t.Visible, character)
	}
}

func (t *TerminalState) GetCharacters(inputChars, innerFillChars, outerFillChars, addedChars bool, sortMode CharacterSort) []*EffectCharacter {
	all := []*EffectCharacter{}
	if inputChars {
		all = append(all, t.InputChars...)
	}
	if innerFillChars {
		all = append(all, t.InnerFillChars...)
	}
	if outerFillChars {
		all = append(all, t.OuterFillChars...)
	}
	if addedChars {
		all = append(all, t.AddedChars...)
	}
	return SortCharacters(all, sortMode)
}

func (t *TerminalState) GetCharactersGrouped(grouping CharacterGroup, inputChars, innerFillChars, outerFillChars, addedChars bool) [][]*EffectCharacter {
	all := []*EffectCharacter{}
	if inputChars {
		all = append(all, t.InputChars...)
	}
	if innerFillChars {
		all = append(all, t.InnerFillChars...)
	}
	if outerFillChars {
		all = append(all, t.OuterFillChars...)
	}
	if addedChars {
		all = append(all, t.AddedChars...)
	}
	return GroupCharactersWithCanvas(all, t.Canvas, grouping)
}

func (t *TerminalState) GetFormattedOutputString() string {
	all := []*EffectCharacter{}
	all = append(all, t.InputChars...)
	all = append(all, t.InnerFillChars...)
	all = append(all, t.OuterFillChars...)
	all = append(all, t.AddedChars...)
	visible := make([]*EffectCharacter, 0, len(all))
	for _, character := range all {
		if character.Visible {
			visible = append(visible, character)
		}
	}
	return RenderFrame(t.Canvas, visible)
}
