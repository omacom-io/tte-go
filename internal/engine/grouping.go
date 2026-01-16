package engine

import (
	"sort"

	"tte-go/internal/terminal"
	"tte-go/internal/utils"
)

type CharacterGroup int

type CharacterSort int

type ColorSort int

const (
	ColumnLeftToRight CharacterGroup = iota + 1
	ColumnRightToLeft
	RowTopToBottom
	RowBottomToTop
	DiagonalTopLeftToBottomRight
	DiagonalBottomLeftToTopRight
	DiagonalTopRightToBottomLeft
	DiagonalBottomRightToTopLeft
	CenterToOutside
	OutsideToCenter
)

const (
	RandomSort CharacterSort = iota + 1
	TopToBottomLeftToRight
	TopToBottomRightToLeft
	BottomToTopLeftToRight
	BottomToTopRightToLeft
	OutsideRowToMiddle
	MiddleRowToOutside
)

const (
	LeastToMost ColorSort = iota + 1
	MostToLeast
	RandomColorSort
)

func GroupCharacters(characters []*EffectCharacter, grouping CharacterGroup) [][]*EffectCharacter {
	return GroupCharactersWithCanvas(characters, nil, grouping)
}

func GroupCharactersWithCanvas(characters []*EffectCharacter, canvas *terminal.Canvas, grouping CharacterGroup) [][]*EffectCharacter {
	groups := map[int][]*EffectCharacter{}
	switch grouping {
	case RowTopToBottom, RowBottomToTop:
		for _, character := range characters {
			groups[character.InputCoord.Row] = append(groups[character.InputCoord.Row], character)
		}
	case ColumnLeftToRight, ColumnRightToLeft:
		for _, character := range characters {
			groups[character.InputCoord.Col] = append(groups[character.InputCoord.Col], character)
		}
	case DiagonalTopLeftToBottomRight, DiagonalBottomRightToTopLeft:
		for _, character := range characters {
			key := character.InputCoord.Col - character.InputCoord.Row
			groups[key] = append(groups[key], character)
		}
	case DiagonalTopRightToBottomLeft, DiagonalBottomLeftToTopRight:
		for _, character := range characters {
			key := character.InputCoord.Row + character.InputCoord.Col
			groups[key] = append(groups[key], character)
		}
	case CenterToOutside, OutsideToCenter:
		center := utils.Coord{Row: 0, Col: 0}
		if canvas != nil {
			center = utils.Coord{Row: canvas.Height / 2, Col: canvas.Width / 2}
		} else {
			for _, character := range characters {
				center.Row += character.InputCoord.Row
				center.Col += character.InputCoord.Col
			}
			if len(characters) > 0 {
				center.Row /= len(characters)
				center.Col /= len(characters)
			}
		}
		for _, character := range characters {
			distance := abs(character.InputCoord.Row-center.Row) + abs(character.InputCoord.Col-center.Col)
			groups[distance] = append(groups[distance], character)
		}
	default:
		for _, character := range characters {
			key := character.InputCoord.Row
			groups[key] = append(groups[key], character)
		}
	}
	keys := make([]int, 0, len(groups))
	for key := range groups {
		keys = append(keys, key)
	}
	sort.Ints(keys)
	if grouping == RowTopToBottom || grouping == ColumnRightToLeft || grouping == DiagonalBottomRightToTopLeft || grouping == DiagonalTopRightToBottomLeft || grouping == OutsideToCenter {
		reverse(keys)
	}
	result := make([][]*EffectCharacter, 0, len(keys))
	for _, key := range keys {
		row := groups[key]
		sort.Slice(row, func(i, j int) bool {
			if row[i].InputCoord.Row == row[j].InputCoord.Row {
				return row[i].InputCoord.Col < row[j].InputCoord.Col
			}
			return row[i].InputCoord.Row < row[j].InputCoord.Row
		})
		result = append(result, row)
	}
	return result
}

func SortCharacters(characters []*EffectCharacter, sortMode CharacterSort) []*EffectCharacter {
	ordered := append([]*EffectCharacter{}, characters...)
	switch sortMode {
	case RandomSort:
		utils.Shuffle(len(ordered), func(i, j int) { ordered[i], ordered[j] = ordered[j], ordered[i] })
	case TopToBottomLeftToRight:
		sort.Slice(ordered, func(i, j int) bool {
			if ordered[i].InputCoord.Row == ordered[j].InputCoord.Row {
				return ordered[i].InputCoord.Col < ordered[j].InputCoord.Col
			}
			return ordered[i].InputCoord.Row > ordered[j].InputCoord.Row
		})
	case TopToBottomRightToLeft:
		sort.Slice(ordered, func(i, j int) bool {
			if ordered[i].InputCoord.Row == ordered[j].InputCoord.Row {
				return ordered[i].InputCoord.Col > ordered[j].InputCoord.Col
			}
			return ordered[i].InputCoord.Row > ordered[j].InputCoord.Row
		})
	case BottomToTopLeftToRight:
		sort.Slice(ordered, func(i, j int) bool {
			if ordered[i].InputCoord.Row == ordered[j].InputCoord.Row {
				return ordered[i].InputCoord.Col < ordered[j].InputCoord.Col
			}
			return ordered[i].InputCoord.Row < ordered[j].InputCoord.Row
		})
	case BottomToTopRightToLeft:
		sort.Slice(ordered, func(i, j int) bool {
			if ordered[i].InputCoord.Row == ordered[j].InputCoord.Row {
				return ordered[i].InputCoord.Col > ordered[j].InputCoord.Col
			}
			return ordered[i].InputCoord.Row < ordered[j].InputCoord.Row
		})
	case OutsideRowToMiddle, MiddleRowToOutside:
		ordered = append([]*EffectCharacter{}, ordered...)
		ordered = sortByRowCol(ordered)
		result := make([]*EffectCharacter, 0, len(ordered))
		for i := 0; i < len(ordered); i++ {
			if i%2 == 0 {
				result = append(result, ordered[0])
				ordered = ordered[1:]
			} else {
				last := ordered[len(ordered)-1]
				result = append(result, last)
				ordered = ordered[:len(ordered)-1]
			}
		}
		if sortMode == MiddleRowToOutside {
			reverseCharacters(result)
		}
		return result
	}
	return ordered
}

func sortByRowCol(chars []*EffectCharacter) []*EffectCharacter {
	sort.Slice(chars, func(i, j int) bool {
		if chars[i].InputCoord.Row == chars[j].InputCoord.Row {
			return chars[i].InputCoord.Col < chars[j].InputCoord.Col
		}
		return chars[i].InputCoord.Row > chars[j].InputCoord.Row
	})
	return chars
}

func reverseCharacters(chars []*EffectCharacter) {
	for i, j := 0, len(chars)-1; i < j; i, j = i+1, j-1 {
		chars[i], chars[j] = chars[j], chars[i]
	}
}

func reverse(values []int) {
	for i, j := 0, len(values)-1; i < j; i, j = i+1, j-1 {
		values[i], values[j] = values[j], values[i]
	}
}

func abs(value int) int {
	if value < 0 {
		return -value
	}
	return value
}
