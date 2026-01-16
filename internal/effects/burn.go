package effects

import (
	"tte-go/internal/config"
	"tte-go/internal/engine"
	"tte-go/internal/utils"
)

// Burn creates an effect where characters are ignited and burn up the screen.
// Uses Prim's spanning tree algorithm to determine the order of burning.
type Burn struct {
	base *BaseEffect

	charLinkOrder    []*engine.EffectCharacter
	activeCharacters map[*engine.EffectCharacter]struct{}
	pendingSmoke     []*engine.EffectCharacter
	smokeParticles   []*engine.EffectCharacter
	smokeIdx         int

	// Config
	startingColor          utils.Color
	burnColors             []utils.Color
	smokeChance            float64
	finalGradientStops     []utils.Color
	finalGradientSteps     []int
	finalGradientDirection utils.GradientDirection
}

func NewBurn(input string, cfg config.TerminalConfig) engine.Effect {
	base := NewBaseEffect(input, cfg)

	// Config defaults
	startingColor := mustColors("837373")[0]
	burnColors := mustColors("ffffff", "fff75d", "fe650d", "8a003c", "510100")
	smokeChance := 0.5

	finalGradientStops := mustColors("00c3ff", "ffff1c")
	finalGradientSteps := []int{12}
	finalGradientDirection := utils.GradientVertical

	b := &Burn{
		base:                   base,
		charLinkOrder:          make([]*engine.EffectCharacter, 0),
		activeCharacters:       make(map[*engine.EffectCharacter]struct{}),
		pendingSmoke:           make([]*engine.EffectCharacter, 0),
		smokeParticles:         make([]*engine.EffectCharacter, 0),
		smokeIdx:               0,
		startingColor:          startingColor,
		burnColors:             burnColors,
		smokeChance:            smokeChance,
		finalGradientStops:     finalGradientStops,
		finalGradientSteps:     finalGradientSteps,
		finalGradientDirection: finalGradientDirection,
	}

	b.makeSmoke()
	b.build()
	return b
}

func (b *Burn) makeSmoke() {
	smokeSymbols := []string{".", ",", "'", "`", "#", "*"}
	smokeGradient, _ := utils.NewGradient(mustColors("504F4F", "C7C7C7"), []int{9}, false)

	for i := 0; i < 2000; i++ {
		symbol := smokeSymbols[utils.RandIntn(len(smokeSymbols))]
		newChar := engine.NewEffectCharacter(symbol, utils.Coord{Row: 0, Col: 0})
		newChar.IsAddedCharacter = true
		newChar.Visible = false
		newChar.Layer = 2

		// Smoke scene - fade through gray gradient
		smokeScene := newChar.Animation.NewScene("smoke")
		for _, color := range smokeGradient.Spectrum {
			colorCopy := color
			_ = smokeScene.AddFrame(symbol, 10, &utils.ColorPair{FG: &colorCopy})
		}

		// Register the smoke character with the terminal
		b.base.Terminal.AddCharacter(newChar)
		b.smokeParticles = append(b.smokeParticles, newChar)
	}
}

func (b *Burn) emitSmoke(origin utils.Coord) {
	if utils.RandFloat64() > b.smokeChance {
		return
	}

	// Get next smoke particle (rotating through pool)
	particle := b.smokeParticles[b.smokeIdx]
	b.smokeIdx = (b.smokeIdx + 1) % len(b.smokeParticles)

	// Reset and position particle
	particle.Motion.SetCoordinate(origin)
	if particle.Animation.ActiveScene != nil {
		particle.Animation.ActiveScene.ResetScene()
	}

	// Create upward path with random horizontal drift
	targetCol := origin.Col + utils.RandIntn(9) - 4 // -4 to +4
	targetRow := b.base.Canvas.Top + 1

	smokePath, _ := particle.Motion.NewPath(0.5, nil, nil, 0, false, "smoke_rise")
	smokePath.AddWaypoint(utils.Coord{Col: targetCol, Row: targetRow})
	particle.Motion.ActivatePath(smokePath)
	particle.Animation.ActivateScene("smoke")

	b.base.Terminal.SetCharacterVisibility(particle, true)
	b.pendingSmoke = append(b.pendingSmoke, particle)
}

func (b *Burn) build() {
	burnCharOrder := []string{"'", ".", "▖", "▙", "█", "▜", "▀", "▝", "."}

	// Build final gradient mapping
	finalGradient, _ := utils.NewGradient(b.finalGradientStops, b.finalGradientSteps, false)
	finalGradientMapping, _ := finalGradient.BuildCoordinateColorMapping(
		b.base.Canvas.TextBottom,
		b.base.Canvas.TextTop,
		b.base.Canvas.TextLeft,
		b.base.Canvas.TextRight,
		b.finalGradientDirection,
	)

	// Build fire gradient
	fireGradient, _ := utils.NewGradient(b.burnColors, []int{10}, false)

	// Run Prim's algorithm to determine burn order
	b.runPrimsAlgorithm()

	// Set up each character
	for _, char := range b.base.Terminal.GetCharacters(true, false, false, false, engine.TopToBottomLeftToRight) {
		charFinalColor := finalGradientMapping[char.InputCoord]

		// Make visible with starting color
		b.base.Terminal.SetCharacterVisibility(char, true)
		startingColorCopy := b.startingColor
		char.SetVisual(engine.CharacterVisual{
			Symbol: char.Symbol,
			Colors: &utils.ColorPair{FG: &startingColorCopy},
		})

		// Burn scene - animate through burn characters with fire gradient
		burnScene := char.Animation.NewScene("burn")
		_ = burnScene.ApplyGradientToSymbols(burnCharOrder, 4, fireGradient, nil)

		// Final color scene - transition from last burn color to final gradient color
		finalColorScene := char.Animation.NewScene("final_color")
		lastBurnColor := fireGradient.Spectrum[len(fireGradient.Spectrum)-1]
		finalTransitionGradient, _ := utils.NewGradient([]utils.Color{lastBurnColor, charFinalColor}, []int{8}, false)
		for _, color := range finalTransitionGradient.Spectrum {
			colorCopy := color
			_ = finalColorScene.AddFrame(char.Symbol, 4, &utils.ColorPair{FG: &colorCopy})
		}

		// Chain: burn complete -> activate final scene
		_ = char.EventHandler.RegisterEvent(
			engine.EventSceneComplete,
			burnScene,
			engine.ActionActivateScene,
			finalColorScene,
		)

		// Callback for smoke emission (stored in char for lookup in tick)
		// We'll handle this in Next() by checking if burn scene just completed
	}
}

// runPrimsAlgorithm runs a simplified Prim's spanning tree algorithm
// to determine the order in which characters will burn.
func (b *Burn) runPrimsAlgorithm() {
	// Get input characters (this is what the Python version does)
	characters := b.base.Terminal.GetCharacters(true, false, false, false, engine.TopToBottomLeftToRight)
	if len(characters) == 0 {
		return
	}

	// Pick a random starting character
	startIdx := utils.RandIntn(len(characters))
	startChar := characters[startIdx]

	linked := make(map[*engine.EffectCharacter]struct{})
	linked[startChar] = struct{}{}
	b.charLinkOrder = append(b.charLinkOrder, startChar)

	edgeChars := []*engine.EffectCharacter{startChar}

	for len(edgeChars) > 0 {
		// Pop random edge character
		idx := utils.RandIntn(len(edgeChars))
		currentChar := edgeChars[idx]
		edgeChars[idx] = edgeChars[len(edgeChars)-1]
		edgeChars = edgeChars[:len(edgeChars)-1]

		// Get unlinked neighbors
		unlinkedNeighbors := b.getUnlinkedNeighbors(currentChar, linked)

		if len(unlinkedNeighbors) > 0 {
			// Link to a random unlinked neighbor
			nextIdx := utils.RandIntn(len(unlinkedNeighbors))
			nextChar := unlinkedNeighbors[nextIdx]

			currentChar.Link(nextChar, true)
			linked[nextChar] = struct{}{}
			b.charLinkOrder = append(b.charLinkOrder, nextChar)

			// Remove the linked neighbor from the list
			unlinkedNeighbors[nextIdx] = unlinkedNeighbors[len(unlinkedNeighbors)-1]
			unlinkedNeighbors = unlinkedNeighbors[:len(unlinkedNeighbors)-1]

			// If current char still has unlinked neighbors, keep it as an edge
			if len(unlinkedNeighbors) > 0 {
				edgeChars = append(edgeChars, currentChar)
			}

			// Check if next char has unlinked neighbors
			nextUnlinked := b.getUnlinkedNeighbors(nextChar, linked)
			if len(nextUnlinked) > 0 {
				edgeChars = append(edgeChars, nextChar)
			}
		}
	}
}

func (b *Burn) getUnlinkedNeighbors(char *engine.EffectCharacter, linked map[*engine.EffectCharacter]struct{}) []*engine.EffectCharacter {
	result := make([]*engine.EffectCharacter, 0)
	for _, neighbor := range char.Neighbors {
		if neighbor == nil {
			continue
		}
		if _, isLinked := linked[neighbor]; isLinked {
			continue
		}
		result = append(result, neighbor)
	}
	return result
}

func (b *Burn) Next() (string, bool) {
	// Check if we have more work to do
	if len(b.charLinkOrder) == 0 && len(b.activeCharacters) == 0 {
		return "", false
	}

	// Add pending smoke to active
	for _, smoke := range b.pendingSmoke {
		b.activeCharacters[smoke] = struct{}{}
	}
	b.pendingSmoke = b.pendingSmoke[:0]

	// Activate 2-4 characters per frame
	numToActivate := 2 + utils.RandIntn(3)
	for i := 0; i < numToActivate && len(b.charLinkOrder) > 0; i++ {
		nextChar := b.charLinkOrder[0]
		b.charLinkOrder = b.charLinkOrder[1:]

		if nextChar.Symbol == " " {
			continue
		}

		nextChar.Animation.ActivateScene("burn")
		b.activeCharacters[nextChar] = struct{}{}
	}

	// Tick all active characters
	for char := range b.activeCharacters {
		// Check if burn scene just completed (for smoke emission)
		wasBurning := false
		if char.Animation.ActiveScene != nil && char.Animation.ActiveScene.ID == "burn" {
			wasBurning = true
		}

		char.Tick()

		// If was burning and now scene changed, emit smoke
		if wasBurning && char.Animation.ActiveScene != nil && char.Animation.ActiveScene.ID != "burn" {
			b.emitSmoke(char.InputCoord)
		}

		// Check if character is still active
		if !char.IsActive() {
			// For smoke particles, hide them when done
			if char.IsAddedCharacter {
				b.base.Terminal.SetCharacterVisibility(char, false)
			}
			delete(b.activeCharacters, char)
		}
	}

	return b.base.Terminal.GetFormattedOutputString(), true
}

func (b *Burn) CanvasHeight() int {
	return b.base.CanvasHeight()
}
