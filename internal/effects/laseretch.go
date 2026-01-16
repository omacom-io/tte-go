package effects

import (
	"tte-go/internal/config"
	"tte-go/internal/engine"
	"tte-go/internal/utils"
)

// LaserEtch creates an effect where a laser beam etches characters onto the terminal.
type LaserEtch struct {
	base *BaseEffect

	pendingChars        []*engine.EffectCharacter
	activeCharacters    map[*engine.EffectCharacter]struct{}
	characterFinalColor map[*engine.EffectCharacter]utils.Color
	charDelay           int

	// Laser beam
	laserBeamChars []*engine.EffectCharacter
	laserPosition  utils.Coord

	// Sparks pool
	sparks      []*engine.EffectCharacter
	sparkIndex  int
	sparkColors []utils.Color

	// Config
	etchSpeed              int
	etchDelay              int
	coolGradientStops      []utils.Color
	laserGradientStops     []utils.Color
	sparkGradientStops     []utils.Color
	sparkCoolingFrames     int
	finalGradientStops     []utils.Color
	finalGradientSteps     []int
	finalGradientFrames    int
	finalGradientDirection utils.GradientDirection
}

func NewLaserEtch(input string, cfg config.TerminalConfig) engine.Effect {
	base := NewBaseEffect(input, cfg)

	// Config defaults
	coolGradientStops := mustColors("ffe680", "ff7b00")
	laserGradientStops := mustColors("ffffff", "376cff")
	sparkGradientStops := mustColors("ffffff", "ffe680", "ff7b00", "1a0900")
	finalGradientStops := mustColors("8A008A", "00D1FF", "ffffff")
	finalGradientSteps := []int{8}
	finalGradientDirection := utils.GradientVertical

	l := &LaserEtch{
		base:                   base,
		pendingChars:           make([]*engine.EffectCharacter, 0),
		activeCharacters:       make(map[*engine.EffectCharacter]struct{}),
		characterFinalColor:    make(map[*engine.EffectCharacter]utils.Color),
		charDelay:              0,
		laserBeamChars:         make([]*engine.EffectCharacter, 0),
		sparks:                 make([]*engine.EffectCharacter, 0),
		sparkIndex:             0,
		etchSpeed:              1,
		etchDelay:              1,
		coolGradientStops:      coolGradientStops,
		laserGradientStops:     laserGradientStops,
		sparkGradientStops:     sparkGradientStops,
		sparkCoolingFrames:     7,
		finalGradientStops:     finalGradientStops,
		finalGradientSteps:     finalGradientSteps,
		finalGradientFrames:    4,
		finalGradientDirection: finalGradientDirection,
	}

	l.build()
	l.initLaser()
	l.initSparks()
	return l
}

func (l *LaserEtch) build() {
	// Build final gradient mapping
	finalGradient, _ := utils.NewGradient(l.finalGradientStops, l.finalGradientSteps, false)
	finalGradientMapping, _ := finalGradient.BuildCoordinateColorMapping(
		l.base.Canvas.TextBottom,
		l.base.Canvas.TextTop,
		l.base.Canvas.TextLeft,
		l.base.Canvas.TextRight,
		l.finalGradientDirection,
	)

	for _, character := range l.base.Terminal.GetCharacters(true, false, false, false, engine.TopToBottomLeftToRight) {
		charFinalColor := finalGradientMapping[character.InputCoord]
		l.characterFinalColor[character] = charFinalColor

		// Create cool gradient from hot to final color
		coolColors := append(l.coolGradientStops, charFinalColor)
		coolGradient, _ := utils.NewGradient(coolColors, []int{8}, false)

		// Spawn scene - start with ^ then cool down to final
		spawnScene := character.Animation.NewScene("spawn")
		hotColor := mustColors("ffe680")[0]
		_ = spawnScene.AddFrame("^", 3, &utils.ColorPair{FG: &hotColor})
		for _, color := range coolGradient.Spectrum {
			colorCopy := color
			_ = spawnScene.AddFrame(character.Symbol, 3, &utils.ColorPair{FG: &colorCopy})
		}
	}

	// Build pending chars using recursive backtracker pattern
	l.pendingChars = l.runRecursiveBacktracker()
}

func (l *LaserEtch) runRecursiveBacktracker() []*engine.EffectCharacter {
	// Get all characters with their neighbors already built
	allChars := l.base.Terminal.GetCharacters(true, true, true, false, engine.TopToBottomLeftToRight)
	if len(allChars) == 0 {
		return nil
	}

	visited := make(map[*engine.EffectCharacter]bool)
	result := make([]*engine.EffectCharacter, 0, len(allChars))
	stack := make([]*engine.EffectCharacter, 0)

	// Start from a random character
	startIdx := utils.RandIntn(len(allChars))
	current := allChars[startIdx]
	visited[current] = true
	result = append(result, current)
	stack = append(stack, current)

	for len(stack) > 0 {
		// Get unvisited neighbors
		unvisitedNeighbors := make([]*engine.EffectCharacter, 0)
		for _, neighbor := range current.Neighbors {
			if neighbor != nil && !visited[neighbor] {
				unvisitedNeighbors = append(unvisitedNeighbors, neighbor)
			}
		}

		if len(unvisitedNeighbors) > 0 {
			// Choose random unvisited neighbor
			nextIdx := utils.RandIntn(len(unvisitedNeighbors))
			next := unvisitedNeighbors[nextIdx]
			visited[next] = true
			result = append(result, next)
			stack = append(stack, next)
			current = next
		} else {
			// Backtrack
			stack = stack[:len(stack)-1]
			if len(stack) > 0 {
				current = stack[len(stack)-1]
			}
		}
	}

	return result
}

func (l *LaserEtch) initLaser() {
	// Create laser beam characters from bottom-left going diagonally up-right
	laserGradient, _ := utils.NewGradient(l.laserGradientStops, []int{6}, true)
	gradientColors := laserGradient.Spectrum

	row := 0
	col := 0
	colorIdx := 0
	for row <= l.base.Canvas.Top {
		symbol := "*"
		if len(l.laserBeamChars) > 0 {
			symbol = "/"
		}
		char := engine.NewEffectCharacter(symbol, utils.Coord{Col: col, Row: row})
		char.Layer = 2
		char.IsAddedCharacter = true
		l.base.Terminal.AddCharacter(char)
		l.base.Terminal.SetCharacterVisibility(char, true)

		// Looping laser scene
		laserScene := char.Animation.NewScene("laser")
		laserScene.IsLooping = true
		for i := 0; i < len(gradientColors); i++ {
			idx := (colorIdx + i) % len(gradientColors)
			colorCopy := gradientColors[idx]
			_ = laserScene.AddFrame(char.Symbol, 3, &utils.ColorPair{FG: &colorCopy})
		}
		char.Animation.ActivateSceneRef(laserScene)

		l.laserBeamChars = append(l.laserBeamChars, char)
		l.activeCharacters[char] = struct{}{}

		row++
		col++
		colorIdx = (colorIdx + 1) % len(gradientColors)
	}
}

func (l *LaserEtch) initSparks() {
	// Create spark gradient
	sparkGradient, _ := utils.NewGradient(l.sparkGradientStops, []int{3, 8}, false)
	l.sparkColors = sparkGradient.Spectrum

	// Pre-create spark pool
	sparkSymbols := []string{".", ",", "*"}
	for i := 0; i < 500; i++ {
		symbol := sparkSymbols[utils.RandIntn(len(sparkSymbols))]
		char := engine.NewEffectCharacter(symbol, utils.Coord{Col: 0, Row: 0})
		char.Layer = 2
		char.Visible = false
		char.IsAddedCharacter = true
		l.base.Terminal.AddCharacter(char)

		// Spark cooling scene
		sparkScene := char.Animation.NewScene("spark")
		for _, color := range l.sparkColors {
			colorCopy := color
			_ = sparkScene.AddFrame(char.Symbol, l.sparkCoolingFrames, &utils.ColorPair{FG: &colorCopy})
		}

		// Register callback to hide spark when scene completes
		terminal := l.base.Terminal
		char.EventHandler.RegisterEvent(
			engine.EventSceneComplete,
			sparkScene,
			engine.ActionCallback,
			engine.Callback{
				Fn: func(c *engine.EffectCharacter, args ...any) {
					if t, ok := args[0].(*engine.TerminalState); ok {
						t.SetCharacterVisibility(c, false)
					}
				},
				Args: []any{terminal},
			},
		)

		l.sparks = append(l.sparks, char)
	}
}

func (l *LaserEtch) repositionLaser(target utils.Coord) {
	l.laserPosition = target
	row := target.Row
	col := target.Col
	for _, char := range l.laserBeamChars {
		char.Motion.SetCoordinate(utils.Coord{Col: col, Row: row})
		char.Coord = utils.Coord{Col: col, Row: row}
		row++
		col++
	}
	l.emitSparks(1)
}

func (l *LaserEtch) emitSparks(count int) {
	for i := 0; i < count; i++ {
		spark := l.sparks[l.sparkIndex]
		l.sparkIndex = (l.sparkIndex + 1) % len(l.sparks)

		spark.Motion.SetCoordinate(l.laserPosition)
		spark.Coord = l.laserPosition

		// Reset animation
		if spark.Animation.ActiveScene != nil {
			spark.Animation.ActiveScene.ResetScene()
		}
		spark.Animation.ActivateScene("spark")

		// Create falling path
		fallTargetCol := l.laserPosition.Col + utils.RandIntn(41) - 20 // -20 to +20
		fallPath, _ := spark.Motion.NewPath(0.3, utils.OutSine, nil, 0, false, "fall")
		fallPath.AddBezierWaypoint(
			utils.Coord{Col: fallTargetCol, Row: l.base.Canvas.Bottom},
			utils.Coord{Col: fallTargetCol, Row: l.laserPosition.Row + utils.RandIntn(31) - 10},
		)
		spark.Motion.ActivatePath(fallPath)

		l.base.Terminal.SetCharacterVisibility(spark, true)
		l.activeCharacters[spark] = struct{}{}
	}
}

func (l *LaserEtch) disableLaser() {
	for _, char := range l.laserBeamChars {
		l.base.Terminal.SetCharacterVisibility(char, false)
		delete(l.activeCharacters, char)
	}
}

func (l *LaserEtch) Next() (string, bool) {
	if len(l.pendingChars) > 0 || len(l.activeCharacters) > 0 {
		if l.charDelay == 0 {
			for i := 0; i < l.etchSpeed; i++ {
				if len(l.pendingChars) == 0 {
					break
				}
				nextChar := l.pendingChars[0]
				l.pendingChars = l.pendingChars[1:]

				// Skip spaces
				for nextChar.Symbol == " " && len(l.pendingChars) > 0 {
					nextChar = l.pendingChars[0]
					l.pendingChars = l.pendingChars[1:]
				}

				l.base.Terminal.SetCharacterVisibility(nextChar, true)
				nextChar.Animation.ActivateScene("spawn")
				l.activeCharacters[nextChar] = struct{}{}
				l.repositionLaser(nextChar.InputCoord)
			}
			l.charDelay = l.etchDelay
		} else {
			l.charDelay--
		}

		// Keep laser active while there are pending chars
		if len(l.pendingChars) > 0 {
			for _, char := range l.laserBeamChars {
				l.activeCharacters[char] = struct{}{}
			}
		} else {
			l.disableLaser()
		}

		// Tick all active characters
		for char := range l.activeCharacters {
			char.Tick()
			if !char.IsActive() {
				delete(l.activeCharacters, char)
			}
		}

		return l.base.Terminal.GetFormattedOutputString(), true
	}

	return l.base.Terminal.GetFormattedOutputString(), false
}

func (l *LaserEtch) CanvasHeight() int {
	return l.base.CanvasHeight()
}
