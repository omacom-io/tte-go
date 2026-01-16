package effects

import (
	"time"

	"tte-go/internal/config"
	"tte-go/internal/engine"
	"tte-go/internal/utils"
)

var matrixSymbols = []string{
	"2", "5", "9", "8", "Z", "*", ")", ":", ".", "\"", "=", "+", "-", "¦", "|", "_",
	"ｦ", "ｱ", "ｳ", "ｴ", "ｵ", "ｶ", "ｷ", "ｹ", "ｺ", "ｻ", "ｼ", "ｽ", "ｾ", "ｿ",
	"ﾀ", "ﾂ", "ﾃ", "ﾅ", "ﾆ", "ﾇ", "ﾈ", "ﾊ", "ﾋ", "ﾎ", "ﾏ", "ﾐ", "ﾑ", "ﾒ", "ﾓ",
	"ﾔ", "ﾕ", "ﾗ", "ﾘ", "ﾜ",
}

// Matrix creates a Matrix digital rain effect.
type Matrix struct {
	base *BaseEffect

	pendingColumns   []*rainColumn
	activeColumns    []*rainColumn
	fullColumns      []*rainColumn
	activeCharacters map[*engine.EffectCharacter]struct{}

	rainColors      []utils.Color
	columnDelay     int
	resolveDelay    int
	phase           string // "rain", "fill", "resolve"
	rainStart       time.Time
	rainComplete    bool
	finalFrameShown bool

	// Config
	highlightColor         utils.Color
	rainColorGradient      []utils.Color
	rainFallDelayRange     [2]int
	rainColumnDelayRange   [2]int
	rainTime               int // seconds
	symbolSwapChance       float64
	colorSwapChance        float64
	resolveDelayFrames     int
	finalGradientStops     []utils.Color
	finalGradientSteps     []int
	finalGradientFrames    int
	finalGradientDirection utils.GradientDirection
}

type rainColumn struct {
	terminal            *engine.TerminalState
	config              *Matrix
	characters          []*engine.EffectCharacter
	pendingCharacters   []*engine.EffectCharacter
	visibleCharacters   []*engine.EffectCharacter
	rainColors          []utils.Color
	baseRainFallDelay   int
	activeRainFallDelay int
	length              int
	holdTime            int
	phase               string
	columnDropChance    float64
}

func newRainColumn(chars []*engine.EffectCharacter, terminal *engine.TerminalState, config *Matrix) *rainColumn {
	rc := &rainColumn{
		terminal:          terminal,
		config:            config,
		characters:        chars,
		pendingCharacters: make([]*engine.EffectCharacter, 0),
		visibleCharacters: make([]*engine.EffectCharacter, 0),
		rainColors:        config.rainColors,
		columnDropChance:  0.08,
	}
	rc.setupColumn("rain")
	return rc
}

func (rc *rainColumn) setupColumn(phase string) {
	rc.pendingCharacters = rc.pendingCharacters[:0]
	rc.phase = phase
	for _, char := range rc.characters {
		rc.terminal.SetCharacterVisibility(char, false)
		rc.pendingCharacters = append(rc.pendingCharacters, char)
		char.Motion.SetCoordinate(char.InputCoord)
		char.Coord = char.InputCoord
	}
	rc.visibleCharacters = rc.visibleCharacters[:0]

	if phase == "fill" {
		minDelay := max(rc.config.rainFallDelayRange[0]/3, 1)
		maxDelay := max(rc.config.rainFallDelayRange[1]/3, 1)
		rc.baseRainFallDelay = minDelay + utils.RandIntn(maxDelay-minDelay+1)
	} else {
		rc.baseRainFallDelay = rc.config.rainFallDelayRange[0] + utils.RandIntn(rc.config.rainFallDelayRange[1]-rc.config.rainFallDelayRange[0]+1)
	}
	rc.activeRainFallDelay = 0

	if phase == "rain" {
		minLen := max(1, len(rc.characters)/10)
		rc.length = minLen + utils.RandIntn(len(rc.characters)-minLen+1)
	} else {
		rc.length = len(rc.characters)
	}

	rc.holdTime = 0
	if rc.length == len(rc.characters) {
		rc.holdTime = 20 + utils.RandIntn(26) // 20-45
	}
}

func (rc *rainColumn) trimColumn() {
	if len(rc.visibleCharacters) == 0 {
		return
	}
	popped := rc.visibleCharacters[0]
	rc.visibleCharacters = rc.visibleCharacters[1:]
	rc.terminal.SetCharacterVisibility(popped, false)
	if len(rc.visibleCharacters) > 1 {
		rc.fadeLastCharacter()
	}
}

func (rc *rainColumn) dropColumn() {
	remaining := make([]*engine.EffectCharacter, 0)
	for _, char := range rc.visibleCharacters {
		newCoord := utils.Coord{Col: char.Coord.Col, Row: char.Coord.Row - 1}
		char.Motion.SetCoordinate(newCoord)
		char.Coord = newCoord
		if newCoord.Row < rc.terminal.Canvas.Bottom {
			rc.terminal.SetCharacterVisibility(char, false)
		} else {
			remaining = append(remaining, char)
		}
	}
	rc.visibleCharacters = remaining
}

func (rc *rainColumn) fadeLastCharacter() {
	if len(rc.visibleCharacters) == 0 {
		return
	}
	// Pick a color from the darker end of the gradient
	colorIdx := len(rc.rainColors) - 1 - utils.RandIntn(min(3, len(rc.rainColors)))
	if colorIdx < 0 {
		colorIdx = 0
	}
	darkerColor := utils.AdjustBrightness(rc.rainColors[colorIdx], 0.65)
	char := rc.visibleCharacters[0]
	visual := char.Visual()
	visual.Colors = &utils.ColorPair{FG: &darkerColor}
	char.SetVisual(visual)
}

func (rc *rainColumn) resolveChar() *engine.EffectCharacter {
	idx := utils.RandIntn(len(rc.visibleCharacters))
	char := rc.visibleCharacters[idx]
	rc.visibleCharacters = append(rc.visibleCharacters[:idx], rc.visibleCharacters[idx+1:]...)
	return char
}

func (rc *rainColumn) tick() {
	if rc.activeRainFallDelay == 0 {
		if len(rc.pendingCharacters) > 0 {
			nextChar := rc.pendingCharacters[0]
			rc.pendingCharacters = rc.pendingCharacters[1:]

			// Set random symbol with highlight color
			symbol := matrixSymbols[utils.RandIntn(len(matrixSymbols))]
			highlightColor := rc.config.highlightColor
			nextChar.SetVisual(engine.CharacterVisual{
				Symbol: symbol,
				Colors: &utils.ColorPair{FG: &highlightColor},
			})

			// Remove highlight from previous character
			if len(rc.visibleCharacters) > 0 {
				prevChar := rc.visibleCharacters[len(rc.visibleCharacters)-1]
				prevVisual := prevChar.Visual()
				rainColor := rc.rainColors[utils.RandIntn(len(rc.rainColors))]
				prevVisual.Colors = &utils.ColorPair{FG: &rainColor}
				prevChar.SetVisual(prevVisual)
			}

			rc.terminal.SetCharacterVisibility(nextChar, true)
			rc.visibleCharacters = append(rc.visibleCharacters, nextChar)
		} else if len(rc.visibleCharacters) > 0 {
			// Remove highlight from bottom character
			bottomChar := rc.visibleCharacters[len(rc.visibleCharacters)-1]
			bottomVisual := bottomChar.Visual()
			if bottomVisual.Colors != nil && bottomVisual.Colors.FG != nil && *bottomVisual.Colors.FG == rc.config.highlightColor {
				rainColor := rc.rainColors[utils.RandIntn(len(rc.rainColors))]
				bottomVisual.Colors = &utils.ColorPair{FG: &rainColor}
				bottomChar.SetVisual(bottomVisual)
			}

			if rc.holdTime > 0 {
				rc.holdTime--
			} else if rc.phase == "rain" {
				if utils.RandFloat64() < rc.columnDropChance {
					rc.dropColumn()
				}
				rc.trimColumn()
			}
		}

		// Trim if longer than preset length
		if len(rc.visibleCharacters) > rc.length {
			rc.trimColumn()
		}
		rc.activeRainFallDelay = rc.baseRainFallDelay
	} else {
		rc.activeRainFallDelay--
	}

	// Randomly swap symbols and colors
	for _, char := range rc.visibleCharacters {
		visual := char.Visual()
		if utils.RandFloat64() < rc.config.symbolSwapChance {
			visual.Symbol = matrixSymbols[utils.RandIntn(len(matrixSymbols))]
		}
		if utils.RandFloat64() < rc.config.colorSwapChance {
			rainColor := rc.rainColors[utils.RandIntn(len(rc.rainColors))]
			visual.Colors = &utils.ColorPair{FG: &rainColor}
		}
		char.SetVisual(visual)
	}
}

func NewMatrix(input string, cfg config.TerminalConfig) engine.Effect {
	base := NewBaseEffect(input, cfg)

	// Config defaults
	highlightColor := mustColors("dbffdb")[0]
	rainColorGradient := mustColors("92be92", "185318")
	finalGradientStops := mustColors("92be92", "336b33")
	finalGradientSteps := []int{12}
	finalGradientDirection := utils.GradientRadial

	// Build rain colors gradient
	rainGradient, _ := utils.NewGradient(rainColorGradient, []int{6}, false)

	m := &Matrix{
		base:                   base,
		pendingColumns:         make([]*rainColumn, 0),
		activeColumns:          make([]*rainColumn, 0),
		fullColumns:            make([]*rainColumn, 0),
		activeCharacters:       make(map[*engine.EffectCharacter]struct{}),
		rainColors:             rainGradient.Spectrum,
		columnDelay:            0,
		resolveDelay:           3,
		phase:                  "rain",
		rainComplete:           false,
		finalFrameShown:        false,
		highlightColor:         highlightColor,
		rainColorGradient:      rainColorGradient,
		rainFallDelayRange:     [2]int{2, 15},
		rainColumnDelayRange:   [2]int{3, 9},
		rainTime:               15,
		symbolSwapChance:       0.005,
		colorSwapChance:        0.001,
		resolveDelayFrames:     3,
		finalGradientStops:     finalGradientStops,
		finalGradientSteps:     finalGradientSteps,
		finalGradientFrames:    3,
		finalGradientDirection: finalGradientDirection,
	}

	m.build()
	m.rainStart = time.Now()
	return m
}

func (m *Matrix) build() {
	// Build final gradient mapping
	finalGradient, _ := utils.NewGradient(m.finalGradientStops, m.finalGradientSteps, false)
	finalGradientMapping, _ := finalGradient.BuildCoordinateColorMapping(
		m.base.Canvas.TextBottom,
		m.base.Canvas.TextTop,
		m.base.Canvas.TextLeft,
		m.base.Canvas.TextRight,
		m.finalGradientDirection,
	)

	// Create resolve scene for each character
	for _, character := range m.base.Terminal.GetCharacters(true, false, false, false, engine.TopToBottomLeftToRight) {
		charFinalColor := finalGradientMapping[character.InputCoord]
		resolveGradient, _ := utils.NewGradient([]utils.Color{m.highlightColor, charFinalColor}, []int{8}, false)
		resolveScene := character.Animation.NewScene("resolve")
		for _, color := range resolveGradient.Spectrum {
			colorCopy := color
			_ = resolveScene.AddFrame(character.Symbol, m.finalGradientFrames, &utils.ColorPair{FG: &colorCopy})
		}
	}

	// Create columns (include fill characters)
	columns := m.base.Terminal.GetCharactersGrouped(engine.ColumnLeftToRight, true, true, true, false)
	for _, colChars := range columns {
		// Reverse so we go from top to bottom
		reversed := make([]*engine.EffectCharacter, len(colChars))
		for i, char := range colChars {
			reversed[len(colChars)-1-i] = char
		}
		m.pendingColumns = append(m.pendingColumns, newRainColumn(reversed, m.base.Terminal, m))
	}

	// Shuffle columns
	for i := len(m.pendingColumns) - 1; i > 0; i-- {
		j := utils.RandIntn(i + 1)
		m.pendingColumns[i], m.pendingColumns[j] = m.pendingColumns[j], m.pendingColumns[i]
	}
}

func (m *Matrix) Next() (string, bool) {
	if m.phase == "rain" || m.phase == "fill" {
		if m.columnDelay == 0 {
			if m.phase == "rain" {
				// Add 1-3 columns
				numToAdd := 1 + utils.RandIntn(3)
				for i := 0; i < numToAdd && len(m.pendingColumns) > 0; i++ {
					col := m.pendingColumns[0]
					m.pendingColumns = m.pendingColumns[1:]
					m.activeColumns = append(m.activeColumns, col)
				}
			} else {
				// Fill phase - add all remaining
				for len(m.pendingColumns) > 0 {
					col := m.pendingColumns[0]
					m.pendingColumns = m.pendingColumns[1:]
					m.activeColumns = append(m.activeColumns, col)
				}
			}
			if m.phase == "rain" {
				m.columnDelay = m.rainColumnDelayRange[0] + utils.RandIntn(m.rainColumnDelayRange[1]-m.rainColumnDelayRange[0]+1)
			} else {
				m.columnDelay = 1
			}
		} else {
			m.columnDelay--
		}

		// Tick all active columns
		remaining := make([]*rainColumn, 0)
		for _, col := range m.activeColumns {
			col.tick()

			if len(col.pendingCharacters) == 0 {
				if col.phase == "fill" {
					// Check if already in fullColumns
					found := false
					for _, fc := range m.fullColumns {
						if fc == col {
							found = true
							break
						}
					}
					if !found {
						m.fullColumns = append(m.fullColumns, col)
					}
				} else if len(col.visibleCharacters) == 0 {
					col.setupColumn(m.phase)
					m.pendingColumns = append(m.pendingColumns, col)
				}
			}

			if len(col.visibleCharacters) > 0 {
				remaining = append(remaining, col)
			}
		}
		m.activeColumns = remaining

		// Check if fill phase is complete
		if m.phase == "fill" && len(m.pendingColumns) == 0 {
			allFillComplete := true
			for _, col := range m.activeColumns {
				if len(col.pendingCharacters) > 0 || col.phase != "fill" {
					allFillComplete = false
					break
				}
			}
			if allFillComplete {
				m.phase = "resolve"
				m.activeColumns = nil
			}
		}

		// Check if rain time is up
		if m.phase == "rain" && m.rainTime > 0 && time.Since(m.rainStart).Seconds() > float64(m.rainTime) {
			m.rainComplete = true
			m.phase = "fill"
			for _, col := range m.activeColumns {
				col.holdTime = 0
				col.columnDropChance = 1
			}
			for _, col := range m.pendingColumns {
				col.setupColumn("fill")
			}
		}
	} else if m.phase == "resolve" {
		// Tick full columns and resolve characters
		for _, col := range m.fullColumns {
			col.tick()
			if len(col.visibleCharacters) > 0 {
				if m.resolveDelay == 0 {
					numToResolve := 1 + utils.RandIntn(4)
					for i := 0; i < numToResolve && len(col.visibleCharacters) > 0; i++ {
						nextChar := col.resolveChar()
						if nextChar.Symbol != " " {
							nextChar.Animation.ActivateScene("resolve")
							m.activeCharacters[nextChar] = struct{}{}
						} else {
							m.base.Terminal.SetCharacterVisibility(nextChar, false)
						}
					}
					m.resolveDelay = m.resolveDelayFrames
				} else {
					m.resolveDelay--
				}
			}
		}

		// Remove empty columns from fullColumns
		remaining := make([]*rainColumn, 0)
		for _, col := range m.fullColumns {
			if len(col.visibleCharacters) > 0 {
				remaining = append(remaining, col)
			}
		}
		m.fullColumns = remaining
	}

	// Tick active characters
	for char := range m.activeCharacters {
		char.Tick()
		if !char.IsActive() {
			delete(m.activeCharacters, char)
		}
	}

	// Check if done
	if len(m.fullColumns) > 0 || len(m.activeColumns) > 0 || len(m.activeCharacters) > 0 || len(m.pendingColumns) > 0 || !m.rainComplete {
		return m.base.Terminal.GetFormattedOutputString(), true
	}

	if !m.finalFrameShown {
		m.finalFrameShown = true
		return m.base.Terminal.GetFormattedOutputString(), true
	}

	return m.base.Terminal.GetFormattedOutputString(), false
}

func (m *Matrix) CanvasHeight() int {
	return m.base.CanvasHeight()
}
