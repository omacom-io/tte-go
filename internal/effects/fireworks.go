package effects

import (
	"math"

	"tte-go/internal/config"
	"tte-go/internal/engine"
	"tte-go/internal/utils"
)

// Fireworks launches characters up the screen where they explode like fireworks and fall into place.
type Fireworks struct {
	base *BaseEffect

	// Character groups
	shells           [][]*engine.EffectCharacter
	activeCharacters map[*engine.EffectCharacter]struct{}

	// Config
	fireworkColors  []utils.Color
	fireworkSymbol  string
	fireworkVolume  int
	explodeDistance int
	launchDelay     int
	launchCounter   int

	// Final gradient mapping
	finalGradientMapping map[utils.Coord]utils.Color
}

func NewFireworks(input string, cfg config.TerminalConfig) engine.Effect {
	base := NewBaseEffect(input, cfg)

	// Configuration
	fireworkColors := mustColors("#88F7E2", "#44D492", "#F5EB67", "#FFA15C", "#FA233E")
	fireworkSymbol := "o"
	fireworkVolume := max(1, int(float64(len(base.Characters))*0.05)) // 5% of chars per shell
	explodeDistance := max(1, min(15, int(float64(base.Canvas.Width)*0.2)))
	launchDelay := 45

	// Build final gradient
	finalGradientStops := mustColors("#8A008A", "#00D1FF", "#FFFFFF")
	finalGradient, _ := utils.NewGradient(finalGradientStops, []int{12}, false)
	finalMapping, _ := finalGradient.BuildCoordinateColorMapping(
		base.Canvas.TextBottom, base.Canvas.TextTop,
		base.Canvas.TextLeft, base.Canvas.TextRight,
		utils.GradientHorizontal,
	)

	f := &Fireworks{
		base:                 base,
		shells:               make([][]*engine.EffectCharacter, 0),
		activeCharacters:     make(map[*engine.EffectCharacter]struct{}),
		fireworkColors:       fireworkColors,
		fireworkSymbol:       fireworkSymbol,
		fireworkVolume:       fireworkVolume,
		explodeDistance:      explodeDistance,
		launchDelay:          launchDelay,
		finalGradientMapping: finalMapping,
	}

	f.prepareWaypoints()
	f.prepareScenes()

	return f
}

func (f *Fireworks) prepareWaypoints() {
	var currentShell []*engine.EffectCharacter
	var originX, originY int
	var originCoord utils.Coord

	for _, character := range f.base.Characters {
		// Start a new shell when needed
		if len(currentShell) == f.fireworkVolume || len(currentShell) == 0 {
			if len(currentShell) > 0 {
				f.shells = append(f.shells, currentShell)
			}
			currentShell = make([]*engine.EffectCharacter, 0, f.fireworkVolume)

			// Random origin x position
			originX = utils.RandIntn(f.base.Canvas.Width) + 1

			// Origin y is random between character's input row and canvas top
			minRow := character.InputCoord.Row
			if minRow < f.base.Canvas.Bottom {
				minRow = f.base.Canvas.Bottom
			}
			originY = minRow + utils.RandIntn(f.base.Canvas.Top-minRow+1)
			originCoord = utils.Coord{Col: originX, Row: originY}
		}

		// Set starting position at bottom of screen
		startCoord := utils.Coord{Col: originX, Row: f.base.Canvas.Bottom}
		character.Motion.SetCoordinate(startCoord)
		character.Coord = startCoord

		// Create apex path (launch upward)
		layer2 := 2
		apexPath, _ := character.Motion.NewPath(0.35, utils.OutExpo, &layer2, 0, false, "apex_pth")
		apexPath.AddWaypoint(originCoord)

		// Find random explode waypoint within circle around origin
		explodeWaypointCoords := findCoordsInCircle(originCoord, f.explodeDistance)
		explodeWaypoint := explodeWaypointCoords[utils.RandIntn(len(explodeWaypointCoords))]

		// Create explode path
		explodeSpeed := 0.2 + utils.RandFloat64()*0.2 // 0.2 to 0.4
		explodePath, _ := character.Motion.NewPath(explodeSpeed, utils.OutCirc, &layer2, 0, false, "explode_pth")
		explodePath.AddWaypoint(explodeWaypoint)

		// Calculate bloom control point (extrapolate along ray from origin to explode waypoint)
		bloomControlPoint := extrapolateAlongRay(originCoord, explodeWaypoint, f.explodeDistance/2)

		// Bloom waypoint is above the control point
		bloomWaypoint := utils.Coord{
			Col: bloomControlPoint.Col,
			Row: max(1, bloomControlPoint.Row-7),
		}
		explodePath.AddBezierWaypoint(bloomWaypoint, bloomControlPoint)

		// Create input path (fall to final position)
		inputPath, _ := character.Motion.NewPath(0.6, utils.InOutQuart, &layer2, 0, false, "input_pth")
		inputControlPoint := utils.Coord{Col: bloomWaypoint.Col, Row: 1}
		inputPath.AddBezierWaypoint(character.InputCoord, inputControlPoint)

		// Register path events
		// apex complete -> activate explode
		_ = character.EventHandler.RegisterEvent(
			engine.EventPathComplete,
			apexPath,
			engine.ActionActivatePath,
			explodePath,
		)
		// explode complete -> activate input
		_ = character.EventHandler.RegisterEvent(
			engine.EventPathComplete,
			explodePath,
			engine.ActionActivatePath,
			inputPath,
		)
		// input complete -> set layer to 0
		_ = character.EventHandler.RegisterEvent(
			engine.EventPathComplete,
			inputPath,
			engine.ActionSetLayer,
			0,
		)

		// Activate apex path (will start when shell is launched)
		character.Motion.ActivatePath(apexPath)

		currentShell = append(currentShell, character)
	}

	// Don't forget the last shell
	if len(currentShell) > 0 {
		f.shells = append(f.shells, currentShell)
	}
}

func (f *Fireworks) prepareScenes() {
	white := mustColors("#FFFFFF")[0]

	for shellIndex, shell := range f.shells {
		// Pick a random shell color
		shellColor := f.fireworkColors[utils.RandIntn(len(f.fireworkColors))]

		// Create shell gradient (color -> white -> color)
		shellGradient, _ := utils.NewGradient([]utils.Color{shellColor, white, shellColor}, []int{5}, false)

		for _, character := range shell {
			finalColor := f.finalGradientMapping[character.InputCoord]

			// Launch scene - firework symbol flashing between shell color and white
			launchScene := character.Animation.NewScene("launch_scn")
			launchScene.IsLooping = true
			shellColorCopy := shellColor
			_ = launchScene.AddFrame(f.fireworkSymbol, 2, &utils.ColorPair{FG: &shellColorCopy})
			whiteCopy := white
			_ = launchScene.AddFrame(f.fireworkSymbol, 1, &utils.ColorPair{FG: &whiteCopy})

			// Bloom scene - character symbol with shell gradient
			bloomScene := character.Animation.NewScene("bloom_scn")
			if shellGradient != nil {
				for _, color := range shellGradient.Spectrum {
					colorCopy := color
					_ = bloomScene.AddFrame(character.Symbol, 2, &utils.ColorPair{FG: &colorCopy})
				}
			}

			// Fall scene - gradient from shell color to final color
			fallScene := character.Animation.NewScene("fall_scn")
			fallGradient, _ := utils.NewGradient([]utils.Color{shellColor, finalColor}, []int{15}, false)
			if fallGradient != nil {
				_ = fallScene.ApplyGradientToSymbols(
					[]string{character.Symbol},
					10,
					fallGradient,
					nil,
				)
			}

			// Register scene events
			// When apex path completes, activate bloom scene
			if apexPath, err := character.Motion.QueryPath("apex_pth"); err == nil {
				_ = character.EventHandler.RegisterEvent(
					engine.EventPathComplete,
					apexPath,
					engine.ActionActivateScene,
					bloomScene,
				)
			}

			// When input path is activated, activate fall scene
			if inputPath, err := character.Motion.QueryPath("input_pth"); err == nil {
				_ = character.EventHandler.RegisterEvent(
					engine.EventPathActivated,
					inputPath,
					engine.ActionActivateScene,
					fallScene,
				)
			}

			// Activate launch scene initially (but character is not visible yet)
			character.Animation.ActivateSceneRef(launchScene)

			// Set character invisible initially (will be made visible when shell launches)
			character.Visible = false

			// Store shell index for later reference
			_ = shellIndex
		}
	}
}

func (f *Fireworks) Next() (string, bool) {
	// Launch new shells periodically
	if len(f.shells) > 0 {
		f.launchCounter--
		if f.launchCounter <= 0 {
			// Pop the last shell
			shell := f.shells[len(f.shells)-1]
			f.shells = f.shells[:len(f.shells)-1]

			// Make all characters in shell visible and active
			for _, character := range shell {
				f.base.Terminal.SetCharacterVisibility(character, true)
				f.activeCharacters[character] = struct{}{}
			}

			// Randomize next launch delay (+/- 50%)
			f.launchCounter = int(float64(f.launchDelay) * (0.5 + utils.RandFloat64()))
		}
	}

	// Check if we're done
	if len(f.shells) == 0 && len(f.activeCharacters) == 0 {
		return f.base.Terminal.GetFormattedOutputString(), false
	}

	// Tick all active characters
	for char := range f.activeCharacters {
		char.Tick()
		if !char.IsActive() {
			delete(f.activeCharacters, char)
		}
	}

	return f.base.Terminal.GetFormattedOutputString(), true
}

func (f *Fireworks) CanvasHeight() int {
	return f.base.CanvasHeight()
}

func (f *Fireworks) CanvasWidth() int {
	return f.base.CanvasWidth()
}

// findCoordsInCircle finds coordinates within a circle (actually an ellipse to account for
// terminal character aspect ratio). Matches Python's geometry.find_coords_in_circle.
func findCoordsInCircle(center utils.Coord, diameter int) []utils.Coord {
	if diameter <= 0 {
		return []utils.Coord{center}
	}

	h, k := center.Col, center.Row
	coords := make([]utils.Coord, 0)

	// The actual shape is an ellipse with major axis = diameter (horizontal)
	// and minor axis = diameter/2 (vertical) to account for terminal char aspect ratio
	aSquared := float64(diameter * diameter)
	bSquared := float64(diameter*diameter) / 4.0

	for x := h - diameter; x <= h+diameter; x++ {
		xComponent := float64((x-h)*(x-h)) / aSquared
		maxYOffset := int(math.Sqrt(bSquared * (1 - xComponent)))
		for y := k - maxYOffset; y <= k+maxYOffset; y++ {
			coords = append(coords, utils.Coord{Col: x, Row: y})
		}
	}

	if len(coords) == 0 {
		coords = append(coords, center)
	}

	return coords
}

// extrapolateAlongRay returns the point that is offsetFromTarget units past the target
// along the ray from origin to target. Matches Python's geometry.extrapolate_along_ray.
func extrapolateAlongRay(origin, target utils.Coord, offsetFromTarget int) utils.Coord {
	lineLength := findLengthOfLine(origin, target)
	totalDistance := lineLength + float64(offsetFromTarget)

	if totalDistance == 0 || origin == target {
		return target
	}

	t := totalDistance / lineLength
	nextCol := (1-t)*float64(origin.Col) + t*float64(target.Col)
	nextRow := (1-t)*float64(origin.Row) + t*float64(target.Row)

	return utils.Coord{
		Col: int(math.Round(nextCol)),
		Row: int(math.Round(nextRow)),
	}
}

// findLengthOfLine returns the length of the line between two coordinates.
func findLengthOfLine(coord1, coord2 utils.Coord) float64 {
	colDiff := float64(coord2.Col - coord1.Col)
	rowDiff := float64(coord2.Row - coord1.Row)
	return math.Hypot(colDiff, rowDiff)
}
