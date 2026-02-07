package effects

import (
	"tte-go/internal/config"
	"tte-go/internal/engine"
	"tte-go/internal/utils"
)

// OrbittingVolley creates an effect where four launchers orbit the canvas
// firing volleys of characters inward to build the input text from the center out.
type OrbittingVolley struct {
	base *BaseEffect

	launchers        []*launcher
	mainLauncher     *launcher
	activeCharacters map[*engine.EffectCharacter]struct{}
	sortedChars      []*engine.EffectCharacter
	delay            int
	complete         bool

	// Config
	launcherMovementSpeed  float64
	characterMovementSpeed float64
	volleySize             float64
	launchDelay            int
	finalGradientStops     []utils.Color
	finalGradientSteps     []int
	finalGradientDirection utils.GradientDirection

	// Gradient maps
	finalGradientMap    map[utils.Coord]utils.Color
	launcherGradientMap map[utils.Coord]utils.Color
}

type launcher struct {
	character *engine.EffectCharacter
	magazine  []*engine.EffectCharacter
	terminal  *engine.TerminalState
}

func newLauncher(terminal *engine.TerminalState, coord utils.Coord, symbol string) *launcher {
	char := engine.NewEffectCharacter(symbol, coord)
	char.Layer = 2
	char.IsAddedCharacter = true
	terminal.AddCharacter(char)
	terminal.SetCharacterVisibility(char, true)
	return &launcher{
		character: char,
		magazine:  make([]*engine.EffectCharacter, 0),
		terminal:  terminal,
	}
}

func (l *launcher) launch() *engine.EffectCharacter {
	if len(l.magazine) == 0 {
		return nil
	}
	nextChar := l.magazine[0]
	l.magazine = l.magazine[1:]
	nextChar.Motion.SetCoordinate(l.character.Coord)
	nextChar.Coord = l.character.Coord
	nextChar.Motion.ActivatePath(nextChar.Motion.Paths["input_path"])
	l.terminal.SetCharacterVisibility(nextChar, true)
	return nextChar
}

func NewOrbittingVolley(input string, cfg config.TerminalConfig) engine.Effect {
	base := NewBaseEffect(input, cfg)

	// Config defaults
	finalGradientStops := mustColors("FFA15C", "44D492")
	finalGradientSteps := []int{12}
	finalGradientDirection := utils.GradientRadial

	o := &OrbittingVolley{
		base:                   base,
		launchers:              make([]*launcher, 0),
		activeCharacters:       make(map[*engine.EffectCharacter]struct{}),
		sortedChars:            make([]*engine.EffectCharacter, 0),
		delay:                  0,
		complete:               false,
		launcherMovementSpeed:  0.8,
		characterMovementSpeed: 1.5,
		volleySize:             0.03,
		launchDelay:            30,
		finalGradientStops:     finalGradientStops,
		finalGradientSteps:     finalGradientSteps,
		finalGradientDirection: finalGradientDirection,
	}

	o.build()
	return o
}

func (o *OrbittingVolley) build() {
	// Build final gradient mappings
	finalGradient, _ := utils.NewGradient(o.finalGradientStops, o.finalGradientSteps, false)
	o.finalGradientMap, _ = finalGradient.BuildCoordinateColorMapping(
		o.base.Canvas.TextBottom,
		o.base.Canvas.TextTop,
		o.base.Canvas.TextLeft,
		o.base.Canvas.TextRight,
		o.finalGradientDirection,
	)
	o.launcherGradientMap, _ = finalGradient.BuildCoordinateColorMapping(
		o.base.Canvas.Bottom,
		o.base.Canvas.Top,
		o.base.Canvas.Left,
		o.base.Canvas.Right,
		o.finalGradientDirection,
	)

	// Set up characters with final colors and input paths
	for _, character := range o.base.Terminal.GetCharacters(true, false, false, false, engine.TopToBottomLeftToRight) {
		charFinalColor := o.finalGradientMap[character.InputCoord]

		// Create input path
		layer := 1
		inputPath, _ := character.Motion.NewPath(o.characterMovementSpeed, utils.OutSine, &layer, 0, false, "input_path")
		inputPath.AddWaypoint(character.InputCoord)

		// Register event to reset layer when path completes
		layer0 := 0
		character.EventHandler.RegisterEvent(
			engine.EventPathComplete,
			inputPath,
			engine.ActionSetLayer,
			layer0,
		)

		// Set initial appearance
		colorCopy := charFinalColor
		character.SetVisual(engine.CharacterVisual{
			Symbol: character.Symbol,
			Colors: &utils.ColorPair{FG: &colorCopy},
		})
	}

	// Create launchers at four corners
	launcherPositions := []utils.Coord{
		{Col: o.base.Canvas.Left, Row: o.base.Canvas.Top},     // top-left
		{Col: o.base.Canvas.Right, Row: o.base.Canvas.Top},    // top-right
		{Col: o.base.Canvas.Right, Row: o.base.Canvas.Bottom}, // bottom-right
		{Col: o.base.Canvas.Left, Row: o.base.Canvas.Bottom},  // bottom-left
	}

	for _, pos := range launcherPositions {
		l := newLauncher(o.base.Terminal, pos, "█")
		o.launchers = append(o.launchers, l)
		o.activeCharacters[l.character] = struct{}{}
	}

	// Main launcher is the first one (top-left)
	o.mainLauncher = o.launchers[0]

	// Set main launcher color
	lastColor := finalGradient.Spectrum[len(finalGradient.Spectrum)-1]
	o.mainLauncher.character.SetVisual(engine.CharacterVisual{
		Symbol: o.mainLauncher.character.Symbol,
		Colors: &utils.ColorPair{FG: &lastColor},
	})

	// Build perimeter path for main launcher (left to right along top edge)
	perimeterPath, _ := o.mainLauncher.character.Motion.NewPath(o.launcherMovementSpeed, nil, nil, 0, false, "perimeter")
	perimeterPath.AddWaypoint(utils.Coord{Col: o.base.Canvas.Left, Row: o.base.Canvas.Top})
	perimeterPath.AddWaypoint(utils.Coord{Col: o.base.Canvas.Right, Row: o.base.Canvas.Top})
	o.mainLauncher.character.Motion.ActivatePath(perimeterPath)

	// Get characters sorted from center outward
	for _, charList := range o.base.Terminal.GetCharactersGrouped(engine.CenterToOutside, true, false, false, false) {
		o.sortedChars = append(o.sortedChars, charList...)
	}

	// Distribute characters to launchers in round-robin
	for i, char := range o.sortedChars {
		launcherIdx := i % len(o.launchers)
		o.launchers[launcherIdx].magazine = append(o.launchers[launcherIdx].magazine, char)
	}
}

func (o *OrbittingVolley) setLauncherCoordinates(parent, child *launcher) {
	// Calculate progress of parent along top edge (0 to 1)
	// Use Motion.CurrentCoord which is updated during path stepping
	parentProgress := float64(parent.character.Motion.CurrentCoord.Col) / float64(o.base.Canvas.Right)

	childInputCoord := child.character.InputCoord

	var newCoord utils.Coord
	if childInputCoord.Col == o.base.Canvas.Right && childInputCoord.Row == o.base.Canvas.Top {
		// Top-right launcher moves down along right edge
		childRow := o.base.Canvas.Top - int(float64(o.base.Canvas.Top)*parentProgress)
		newCoord = utils.Coord{Col: o.base.Canvas.Right, Row: max(1, childRow)}
	} else if childInputCoord.Col == o.base.Canvas.Right && childInputCoord.Row == o.base.Canvas.Bottom {
		// Bottom-right launcher moves left along bottom edge
		childCol := o.base.Canvas.Right - int(float64(o.base.Canvas.Right)*parentProgress)
		newCoord = utils.Coord{Col: max(1, childCol), Row: o.base.Canvas.Bottom}
	} else if childInputCoord.Col == o.base.Canvas.Left && childInputCoord.Row == o.base.Canvas.Bottom {
		// Bottom-left launcher moves up along left edge
		childRow := o.base.Canvas.Bottom + int(float64(o.base.Canvas.Top)*parentProgress)
		newCoord = utils.Coord{Col: o.base.Canvas.Left, Row: min(o.base.Canvas.Top, childRow)}
	}

	child.character.Motion.SetCoordinate(newCoord)
	child.character.Coord = newCoord

	// Set color based on position
	if color, ok := o.launcherGradientMap[newCoord]; ok {
		child.character.SetVisual(engine.CharacterVisual{
			Symbol: child.character.Symbol,
			Colors: &utils.ColorPair{FG: &color},
		})
	}
}

func (o *OrbittingVolley) Next() (string, bool) {
	// Check if any launcher has characters or if there are active characters (besides launchers)
	hasCharacters := false
	for _, l := range o.launchers {
		if len(l.magazine) > 0 {
			hasCharacters = true
			break
		}
	}

	activeNonLaunchers := 0
	for char := range o.activeCharacters {
		isLauncher := false
		for _, l := range o.launchers {
			if char == l.character {
				isLauncher = true
				break
			}
		}
		if !isLauncher {
			activeNonLaunchers++
		}
	}

	if hasCharacters || activeNonLaunchers > 0 {
		// Keep main launcher moving along perimeter
		if o.mainLauncher.character.Motion.ActivePath == nil {
			// Reactivate the existing perimeter path from the starting position
			perimeterPath, _ := o.mainLauncher.character.Motion.QueryPath("perimeter")
			if perimeterPath != nil {
				o.mainLauncher.character.Motion.SetCoordinate(utils.Coord{Col: o.base.Canvas.Left, Row: o.base.Canvas.Top})
				o.mainLauncher.character.Coord = utils.Coord{Col: o.base.Canvas.Left, Row: o.base.Canvas.Top}
				o.mainLauncher.character.Motion.ActivatePath(perimeterPath)
				o.activeCharacters[o.mainLauncher.character] = struct{}{}
			}
		}

		// Launch characters when delay is 0
		if o.delay == 0 {
			totalChars := len(o.base.Terminal.GetCharacters(true, false, false, false, engine.TopToBottomLeftToRight))
			charsToLaunch := max(int(o.volleySize*float64(totalChars)/4), 1)

			for _, l := range o.launchers {
				for i := 0; i < charsToLaunch; i++ {
					if nextChar := l.launch(); nextChar != nil {
						o.activeCharacters[nextChar] = struct{}{}
					}
				}
			}
			o.delay = o.launchDelay
		} else {
			o.delay--
		}

		// Tick all active characters - this updates Motion.CurrentCoord
		for char := range o.activeCharacters {
			char.Tick()
			if !char.IsActive() {
				// Don't remove launchers from active set
				isLauncher := false
				for _, l := range o.launchers {
					if char == l.character {
						isLauncher = true
						break
					}
				}
				if !isLauncher {
					delete(o.activeCharacters, char)
				}
			}
		}

		// Update main launcher color based on current position (after tick)
		if color, ok := o.launcherGradientMap[o.mainLauncher.character.Motion.CurrentCoord]; ok {
			o.mainLauncher.character.SetVisual(engine.CharacterVisual{
				Symbol: o.mainLauncher.character.Symbol,
				Colors: &utils.ColorPair{FG: &color},
			})
		}

		// Update other launcher positions based on main launcher's current position (after tick)
		for _, l := range o.launchers[1:] {
			o.setLauncherCoordinates(o.mainLauncher, l)
		}

		return o.base.Terminal.GetFormattedOutputString(), true
	}

	// Effect complete - hide launchers
	if !o.complete {
		o.complete = true
		for _, l := range o.launchers {
			o.base.Terminal.SetCharacterVisibility(l.character, false)
		}
		return o.base.Terminal.GetFormattedOutputString(), true
	}

	return o.base.Terminal.GetFormattedOutputString(), false
}

func (o *OrbittingVolley) CanvasHeight() int {
	return o.base.CanvasHeight()
}

func (o *OrbittingVolley) CanvasWidth() int {
	return o.base.CanvasWidth()
}
