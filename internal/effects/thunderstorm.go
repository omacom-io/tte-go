package effects

import (
	"math"
	"time"

	"tte-go/internal/config"
	"tte-go/internal/engine"
	"tte-go/internal/utils"
)

// Thunderstorm creates a thunderstorm effect with rain, lightning, and sparks.
type Thunderstorm struct {
	base                 *BaseEffect
	finalGradientMapping map[utils.Coord]utils.Color

	// Rain system
	rainDrops     []*rainDrop
	availableRain []*rainDrop
	rainDelay     int

	// Lightning strike system
	availableStrike []*engine.EffectCharacter
	pendingStrike   []*engine.EffectCharacter
	activeStrike    []*engine.EffectCharacter
	strikeCoords    []utils.Coord // Track coords for selective glow

	// Spark system
	availableSparks []*sparkChar
	activeSparks    []*sparkChar

	// State
	phase               string
	stormStartTime      time.Time
	stormDuration       time.Duration
	strikeInProgress    bool
	strikeFrame         int
	strikeProgressDelay int
	strikeBranchChance  float64

	// Config
	lightningColor  utils.Color
	raindropSymbols []string
	sparkSymbols    []string
	sparkGlowColor  utils.Color

	// Spawn row for rain/lightning (above text area)
	spawnRow int
}

// rainDrop represents a raindrop with motion
type rainDrop struct {
	char       *engine.EffectCharacter
	startCoord utils.Coord
	endCoord   utils.Coord
	x, y       float64
	speed      float64
	active     bool
}

// sparkChar represents a spark with arc motion
type sparkChar struct {
	char           *engine.EffectCharacter
	x, y           float64 // Current position
	startX, startY float64
	endX, endY     float64
	ctrlX, ctrlY   float64 // Bezier control point
	t              float64 // Progress 0-1
	speed          float64
	active         bool
	holdFrames     int
}

func NewThunderstorm(input string, cfg config.TerminalConfig) engine.Effect {
	base := NewBaseEffect(input, cfg)

	// Calculate spawn row for rain/lightning
	// Use canvas top if it's larger than text, otherwise add padding above text
	const minPadding = 8
	spawnRow := base.Canvas.Top
	if spawnRow <= base.Canvas.TextTop+2 {
		// Canvas is close to text height, use padding above text for spawn
		spawnRow = base.Canvas.TextTop + minPadding
	}

	// Configuration
	lightningColors := mustColors("#68A3E8")
	lightningColor := lightningColors[0]
	raindropSymbols := []string{"\\", ".", ","}
	sparkSymbols := []string{"*", ".", "'"}
	sparkGlowColors := mustColors("#ff4d00")
	stormDuration := 12 * time.Second

	// Build final gradient
	finalGradientStops := mustColors("#8A008A", "#00D1FF", "#FFFFFF")
	finalGradient, _ := utils.NewGradient(finalGradientStops, []int{12}, false)
	finalMapping, _ := finalGradient.BuildCoordinateColorMapping(
		base.Canvas.TextBottom, base.Canvas.TextTop,
		base.Canvas.TextLeft, base.Canvas.TextRight,
		utils.GradientVertical,
	)

	t := &Thunderstorm{
		base:                 base,
		finalGradientMapping: finalMapping,
		rainDrops:            make([]*rainDrop, 0),
		availableRain:        make([]*rainDrop, 0),
		availableStrike:      make([]*engine.EffectCharacter, 0),
		pendingStrike:        make([]*engine.EffectCharacter, 0),
		activeStrike:         make([]*engine.EffectCharacter, 0),
		strikeCoords:         make([]utils.Coord, 0),
		availableSparks:      make([]*sparkChar, 0),
		activeSparks:         make([]*sparkChar, 0),
		phase:                "pre-storm",
		stormDuration:        stormDuration,
		lightningColor:       lightningColor,
		raindropSymbols:      raindropSymbols,
		sparkSymbols:         sparkSymbols,
		sparkGlowColor:       sparkGlowColors[0],
		strikeBranchChance:   0.05,
		spawnRow:             spawnRow,
	}

	// Setup text characters with scenes
	for _, char := range base.Characters {
		t.setupTextCharacter(char)
		char.Visible = true
	}

	// Pre-build rain and spark pools
	t.buildRaindrops(50)
	t.buildSparks(100)
	t.buildStrikeChars(200)

	return t
}

func (t *Thunderstorm) setupTextCharacter(char *engine.EffectCharacter) {
	finalColor := t.finalGradientMapping[char.InputCoord]
	fadedColor := utils.AdjustBrightness(finalColor, 0.5)

	// Fade scene (pre-storm)
	fadeScene := char.Animation.NewScene("fade")
	fadeGradient, _ := utils.NewGradient([]utils.Color{finalColor, fadedColor}, []int{7}, false)
	for _, color := range fadeGradient.Spectrum {
		c := color
		fadeScene.AddFrame(char.Symbol, 12, &utils.ColorPair{FG: &c})
	}

	// Glow scene (after lightning strike)
	glowColors := mustColors("#EF5411")
	glowScene := char.Animation.NewScene("glow")
	glowGradient, _ := utils.NewGradient([]utils.Color{glowColors[0], fadedColor}, []int{7}, false)
	for _, color := range glowGradient.Spectrum {
		c := color
		glowScene.AddFrame(char.Symbol, 6, &utils.ColorPair{FG: &c})
	}

	// Unfade scene (post-storm)
	unfadeScene := char.Animation.NewScene("unfade")
	unfadeGradient, _ := utils.NewGradient([]utils.Color{fadedColor, finalColor}, []int{7}, false)
	for _, color := range unfadeGradient.Spectrum {
		c := color
		unfadeScene.AddFrame(char.Symbol, 12, &utils.ColorPair{FG: &c})
	}

	// Flash scene (during lightning)
	flashColor := utils.AdjustBrightness(finalColor, 1.7)
	flashScene := char.Animation.NewScene("flash")
	flashGradient, _ := utils.NewGradient([]utils.Color{fadedColor, flashColor, fadedColor}, []int{4, 4}, false)
	for _, color := range flashGradient.Spectrum {
		c := color
		flashScene.AddFrame(char.Symbol, 3, &utils.ColorPair{FG: &c})
	}
}

func (t *Thunderstorm) buildRaindrops(count int) {
	rainColors := mustColors("#aaaaff")
	rainColor := rainColors[0]

	for i := 0; i < count; i++ {
		symbol := t.raindropSymbols[utils.RandIntn(len(t.raindropSymbols))]
		char := engine.NewEffectCharacter(symbol, utils.Coord{Row: 0, Col: 0})
		char.IsAddedCharacter = true
		char.Layer = 1

		visual := char.Visual()
		visual.Colors = &utils.ColorPair{FG: &rainColor}
		char.SetVisual(visual)

		t.base.Terminal.AddCharacter(char)
		// AddCharacter sets Visible=true, but we want them hidden until activated
		char.Visible = false

		drop := &rainDrop{
			char:   char,
			active: false,
		}
		t.availableRain = append(t.availableRain, drop)
	}
}

func (t *Thunderstorm) buildSparks(count int) {
	bgColor := utils.Color{R: 0, G: 0, B: 0}

	for i := 0; i < count; i++ {
		symbol := t.sparkSymbols[utils.RandIntn(len(t.sparkSymbols))]
		char := engine.NewEffectCharacter(symbol, utils.Coord{Row: 0, Col: 0})
		char.IsAddedCharacter = true
		char.Layer = 2

		// Create glow scene
		glowScene := char.Animation.NewScene("glow")
		glowGradient, _ := utils.NewGradient([]utils.Color{t.sparkGlowColor, bgColor}, []int{7}, false)
		for _, color := range glowGradient.Spectrum {
			c := color
			glowScene.AddFrame(symbol, 18, &utils.ColorPair{FG: &c})
		}

		t.base.Terminal.AddCharacter(char)
		// AddCharacter sets Visible=true, but we want them hidden until activated
		char.Visible = false

		spark := &sparkChar{
			char:   char,
			active: false,
		}
		t.availableSparks = append(t.availableSparks, spark)
	}
}

func (t *Thunderstorm) buildStrikeChars(count int) {
	for i := 0; i < count; i++ {
		char := engine.NewEffectCharacter("|", utils.Coord{Row: 0, Col: 0})
		char.IsAddedCharacter = true
		char.Layer = 3
		t.base.Terminal.AddCharacter(char)
		// AddCharacter sets Visible=true, but we want them hidden until activated
		char.Visible = false
		t.availableStrike = append(t.availableStrike, char)
	}
}

func (t *Thunderstorm) getStrikeChar() *engine.EffectCharacter {
	if len(t.availableStrike) == 0 {
		t.buildStrikeChars(20)
	}
	char := t.availableStrike[len(t.availableStrike)-1]
	t.availableStrike = t.availableStrike[:len(t.availableStrike)-1]
	// Clear any existing scenes
	char.Animation.Scenes = make(map[string]*engine.Scene)
	return char
}

func (t *Thunderstorm) getSpark() *sparkChar {
	if len(t.availableSparks) == 0 {
		t.buildSparks(20)
	}
	spark := t.availableSparks[len(t.availableSparks)-1]
	t.availableSparks = t.availableSparks[:len(t.availableSparks)-1]
	return spark
}

func (t *Thunderstorm) setupLightningStrike(branchNeighbor *engine.EffectCharacter) {
	var col, row int

	if branchNeighbor != nil {
		col = branchNeighbor.Coord.Col
		row = branchNeighbor.Coord.Row
	} else {
		col = utils.RandIntn(t.base.Canvas.Width) + 1
		row = t.spawnRow
		t.strikeCoords = t.strikeCoords[:0] // Clear previous coords
	}

	for row >= 1 {
		var symbol string
		var delta int

		if branchNeighbor != nil {
			// Branch follows parent's direction tendency
			switch branchNeighbor.Symbol {
			case "/":
				col++
				if utils.RandFloat64() < 0.5 {
					symbol = "|"
				} else {
					symbol = "\\"
				}
			case "\\":
				col--
				if utils.RandFloat64() < 0.5 {
					symbol = "|"
				} else {
					symbol = "/"
				}
			default:
				delta = 1
				if utils.RandFloat64() < 0.5 {
					delta = -1
				}
				col += delta
				if delta == 1 {
					symbol = "\\"
				} else {
					symbol = "/"
				}
			}
			branchNeighbor = nil // Only use for first segment
		} else {
			r := utils.RandFloat64()
			if r < 0.33 {
				symbol = "\\"
				delta = 1
			} else if r < 0.66 {
				symbol = "/"
				delta = -1
			} else {
				symbol = "|"
				delta = 0
			}
		}

		// Keep in bounds
		if col < 1 {
			col = 1
		}
		if col > t.base.Canvas.Width {
			col = t.base.Canvas.Width
		}

		char := t.getStrikeChar()
		strikeCoord := utils.Coord{Row: row, Col: col}
		char.Coord = strikeCoord
		if char.Motion != nil {
			char.Motion.CurrentCoord = strikeCoord
		}
		char.Symbol = symbol

		// Setup flash and fade scenes
		flashColor := utils.AdjustBrightness(t.lightningColor, 1.7)
		bgColor := utils.Color{R: 0, G: 0, B: 0}

		// Flash scene
		flashScene := char.Animation.NewScene("flash")
		flashGradient, _ := utils.NewGradient([]utils.Color{t.lightningColor, flashColor}, []int{7}, false)
		for _, color := range flashGradient.Spectrum {
			c := color
			flashScene.AddFrame(symbol, 6, &utils.ColorPair{FG: &c})
		}
		// Loop the flash
		for _, color := range flashGradient.Spectrum {
			c := color
			flashScene.AddFrame(symbol, 6, &utils.ColorPair{FG: &c})
		}

		// Fade scene
		fadeScene := char.Animation.NewScene("fade")
		fadeGradient, _ := utils.NewGradient([]utils.Color{t.lightningColor, bgColor}, []int{6}, false)
		for _, color := range fadeGradient.Spectrum {
			c := color
			fadeScene.AddFrame(symbol, 2, &utils.ColorPair{FG: &c})
		}

		visual := char.Visual()
		visual.Symbol = symbol
		visual.Colors = &utils.ColorPair{FG: &t.lightningColor}
		char.SetVisual(visual)

		t.pendingStrike = append(t.pendingStrike, char)
		t.strikeCoords = append(t.strikeCoords, utils.Coord{Row: row, Col: col})

		// Check for branch
		if utils.RandFloat64() < t.strikeBranchChance && branchNeighbor == nil {
			t.strikeBranchChance -= 0.01
			t.setupLightningStrike(char)
		}

		row--
		if symbol == "\\" {
			col++
		} else if symbol == "/" {
			col--
		}
	}

	// Reset branch chance after main bolt
	if branchNeighbor == nil {
		t.strikeBranchChance = 0.05
		t.setupSparksForImpact()
	}
}

func (t *Thunderstorm) setupSparksForImpact() {
	if len(t.pendingStrike) == 0 {
		return
	}

	// Get the last strike char's position (bottom of bolt)
	lastStrike := t.pendingStrike[len(t.pendingStrike)-1]
	impactCol := float64(lastStrike.Coord.Col)
	impactRow := float64(lastStrike.Coord.Row)

	// Create 6-10 sparks
	sparkCount := 6 + utils.RandIntn(5)
	for i := 0; i < sparkCount; i++ {
		spark := t.getSpark()

		// Start at impact point
		spark.startX = impactCol
		spark.startY = impactRow
		spark.x = impactCol
		spark.y = impactRow

		// End at bottom, spread horizontally
		spread := float64(4 + utils.RandIntn(17)) // 4-20
		if utils.RandFloat64() < 0.5 {
			spread = -spread
		}
		spark.endX = impactCol + spread
		spark.endY = float64(1) // Bottom of canvas

		// Control point for bezier arc (upward arc)
		spark.ctrlX = impactCol + spread/2
		spark.ctrlY = float64(t.spawnRow + utils.RandIntn(t.spawnRow/2+1))

		spark.t = 0
		spark.speed = 0.01 + utils.RandFloat64()*0.015 // 0.01-0.025
		spark.holdFrames = 30
		spark.active = true

		sparkCoord := utils.Coord{Row: int(impactRow), Col: int(impactCol)}
		spark.char.Coord = sparkCoord
		if spark.char.Motion != nil {
			spark.char.Motion.CurrentCoord = sparkCoord
		}
		spark.char.Animation.ActivateScene("glow")

		t.activeSparks = append(t.activeSparks, spark)
	}
}

// Quadratic bezier interpolation
func bezierPoint(t, start, ctrl, end float64) float64 {
	oneMinusT := 1 - t
	return oneMinusT*oneMinusT*start + 2*oneMinusT*t*ctrl + t*t*end
}

func (t *Thunderstorm) stepSparks() {
	stillActive := make([]*sparkChar, 0, len(t.activeSparks))

	for _, spark := range t.activeSparks {
		if !spark.active {
			continue
		}

		if spark.t < 1.0 {
			// Advance along bezier curve
			spark.t += spark.speed
			if spark.t > 1.0 {
				spark.t = 1.0
			}

			// Apply easing (out_quint)
			eased := 1 - math.Pow(1-spark.t, 5)

			spark.x = bezierPoint(eased, spark.startX, spark.ctrlX, spark.endX)
			spark.y = bezierPoint(eased, spark.startY, spark.ctrlY, spark.endY)

			sparkCoord := utils.Coord{
				Row: int(math.Round(spark.y)),
				Col: int(math.Round(spark.x)),
			}
			spark.char.Coord = sparkCoord
			if spark.char.Motion != nil {
				spark.char.Motion.CurrentCoord = sparkCoord
			}
			spark.char.Visible = true
			stillActive = append(stillActive, spark)
		} else if spark.holdFrames > 0 {
			spark.holdFrames--
			stillActive = append(stillActive, spark)
		} else {
			// Done - recycle
			spark.char.Visible = false
			spark.active = false
			t.availableSparks = append(t.availableSparks, spark)
		}
	}

	t.activeSparks = stillActive
}

func (t *Thunderstorm) rain() {
	if t.rainDelay > 0 {
		t.rainDelay--
		return
	}

	// Update active raindrops
	stillActive := make([]*rainDrop, 0)
	for _, drop := range t.rainDrops {
		if !drop.active {
			continue
		}

		// Move along path
		drop.y -= drop.speed
		drop.x += drop.speed // Diagonal movement

		coord := utils.Coord{
			Row: int(math.Round(drop.y)),
			Col: int(math.Round(drop.x)),
		}
		drop.char.Coord = coord
		if drop.char.Motion != nil {
			drop.char.Motion.CurrentCoord = coord
		}

		if drop.y < 1 {
			// Reached bottom - recycle
			drop.char.Visible = false
			drop.active = false
			t.availableRain = append(t.availableRain, drop)
		} else {
			stillActive = append(stillActive, drop)
		}
	}
	t.rainDrops = stillActive

	// Spawn new raindrops
	spawnCount := 1 + utils.RandIntn(6) // 1-6 drops
	for i := 0; i < spawnCount; i++ {
		if len(t.availableRain) == 0 {
			break
		}

		drop := t.availableRain[len(t.availableRain)-1]
		t.availableRain = t.availableRain[:len(t.availableRain)-1]

		// Spawn at top, random column (can be off-left to account for diagonal)
		spawnCol := utils.RandIntn(t.base.Canvas.Width+t.spawnRow) - t.spawnRow + 1
		drop.x = float64(spawnCol)
		drop.y = float64(t.spawnRow)
		drop.speed = 0.5 + utils.RandFloat64() // 0.5-1.5
		drop.active = true
		drop.char.Visible = true
		spawnCoord := utils.Coord{Row: t.spawnRow, Col: spawnCol}
		drop.char.Coord = spawnCoord
		if drop.char.Motion != nil {
			drop.char.Motion.CurrentCoord = spawnCoord
		}

		t.rainDrops = append(t.rainDrops, drop)
	}

	t.rainDelay = 1 + utils.RandIntn(7) // 1-7 frame delay
}

func (t *Thunderstorm) stepLightningStrike() {
	if t.strikeProgressDelay > 0 {
		t.strikeProgressDelay--
		return
	}

	// Reveal pending strike chars progressively
	if len(t.pendingStrike) > 0 {
		revealCount := 1 + utils.RandIntn(3) // 1-3 at a time
		for i := 0; i < revealCount && len(t.pendingStrike) > 0; i++ {
			char := t.pendingStrike[0]
			t.pendingStrike = t.pendingStrike[1:]
			char.Visible = true
			t.activeStrike = append(t.activeStrike, char)
		}
		t.strikeProgressDelay = 1

		// When all revealed, activate flash
		if len(t.pendingStrike) == 0 {
			// Make sparks visible
			for _, spark := range t.activeSparks {
				spark.char.Visible = true
			}

			// Flash all strike chars
			for _, char := range t.activeStrike {
				char.Animation.ActivateScene("flash")
			}

			// Flash all text chars
			for _, char := range t.base.Characters {
				char.Animation.ActivateScene("flash")
			}
		}
		return
	}

	// Check if flash scenes are complete, then start fade
	allFlashDone := true
	for _, char := range t.activeStrike {
		if char.Animation.ActiveScene != nil {
			sceneName := ""
			for name, scene := range char.Animation.Scenes {
				if scene == char.Animation.ActiveScene {
					sceneName = name
					break
				}
			}
			if sceneName == "flash" && !char.Animation.ActiveScene.IsComplete() {
				allFlashDone = false
				break
			}
		}
	}

	if allFlashDone && len(t.activeStrike) > 0 {
		// Check if any are still in flash scene
		anyInFlash := false
		for _, char := range t.activeStrike {
			if char.Animation.ActiveScene != nil {
				for name, scene := range char.Animation.Scenes {
					if scene == char.Animation.ActiveScene && name == "flash" {
						anyInFlash = true
						break
					}
				}
			}
		}

		if anyInFlash || t.strikeFrame == 0 {
			// Transition to fade
			for _, char := range t.activeStrike {
				char.Animation.ActivateScene("fade")
			}
			t.strikeFrame = 1
		}
	}

	// Check if fade is complete
	if t.strikeFrame > 0 {
		allFadeDone := true
		for _, char := range t.activeStrike {
			if char.Animation.ActiveScene != nil && !char.Animation.ActiveScene.IsComplete() {
				allFadeDone = false
				break
			}
		}

		if allFadeDone {
			// Glow only chars that were behind the lightning
			t.glowAffectedChars()

			// Recycle strike chars
			for _, char := range t.activeStrike {
				char.Visible = false
				t.availableStrike = append(t.availableStrike, char)
			}
			t.activeStrike = t.activeStrike[:0]
			t.strikeFrame = 0
			t.strikeInProgress = false
		}
	}
}

func (t *Thunderstorm) glowAffectedChars() {
	// Build a set of strike coordinates for quick lookup
	strikeCoordSet := make(map[utils.Coord]bool)
	for _, coord := range t.strikeCoords {
		strikeCoordSet[coord] = true
	}

	// Only glow text chars that were behind lightning
	for _, char := range t.base.Characters {
		if strikeCoordSet[char.InputCoord] {
			char.Animation.ActivateScene("glow")
		}
	}
}

func (t *Thunderstorm) Next() (string, bool) {
	switch t.phase {
	case "pre-storm":
		// Activate fade on all text
		for _, char := range t.base.Characters {
			char.Animation.ActivateScene("fade")
		}
		t.phase = "waiting"

	case "waiting":
		// Check if all fade animations complete
		allDone := true
		for _, char := range t.base.Characters {
			if char.Animation.ActiveScene != nil && !char.Animation.ActiveScene.IsComplete() {
				allDone = false
				break
			}
		}
		if allDone {
			t.phase = "storm"
			t.stormStartTime = time.Now()
		}

	case "storm":
		t.rain()
		t.stepSparks()

		// Random chance for lightning strike
		if !t.strikeInProgress && utils.RandFloat64() < 0.008 {
			t.strikeInProgress = true
			t.strikeFrame = 0
			t.setupLightningStrike(nil)
		}

		if t.strikeInProgress {
			t.stepLightningStrike()
		}

		// Check storm duration
		if time.Since(t.stormStartTime) >= t.stormDuration && !t.strikeInProgress {
			// Hide remaining rain
			for _, drop := range t.rainDrops {
				drop.char.Visible = false
				drop.active = false
			}
			t.rainDrops = t.rainDrops[:0]

			// Unfade all text
			for _, char := range t.base.Characters {
				char.Animation.ActivateScene("unfade")
			}
			t.phase = "complete"
		}

	case "complete":
		// Check if all text animations are done
		allDone := true
		for _, char := range t.base.Characters {
			if char.Animation.ActiveScene != nil && !char.Animation.ActiveScene.IsComplete() {
				allDone = false
				break
			}
		}
		// Also check sparks
		if len(t.activeSparks) > 0 {
			allDone = false
			t.stepSparks()
		}
		if allDone {
			return "", false
		}
	}

	// Tick all text characters
	for _, char := range t.base.Characters {
		char.Tick()
	}

	// Tick strike characters
	for _, char := range t.activeStrike {
		if char.Visible {
			char.Tick()
		}
	}

	// Tick spark characters
	for _, spark := range t.activeSparks {
		if spark.char.Visible {
			spark.char.Tick()
		}
	}

	allChars := t.base.Terminal.GetCharacters(true, true, true, true, engine.TopToBottomLeftToRight)
	frame := engine.RenderFrame(t.base.Canvas, allChars)
	return frame, true
}

func (t *Thunderstorm) CanvasHeight() int {
	return t.base.CanvasHeight()
}
