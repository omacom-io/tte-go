package effects

import (
	"tte-go/internal/config"
	"tte-go/internal/engine"
	"tte-go/internal/utils"
)

type SliceConfig struct {
	SliceDirection     string
	MovementSpeed      float64
	MovementEasing     utils.EasingFunction
	FinalGradientStops []utils.Color
	FinalGradientSteps []int
	FinalGradientDir   utils.GradientDirection
}

type Slice struct {
	base             *BaseEffect
	config           SliceConfig
	activeCharacters map[*engine.EffectCharacter]struct{}
}

func NewSlice(input string, cfg config.TerminalConfig) engine.Effect {
	base := NewBaseEffect(input, cfg)
	slice := &Slice{
		base:             base,
		config:           defaultSliceConfig(),
		activeCharacters: map[*engine.EffectCharacter]struct{}{},
	}
	slice.build()
	return slice
}

func defaultSliceConfig() SliceConfig {
	return SliceConfig{
		SliceDirection:     "vertical",
		MovementSpeed:      0.25,
		MovementEasing:     utils.InOutExpo,
		FinalGradientStops: mustColors("8A008A", "00D1FF", "FFFFFF"),
		FinalGradientSteps: []int{12},
		FinalGradientDir:   utils.GradientDiagonal,
	}
}

func (s *Slice) build() {
	directionMap := map[string]engine.CharacterGroup{
		"vertical":   engine.RowBottomToTop,
		"horizontal": engine.ColumnRightToLeft,
		"diagonal":   engine.DiagonalBottomLeftToTopRight,
	}
	finalGradient, _ := utils.NewGradient(s.config.FinalGradientStops, s.config.FinalGradientSteps, false)
	finalMapping, _ := finalGradient.BuildCoordinateColorMapping(
		s.base.Canvas.TextBottom,
		s.base.Canvas.TextTop,
		s.base.Canvas.TextLeft,
		s.base.Canvas.TextRight,
		s.config.FinalGradientDir,
	)
	for _, character := range s.base.Terminal.GetCharacters(true, false, false, false, engine.TopToBottomLeftToRight) {
		color := finalMapping[character.InputCoord]
		scene := character.Animation.NewScene("final")
		_ = scene.AddFrame(character.Symbol, 1, &utils.ColorPair{FG: &color})
		character.Animation.ActivateScene("final")
	}

	switch s.config.SliceDirection {
	case "vertical":
		rows := s.base.Terminal.GetCharactersGrouped(directionMap[s.config.SliceDirection], true, false, false, false)
		for rowIndex, row := range rows {
			leftHalf := []*engine.EffectCharacter{}
			rightHalf := []*engine.EffectCharacter{}
			for _, character := range row {
				if character.InputCoord.Col <= s.base.Canvas.TextCenter.Col {
					leftHalf = append(leftHalf, character)
				} else {
					rightHalf = append(rightHalf, character)
				}
			}
			for _, character := range leftHalf {
				character.Motion.SetCoordinate(utils.Coord{Row: s.base.Canvas.Top + 1, Col: character.InputCoord.Col})
				path, _ := character.Motion.NewPath(s.config.MovementSpeed, s.config.MovementEasing, nil, 0, false, "")
				path.AddWaypoint(character.InputCoord)
				character.Motion.ActivatePath(path)
				s.activeCharacters[character] = struct{}{}
			}
			oppositeRow := rows[len(rows)-(rowIndex+1)]
			rightHalf = []*engine.EffectCharacter{}
			for _, character := range oppositeRow {
				if character.InputCoord.Col > s.base.Canvas.TextCenter.Col {
					rightHalf = append(rightHalf, character)
				}
			}
			for _, character := range rightHalf {
				character.Motion.SetCoordinate(utils.Coord{Row: s.base.Canvas.Bottom - 1, Col: character.InputCoord.Col})
				path, _ := character.Motion.NewPath(s.config.MovementSpeed, s.config.MovementEasing, nil, 0, false, "")
				path.AddWaypoint(character.InputCoord)
				character.Motion.ActivatePath(path)
				s.activeCharacters[character] = struct{}{}
			}
		}
	case "horizontal":
		s.config.MovementSpeed *= 2
		columns := s.base.Terminal.GetCharactersGrouped(directionMap[s.config.SliceDirection], true, true, true, false)
		trimmed := [][]*engine.EffectCharacter{}
		for _, column := range columns {
			newColumn := []*engine.EffectCharacter{}
			for _, character := range column {
				if character.InputCoord.Col >= s.base.Canvas.TextLeft && character.InputCoord.Col <= s.base.Canvas.TextRight && character.InputCoord.Row >= s.base.Canvas.TextBottom && character.InputCoord.Row <= s.base.Canvas.TextTop {
					newColumn = append(newColumn, character)
				}
			}
			if len(newColumn) > 0 {
				trimmed = append(trimmed, newColumn)
			}
		}
		columns = trimmed
		midPoint := s.base.Canvas.TextCenter.Row
		for colIndex, column := range columns {
			bottomHalf := []*engine.EffectCharacter{}
			for _, character := range column {
				if character.InputCoord.Row <= midPoint {
					bottomHalf = append(bottomHalf, character)
				}
			}
			for _, character := range bottomHalf {
				character.Motion.SetCoordinate(utils.Coord{Row: character.InputCoord.Row, Col: s.base.Canvas.Left - 1})
				path, _ := character.Motion.NewPath(s.config.MovementSpeed, s.config.MovementEasing, nil, 0, false, "")
				path.AddWaypoint(character.InputCoord)
				character.Motion.ActivatePath(path)
				s.activeCharacters[character] = struct{}{}
			}
			opposite := columns[len(columns)-(colIndex+1)]
			topHalf := []*engine.EffectCharacter{}
			for _, character := range opposite {
				if character.InputCoord.Row > midPoint {
					topHalf = append(topHalf, character)
				}
			}
			for _, character := range topHalf {
				character.Motion.SetCoordinate(utils.Coord{Row: character.InputCoord.Row, Col: s.base.Canvas.Right + 1})
				path, _ := character.Motion.NewPath(s.config.MovementSpeed, s.config.MovementEasing, nil, 0, false, "")
				path.AddWaypoint(character.InputCoord)
				character.Motion.ActivatePath(path)
				s.activeCharacters[character] = struct{}{}
			}
		}
	case "diagonal":
		diagonals := s.base.Terminal.GetCharactersGrouped(directionMap[s.config.SliceDirection], true, false, false, false)
		left := diagonals[:len(diagonals)/2]
		right := diagonals[len(diagonals)/2:]
		for len(left) > 0 || len(right) > 0 {
			group := []*engine.EffectCharacter{}
			if len(left) > 0 {
				leftGroup := left[0]
				left = left[1:]
				origin := utils.Coord{Row: s.base.Canvas.Bottom - 1, Col: leftGroup[0].InputCoord.Col}
				for _, character := range leftGroup {
					character.Motion.SetCoordinate(origin)
					path, _ := character.Motion.NewPath(s.config.MovementSpeed, s.config.MovementEasing, nil, 0, false, "")
					path.AddWaypoint(character.InputCoord)
					character.Motion.ActivatePath(path)
					group = append(group, character)
				}
			}
			if len(right) > 0 {
				rightGroup := right[0]
				right = right[1:]
				origin := utils.Coord{Row: s.base.Canvas.Top + 1, Col: rightGroup[len(rightGroup)-1].InputCoord.Col}
				for _, character := range rightGroup {
					character.Motion.SetCoordinate(origin)
					path, _ := character.Motion.NewPath(s.config.MovementSpeed, s.config.MovementEasing, nil, 0, false, "")
					path.AddWaypoint(character.InputCoord)
					character.Motion.ActivatePath(path)
					group = append(group, character)
				}
			}
			for _, character := range group {
				s.activeCharacters[character] = struct{}{}
			}
		}
	}

	for character := range s.activeCharacters {
		s.base.Terminal.SetCharacterVisibility(character, true)
	}
}

func (s *Slice) Next() (string, bool) {
	if len(s.activeCharacters) == 0 {
		return "", false
	}
	for character := range s.activeCharacters {
		character.Tick()
		if !character.IsActive() {
			delete(s.activeCharacters, character)
		}
	}
	return s.base.Terminal.GetFormattedOutputString(), true
}
