package engine

import "tte-go/internal/utils"

func BuildFillCharacters(canvasLeft, canvasRight, canvasBottom, canvasTop int, existing []*EffectCharacter) (inner []*EffectCharacter, outer []*EffectCharacter) {
	occupied := map[utils.Coord]struct{}{}
	for _, character := range existing {
		occupied[character.InputCoord] = struct{}{}
	}
	for row := canvasBottom; row <= canvasTop; row++ {
		for col := canvasLeft; col <= canvasRight; col++ {
			coord := utils.Coord{Row: row, Col: col}
			if _, ok := occupied[coord]; ok {
				continue
			}
			fill := NewEffectCharacter(" ", coord)
			fill.IsFillCharacter = true
			inner = append(inner, fill)
		}
	}
	return inner, outer
}
