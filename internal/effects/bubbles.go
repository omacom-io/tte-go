package effects

import (
	"tte-go/internal/config"
	"tte-go/internal/engine"
	"tte-go/internal/utils"
)

type BubblesConfig struct {
	Rainbow            bool
	BubbleColors       []utils.Color
	PopColor           utils.Color
	BubbleSpeed        float64
	BubbleDelay        int
	PopCondition       string // "row", "bottom", "anywhere"
	MovementEasing     utils.EasingFunction
	FinalGradientStops []utils.Color
	FinalGradientSteps []int
	FinalGradientDir   utils.GradientDirection
}

type Bubble struct {
	effect     *Bubbles
	characters []*engine.EffectCharacter
	radius     int
	origin     utils.Coord
	anchor     *engine.EffectCharacter
	lowestRow  int
	landed     bool
}

type Bubbles struct {
	base                 *BaseEffect
	config               BubblesConfig
	bubbles              []*Bubble
	animatingBubbles     []*Bubble
	activeCharacters     map[*engine.EffectCharacter]struct{}
	finalColors          map[*engine.EffectCharacter]utils.Color
	rainbowGradient      *utils.Gradient
	stepsSinceLastBubble int
}

func NewBubbles(input string, cfg config.TerminalConfig) engine.Effect {
	base := NewBaseEffect(input, cfg)
	b := &Bubbles{
		base:                 base,
		config:               defaultBubblesConfig(),
		bubbles:              []*Bubble{},
		animatingBubbles:     []*Bubble{},
		activeCharacters:     map[*engine.EffectCharacter]struct{}{},
		finalColors:          map[*engine.EffectCharacter]utils.Color{},
		stepsSinceLastBubble: 0,
	}
	b.buildRainbowGradient()
	b.build()
	return b
}

func defaultBubblesConfig() BubblesConfig {
	return BubblesConfig{
		Rainbow:            false,
		BubbleColors:       mustColors("d33aff", "7395c4", "43c2a7", "02ff7f"),
		PopColor:           mustColors("ffffff")[0],
		BubbleSpeed:        0.5,
		BubbleDelay:        20,
		PopCondition:       "row",
		MovementEasing:     utils.InOutSine,
		FinalGradientStops: mustColors("d33aff", "02ff7f"),
		FinalGradientSteps: []int{12},
		FinalGradientDir:   utils.GradientDiagonal,
	}
}

func (b *Bubbles) buildRainbowGradient() {
	rainbowColors := mustColors("e81416", "ffa500", "faeb36", "79c314", "487de7", "4b369d", "70369d")
	b.rainbowGradient, _ = utils.NewGradient(rainbowColors, []int{5}, false)
}

func (b *Bubbles) build() {
	// Build final gradient mapping for characters
	finalGradient, _ := utils.NewGradient(b.config.FinalGradientStops, b.config.FinalGradientSteps, false)
	finalMapping, _ := finalGradient.BuildCoordinateColorMapping(
		b.base.Canvas.TextBottom,
		b.base.Canvas.TextTop,
		b.base.Canvas.TextLeft,
		b.base.Canvas.TextRight,
		b.config.FinalGradientDir,
	)

	// Set up each character with animation scenes and motion paths
	for _, character := range b.base.Terminal.GetCharacters(true, false, false, false, engine.TopToBottomLeftToRight) {
		b.finalColors[character] = finalMapping[character.InputCoord]
		character.Layer = 1

		// Pop scenes
		pop1Scene := character.Animation.NewScene("pop_1")
		pop2Scene := character.Animation.NewScene("pop_2")
		popColorPair := &utils.ColorPair{FG: &b.config.PopColor}
		_ = pop1Scene.AddFrame("*", 9, popColorPair)
		_ = pop2Scene.AddFrame("'", 9, popColorPair)

		// Final scene - gradient from pop color to final color
		finalScene := character.Animation.NewScene("final")
		charFinalGradient, _ := utils.NewGradient(
			[]utils.Color{b.config.PopColor, b.finalColors[character]},
			[]int{8},
			false,
		)
		_ = finalScene.ApplyGradientToSymbols([]string{character.Symbol}, 6, charFinalGradient, nil)

		// Chain pop scenes
		_ = character.EventHandler.RegisterEvent(engine.EventSceneComplete, pop1Scene, engine.ActionActivateScene, pop2Scene)
		_ = character.EventHandler.RegisterEvent(engine.EventSceneComplete, pop2Scene, engine.ActionActivateScene, finalScene)

		// Final path - character returns to its original position
		finalPath, _ := character.Motion.NewPath(0.3, utils.InOutExpo, nil, 0, false, "final")
		finalPath.AddWaypoint(character.InputCoord)
		_ = character.EventHandler.RegisterEvent(engine.EventPathComplete, finalPath, engine.ActionSetLayer, 0)
	}

	// Group characters from bottom to top rows, then form bubbles
	unbubbledChars := []*engine.EffectCharacter{}
	for _, charList := range b.base.Terminal.GetCharactersGrouped(engine.RowBottomToTop, true, false, false, false) {
		unbubbledChars = append(unbubbledChars, charList...)
	}

	// Form bubbles from character groups
	for len(unbubbledChars) > 0 {
		bubbleGroup := []*engine.EffectCharacter{}
		if len(unbubbledChars) < 5 {
			bubbleGroup = append(bubbleGroup, unbubbledChars...)
			unbubbledChars = nil
		} else {
			// Random bubble size between 5 and min(20, remaining)
			maxSize := len(unbubbledChars)
			if maxSize > 20 {
				maxSize = 20
			}
			bubbleSize := 5 + utils.RandIntn(maxSize-5+1)
			for i := 0; i < bubbleSize && len(unbubbledChars) > 0; i++ {
				bubbleGroup = append(bubbleGroup, unbubbledChars[0])
				unbubbledChars = unbubbledChars[1:]
			}
		}

		// Create bubble origin - random column, above the canvas
		bubbleOrigin := utils.Coord{
			Col: b.base.Canvas.Left + utils.RandIntn(b.base.Canvas.Right-b.base.Canvas.Left+1),
			Row: b.base.Canvas.Top + 10,
		}

		bubble := b.newBubble(bubbleOrigin, bubbleGroup)
		b.bubbles = append(b.bubbles, bubble)
	}
}

func (b *Bubbles) newBubble(origin utils.Coord, characters []*engine.EffectCharacter) *Bubble {
	// Create anchor character for bubble movement
	anchor := engine.NewEffectCharacter(" ", origin)
	anchor.Visible = false

	radius := len(characters) / 5
	if radius < 1 {
		radius = 1
	}

	// Determine lowest row based on pop condition
	lowestRow := b.base.Canvas.Bottom
	if b.config.PopCondition == "row" {
		lowestRow = characters[0].InputCoord.Row
		for _, char := range characters {
			if char.InputCoord.Row < lowestRow {
				lowestRow = char.InputCoord.Row
			}
		}
	}

	bubble := &Bubble{
		effect:     b,
		characters: characters,
		radius:     radius,
		origin:     origin,
		anchor:     anchor,
		lowestRow:  lowestRow,
		landed:     false,
	}

	// Set initial character coordinates on the bubble circle
	bubble.setCharacterCoordinates()

	// Create waypoint path for the anchor
	bubble.makeWaypoints()

	// Set up gradients/colors for bubble characters
	bubble.makeGradients()

	return bubble
}

func (bubble *Bubble) setCharacterCoordinates() {
	circlePoints := utils.FindCoordsOnCircle(
		bubble.anchor.Motion.CurrentCoord,
		bubble.radius,
		len(bubble.characters),
		false,
	)

	for i, char := range bubble.characters {
		if i < len(circlePoints) {
			point := circlePoints[i]
			char.Motion.SetCoordinate(point)
			char.Coord = point
			if point.Row == bubble.lowestRow {
				bubble.landed = true
			}
		}
	}

	// Random pop for "anywhere" condition
	if bubble.effect.config.PopCondition == "anywhere" && utils.RandFloat64() < 0.002 {
		bubble.landed = true
	}
}

func (bubble *Bubble) makeWaypoints() {
	waypointColumn := bubble.effect.base.Canvas.Left + utils.RandIntn(bubble.effect.base.Canvas.Right-bubble.effect.base.Canvas.Left+1)
	floorPath, _ := bubble.anchor.Motion.NewPath(bubble.effect.config.BubbleSpeed, nil, nil, 0, false, "floor")
	floorPath.AddWaypoint(utils.Coord{Col: waypointColumn, Row: bubble.lowestRow})
	bubble.anchor.Motion.ActivatePath(floorPath)
}

func (bubble *Bubble) makeGradients() {
	if bubble.effect.config.Rainbow {
		// Rainbow mode - rotating rainbow gradient
		rainbowSpectrum := make([]utils.Color, len(bubble.effect.rainbowGradient.Spectrum))
		copy(rainbowSpectrum, bubble.effect.rainbowGradient.Spectrum)

		gradientOffset := 0
		for _, character := range bubble.characters {
			sheenScene := character.Animation.NewScene("sheen")
			for _, step := range rainbowSpectrum {
				colorCopy := step
				_ = sheenScene.AddFrame(character.Symbol, 4, &utils.ColorPair{FG: &colorCopy})
			}
			sheenScene.IsLooping = true
			character.Animation.ActivateScene("sheen")

			gradientOffset += 2
			gradientOffset %= len(rainbowSpectrum)
			// Rotate the spectrum for the next character
			rotated := make([]utils.Color, len(rainbowSpectrum))
			copy(rotated, rainbowSpectrum[gradientOffset:])
			copy(rotated[len(rainbowSpectrum)-gradientOffset:], rainbowSpectrum[:gradientOffset])
			rainbowSpectrum = rotated
		}
	} else {
		// Pick a random bubble color for all characters in this bubble
		bubbleColor := bubble.effect.config.BubbleColors[utils.RandIntn(len(bubble.effect.config.BubbleColors))]
		for _, character := range bubble.characters {
			sheenScene := character.Animation.NewScene("sheen")
			colorCopy := bubbleColor
			_ = sheenScene.AddFrame(character.Symbol, 1, &utils.ColorPair{FG: &colorCopy})
			character.Animation.ActivateScene("sheen")
		}
	}
}

func (bubble *Bubble) pop() {
	// Calculate pop-out positions (radius + 3)
	popOutPoints := utils.FindCoordsOnCircle(
		bubble.anchor.Motion.CurrentCoord,
		bubble.radius+3,
		len(bubble.characters),
		false,
	)

	for i, char := range bubble.characters {
		if i < len(popOutPoints) {
			point := popOutPoints[i]
			popOutPath, _ := char.Motion.NewPath(0.3, utils.OutExpo, nil, 0, false, "pop_out")
			popOutPath.AddWaypoint(point)

			// Chain: pop_out complete -> activate final path
			finalPath, _ := char.Motion.QueryPath("final")
			if finalPath != nil {
				_ = char.EventHandler.RegisterEvent(engine.EventPathComplete, popOutPath, engine.ActionActivatePath, finalPath)
			}
		}

		// Activate pop animation and movement
		char.Animation.ActivateScene("pop_1")
		if popOutPath, _ := char.Motion.QueryPath("pop_out"); popOutPath != nil {
			char.Motion.ActivatePath(popOutPath)
		}
	}
}

func (bubble *Bubble) activate() {
	for _, char := range bubble.characters {
		bubble.effect.base.Terminal.SetCharacterVisibility(char, true)
	}
}

func (bubble *Bubble) move() {
	bubble.anchor.Motion.Move()
	bubble.anchor.Coord = bubble.anchor.Motion.CurrentCoord
	bubble.setCharacterCoordinates()
	for _, character := range bubble.characters {
		if character.Animation != nil && character.Animation.ActiveScene != nil {
			visual, ok := character.Animation.Next()
			if ok {
				character.SetVisual(visual)
			}
		}
	}
}

func (b *Bubbles) Next() (string, bool) {
	hasWork := len(b.animatingBubbles) > 0 || len(b.activeCharacters) > 0 || len(b.bubbles) > 0

	if !hasWork {
		return "", false
	}

	// Release next bubble if delay has passed
	if len(b.bubbles) > 0 && b.stepsSinceLastBubble >= b.config.BubbleDelay {
		nextBubble := b.bubbles[0]
		b.bubbles = b.bubbles[1:]
		nextBubble.activate()
		b.animatingBubbles = append(b.animatingBubbles, nextBubble)
		b.stepsSinceLastBubble = 0
	}
	b.stepsSinceLastBubble++

	// Check for landed bubbles and pop them
	stillAnimating := []*Bubble{}
	for _, bubble := range b.animatingBubbles {
		if bubble.landed {
			bubble.pop()
			for _, char := range bubble.characters {
				b.activeCharacters[char] = struct{}{}
			}
		} else {
			stillAnimating = append(stillAnimating, bubble)
		}
	}
	b.animatingBubbles = stillAnimating

	// Move remaining bubbles
	for _, bubble := range b.animatingBubbles {
		bubble.move()
	}

	// Tick active characters
	for character := range b.activeCharacters {
		character.Tick()
		if !character.IsActive() {
			delete(b.activeCharacters, character)
		}
	}

	return b.base.Terminal.GetFormattedOutputString(), true
}

func (b *Bubbles) CanvasHeight() int {
	return b.base.CanvasHeight()
}

func (b *Bubbles) CanvasWidth() int {
	return b.base.CanvasWidth()
}
