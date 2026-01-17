package effects

import (
	"tte-go/internal/config"
	"tte-go/internal/engine"
	"tte-go/internal/utils"
)

// Unstable spawns characters jumbled, explodes them to the edge of the canvas, then reassembles them.
type Unstable struct {
	base                 *BaseEffect
	finalGradientMapping map[utils.Coord]utils.Color
	characterFinalColor  map[*engine.EffectCharacter]utils.Color
	jumbledCoords        map[*engine.EffectCharacter]utils.Coord
	activeCharacters     map[*engine.EffectCharacter]struct{}
	phase                string
	rumbleSteps          int
	maxRumbleSteps       int
	rumbleModDelay       int
	explosionHoldTime    int
	unstableColor        utils.Color
}

func NewUnstable(input string, cfg config.TerminalConfig) engine.Effect {
	base := NewBaseEffect(input, cfg)

	// Configuration
	unstableColors := mustColors("#ff9200")
	unstableColor := unstableColors[0]

	// Build final gradient
	finalGradientStops := mustColors("#8A008A", "#00D1FF", "#FFFFFF")
	finalGradient, _ := utils.NewGradient(finalGradientStops, []int{12}, false)
	finalMapping, _ := finalGradient.BuildCoordinateColorMapping(
		base.Canvas.TextBottom, base.Canvas.TextTop,
		base.Canvas.TextLeft, base.Canvas.TextRight,
		utils.GradientVertical,
	)

	u := &Unstable{
		base:                 base,
		finalGradientMapping: finalMapping,
		characterFinalColor:  make(map[*engine.EffectCharacter]utils.Color),
		jumbledCoords:        make(map[*engine.EffectCharacter]utils.Coord),
		activeCharacters:     make(map[*engine.EffectCharacter]struct{}),
		phase:                "rumble",
		maxRumbleSteps:       150,
		rumbleModDelay:       18,
		explosionHoldTime:    30,
		unstableColor:        unstableColor,
	}

	// Store final colors for each character
	for _, char := range base.Characters {
		u.characterFinalColor[char] = finalMapping[char.InputCoord]
	}

	// Create list of all input coords for jumbling
	coords := make([]utils.Coord, 0, len(base.Characters))
	for _, char := range base.Characters {
		coords = append(coords, char.InputCoord)
	}

	// Shuffle coords to create jumbled positions
	for i := len(coords) - 1; i > 0; i-- {
		j := utils.RandIntn(i + 1)
		coords[i], coords[j] = coords[j], coords[i]
	}

	// Setup each character
	for i, char := range base.Characters {
		// Assign jumbled position from shuffled coords
		jumbledCoord := coords[i]
		u.jumbledCoords[char] = jumbledCoord

		// Set initial position using Motion (this is critical!)
		char.Motion.SetCoordinate(jumbledCoord)
		char.Coord = jumbledCoord

		// Determine explosion target (random edge)
		pos := utils.RandIntn(4)
		var explosionTarget utils.Coord
		switch pos {
		case 0: // left edge
			explosionTarget = utils.Coord{Row: utils.RandIntn(base.Canvas.Height) + 1, Col: base.Canvas.Left}
		case 1: // right edge
			explosionTarget = utils.Coord{Row: utils.RandIntn(base.Canvas.Height) + 1, Col: base.Canvas.Right}
		case 2: // bottom edge
			explosionTarget = utils.Coord{Row: base.Canvas.Bottom, Col: utils.RandIntn(base.Canvas.Width) + 1}
		case 3: // top edge
			explosionTarget = utils.Coord{Row: base.Canvas.Top, Col: utils.RandIntn(base.Canvas.Width) + 1}
		}

		// Create explosion path (from jumbled to edge)
		explosionPath, _ := char.Motion.NewPath(1.0, utils.OutExpo, nil, 0, false, "explosion")
		explosionPath.AddWaypoint(explosionTarget)

		// Create reassembly path (from edge to input coord)
		reassemblyPath, _ := char.Motion.NewPath(1.0, utils.OutExpo, nil, 0, false, "reassembly")
		reassemblyPath.AddWaypoint(char.InputCoord)

		// Setup animation scenes
		u.setupCharacterScenes(char)

		char.Visible = true
	}

	return u
}

func (u *Unstable) setupCharacterScenes(char *engine.EffectCharacter) {
	finalColor := u.characterFinalColor[char]

	// Rumble scene: transition from final color to unstable color
	rumbleScene := char.Animation.NewScene("rumble")
	rumbleGradient, _ := utils.NewGradient([]utils.Color{finalColor, u.unstableColor}, []int{12}, false)
	for _, color := range rumbleGradient.Spectrum {
		c := color
		rumbleScene.AddFrame(char.Symbol, 10, &utils.ColorPair{FG: &c})
	}

	// Final scene: transition from unstable color back to final color
	finalScene := char.Animation.NewScene("final")
	finalGradient, _ := utils.NewGradient([]utils.Color{u.unstableColor, finalColor}, []int{12}, false)
	for _, color := range finalGradient.Spectrum {
		c := color
		finalScene.AddFrame(char.Symbol, 3, &utils.ColorPair{FG: &c})
	}

	// Start with rumble animation
	char.Animation.ActivateScene("rumble")
}

func (u *Unstable) Next() (string, bool) {
	switch u.phase {
	case "rumble":
		u.rumbleSteps++

		// Check if rumble phase is complete FIRST
		if u.rumbleSteps >= u.maxRumbleSteps {
			// Transition to explosion phase
			u.phase = "explosion"
			// Activate explosion paths for all characters
			for _, char := range u.base.Characters {
				explosionPath, _ := char.Motion.QueryPath("explosion")
				char.Motion.ActivatePath(explosionPath)
				u.activeCharacters[char] = struct{}{}
			}
			// Fall through to explosion phase handling below
		} else {
			// Periodic jitter effect (after step 30)
			if u.rumbleSteps > 30 && u.rumbleSteps%u.rumbleModDelay == 0 {
				// Apply random offset to all characters
				rowOffset := utils.RandIntn(3) - 1 // -1, 0, or 1
				colOffset := utils.RandIntn(3) - 1

				for _, char := range u.base.Characters {
					jumbled := u.jumbledCoords[char]
					offsetCoord := utils.Coord{
						Row: jumbled.Row + rowOffset,
						Col: jumbled.Col + colOffset,
					}
					// Set both Motion.CurrentCoord and Coord for proper rendering
					char.Motion.SetCoordinate(offsetCoord)
					char.Coord = offsetCoord
					// Step animation only (not motion) during rumble
					if visual, ok := char.Animation.Next(); ok {
						char.SetVisual(visual)
					}
				}

				// Render the jittered frame
				frame := engine.RenderFrame(u.base.Canvas, u.base.Characters)

				// Reset to jumbled positions
				for _, char := range u.base.Characters {
					jumbled := u.jumbledCoords[char]
					char.Motion.SetCoordinate(jumbled)
					char.Coord = jumbled
				}

				// Speed up rumble over time
				if u.rumbleModDelay > 1 {
					u.rumbleModDelay--
				}

				return frame, true
			}

			// Regular frame - just step animations (not motion)
			for _, char := range u.base.Characters {
				if visual, ok := char.Animation.Next(); ok {
					char.SetVisual(visual)
				}
			}
		}

	case "explosion":
		if len(u.activeCharacters) > 0 {
			// Tick all active characters (moves along path + steps animation)
			for char := range u.activeCharacters {
				char.Tick()
			}

			// Remove characters that have completed explosion path
			for char := range u.activeCharacters {
				if char.Motion.MovementComplete() {
					delete(u.activeCharacters, char)
				}
			}
		} else if u.explosionHoldTime > 0 {
			// Hold at edge position
			u.explosionHoldTime--
		} else {
			// Transition to reassembly phase
			u.phase = "reassembly"
			for _, char := range u.base.Characters {
				// Activate final color animation
				char.Animation.ActivateScene("final")
				// Activate reassembly path
				reassemblyPath, _ := char.Motion.QueryPath("reassembly")
				char.Motion.ActivatePath(reassemblyPath)
				u.activeCharacters[char] = struct{}{}
			}
		}

	case "reassembly":
		if len(u.activeCharacters) > 0 {
			// Tick all active characters
			for char := range u.activeCharacters {
				char.Tick()
			}

			// Remove characters that have completed both movement and animation
			for char := range u.activeCharacters {
				movementDone := char.Motion.MovementComplete()
				animationDone := char.Animation.ActiveScene == nil || char.Animation.ActiveScene.IsComplete()
				if movementDone && animationDone {
					delete(u.activeCharacters, char)
				}
			}
		}

		if len(u.activeCharacters) == 0 {
			u.phase = "complete"
		}

	case "complete":
		return "", false
	}

	frame := engine.RenderFrame(u.base.Canvas, u.base.Characters)
	return frame, true
}

func (u *Unstable) CanvasHeight() int {
	return u.base.CanvasHeight()
}
