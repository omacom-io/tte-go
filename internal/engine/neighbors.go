package engine

import "tte-go/internal/utils"

func BuildNeighbors(characters []*EffectCharacter) {
	lookup := map[utils.Coord]*EffectCharacter{}
	for _, character := range characters {
		lookup[character.InputCoord] = character
	}
	for _, character := range characters {
		row := character.InputCoord.Row
		col := character.InputCoord.Col
		character.Neighbors["n"] = lookup[utils.Coord{Row: row + 1, Col: col}]
		character.Neighbors["s"] = lookup[utils.Coord{Row: row - 1, Col: col}]
		character.Neighbors["e"] = lookup[utils.Coord{Row: row, Col: col + 1}]
		character.Neighbors["w"] = lookup[utils.Coord{Row: row, Col: col - 1}]
		character.Neighbors["ne"] = lookup[utils.Coord{Row: row + 1, Col: col + 1}]
		character.Neighbors["nw"] = lookup[utils.Coord{Row: row + 1, Col: col - 1}]
		character.Neighbors["se"] = lookup[utils.Coord{Row: row - 1, Col: col + 1}]
		character.Neighbors["sw"] = lookup[utils.Coord{Row: row - 1, Col: col - 1}]
	}
}
