package effects

import (
	"tte-go/internal/config"
	"tte-go/internal/engine"
	"tte-go/internal/utils"
)

// Smoke creates an effect where smoke floods the canvas colorizing characters.
type Smoke struct {
	base *BaseEffect

	activeCharacters map[*engine.EffectCharacter]struct{}

	// BFS flood state
	frontier  []*engine.EffectCharacter
	visited   map[*engine.EffectCharacter]bool
	treeEdges map[*engine.EffectCharacter][]*engine.EffectCharacter // spanning tree edges

	// Config
	startingColor          utils.Color
	smokeSymbols           []string
	smokeGradientStops     []utils.Color
	finalGradientStops     []utils.Color
	finalGradientSteps     []int
	finalGradientDirection utils.GradientDirection
	useWholeCanvas         bool
}

func NewSmoke(input string, cfg config.TerminalConfig) engine.Effect {
	base := NewBaseEffect(input, cfg)

	// Config defaults matching Python
	startingColor := mustColors("7A7A7A")[0]
	smokeSymbols := []string{"░", "▒", "▓", "▒", "░"}
	smokeGradientStops := mustColors("242424", "FFFFFF")
	finalGradientStops := mustColors("8A008A", "00D1FF", "FFFFFF")
	finalGradientSteps := []int{12}
	finalGradientDirection := utils.GradientVertical

	s := &Smoke{
		base:                   base,
		activeCharacters:       make(map[*engine.EffectCharacter]struct{}),
		frontier:               make([]*engine.EffectCharacter, 0),
		visited:                make(map[*engine.EffectCharacter]bool),
		treeEdges:              make(map[*engine.EffectCharacter][]*engine.EffectCharacter),
		startingColor:          startingColor,
		smokeSymbols:           smokeSymbols,
		smokeGradientStops:     smokeGradientStops,
		finalGradientStops:     finalGradientStops,
		finalGradientSteps:     finalGradientSteps,
		finalGradientDirection: finalGradientDirection,
		useWholeCanvas:         false,
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

	// Build smoke gradient: smoke_stops + reversed(final_stops)
	// Python: Gradient(*smoke_gradient_stops, *final_gradient_stops[::-1], steps=(3, 4))
	smokeStops := make([]utils.Color, 0, len(s.smokeGradientStops)+len(s.finalGradientStops))
	smokeStops = append(smokeStops, s.smokeGradientStops...)
	// Append reversed final gradient stops
	for i := len(s.finalGradientStops) - 1; i >= 0; i-- {
		smokeStops = append(smokeStops, s.finalGradientStops[i])
	}
	// Steps: 3 for first segment, 4 for rest (approximation)
	smokeSteps := make([]int, len(smokeStops)-1)
	for i := range smokeSteps {
		if i == 0 {
			smokeSteps[i] = 3
		} else {
			smokeSteps[i] = 4
		}
	}
	smokeGradient, _ := utils.NewGradient(smokeStops, smokeSteps, false)

	// Get all characters (including fill chars)
	allChars := s.base.Terminal.GetCharacters(true, true, true, false, engine.TopToBottomLeftToRight)
	if len(allChars) == 0 {
		return
	}

	// Python: All characters start VISIBLE with starting color
	for _, character := range allChars {
		s.base.Terminal.SetCharacterVisibility(character, true)
		startingColorCopy := s.startingColor
		character.SetVisual(engine.CharacterVisual{
			Symbol: character.Symbol,
			Colors: &utils.ColorPair{FG: &startingColorCopy},
		})
	}

	// Build spanning tree using simplified Prim's algorithm for organic spread
	// Returns the starting character used for the tree
	startChar := s.buildSpanningTree(allChars)

	// Set up scenes for each character
	blackColor := mustColors("000000")[0]
	for _, character := range allChars {
		charFinalColor, ok := finalGradientMapping[character.InputCoord]
		if !ok {
			charFinalColor = blackColor
		}

		// Paint scene: final_gradient_stops -> char's final color
		// Python: Gradient(*final_gradient_stops, char_final_color, steps=5)
		paintStops := make([]utils.Color, 0, len(s.finalGradientStops)+1)
		paintStops = append(paintStops, s.finalGradientStops...)
		paintStops = append(paintStops, charFinalColor)
		paintSteps := make([]int, len(paintStops)-1)
		for i := range paintSteps {
			paintSteps[i] = 5
		}
		paintGradient, _ := utils.NewGradient(paintStops, paintSteps, false)

		paintScene := character.Animation.NewScene("paint")
		// Apply gradient across frames with character's symbol
		for _, color := range paintGradient.Spectrum {
			colorCopy := color
			_ = paintScene.AddFrame(character.Symbol, 5, &utils.ColorPair{FG: &colorCopy})
		}

		// Smoke scene: cycle through smoke symbols with smoke gradient colors
		// Python: apply_gradient_to_symbols(smoke_symbols, duration=3, fg_gradient=smoke_gradient)
		smokeScene := character.Animation.NewScene("smoke")
		// Drive frames by gradient spectrum, cycling through symbols
		for i, color := range smokeGradient.Spectrum {
			symbolIdx := i % len(s.smokeSymbols)
			colorCopy := color
			_ = smokeScene.AddFrame(s.smokeSymbols[symbolIdx], 3, &utils.ColorPair{FG: &colorCopy})
		}

		// Event: smoke complete -> activate paint
		character.EventHandler.RegisterEvent(
			engine.EventSceneComplete,
			smokeScene,
			engine.ActionActivateScene,
			paintScene,
		)
	}

	// Activate smoke on starting char immediately (use same start as spanning tree)
	startChar.Animation.ActivateScene("smoke")
	s.activeCharacters[startChar] = struct{}{}
	s.visited[startChar] = true
	s.frontier = append(s.frontier, startChar)
}

// buildSpanningTree creates a randomized spanning tree for organic smoke spread
// Returns the starting character used for the tree
func (s *Smoke) buildSpanningTree(chars []*engine.EffectCharacter) *engine.EffectCharacter {
	if len(chars) == 0 {
		return nil
	}

	// Build coord -> char lookup
	lookup := make(map[utils.Coord]*engine.EffectCharacter)
	for _, c := range chars {
		lookup[c.InputCoord] = c
	}

	// Simple randomized Prim's: start from random char, grow tree
	inTree := make(map[*engine.EffectCharacter]bool)
	startIdx := utils.RandIntn(len(chars))
	start := chars[startIdx]
	inTree[start] = true

	// Edge candidates: (from, to) pairs
	type edge struct {
		from, to *engine.EffectCharacter
		weight   float64
	}
	edges := make([]edge, 0)

	// Add initial edges from start
	addEdges := func(c *engine.EffectCharacter) {
		for _, neighbor := range c.Neighbors {
			if neighbor != nil && !inTree[neighbor] {
				edges = append(edges, edge{from: c, to: neighbor, weight: utils.RandFloat64()})
			}
		}
	}
	addEdges(start)

	for len(edges) > 0 {
		// Pick random edge (weighted selection approximation - just pick random)
		idx := utils.RandIntn(len(edges))
		e := edges[idx]
		// Remove this edge
		edges[idx] = edges[len(edges)-1]
		edges = edges[:len(edges)-1]

		if inTree[e.to] {
			continue
		}

		// Add to tree - store edge bidirectionally for BFS from any start
		inTree[e.to] = true
		s.treeEdges[e.from] = append(s.treeEdges[e.from], e.to)
		s.treeEdges[e.to] = append(s.treeEdges[e.to], e.from)

		// Add new edges from this node
		addEdges(e.to)
	}

	return start
}

func (s *Smoke) Next() (string, bool) {
	// BFS wave expansion using spanning tree edges
	if len(s.frontier) > 0 {
		nextFrontier := make([]*engine.EffectCharacter, 0)

		for _, char := range s.frontier {
			// Get tree children (not all neighbors)
			children := s.treeEdges[char]
			for _, child := range children {
				if !s.visited[child] {
					s.visited[child] = true
					child.Animation.ActivateScene("smoke")
					s.activeCharacters[child] = struct{}{}
					nextFrontier = append(nextFrontier, child)
				}
			}
		}

		s.frontier = nextFrontier
	}

	// Tick all active characters
	for char := range s.activeCharacters {
		char.Tick()
		if !char.IsActive() {
			delete(s.activeCharacters, char)
		}
	}

	if len(s.frontier) > 0 || len(s.activeCharacters) > 0 {
		return s.base.Terminal.GetFormattedOutputString(), true
	}

	return s.base.Terminal.GetFormattedOutputString(), false
}

func (s *Smoke) CanvasHeight() int {
	return s.base.CanvasHeight()
}

func (s *Smoke) CanvasWidth() int {
	return s.base.CanvasWidth()
}
