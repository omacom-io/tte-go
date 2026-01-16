package effects

import (
	"tte-go/internal/config"
	"tte-go/internal/engine"
	"tte-go/internal/utils"
)

// Smoke creates an effect where smoke floods the canvas colorizing characters.
type Smoke struct {
	base *BaseEffect

	pendingChars     []*engine.EffectCharacter
	activeCharacters map[*engine.EffectCharacter]struct{}

	// Config
	startingColor          utils.Color
	smokeSymbols           []string
	smokeGradientStops     []utils.Color
	finalGradientStops     []utils.Color
	finalGradientSteps     []int
	finalGradientDirection utils.GradientDirection
}

func NewSmoke(input string, cfg config.TerminalConfig) engine.Effect {
	base := NewBaseEffect(input, cfg)

	// Config defaults
	startingColor := mustColors("7A7A7A")[0]
	smokeSymbols := []string{"░", "▒", "▓", "▒", "░"}
	smokeGradientStops := mustColors("242424", "FFFFFF")
	finalGradientStops := mustColors("8A008A", "00D1FF", "FFFFFF")
	finalGradientSteps := []int{12}
	finalGradientDirection := utils.GradientVertical

	s := &Smoke{
		base:                   base,
		pendingChars:           make([]*engine.EffectCharacter, 0),
		activeCharacters:       make(map[*engine.EffectCharacter]struct{}),
		startingColor:          startingColor,
		smokeSymbols:           smokeSymbols,
		smokeGradientStops:     smokeGradientStops,
		finalGradientStops:     finalGradientStops,
		finalGradientSteps:     finalGradientSteps,
		finalGradientDirection: finalGradientDirection,
	}

	s.build()
	return s
}

func (s *Smoke) build() {
	// Build final gradient mapping
	finalGradient, _ := utils.NewGradient(s.finalGradientStops, s.finalGradientSteps, false)
	finalGradientMapping, _ := finalGradient.BuildCoordinateColorMapping(
		s.base.Canvas.TextBottom,
		s.base.Canvas.TextTop,
		s.base.Canvas.TextLeft,
		s.base.Canvas.TextRight,
		s.finalGradientDirection,
	)

	// Build smoke gradient
	smokeGradient, _ := utils.NewGradient(s.smokeGradientStops, []int{8}, false)

	// Get all characters using BFS from a random starting point
	allChars := s.base.Terminal.GetCharacters(true, true, true, false, engine.TopToBottomLeftToRight)
	if len(allChars) == 0 {
		return
	}

	// Build BFS order from center
	visited := make(map[*engine.EffectCharacter]bool)
	queue := make([]*engine.EffectCharacter, 0)

	// Find character closest to center
	center := s.base.Canvas.Center
	var startChar *engine.EffectCharacter
	minDist := float64(1000000)
	for _, char := range allChars {
		dx := float64(char.InputCoord.Col - center.Col)
		dy := float64(char.InputCoord.Row - center.Row)
		dist := dx*dx + dy*dy
		if dist < minDist {
			minDist = dist
			startChar = char
		}
	}

	if startChar == nil {
		startChar = allChars[0]
	}

	queue = append(queue, startChar)
	visited[startChar] = true

	for len(queue) > 0 {
		current := queue[0]
		queue = queue[1:]
		s.pendingChars = append(s.pendingChars, current)

		// Add unvisited neighbors
		for _, neighbor := range current.Neighbors {
			if !visited[neighbor] {
				visited[neighbor] = true
				queue = append(queue, neighbor)
			}
		}
	}

	// Add any remaining unvisited chars (disconnected regions)
	for _, char := range allChars {
		if !visited[char] {
			s.pendingChars = append(s.pendingChars, char)
		}
	}

	// Set up each character with smoke and final scenes
	for _, character := range s.pendingChars {
		charFinalColor := finalGradientMapping[character.InputCoord]

		// Initial appearance with starting color
		startingColorCopy := s.startingColor
		character.SetVisual(engine.CharacterVisual{
			Symbol: character.Symbol,
			Colors: &utils.ColorPair{FG: &startingColorCopy},
		})

		// Smoke scene - animate through smoke symbols with smoke gradient
		smokeScene := character.Animation.NewScene("smoke")
		for i, symbol := range s.smokeSymbols {
			colorIdx := i * len(smokeGradient.Spectrum) / len(s.smokeSymbols)
			if colorIdx >= len(smokeGradient.Spectrum) {
				colorIdx = len(smokeGradient.Spectrum) - 1
			}
			colorCopy := smokeGradient.Spectrum[colorIdx]
			_ = smokeScene.AddFrame(symbol, 3, &utils.ColorPair{FG: &colorCopy})
		}

		// Final scene - fade to final color
		finalScene := character.Animation.NewScene("final")
		finalColorGradient, _ := utils.NewGradient([]utils.Color{smokeGradient.Spectrum[len(smokeGradient.Spectrum)-1], charFinalColor}, []int{8}, false)
		for _, color := range finalColorGradient.Spectrum {
			colorCopy := color
			_ = finalScene.AddFrame(character.Symbol, 3, &utils.ColorPair{FG: &colorCopy})
		}

		// Event: smoke complete -> activate final
		character.EventHandler.RegisterEvent(
			engine.EventSceneComplete,
			smokeScene,
			engine.ActionActivateScene,
			finalScene,
		)
	}
}

func (s *Smoke) Next() (string, bool) {
	// Add characters to active set
	charsToAdd := 3 + utils.RandIntn(5) // 3-7 per frame
	for i := 0; i < charsToAdd && len(s.pendingChars) > 0; i++ {
		nextChar := s.pendingChars[0]
		s.pendingChars = s.pendingChars[1:]
		s.base.Terminal.SetCharacterVisibility(nextChar, true)
		nextChar.Animation.ActivateScene("smoke")
		s.activeCharacters[nextChar] = struct{}{}
	}

	// Tick all active characters
	for char := range s.activeCharacters {
		char.Tick()
		if !char.IsActive() {
			delete(s.activeCharacters, char)
		}
	}

	if len(s.pendingChars) > 0 || len(s.activeCharacters) > 0 {
		return s.base.Terminal.GetFormattedOutputString(), true
	}

	return s.base.Terminal.GetFormattedOutputString(), false
}

func (s *Smoke) CanvasHeight() int {
	return s.base.CanvasHeight()
}
