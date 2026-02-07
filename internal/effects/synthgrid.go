package effects

import (
	"tte-go/internal/config"
	"tte-go/internal/engine"
	"tte-go/internal/utils"
)

// SynthGrid creates a grid which fills with characters dissolving into the final text.
type SynthGrid struct {
	base                  *BaseEffect
	gridLines             []*gridLine
	pendingGroups         []characterGroup
	activeGroups          map[int]int // groupNumber -> activeCount
	phase                 string
	textGradient          *utils.Gradient
	textGradientMapping   map[utils.Coord]utils.Color
	maxActiveBlocks       float64
	textGenerationSymbols []string
}

type gridLine struct {
	characters []*engine.EffectCharacter
	collapsed  []*engine.EffectCharacter
	extended   []*engine.EffectCharacter
	direction  string
	extendRate int
}

type characterGroup struct {
	number     int
	characters []*engine.EffectCharacter
}

func NewSynthGrid(input string, cfg config.TerminalConfig) engine.Effect {
	base := NewBaseEffect(input, cfg)

	// Configuration
	gridGradientStops := mustColors("#CC00CC", "#ffffff")
	textGradientStops := mustColors("#8A008A", "#00D1FF", "#FFFFFF")
	gridRowSymbol := "─"
	gridColumnSymbol := "│"
	textGenerationSymbols := []string{"░", "▒", "▓"}
	maxActiveBlocks := 0.1

	// Build gradients
	gridGradient, _ := utils.NewGradient(gridGradientStops, []int{12}, false)
	gridMapping, _ := gridGradient.BuildCoordinateColorMapping(
		1, base.Canvas.Height, 1, base.Canvas.Width,
		utils.GradientDiagonal,
	)

	textGradient, _ := utils.NewGradient(textGradientStops, []int{12}, false)
	textGradientMapping, _ := textGradient.BuildCoordinateColorMapping(
		base.Canvas.TextBottom, base.Canvas.TextTop,
		base.Canvas.TextLeft, base.Canvas.TextRight,
		utils.GradientVertical,
	)

	s := &SynthGrid{
		base:                  base,
		gridLines:             make([]*gridLine, 0),
		pendingGroups:         make([]characterGroup, 0),
		activeGroups:          make(map[int]int),
		phase:                 "grid_expand",
		textGradient:          textGradient,
		textGradientMapping:   textGradientMapping,
		maxActiveBlocks:       maxActiveBlocks,
		textGenerationSymbols: textGenerationSymbols,
	}

	// Create border grid lines
	// Bottom horizontal line
	s.gridLines = append(s.gridLines, s.createGridLine(gridRowSymbol, "horizontal", 1, gridMapping))
	// Top horizontal line
	s.gridLines = append(s.gridLines, s.createGridLine(gridRowSymbol, "horizontal", base.Canvas.Height, gridMapping))
	// Left vertical line
	s.gridLines = append(s.gridLines, s.createGridLine(gridColumnSymbol, "vertical", 1, gridMapping))
	// Right vertical line
	s.gridLines = append(s.gridLines, s.createGridLine(gridColumnSymbol, "vertical", base.Canvas.Width, gridMapping))

	// Calculate grid gaps
	columnGap, rowGap := s.findGridGaps()

	// Create interior horizontal lines
	for row := 1 + rowGap; row < base.Canvas.Height; row += max(rowGap, 1) {
		if base.Canvas.Height-row < 2 {
			continue
		}
		s.gridLines = append(s.gridLines, s.createGridLine(gridRowSymbol, "horizontal", row, gridMapping))
	}

	// Create interior vertical lines
	for col := 1 + columnGap; col < base.Canvas.Width; col += max(columnGap, 1) {
		if base.Canvas.Width-col < 2 {
			continue
		}
		s.gridLines = append(s.gridLines, s.createGridLine(gridColumnSymbol, "vertical", col, gridMapping))
	}

	// Create character groups based on grid blocks
	s.createCharacterGroups(columnGap, rowGap)

	// Setup character animations
	for _, group := range s.pendingGroups {
		s.activeGroups[group.number] = 0
		for _, char := range group.characters {
			s.setupCharacterAnimation(char, group.number)
		}
	}

	// Shuffle groups
	shuffleGroups(s.pendingGroups)

	// Hide all text characters initially
	for _, char := range base.Characters {
		char.Visible = false
	}

	return s
}

func shuffleGroups(groups []characterGroup) {
	n := len(groups)
	for i := n - 1; i > 0; i-- {
		j := utils.RandIntn(i + 1)
		groups[i], groups[j] = groups[j], groups[i]
	}
}

func (s *SynthGrid) createGridLine(symbol, direction string, position int, colorMapping map[utils.Coord]utils.Color) *gridLine {
	gl := &gridLine{
		direction:  direction,
		characters: make([]*engine.EffectCharacter, 0),
		collapsed:  make([]*engine.EffectCharacter, 0),
		extended:   make([]*engine.EffectCharacter, 0),
		extendRate: 3,
	}

	if direction == "vertical" {
		gl.extendRate = 1
	}

	if direction == "horizontal" {
		for col := 1; col <= s.base.Canvas.Width; col++ {
			coord := utils.Coord{Row: position, Col: col}
			char := engine.NewEffectCharacter(symbol, coord)
			char.IsAddedCharacter = true
			char.Layer = 2
			char.Visible = false

			// Set color from gradient
			if color, ok := colorMapping[coord]; ok {
				visual := char.Visual()
				visual.Colors = &utils.ColorPair{FG: &color}
				char.SetVisual(visual)
			}

			s.base.Terminal.AddCharacter(char)
			gl.characters = append(gl.characters, char)
			gl.collapsed = append(gl.collapsed, char)
		}
	} else { // vertical
		for row := 1; row <= s.base.Canvas.Height; row++ {
			coord := utils.Coord{Row: row, Col: position}
			char := engine.NewEffectCharacter(symbol, coord)
			char.IsAddedCharacter = true
			char.Layer = 2
			char.Visible = false

			// Set color from gradient
			if color, ok := colorMapping[coord]; ok {
				visual := char.Visual()
				visual.Colors = &utils.ColorPair{FG: &color}
				char.SetVisual(visual)
			}

			s.base.Terminal.AddCharacter(char)
			gl.characters = append(gl.characters, char)
			gl.collapsed = append(gl.collapsed, char)
		}
	}

	return gl
}

func (s *SynthGrid) findGridGaps() (columnGap, rowGap int) {
	// Calculate gaps based on canvas dimensions
	if s.base.Canvas.Height > 2*s.base.Canvas.Width {
		rowGap = s.findEvenGap(s.base.Canvas.Height) + 1
		columnGap = rowGap * 2
	} else {
		columnGap = s.findEvenGap(s.base.Canvas.Width) + 1
		rowGap = columnGap / 2
	}
	return columnGap, max(rowGap, 1)
}

func (s *SynthGrid) findEvenGap(dimension int) int {
	dimension = dimension - 2
	if dimension <= 0 {
		return 0
	}

	// Find gaps that evenly divide the dimension
	var potentialGaps []int
	for i := dimension; i > 4; i-- {
		if dimension%i <= 1 {
			potentialGaps = append(potentialGaps, i)
		}
	}

	if len(potentialGaps) == 0 {
		return 4
	}

	// Find the gap closest to 20% of dimension
	targetGap := dimension / 5
	bestGap := potentialGaps[0]
	bestDiff := absInt(bestGap - targetGap)

	for _, gap := range potentialGaps {
		diff := absInt(gap - targetGap)
		if diff < bestDiff {
			bestGap = gap
			bestDiff = diff
		}
	}

	return bestGap
}

func (s *SynthGrid) createCharacterGroups(columnGap, rowGap int) {
	// Build a map of input coord to character for fast lookup
	inputCharMap := make(map[utils.Coord]*engine.EffectCharacter)
	for _, char := range s.base.Characters {
		inputCharMap[char.InputCoord] = char
	}

	// Build row and column indexes for blocks
	var rowIndexes []int
	var columnIndexes []int

	for row := 1 + rowGap; row < s.base.Canvas.Height; row += max(rowGap, 1) {
		if s.base.Canvas.Height-row >= 2 {
			rowIndexes = append(rowIndexes, row)
		}
	}
	rowIndexes = append(rowIndexes, s.base.Canvas.Height+1)

	for col := 1 + columnGap; col < s.base.Canvas.Width; col += max(columnGap, 1) {
		if s.base.Canvas.Width-col >= 2 {
			columnIndexes = append(columnIndexes, col)
		}
	}
	columnIndexes = append(columnIndexes, s.base.Canvas.Width+1)

	// Create groups for each block
	groupNum := 0
	prevRow := 1
	for _, rowIdx := range rowIndexes {
		prevCol := 1
		for _, colIdx := range columnIndexes {
			// Find characters in this block
			var charsInBlock []*engine.EffectCharacter
			for row := prevRow; row < rowIdx; row++ {
				for col := prevCol; col < colIdx; col++ {
					coord := utils.Coord{Row: row, Col: col}
					if char, ok := inputCharMap[coord]; ok {
						charsInBlock = append(charsInBlock, char)
					}
				}
			}

			if len(charsInBlock) > 0 {
				s.pendingGroups = append(s.pendingGroups, characterGroup{
					number:     groupNum,
					characters: charsInBlock,
				})
				groupNum++
			}
			prevCol = colIdx
		}
		prevRow = rowIdx
	}
}

func (s *SynthGrid) setupCharacterAnimation(char *engine.EffectCharacter, groupNumber int) {
	// Create dissolve scene with random symbols transitioning to final character
	dissolveScene := char.Animation.NewScene("dissolve")
	spectrum := s.textGradient.Spectrum

	frameCount := 15 + utils.RandIntn(16) // 15-30 frames
	for i := 0; i < frameCount; i++ {
		symbol := s.textGenerationSymbols[utils.RandIntn(len(s.textGenerationSymbols))]
		color := spectrum[utils.RandIntn(len(spectrum))]
		dissolveScene.AddFrame(symbol, 2, &utils.ColorPair{FG: &color})
	}

	// Final frame with character's symbol and color
	if char.Symbol == " " {
		dissolveScene.AddFrame(char.Symbol, 1, nil)
	} else {
		finalColor := s.textGradientMapping[char.InputCoord]
		dissolveScene.AddFrame(char.Symbol, 1, &utils.ColorPair{FG: &finalColor})
	}

	char.Animation.ActivateSceneRef(dissolveScene)

	// Register event to track group completion
	// Use a callback that decrements the group counter
	gn := groupNumber // capture for closure
	char.EventHandler.RegisterEvent(
		engine.EventSceneComplete,
		dissolveScene,
		engine.ActionCallback,
		engine.Callback{
			Fn: func(c *engine.EffectCharacter, args ...any) {
				s.activeGroups[gn]--
			},
		},
	)
}

func (gl *gridLine) extend() {
	for i := 0; i < gl.extendRate && len(gl.collapsed) > 0; i++ {
		char := gl.collapsed[0]
		gl.collapsed = gl.collapsed[1:]
		char.Visible = true
		gl.extended = append(gl.extended, char)
	}
}

func (gl *gridLine) collapse() {
	// On first collapse call, reverse extended list
	if len(gl.collapsed) == 0 && len(gl.extended) > 0 {
		// Reverse extended
		for i, j := 0, len(gl.extended)-1; i < j; i, j = i+1, j-1 {
			gl.extended[i], gl.extended[j] = gl.extended[j], gl.extended[i]
		}
	}

	for i := 0; i < gl.extendRate && len(gl.extended) > 0; i++ {
		char := gl.extended[0]
		gl.extended = gl.extended[1:]
		char.Visible = false
		gl.collapsed = append(gl.collapsed, char)
	}
}

func (gl *gridLine) isExtended() bool {
	return len(gl.collapsed) == 0
}

func (gl *gridLine) isCollapsed() bool {
	return len(gl.extended) == 0
}

func (s *SynthGrid) Next() (string, bool) {
	if s.phase == "complete" {
		return "", false
	}

	switch s.phase {
	case "grid_expand":
		allExtended := true
		for _, gl := range s.gridLines {
			if !gl.isExtended() {
				gl.extend()
				allExtended = false
			}
		}
		if allExtended {
			s.phase = "add_chars"
		}

	case "add_chars":
		// Count active groups
		activeGroupCount := 0
		for _, count := range s.activeGroups {
			if count > 0 {
				activeGroupCount++
			}
		}

		// Add new groups if under threshold
		totalGroups := len(s.pendingGroups) + activeGroupCount
		if totalGroups == 0 {
			totalGroups = 1
		}
		maxActive := int(float64(totalGroups) * s.maxActiveBlocks)
		if maxActive < 1 {
			maxActive = 1
		}

		if len(s.pendingGroups) > 0 && activeGroupCount < maxActive {
			group := s.pendingGroups[0]
			s.pendingGroups = s.pendingGroups[1:]
			for _, char := range group.characters {
				char.Visible = true
				s.activeGroups[group.number]++
			}
		}

		// Check if all groups are done
		if len(s.pendingGroups) == 0 {
			allDone := true
			for _, count := range s.activeGroups {
				if count > 0 {
					allDone = false
					break
				}
			}
			if allDone {
				s.phase = "collapse"
			}
		}

	case "collapse":
		allCollapsed := true
		for _, gl := range s.gridLines {
			if !gl.isCollapsed() {
				gl.collapse()
				allCollapsed = false
			}
		}
		if allCollapsed {
			s.phase = "complete"
		}
	}

	// Step all character animations
	for _, char := range s.base.Characters {
		if char.Visible {
			char.Tick()
		}
	}

	// Collect all characters for rendering
	allChars := s.base.Terminal.GetCharacters(true, true, true, true, engine.TopToBottomLeftToRight)
	frame := engine.RenderFrame(s.base.Canvas, allChars)
	return frame, s.phase != "complete"
}

func (s *SynthGrid) CanvasHeight() int {
	return s.base.CanvasHeight()
}

func (s *SynthGrid) CanvasWidth() int {
	return s.base.CanvasWidth()
}

func absInt(x int) int {
	if x < 0 {
		return -x
	}
	return x
}
