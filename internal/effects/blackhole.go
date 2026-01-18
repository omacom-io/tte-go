package effects

import (
	"math"

	"tte-go/internal/config"
	"tte-go/internal/engine"
	"tte-go/internal/utils"
)

// Blackhole creates a blackhole in a starfield, consumes the stars, then explodes
// the input data back into position.
//
// Phases:
//   - forming: Blackhole ring characters fly to their positions one by one
//   - consuming: Ring rotates while star characters are pulled to center and fade
//   - collapsing: Ring expands then collapses to center with unstable animation
//   - exploding: All characters explode outward to nearby positions, then settle to input coords
//   - complete: Effect finished
type Blackhole struct {
	base *BaseEffect

	// Character groups
	blackholeChars           []*engine.EffectCharacter
	awaitingConsumptionChars []*engine.EffectCharacter
	activeCharacters         map[*engine.EffectCharacter]struct{}

	// Config
	blackholeRadius      int
	blackholeColor       utils.Color
	starColors           []utils.Color
	finalGradientMapping map[utils.Coord]utils.Color

	// Phase tracking
	phase                  string
	awaitingBlackholeChars []*engine.EffectCharacter
	formationDelay         int
	formationCounter       int
}

func NewBlackhole(input string, cfg config.TerminalConfig) engine.Effect {
	base := NewBaseEffect(input, cfg)

	// Configuration
	blackholeColor := mustColors("#ffffff")[0]
	starColors := mustColors("#ffcc0d", "#ff7326", "#ff194d", "#bf2669", "#702a8c", "#049dbf")

	// Build final gradient
	finalGradientStops := mustColors("#8A008A", "#00D1FF", "#ffffff")
	finalGradient, _ := utils.NewGradient(finalGradientStops, []int{9}, false)
	finalMapping, _ := finalGradient.BuildCoordinateColorMapping(
		base.Canvas.TextBottom, base.Canvas.TextTop,
		base.Canvas.TextLeft, base.Canvas.TextRight,
		utils.GradientDiagonal,
	)

	// Calculate blackhole radius
	blackholeRadius := max(
		min(
			int(math.Round(float64(base.Canvas.Width)*0.3)),
			int(math.Round(float64(base.Canvas.Height)*0.20)),
		),
		3,
	)

	b := &Blackhole{
		base:                     base,
		blackholeChars:           make([]*engine.EffectCharacter, 0),
		awaitingConsumptionChars: make([]*engine.EffectCharacter, 0),
		activeCharacters:         make(map[*engine.EffectCharacter]struct{}),
		blackholeRadius:          blackholeRadius,
		blackholeColor:           blackholeColor,
		starColors:               starColors,
		finalGradientMapping:     finalMapping,
		phase:                    "forming",
	}

	b.prepareBlackhole()
	return b
}

func (b *Blackhole) prepareBlackhole() {
	starSymbols := []string{"*", "'", "`", "¤", "•", "°", "·"}

	// Starfield color gradient (dim gray to white)
	starfieldColors := mustColors("#4a4a4d", "#666666", "#888888", "#aaaaaa", "#cccccc", "#ffffff")

	// Build fade-to-black gradients for each starfield color
	gradientMap := make(map[utils.Color]*utils.Gradient)
	for _, color := range starfieldColors {
		fadeGradient, _ := utils.NewGradient([]utils.Color{color, mustColors("#000000")[0]}, []int{10}, false)
		gradientMap[color] = fadeGradient
	}

	center := b.base.Canvas.Center

	// Select random characters to form the blackhole ring
	availableChars := make([]*engine.EffectCharacter, len(b.base.Characters))
	copy(availableChars, b.base.Characters)
	shuffleChars(availableChars)

	blackholeCount := min(b.blackholeRadius*3, len(availableChars))
	b.blackholeChars = availableChars[:blackholeCount]

	// Calculate ring positions for blackhole characters
	ringPositions := findCoordsOnCircle(center, b.blackholeRadius, len(b.blackholeChars))

	// Setup blackhole characters
	for posIndex, character := range b.blackholeChars {
		startingPos := ringPositions[posIndex]

		// Create path to fly to ring position
		blackholePath, _ := character.Motion.NewPath(0.7, utils.InOutSine, nil, 0, false, "blackhole")
		blackholePath.AddWaypoint(startingPos)

		// Create blackhole appearance scene
		blackholeScene := character.Animation.NewScene("blackhole")
		bhColor := b.blackholeColor
		_ = blackholeScene.AddFrame("*", 1, &utils.ColorPair{FG: &bhColor})

		// Register event to set layer when path activates
		layer := 1
		_ = character.EventHandler.RegisterEvent(
			engine.EventPathActivated,
			blackholePath,
			engine.ActionSetLayer,
			layer,
		)

		// Create rotation path (loops through all ring positions starting from this one)
		rotationPath, _ := character.Motion.NewPath(0.45, nil, nil, 0, true, "blackhole_rotation")
		for i := 0; i < len(ringPositions); i++ {
			idx := (posIndex + i) % len(ringPositions)
			rotationPath.AddWaypoint(ringPositions[idx])
		}
	}

	// Setup all characters with star appearance and visibility
	for _, character := range b.base.Terminal.GetCharacters(true, false, false, false, engine.TopToBottomLeftToRight) {
		b.base.Terminal.SetCharacterVisibility(character, true)

		// Random star symbol and color
		starSymbol := starSymbols[utils.RandIntn(len(starSymbols))]
		starColor := starfieldColors[utils.RandIntn(len(starfieldColors))]

		// Create starting scene with star appearance
		startingScene := character.Animation.NewScene("starting")
		starColorCopy := starColor
		_ = startingScene.AddFrame(starSymbol, 1, &utils.ColorPair{FG: &starColorCopy})
		character.Animation.ActivateScene("starting")

		// Non-blackhole characters get starfield treatment
		if !containsChar(b.blackholeChars, character) {
			// Random starfield position
			starfieldCoord := utils.Coord{
				Row: utils.RandIntn(b.base.Canvas.Height) + 1,
				Col: utils.RandIntn(b.base.Canvas.Width) + 1,
			}
			character.Motion.SetCoordinate(starfieldCoord)
			character.Coord = starfieldCoord

			// Create path to singularity (center)
			speed := 0.17 + utils.RandFloat64()*0.13 // 0.17 to 0.30
			singularityPath, _ := character.Motion.NewPath(speed, utils.InExpo, nil, 0, false, "singularity")
			singularityPath.AddWaypoint(center)

			// Create consumed scene (fade to black then disappear)
			consumedScene := character.Animation.NewScene("consumed")
			if fadeGradient, ok := gradientMap[starColor]; ok && fadeGradient != nil {
				for _, color := range fadeGradient.Spectrum {
					colorCopy := color
					_ = consumedScene.AddFrame(starSymbol, 1, &utils.ColorPair{FG: &colorCopy})
				}
			}
			_ = consumedScene.AddFrame(" ", 1, nil)
			// Note: Python uses SyncMetric.DISTANCE to sync animation with motion
			// This is not yet implemented in Go engine

			// Register events for singularity path
			layer := 2
			_ = character.EventHandler.RegisterEvent(
				engine.EventPathActivated,
				singularityPath,
				engine.ActionSetLayer,
				layer,
			)
			_ = character.EventHandler.RegisterEvent(
				engine.EventPathActivated,
				singularityPath,
				engine.ActionActivateScene,
				consumedScene,
			)

			b.awaitingConsumptionChars = append(b.awaitingConsumptionChars, character)
		}
	}

	// Shuffle consumption order
	shuffleChars(b.awaitingConsumptionChars)

	// Setup formation tracking
	b.formationDelay = max(100/len(b.blackholeChars), 6)
	b.formationCounter = b.formationDelay
	b.awaitingBlackholeChars = make([]*engine.EffectCharacter, len(b.blackholeChars))
	copy(b.awaitingBlackholeChars, b.blackholeChars)
}

func (b *Blackhole) rotateBlackhole() {
	for _, character := range b.blackholeChars {
		if path, err := character.Motion.QueryPath("blackhole_rotation"); err == nil {
			character.Motion.ActivatePath(path)
			b.activeCharacters[character] = struct{}{}
		}
	}
}

func (b *Blackhole) collapseBlackhole() {
	center := b.base.Canvas.Center

	// Expanded ring positions
	expandedRingPositions := findCoordsOnCircle(center, b.blackholeRadius+3, len(b.blackholeChars))

	unstableSymbols := []string{"◦", "◎", "◉", "●", "◉", "◎", "◦"}
	pointCharMade := false

	for i, character := range b.blackholeChars {
		nextPos := expandedRingPositions[i]

		// Create expand path
		expandPath, _ := character.Motion.NewPath(0.2, utils.InExpo, nil, 0, false, "expand")
		expandPath.AddWaypoint(nextPos)

		// Create collapse path
		collapsePath, _ := character.Motion.NewPath(0.3, utils.InExpo, nil, 0, false, "collapse")
		collapsePath.AddWaypoint(center)

		// Register event: when expand completes, activate collapse
		_ = character.EventHandler.RegisterEvent(
			engine.EventPathComplete,
			expandPath,
			engine.ActionActivatePath,
			collapsePath,
		)

		// First character gets the unstable point animation
		if !pointCharMade {
			pointScene := character.Animation.NewScene("point")
			for rep := 0; rep < 3; rep++ {
				for _, symbol := range unstableSymbols {
					randColor := b.starColors[utils.RandIntn(len(b.starColors))]
					_ = pointScene.AddFrame(symbol, 3, &utils.ColorPair{FG: &randColor})
				}
			}

			_ = character.EventHandler.RegisterEvent(
				engine.EventPathComplete,
				collapsePath,
				engine.ActionActivateScene,
				pointScene,
			)
			layer := 3
			_ = character.EventHandler.RegisterEvent(
				engine.EventPathComplete,
				collapsePath,
				engine.ActionSetLayer,
				layer,
			)
			pointCharMade = true
		}

		character.Motion.ActivatePath(expandPath)
		b.activeCharacters[character] = struct{}{}
	}
}

func (b *Blackhole) explodeSingularity() {
	center := b.base.Canvas.Center

	for _, character := range b.base.Terminal.GetCharacters(true, false, false, false, engine.TopToBottomLeftToRight) {
		// Find a nearby coord (explosion destination)
		nearbyCoords := findCoordsOnCircle(character.InputCoord, 3, 5)
		nearbyCoord := nearbyCoords[utils.RandIntn(len(nearbyCoords))]

		// Create path to nearby position (fast, out_expo)
		speed := float64(utils.RandIntn(2)+3) / 10.0 // 0.3 to 0.4
		nearbyPath, _ := character.Motion.NewPath(speed, utils.OutExpo, nil, 0, false, "nearby")
		nearbyPath.AddWaypoint(nearbyCoord)

		// Create path to input position (slow, in_cubic)
		inputSpeed := float64(utils.RandIntn(3)+4) / 100.0 // 0.04 to 0.06
		inputPath, _ := character.Motion.NewPath(inputSpeed, utils.InCubic, nil, 0, false, "input")
		inputPath.AddWaypoint(character.InputCoord)

		// Explode scene (star color)
		explodeScene := character.Animation.NewScene("explode")
		explodeStarColor := b.starColors[utils.RandIntn(len(b.starColors))]
		_ = explodeScene.AddFrame(character.Symbol, 1, &utils.ColorPair{FG: &explodeStarColor})

		// Cooling scene (transition from star color to final color)
		coolingScene := character.Animation.NewScene("cooling")
		finalColor := b.finalGradientMapping[character.InputCoord]
		coolingGradient, _ := utils.NewGradient([]utils.Color{explodeStarColor, finalColor}, []int{10}, false)
		if coolingGradient != nil {
			_ = coolingScene.ApplyGradientToSymbols(
				[]string{character.Symbol},
				20,
				coolingGradient,
				nil,
			)
		}

		// Register events
		_ = character.EventHandler.RegisterEvent(
			engine.EventPathComplete,
			nearbyPath,
			engine.ActionActivatePath,
			inputPath,
		)
		_ = character.EventHandler.RegisterEvent(
			engine.EventPathComplete,
			nearbyPath,
			engine.ActionActivateScene,
			coolingScene,
		)

		// Start from center, activate explode scene and nearby path
		character.Motion.SetCoordinate(center)
		character.Coord = center
		character.Animation.ActivateSceneRef(explodeScene)
		character.Motion.ActivatePath(nearbyPath)
		b.activeCharacters[character] = struct{}{}
	}
}

func (b *Blackhole) Next() (string, bool) {
	if len(b.activeCharacters) == 0 && b.phase == "complete" {
		return b.base.Terminal.GetFormattedOutputString(), false
	}

	switch b.phase {
	case "forming":
		if len(b.awaitingBlackholeChars) > 0 {
			b.formationCounter--
			if b.formationCounter <= 0 {
				nextChar := b.awaitingBlackholeChars[0]
				b.awaitingBlackholeChars = b.awaitingBlackholeChars[1:]

				if path, err := nextChar.Motion.QueryPath("blackhole"); err == nil {
					nextChar.Motion.ActivatePath(path)
				}
				nextChar.Animation.ActivateScene("blackhole")
				b.activeCharacters[nextChar] = struct{}{}
				b.formationCounter = b.formationDelay
			}
		} else if len(b.activeCharacters) == 0 {
			b.rotateBlackhole()
			b.phase = "consuming"
		}

	case "consuming":
		if len(b.awaitingConsumptionChars) > 0 {
			// Activate all consumption paths at once
			for _, char := range b.awaitingConsumptionChars {
				if path, err := char.Motion.QueryPath("singularity"); err == nil {
					char.Motion.ActivatePath(path)
				}
				b.activeCharacters[char] = struct{}{}
			}
			b.awaitingConsumptionChars = nil
		} else {
			// Check if all non-blackhole chars have finished (consumed)
			// Blackhole chars are always "active" due to looping rotation, so we
			// check if there are any non-blackhole chars still active
			hasNonBlackhole := false
			for char := range b.activeCharacters {
				if !containsChar(b.blackholeChars, char) {
					hasNonBlackhole = true
					break
				}
			}
			if !hasNonBlackhole {
				b.phase = "collapsing"
			}
		}

	case "collapsing":
		b.collapseBlackhole()
		b.phase = "exploding"

	case "exploding":
		// Wait for all blackhole characters to finish their collapse animation
		allBlackholesDone := true
		for _, char := range b.blackholeChars {
			hasPath := char.Motion.ActivePath != nil
			hasScene := char.Animation.ActiveScene != nil
			if hasPath || hasScene {
				allBlackholesDone = false
				break
			}
		}
		if allBlackholesDone {
			b.explodeSingularity()
			b.phase = "complete"
		}
	}

	// Tick all active characters
	for char := range b.activeCharacters {
		char.Tick()
		if !char.IsActive() {
			delete(b.activeCharacters, char)
		}
	}

	return b.base.Terminal.GetFormattedOutputString(), true
}

func (b *Blackhole) CanvasHeight() int {
	return b.base.CanvasHeight()
}

// Helper functions

func shuffleChars(chars []*engine.EffectCharacter) {
	for i := len(chars) - 1; i > 0; i-- {
		j := utils.RandIntn(i + 1)
		chars[i], chars[j] = chars[j], chars[i]
	}
}

func containsChar(chars []*engine.EffectCharacter, char *engine.EffectCharacter) bool {
	for _, c := range chars {
		if c == char {
			return true
		}
	}
	return false
}

// findCoordsOnCircle finds points on a circle, matching Python's geometry.find_coords_on_circle
func findCoordsOnCircle(center utils.Coord, radius, count int) []utils.Coord {
	if radius == 0 || count == 0 {
		return nil
	}

	coords := make([]utils.Coord, 0, count)
	seen := make(map[utils.Coord]bool)

	angleStep := 2 * math.Pi / float64(count)
	for i := 0; i < count; i++ {
		angle := angleStep * float64(i)
		x := float64(center.Col) + float64(radius)*math.Cos(angle)
		// Correct for terminal character height/width ratio by doubling x distance from origin
		xDiff := x - float64(center.Col)
		x += xDiff
		y := float64(center.Row) + float64(radius)*math.Sin(angle)

		coord := utils.Coord{Col: int(math.Round(x)), Row: int(math.Round(y))}
		if !seen[coord] {
			coords = append(coords, coord)
			seen[coord] = true
		}
	}

	return coords
}
