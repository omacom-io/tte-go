package effects

import (
	"tte-go/internal/config"
	"tte-go/internal/engine"
	"tte-go/internal/utils"
)

// Spray creates an effect where characters are sprayed from a single point.
type Spray struct {
	base *BaseEffect

	pendingChars     []*engine.EffectCharacter
	activeCharacters map[*engine.EffectCharacter]struct{}
	sprayOrigin      utils.Coord

	// Config
	sprayPosition          string // "n", "ne", "e", "se", "s", "sw", "w", "nw", "center"
	sprayVolume            float64
	movementSpeedRange     [2]float64
	finalGradientStops     []utils.Color
	finalGradientSteps     []int
	finalGradientDirection utils.GradientDirection
}

func NewSpray(input string, cfg config.TerminalConfig) engine.Effect {
	base := NewBaseEffect(input, cfg)

	// Config defaults
	finalGradientStops := mustColors("8A008A", "00D1FF", "FFFFFF")
	finalGradientSteps := []int{12}
	finalGradientDirection := utils.GradientVertical

	s := &Spray{
		base:                   base,
		pendingChars:           make([]*engine.EffectCharacter, 0),
		activeCharacters:       make(map[*engine.EffectCharacter]struct{}),
		sprayPosition:          "e",
		sprayVolume:            0.005,
		movementSpeedRange:     [2]float64{0.6, 1.4},
		finalGradientStops:     finalGradientStops,
		finalGradientSteps:     finalGradientSteps,
		finalGradientDirection: finalGradientDirection,
	}

	s.build()
	return s
}

func (s *Spray) build() {
	// Calculate spray origin based on position
	switch s.sprayPosition {
	case "n":
		s.sprayOrigin = utils.Coord{Col: s.base.Canvas.Center.Col, Row: s.base.Canvas.Top}
	case "ne":
		s.sprayOrigin = utils.Coord{Col: s.base.Canvas.Right, Row: s.base.Canvas.Top}
	case "e":
		s.sprayOrigin = utils.Coord{Col: s.base.Canvas.Right, Row: s.base.Canvas.Center.Row}
	case "se":
		s.sprayOrigin = utils.Coord{Col: s.base.Canvas.Right, Row: s.base.Canvas.Bottom}
	case "s":
		s.sprayOrigin = utils.Coord{Col: s.base.Canvas.Center.Col, Row: s.base.Canvas.Bottom}
	case "sw":
		s.sprayOrigin = utils.Coord{Col: s.base.Canvas.Left, Row: s.base.Canvas.Bottom}
	case "w":
		s.sprayOrigin = utils.Coord{Col: s.base.Canvas.Left, Row: s.base.Canvas.Center.Row}
	case "nw":
		s.sprayOrigin = utils.Coord{Col: s.base.Canvas.Left, Row: s.base.Canvas.Top}
	case "center":
		s.sprayOrigin = s.base.Canvas.Center
	default:
		s.sprayOrigin = utils.Coord{Col: s.base.Canvas.Right, Row: s.base.Canvas.Center.Row}
	}

	// Build final gradient mapping
	finalGradient, _ := utils.NewGradient(s.finalGradientStops, s.finalGradientSteps, false)
	finalGradientMapping, _ := finalGradient.BuildCoordinateColorMapping(
		s.base.Canvas.TextBottom,
		s.base.Canvas.TextTop,
		s.base.Canvas.TextLeft,
		s.base.Canvas.TextRight,
		s.finalGradientDirection,
	)

	// Set up all characters
	for _, character := range s.base.Terminal.GetCharacters(true, false, false, false, engine.TopToBottomLeftToRight) {
		// Set starting position to spray origin
		character.Motion.SetCoordinate(s.sprayOrigin)
		character.Coord = s.sprayOrigin

		// Create path to input position
		speed := s.movementSpeedRange[0] + utils.RandFloat64()*(s.movementSpeedRange[1]-s.movementSpeedRange[0])
		inputPath, _ := character.Motion.NewPath(speed, utils.OutExpo, nil, 0, false, "input")
		inputPath.AddWaypoint(character.InputCoord)

		// Set final color appearance
		charFinalColor := finalGradientMapping[character.InputCoord]
		colorCopy := charFinalColor
		character.SetVisual(engine.CharacterVisual{
			Symbol: character.Symbol,
			Colors: &utils.ColorPair{FG: &colorCopy},
		})

		s.pendingChars = append(s.pendingChars, character)
	}

	// Shuffle pending chars for random spray order
	for i := len(s.pendingChars) - 1; i > 0; i-- {
		j := utils.RandIntn(i + 1)
		s.pendingChars[i], s.pendingChars[j] = s.pendingChars[j], s.pendingChars[i]
	}
}

func (s *Spray) Next() (string, bool) {
	if len(s.pendingChars) > 0 || len(s.activeCharacters) > 0 {
		// Spray characters based on volume
		totalChars := len(s.base.Terminal.GetCharacters(true, false, false, false, engine.TopToBottomLeftToRight))
		charsToSpray := max(int(s.sprayVolume*float64(totalChars)), 1)

		for i := 0; i < charsToSpray && len(s.pendingChars) > 0; i++ {
			nextChar := s.pendingChars[0]
			s.pendingChars = s.pendingChars[1:]
			s.base.Terminal.SetCharacterVisibility(nextChar, true)
			nextChar.Motion.ActivatePath(nextChar.Motion.Paths["input"])
			s.activeCharacters[nextChar] = struct{}{}
		}

		// Tick all active characters
		for char := range s.activeCharacters {
			char.Tick()
			if !char.IsActive() {
				delete(s.activeCharacters, char)
			}
		}

		return s.base.Terminal.GetFormattedOutputString(), true
	}

	return s.base.Terminal.GetFormattedOutputString(), false
}

func (s *Spray) CanvasHeight() int {
	return s.base.CanvasHeight()
}
