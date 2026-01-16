package effects

import (
	"tte-go/internal/config"
	"tte-go/internal/engine"
	"tte-go/internal/utils"
)

// Pour creates an effect where characters pour back and forth from the top, bottom, left, or right.
type Pour struct {
	base *BaseEffect

	pendingGroups    [][]*engine.EffectCharacter
	currentGroup     []*engine.EffectCharacter
	activeCharacters map[*engine.EffectCharacter]struct{}
	gap              int

	// Config
	pourDirection          string // "down", "up", "left", "right"
	pourSpeed              int
	movementSpeedRange     [2]float64
	gapFrames              int
	startingColor          utils.Color
	finalGradientStops     []utils.Color
	finalGradientSteps     []int
	finalGradientFrames    int
	finalGradientDirection utils.GradientDirection
}

func NewPour(input string, cfg config.TerminalConfig) engine.Effect {
	base := NewBaseEffect(input, cfg)

	// Config defaults
	startingColor := mustColors("ffffff")[0]
	finalGradientStops := mustColors("8A008A", "00D1FF", "FFFFFF")
	finalGradientSteps := []int{12}
	finalGradientDirection := utils.GradientVertical

	p := &Pour{
		base:                   base,
		pendingGroups:          make([][]*engine.EffectCharacter, 0),
		currentGroup:           make([]*engine.EffectCharacter, 0),
		activeCharacters:       make(map[*engine.EffectCharacter]struct{}),
		gap:                    0,
		pourDirection:          "down",
		pourSpeed:              2,
		movementSpeedRange:     [2]float64{0.4, 0.6},
		gapFrames:              1,
		startingColor:          startingColor,
		finalGradientStops:     finalGradientStops,
		finalGradientSteps:     finalGradientSteps,
		finalGradientFrames:    6,
		finalGradientDirection: finalGradientDirection,
	}

	p.build()
	return p
}

func (p *Pour) build() {
	// Build final gradient mapping
	finalGradient, _ := utils.NewGradient(p.finalGradientStops, p.finalGradientSteps, false)
	finalGradientMapping, _ := finalGradient.BuildCoordinateColorMapping(
		p.base.Canvas.TextBottom,
		p.base.Canvas.TextTop,
		p.base.Canvas.TextLeft,
		p.base.Canvas.TextRight,
		p.finalGradientDirection,
	)

	// Get character groups based on pour direction
	var groups [][]*engine.EffectCharacter
	switch p.pourDirection {
	case "down":
		groups = p.base.Terminal.GetCharactersGrouped(engine.RowBottomToTop, true, false, false, false)
	case "up":
		groups = p.base.Terminal.GetCharactersGrouped(engine.RowTopToBottom, true, false, false, false)
	case "left":
		groups = p.base.Terminal.GetCharactersGrouped(engine.ColumnLeftToRight, true, false, false, false)
	case "right":
		groups = p.base.Terminal.GetCharactersGrouped(engine.ColumnRightToLeft, true, false, false, false)
	default:
		groups = p.base.Terminal.GetCharactersGrouped(engine.RowBottomToTop, true, false, false, false)
	}

	for i, group := range groups {
		for _, character := range group {
			p.base.Terminal.SetCharacterVisibility(character, false)

			// Set starting position based on pour direction
			var startCoord utils.Coord
			switch p.pourDirection {
			case "down":
				startCoord = utils.Coord{Col: character.InputCoord.Col, Row: p.base.Canvas.Top}
			case "up":
				startCoord = utils.Coord{Col: character.InputCoord.Col, Row: p.base.Canvas.Bottom}
			case "left":
				startCoord = utils.Coord{Col: p.base.Canvas.Right, Row: character.InputCoord.Row}
			case "right":
				startCoord = utils.Coord{Col: p.base.Canvas.Left, Row: character.InputCoord.Row}
			default:
				startCoord = utils.Coord{Col: character.InputCoord.Col, Row: p.base.Canvas.Top}
			}
			character.Motion.SetCoordinate(startCoord)
			character.Coord = startCoord

			// Create path to input position
			speed := p.movementSpeedRange[0] + utils.RandFloat64()*(p.movementSpeedRange[1]-p.movementSpeedRange[0])
			inputPath, _ := character.Motion.NewPath(speed, utils.InQuad, nil, 0, false, "input")
			inputPath.AddWaypoint(character.InputCoord)
			character.Motion.ActivatePath(inputPath)

			// Create pour gradient scene
			charFinalColor := finalGradientMapping[character.InputCoord]
			pourGradient, _ := utils.NewGradient([]utils.Color{p.startingColor, charFinalColor}, p.finalGradientSteps, false)
			pourScene := character.Animation.NewScene("pour")
			symbols := make([]string, len(pourGradient.Spectrum))
			for j := range symbols {
				symbols[j] = character.Symbol
			}
			_ = pourScene.ApplyGradientToSymbols(symbols, p.finalGradientFrames, pourGradient, nil)
			character.Animation.ActivateSceneRef(pourScene)
		}

		// Alternate direction for back-and-forth effect
		if i%2 == 0 {
			p.pendingGroups = append(p.pendingGroups, group)
		} else {
			// Reverse the group
			reversed := make([]*engine.EffectCharacter, len(group))
			for j, char := range group {
				reversed[len(group)-1-j] = char
			}
			p.pendingGroups = append(p.pendingGroups, reversed)
		}
	}

	// Initialize current group
	if len(p.pendingGroups) > 0 {
		p.currentGroup = p.pendingGroups[0]
		p.pendingGroups = p.pendingGroups[1:]
	}
}

func (p *Pour) Next() (string, bool) {
	if len(p.pendingGroups) > 0 || len(p.activeCharacters) > 0 || len(p.currentGroup) > 0 {
		// Get next group if current is empty
		if len(p.currentGroup) == 0 && len(p.pendingGroups) > 0 {
			p.currentGroup = p.pendingGroups[0]
			p.pendingGroups = p.pendingGroups[1:]
		}

		// Pour characters from current group
		if len(p.currentGroup) > 0 {
			if p.gap == 0 {
				for i := 0; i < p.pourSpeed && len(p.currentGroup) > 0; i++ {
					nextChar := p.currentGroup[0]
					p.currentGroup = p.currentGroup[1:]
					p.base.Terminal.SetCharacterVisibility(nextChar, true)
					p.activeCharacters[nextChar] = struct{}{}
				}
				p.gap = p.gapFrames
			} else {
				p.gap--
			}
		}

		// Tick all active characters
		for char := range p.activeCharacters {
			char.Tick()
			if !char.IsActive() {
				delete(p.activeCharacters, char)
			}
		}

		return p.base.Terminal.GetFormattedOutputString(), true
	}

	return p.base.Terminal.GetFormattedOutputString(), false
}

func (p *Pour) CanvasHeight() int {
	return p.base.CanvasHeight()
}
