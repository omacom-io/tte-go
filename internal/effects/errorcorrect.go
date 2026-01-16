package effects

import (
	"tte-go/internal/config"
	"tte-go/internal/engine"
	"tte-go/internal/utils"
)

// ErrorCorrect creates an effect where some characters start in wrong positions
// and are corrected in sequence with error/correction animations.
type ErrorCorrect struct {
	base *BaseEffect

	swapped          [][2]*engine.EffectCharacter
	activeCharacters map[*engine.EffectCharacter]struct{}
	swapDelay        int

	// Config
	errorPairs             float64
	swapDelayFrames        int
	errorColor             utils.Color
	correctColor           utils.Color
	movementSpeed          float64
	finalGradientStops     []utils.Color
	finalGradientSteps     []int
	finalGradientDirection utils.GradientDirection
}

func NewErrorCorrect(input string, cfg config.TerminalConfig) engine.Effect {
	base := NewBaseEffect(input, cfg)

	// Config defaults
	errorColor := mustColors("e74c3c")[0]
	correctColor := mustColors("45bf55")[0]
	finalGradientStops := mustColors("8A008A", "00D1FF", "FFFFFF")
	finalGradientSteps := []int{12}
	finalGradientDirection := utils.GradientVertical

	e := &ErrorCorrect{
		base:                   base,
		swapped:                make([][2]*engine.EffectCharacter, 0),
		activeCharacters:       make(map[*engine.EffectCharacter]struct{}),
		swapDelay:              0,
		errorPairs:             0.1,
		swapDelayFrames:        6,
		errorColor:             errorColor,
		correctColor:           correctColor,
		movementSpeed:          0.9,
		finalGradientStops:     finalGradientStops,
		finalGradientSteps:     finalGradientSteps,
		finalGradientDirection: finalGradientDirection,
	}

	e.build()
	return e
}

func (e *ErrorCorrect) build() {
	// Build final gradient mapping
	finalGradient, _ := utils.NewGradient(e.finalGradientStops, e.finalGradientSteps, false)
	finalGradientMapping, _ := finalGradient.BuildCoordinateColorMapping(
		e.base.Canvas.TextBottom,
		e.base.Canvas.TextTop,
		e.base.Canvas.TextLeft,
		e.base.Canvas.TextRight,
		e.finalGradientDirection,
	)

	// Make all characters visible with their final color initially
	for _, character := range e.base.Terminal.GetCharacters(true, false, false, false, engine.TopToBottomLeftToRight) {
		charFinalColor := finalGradientMapping[character.InputCoord]
		spawnScene := character.Animation.NewScene("spawn")
		colorCopy := charFinalColor
		_ = spawnScene.AddFrame(character.Symbol, 1, &utils.ColorPair{FG: &colorCopy})
		character.Animation.ActivateSceneRef(spawnScene)
		// Tick once to set the current visual from the scene
		character.Tick()
		e.base.Terminal.SetCharacterVisibility(character, true)
	}

	// Get all characters for swapping
	allCharacters := make([]*engine.EffectCharacter, 0)
	for _, char := range e.base.Terminal.GetCharacters(true, false, false, false, engine.TopToBottomLeftToRight) {
		allCharacters = append(allCharacters, char)
	}

	// Shuffle for random selection
	for i := len(allCharacters) - 1; i > 0; i-- {
		j := utils.RandIntn(i + 1)
		allCharacters[i], allCharacters[j] = allCharacters[j], allCharacters[i]
	}

	// Create gradients
	correctingGradient, _ := utils.NewGradient([]utils.Color{e.errorColor, e.correctColor}, []int{10}, false)

	blockSymbol := "▓"
	blockWipeStart := []string{"▁", "▂", "▃", "▄", "▅", "▆", "▇", "█"}
	blockWipeEnd := []string{"▇", "▆", "▅", "▄", "▃", "▂", "▁"}

	// Create error pairs
	numPairs := int(e.errorPairs * float64(len(e.base.Terminal.GetCharacters(true, false, false, false, engine.TopToBottomLeftToRight))))
	for i := 0; i < numPairs; i++ {
		if len(allCharacters) < 2 {
			break
		}

		// Pop two random characters
		char1 := allCharacters[0]
		allCharacters = allCharacters[1:]
		char2 := allCharacters[0]
		allCharacters = allCharacters[1:]

		// Swap their positions
		char1.Motion.SetCoordinate(char2.InputCoord)
		char1.Coord = char2.InputCoord
		char2.Motion.SetCoordinate(char1.InputCoord)
		char2.Coord = char1.InputCoord

		// Create paths back to original positions
		char1InputPath, _ := char1.Motion.NewPath(e.movementSpeed, nil, nil, 0, false, "input_coord")
		char1InputPath.AddWaypoint(char1.InputCoord)

		char2InputPath, _ := char2.Motion.NewPath(e.movementSpeed, nil, nil, 0, false, "input_coord")
		char2InputPath.AddWaypoint(char2.InputCoord)

		e.swapped = append(e.swapped, [2]*engine.EffectCharacter{char1, char2})

		// Create animation scenes for both characters
		for _, character := range []*engine.EffectCharacter{char1, char2} {
			charFinalColor := finalGradientMapping[character.InputCoord]
			inputPath := character.Motion.Paths["input_coord"]

			// First block wipe scene
			firstBlockWipe := character.Animation.NewScene("first_block_wipe")
			for _, block := range blockWipeStart {
				colorCopy := e.errorColor
				_ = firstBlockWipe.AddFrame(block, 3, &utils.ColorPair{FG: &colorCopy})
			}

			// Last block wipe scene
			lastBlockWipe := character.Animation.NewScene("last_block_wipe")
			for _, block := range blockWipeEnd {
				colorCopy := e.correctColor
				_ = lastBlockWipe.AddFrame(block, 3, &utils.ColorPair{FG: &colorCopy})
			}

			// Initial scene - show character in error color
			initialScene := character.Animation.NewScene("initial")
			errorColorCopy := e.errorColor
			_ = initialScene.AddFrame(character.Symbol, 1, &utils.ColorPair{FG: &errorColorCopy})
			character.Animation.ActivateSceneRef(initialScene)

			// Error scene - flashing block/character
			errorScene := character.Animation.NewScene("error")
			white := mustColors("ffffff")[0]
			for j := 0; j < 10; j++ {
				errorColorCopy := e.errorColor
				_ = errorScene.AddFrame(blockSymbol, 3, &utils.ColorPair{FG: &errorColorCopy})
				whiteCopy := white
				_ = errorScene.AddFrame(character.Symbol, 3, &utils.ColorPair{FG: &whiteCopy})
			}

			// Correcting scene - gradient while moving
			correctingScene := character.Animation.NewScene("correcting")
			symbols := make([]string, len(correctingGradient.Spectrum))
			for j := range symbols {
				symbols[j] = "█"
			}
			_ = correctingScene.ApplyGradientToSymbols(symbols, 3, correctingGradient, nil)

			// Final scene - fade from correct color to final color
			finalScene := character.Animation.NewScene("final")
			charFinalGradient, _ := utils.NewGradient([]utils.Color{e.correctColor, charFinalColor}, []int{10}, false)
			finalSymbols := make([]string, len(charFinalGradient.Spectrum))
			for j := range finalSymbols {
				finalSymbols[j] = character.Symbol
			}
			_ = finalScene.ApplyGradientToSymbols(finalSymbols, 3, charFinalGradient, nil)

			// Event chain: error -> first_block_wipe -> (correcting + activate path) -> path complete -> last_block_wipe -> final
			character.EventHandler.RegisterEvent(
				engine.EventSceneComplete,
				errorScene,
				engine.ActionActivateScene,
				firstBlockWipe,
			)
			character.EventHandler.RegisterEvent(
				engine.EventSceneComplete,
				firstBlockWipe,
				engine.ActionActivateScene,
				correctingScene,
			)
			character.EventHandler.RegisterEvent(
				engine.EventSceneComplete,
				firstBlockWipe,
				engine.ActionActivatePath,
				inputPath,
			)
			// Set layer when path activates
			layer1 := 1
			character.EventHandler.RegisterEvent(
				engine.EventPathActivated,
				inputPath,
				engine.ActionSetLayer,
				layer1,
			)
			// Reset layer when path completes
			layer0 := 0
			character.EventHandler.RegisterEvent(
				engine.EventPathComplete,
				inputPath,
				engine.ActionSetLayer,
				layer0,
			)
			character.EventHandler.RegisterEvent(
				engine.EventPathComplete,
				inputPath,
				engine.ActionActivateScene,
				lastBlockWipe,
			)
			character.EventHandler.RegisterEvent(
				engine.EventSceneComplete,
				lastBlockWipe,
				engine.ActionActivateScene,
				finalScene,
			)
		}
	}
}

func (e *ErrorCorrect) Next() (string, bool) {
	// Activate next swap pair if delay is 0
	if len(e.swapped) > 0 && e.swapDelay == 0 {
		nextPair := e.swapped[0]
		e.swapped = e.swapped[1:]
		for _, char := range nextPair {
			char.Animation.ActivateScene("error")
			e.activeCharacters[char] = struct{}{}
		}
		e.swapDelay = e.swapDelayFrames
	} else if e.swapDelay > 0 {
		e.swapDelay--
	}

	// Tick all active characters
	for char := range e.activeCharacters {
		char.Tick()
		if !char.IsActive() {
			delete(e.activeCharacters, char)
		}
	}

	if len(e.activeCharacters) > 0 || len(e.swapped) > 0 {
		return e.base.Terminal.GetFormattedOutputString(), true
	}

	return e.base.Terminal.GetFormattedOutputString(), false
}

func (e *ErrorCorrect) CanvasHeight() int {
	return e.base.CanvasHeight()
}
