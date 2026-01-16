package effects

import (
	"tte-go/internal/config"
	"tte-go/internal/engine"
	"tte-go/internal/utils"
)

// Crumble creates an effect where characters crumble into dust before being
// vacuumed up and reformed.
type Crumble struct {
	base *BaseEffect

	pendingChars        []*engine.EffectCharacter
	activeCharacters    map[*engine.EffectCharacter]struct{}
	unvacuumedChars     []*engine.EffectCharacter
	characterFinalColor map[*engine.EffectCharacter]utils.Color

	// Stage management
	stage            string // "falling", "vacuuming", "resetting", "complete"
	fallDelay        int
	maxFallDelay     int
	minFallDelay     int
	fallGroupMaxSize int
	resetDone        bool

	// Config
	finalGradientStops     []utils.Color
	finalGradientSteps     []int
	finalGradientDirection utils.GradientDirection
}

func NewCrumble(input string, cfg config.TerminalConfig) engine.Effect {
	base := NewBaseEffect(input, cfg)

	// Config defaults
	finalGradientStops := mustColors("5CE1FF", "FF8C00")
	finalGradientSteps := []int{12}
	finalGradientDirection := utils.GradientDiagonal

	c := &Crumble{
		base:                   base,
		pendingChars:           make([]*engine.EffectCharacter, 0),
		activeCharacters:       make(map[*engine.EffectCharacter]struct{}),
		unvacuumedChars:        make([]*engine.EffectCharacter, 0),
		characterFinalColor:    make(map[*engine.EffectCharacter]utils.Color),
		stage:                  "falling",
		fallDelay:              12,
		maxFallDelay:           12,
		minFallDelay:           9,
		fallGroupMaxSize:       1,
		resetDone:              false,
		finalGradientStops:     finalGradientStops,
		finalGradientSteps:     finalGradientSteps,
		finalGradientDirection: finalGradientDirection,
	}

	c.build()
	return c
}

func (c *Crumble) build() {
	// Build final gradient mapping
	finalGradient, _ := utils.NewGradient(c.finalGradientStops, c.finalGradientSteps, false)
	finalGradientMapping, _ := finalGradient.BuildCoordinateColorMapping(
		c.base.Canvas.TextBottom,
		c.base.Canvas.TextTop,
		c.base.Canvas.TextLeft,
		c.base.Canvas.TextRight,
		c.finalGradientDirection,
	)

	dustChars := []string{"*", ".", ","}

	for _, character := range c.base.Terminal.GetCharacters(true, false, false, false, engine.TopToBottomLeftToRight) {
		charFinalColor := finalGradientMapping[character.InputCoord]
		c.characterFinalColor[character] = charFinalColor

		// Calculate dimmed colors
		weakColor := utils.AdjustBrightness(charFinalColor, 0.65)
		dustColor := utils.AdjustBrightness(charFinalColor, 0.55)

		// Create gradients
		strengthenFlashGradient, _ := utils.NewGradient([]utils.Color{charFinalColor, mustColors("ffffff")[0]}, []int{6}, false)
		strengthenGradient, _ := utils.NewGradient([]utils.Color{mustColors("ffffff")[0], charFinalColor}, []int{9}, false)
		weakenGradient, _ := utils.NewGradient([]utils.Color{weakColor, dustColor}, []int{9}, false)

		// Make character visible with initial weak color
		c.base.Terminal.SetCharacterVisibility(character, true)
		weakColorCopy := weakColor
		character.SetVisual(engine.CharacterVisual{
			Symbol: character.Symbol,
			Colors: &utils.ColorPair{FG: &weakColorCopy},
		})

		// Initial scene (just weak color frame)
		initialScene := character.Animation.NewScene("initial")
		weakColorCopy2 := weakColor
		_ = initialScene.AddFrame(character.Symbol, 1, &utils.ColorPair{FG: &weakColorCopy2})
		character.Animation.ActivateSceneRef(initialScene)

		// Weaken scene - transition from weak to dust color
		weakenScene := character.Animation.NewScene("weaken")
		_ = weakenScene.ApplyGradientToSymbols([]string{character.Symbol}, 4, weakenGradient, nil)

		// Fall path - character falls to bottom with bounce easing
		fallPath, _ := character.Motion.NewPath(0.65, utils.OutBounce, nil, 0, false, "fall")
		fallPath.AddWaypoint(utils.Coord{Col: character.InputCoord.Col, Row: c.base.Canvas.Bottom})

		// Dust scene - cycles through dust characters (simulating sync-to-distance)
		dustScene := character.Animation.NewScene("dust")
		dustScene.IsLooping = true
		for i := 0; i < 5; i++ {
			dustColorCopy := dustColor
			dustChar := dustChars[utils.RandIntn(len(dustChars))]
			_ = dustScene.AddFrame(dustChar, 1, &utils.ColorPair{FG: &dustColorCopy})
		}

		// Top path - character gets vacuumed up to top via bezier curve through canvas center
		layer := 1
		topPath, _ := character.Motion.NewPath(1.0, utils.OutQuint, &layer, 0, false, "top")
		topPath.AddBezierWaypoint(
			utils.Coord{Col: character.InputCoord.Col, Row: c.base.Canvas.Top},
			utils.Coord{Col: c.base.Canvas.Center.Col, Row: c.base.Canvas.Center.Row},
		)

		// Input path - return to original position
		inputPath, _ := character.Motion.NewPath(1.0, nil, nil, 0, false, "input")
		inputPath.AddWaypoint(character.InputCoord)

		// Strengthen flash scene - flash white before returning to final color
		strengthenFlashScene := character.Animation.NewScene("strengthen_flash")
		_ = strengthenFlashScene.ApplyGradientToSymbols([]string{character.Symbol}, 4, strengthenFlashGradient, nil)

		// Strengthen scene - transition from white to final color
		strengthenScene := character.Animation.NewScene("strengthen")
		_ = strengthenScene.ApplyGradientToSymbols([]string{character.Symbol}, 4, strengthenGradient, nil)

		// Event chain: weaken complete -> activate fall path + dust scene + set layer
		_ = character.EventHandler.RegisterEvent(
			engine.EventSceneComplete,
			weakenScene,
			engine.ActionActivatePath,
			fallPath,
		)
		_ = character.EventHandler.RegisterEvent(
			engine.EventSceneComplete,
			weakenScene,
			engine.ActionActivateScene,
			dustScene,
		)
		_ = character.EventHandler.RegisterEvent(
			engine.EventSceneComplete,
			weakenScene,
			engine.ActionSetLayer,
			1,
		)

		// Event chain: fall path complete -> deactivate dust scene (so char becomes inactive)
		_ = character.EventHandler.RegisterEvent(
			engine.EventPathComplete,
			fallPath,
			engine.ActionDeactivateScene,
			nil,
		)

		// Event chain: input path complete -> activate strengthen flash scene
		_ = character.EventHandler.RegisterEvent(
			engine.EventPathComplete,
			inputPath,
			engine.ActionActivateScene,
			strengthenFlashScene,
		)

		// Event chain: strengthen flash complete -> activate strengthen scene
		_ = character.EventHandler.RegisterEvent(
			engine.EventSceneComplete,
			strengthenFlashScene,
			engine.ActionActivateScene,
			strengthenScene,
		)

		c.pendingChars = append(c.pendingChars, character)
	}

	// Shuffle pending chars for random fall order
	for i := len(c.pendingChars) - 1; i > 0; i-- {
		j := utils.RandIntn(i + 1)
		c.pendingChars[i], c.pendingChars[j] = c.pendingChars[j], c.pendingChars[i]
	}

	// Copy input characters for vacuuming stage (shuffled)
	inputChars := c.base.Terminal.GetCharacters(true, false, false, false, engine.TopToBottomLeftToRight)
	c.unvacuumedChars = make([]*engine.EffectCharacter, len(inputChars))
	copy(c.unvacuumedChars, inputChars)
	for i := len(c.unvacuumedChars) - 1; i > 0; i-- {
		j := utils.RandIntn(i + 1)
		c.unvacuumedChars[i], c.unvacuumedChars[j] = c.unvacuumedChars[j], c.unvacuumedChars[i]
	}
}

func (c *Crumble) Next() (string, bool) {
	if c.stage == "complete" {
		return "", false
	}

	if c.stage == "falling" {
		if len(c.pendingChars) > 0 {
			if c.fallDelay == 0 {
				// Determine the size of the next group of falling characters
				fallGroupSize := 1 + utils.RandIntn(c.fallGroupMaxSize)
				// Add the next group of falling characters to the active set
				for i := 0; i < fallGroupSize && len(c.pendingChars) > 0; i++ {
					nextChar := c.pendingChars[0]
					c.pendingChars = c.pendingChars[1:]
					nextChar.Animation.ActivateScene("weaken")
					c.activeCharacters[nextChar] = struct{}{}
				}
				// Reset the fall delay and adjust the fall group size and delay range
				c.fallDelay = c.minFallDelay + utils.RandIntn(c.maxFallDelay-c.minFallDelay+1)
				if utils.RandIntn(10) > 3 { // 60% chance to modify fall delay and group size
					c.fallGroupMaxSize++
					c.minFallDelay = max(0, c.minFallDelay-1)
					c.maxFallDelay = max(0, c.maxFallDelay-1)
				}
			} else {
				c.fallDelay--
			}
		}
		if len(c.pendingChars) == 0 && len(c.activeCharacters) == 0 {
			c.stage = "vacuuming"
		}
	} else if c.stage == "vacuuming" {
		if len(c.unvacuumedChars) > 0 {
			// Vacuum 3-10 characters per frame
			numToVacuum := 3 + utils.RandIntn(8) // 3-10
			for i := 0; i < numToVacuum && len(c.unvacuumedChars) > 0; i++ {
				nextChar := c.unvacuumedChars[0]
				c.unvacuumedChars = c.unvacuumedChars[1:]
				nextChar.Motion.ActivatePath(nextChar.Motion.Paths["top"])
				c.activeCharacters[nextChar] = struct{}{}
			}
		}
		if len(c.unvacuumedChars) == 0 && len(c.activeCharacters) == 0 {
			c.stage = "resetting"
		}
	} else if c.stage == "resetting" {
		if !c.resetDone {
			for _, character := range c.base.Terminal.GetCharacters(true, false, false, false, engine.TopToBottomLeftToRight) {
				character.Motion.ActivatePath(character.Motion.Paths["input"])
				c.activeCharacters[character] = struct{}{}
			}
			c.resetDone = true
		}
		if len(c.activeCharacters) == 0 {
			c.stage = "complete"
		}
	}

	// Tick all active characters
	for char := range c.activeCharacters {
		char.Tick()
		if !char.IsActive() {
			delete(c.activeCharacters, char)
		}
	}

	return c.base.Terminal.GetFormattedOutputString(), true
}

func (c *Crumble) CanvasHeight() int {
	return c.base.CanvasHeight()
}
