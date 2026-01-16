package effects

import (
	"tte-go/internal/config"
	"tte-go/internal/engine"
	"tte-go/internal/utils"
)

type SlideConfig struct {
	MovementSpeed       float64
	Grouping            string
	Gap                 int
	ReverseDirection    bool
	Merge               bool
	MovementEasing      utils.EasingFunction
	FinalGradientStops  []utils.Color
	FinalGradientSteps  []int
	FinalGradientFrames int
	FinalGradientDir    utils.GradientDirection
}

type Slide struct {
	base             *BaseEffect
	config           SlideConfig
	pendingGroups    [][]*engine.EffectCharacter
	activeGroups     [][]*engine.EffectCharacter
	currentGap       int
	activeCharacters map[*engine.EffectCharacter]struct{}
	finalColors      map[*engine.EffectCharacter]utils.Color
}

func NewSlide(input string, cfg config.TerminalConfig) engine.Effect {
	base := NewBaseEffect(input, cfg)
	slide := &Slide{
		base:             base,
		config:           defaultSlideConfig(),
		activeCharacters: map[*engine.EffectCharacter]struct{}{},
		finalColors:      map[*engine.EffectCharacter]utils.Color{},
	}
	slide.build()
	return slide
}

func defaultSlideConfig() SlideConfig {
	return SlideConfig{
		MovementSpeed:       0.8,
		Grouping:            "row",
		Gap:                 2,
		ReverseDirection:    false,
		Merge:               false,
		MovementEasing:      utils.InOutQuad,
		FinalGradientStops:  mustColors("833ab4", "fd1d1d", "fcb045"),
		FinalGradientSteps:  []int{12},
		FinalGradientFrames: 6,
		FinalGradientDir:    utils.GradientVertical,
	}
}

func (s *Slide) build() {
	finalGradient, _ := utils.NewGradient(s.config.FinalGradientStops, s.config.FinalGradientSteps, false)
	finalMapping, _ := finalGradient.BuildCoordinateColorMapping(
		s.base.Canvas.TextBottom,
		s.base.Canvas.TextTop,
		s.base.Canvas.TextLeft,
		s.base.Canvas.TextRight,
		s.config.FinalGradientDir,
	)
	for _, character := range s.base.Terminal.GetCharacters(true, false, false, false, engine.TopToBottomLeftToRight) {
		s.finalColors[character] = finalMapping[character.InputCoord]
	}

	groups := [][]*engine.EffectCharacter{}
	switch s.config.Grouping {
	case "column":
		groups = s.base.Terminal.GetCharactersGrouped(engine.ColumnLeftToRight, true, false, false, false)
	case "diagonal":
		groups = s.base.Terminal.GetCharactersGrouped(engine.DiagonalTopLeftToBottomRight, true, false, false, false)
	default:
		groups = s.base.Terminal.GetCharactersGrouped(engine.RowTopToBottom, true, false, false, false)
	}

	for _, group := range groups {
		for _, character := range group {
			path, _ := character.Motion.NewPath(s.config.MovementSpeed, s.config.MovementEasing, nil, 0, false, "input_path")
			path.AddWaypoint(character.InputCoord)
		}
	}

	for index, group := range groups {
		switch s.config.Grouping {
		case "row":
			startingColumn := s.base.Canvas.Left - 1
			if s.config.Merge && index%2 == 0 {
				startingColumn = s.base.Canvas.Right + 1
			} else {
				reverseEffectCharacters(group)
			}
			if s.config.ReverseDirection && !s.config.Merge {
				reverseEffectCharacters(group)
				startingColumn = s.base.Canvas.Right + 1
			}
			for _, character := range group {
				character.Motion.SetCoordinate(utils.Coord{Row: character.InputCoord.Row, Col: startingColumn})
			}
		case "column":
			startingRow := s.base.Canvas.Top + 1
			if s.config.Merge && index%2 == 0 {
				startingRow = s.base.Canvas.Bottom - 1
			} else {
				reverseEffectCharacters(group)
			}
			if s.config.ReverseDirection && !s.config.Merge {
				reverseEffectCharacters(group)
				startingRow = s.base.Canvas.Bottom - 1
			}
			for _, character := range group {
				character.Motion.SetCoordinate(utils.Coord{Row: startingRow, Col: character.InputCoord.Col})
			}
		case "diagonal":
			distance := group[len(group)-1].InputCoord.Row - (s.base.Canvas.Bottom - 1)
			startingCoord := utils.Coord{Row: group[len(group)-1].InputCoord.Row - distance, Col: group[len(group)-1].InputCoord.Col - distance}
			if s.config.Merge && index%2 == 0 {
				reverseEffectCharacters(group)
				distance = (s.base.Canvas.Top + 1) - group[0].InputCoord.Row
				startingCoord = utils.Coord{Row: group[0].InputCoord.Row + distance, Col: group[0].InputCoord.Col + distance}
			}
			if s.config.ReverseDirection && !s.config.Merge {
				reverseEffectCharacters(group)
				distance = (s.base.Canvas.Top + 1) - group[0].InputCoord.Row
				startingCoord = utils.Coord{Row: group[0].InputCoord.Row + distance, Col: group[0].InputCoord.Col + distance}
			}
			for _, character := range group {
				character.Motion.SetCoordinate(startingCoord)
			}
		}

		for _, character := range group {
			gradientScene := character.Animation.NewScene("gradient")
			charGradient, _ := utils.NewGradient([]utils.Color{s.config.FinalGradientStops[0], s.finalColors[character]}, []int{10}, false)
			_ = gradientScene.ApplyGradientToSymbols([]string{character.Symbol}, s.config.FinalGradientFrames, charGradient, nil)
			character.Animation.ActivateScene("gradient")
		}
	}

	s.pendingGroups = groups
	s.currentGap = 0
}

func (s *Slide) Next() (string, bool) {
	if len(s.pendingGroups) == 0 && len(s.activeCharacters) == 0 && len(s.activeGroups) == 0 {
		return "", false
	}
	if s.currentGap == s.config.Gap && len(s.pendingGroups) > 0 {
		s.activeGroups = append(s.activeGroups, s.pendingGroups[0])
		s.pendingGroups = s.pendingGroups[1:]
		s.currentGap = 0
	} else if len(s.pendingGroups) > 0 {
		s.currentGap++
	}
	for i, group := range s.activeGroups {
		if len(group) > 0 {
			next := group[0]
			group = group[1:]
			s.activeGroups[i] = group
			s.base.Terminal.SetCharacterVisibility(next, true)
			if path, ok := next.Motion.Paths["input_path"]; ok {
				next.Motion.ActivatePath(path)
			}
			s.activeCharacters[next] = struct{}{}
		}
	}
	remaining := [][]*engine.EffectCharacter{}
	for _, group := range s.activeGroups {
		if len(group) > 0 {
			remaining = append(remaining, group)
		}
	}
	s.activeGroups = remaining
	for character := range s.activeCharacters {
		character.Tick()
		if !character.IsActive() {
			delete(s.activeCharacters, character)
		}
	}
	return s.base.Terminal.GetFormattedOutputString(), true
}
