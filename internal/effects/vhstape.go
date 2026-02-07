package effects

import (
	"math"
	"sort"

	"tte-go/internal/config"
	"tte-go/internal/engine"
	"tte-go/internal/utils"
)

// VHSTape is a port of Python terminaltexteffects/effect_vhstape.py.
//
// High-level behavior (matching Python):
//   1) Glitching phase (totalGlitchTime frames):
//        - A 3-line "glitch wave" continuously moves down the text.
//        - Up to 3 additional random single-line glitches can occur.
//        - Rarely, a global "snow" (noise) animation plays.
//        - Glitches move the whole line horizontally, hold briefly, then move back.
//   2) Noise phase:
//        - After glitching ends, all motion is restored.
//        - When everything settles, final snow plays on all characters.
//   3) Redraw phase:
//        - Lines are redrawn one by one from top to bottom.
//
// NOTE: The Go engine's Path HoldTime currently behaves like an initial delay for
// single-waypoint paths, not an end-of-path hold (unlike Python). To preserve the
// Python feel, we implement the "hold" portion as effect-managed frames after the
// motion completes.

type vhsLinePhase int

const (
	lineIdle vhsLinePhase = iota
	lineGlitchMoving
	lineHolding
	lineRestoring
	lineWaveMoving
)

type vhsLine struct {
	chars []*engine.EffectCharacter
	row   int

	// Constant per-line glitch offset (picked once at build like Python).
	glitchOffset int

	phase         vhsLinePhase
	holdRemaining int
	inWave        bool
}

func (l *vhsLine) movementComplete() bool {
	for _, ch := range l.chars {
		if ch.Motion != nil && !ch.Motion.MovementComplete() {
			return false
		}
	}
	return true
}

func (l *vhsLine) activateScene(sceneID string, reset bool) {
	for _, ch := range l.chars {
		if scn := ch.Animation.QueryScene(sceneID); scn != nil && reset {
			scn.ResetScene()
		}
		ch.Animation.ActivateScene(sceneID)
	}
}

func (l *vhsLine) activatePath(pathID string) {
	for _, ch := range l.chars {
		path, err := ch.Motion.QueryPath(pathID)
		if err != nil {
			continue
		}
		ch.Motion.ActivatePath(path)
	}
}

func (l *vhsLine) setPathSpeed(pathID string, speed float64) {
	for _, ch := range l.chars {
		path, err := ch.Motion.QueryPath(pathID)
		if err != nil {
			continue
		}
		path.Speed = speed
	}
}

// VHSTape creates a VHS tape glitch effect with horizontal line glitches and noise.
// This implementation aims to mirror the Python behavior closely.
//
// Effort: L (1-3h) for testing/tuning, but core logic is straightforward.

type VHSTape struct {
	base *BaseEffect

	finalGradientMapping map[utils.Coord]utils.Color

	// Lines keyed by row coordinate.
	linesByRow map[int]*vhsLine
	rows       []int // sorted ascending

	// Effect phases: "glitching", "noise", "redraw", "complete"
	phase string

	glitchingSteps  int
	totalGlitchTime int

	glitchLineChance float64
	noiseChance      float64

	glitchLineColors []utils.Color
	noiseColors      []utils.Color
	snowChars        []string

	// Glitch wave (row coordinate of the TOP of the 3-line wave).
	glitchWaveTop   int
	glitchWaveLines []*vhsLine

	// Additional random single-line glitches.
	activeGlitchLines []*vhsLine

	// Noise/redraw bookkeeping.
	finalSnowStarted bool
	finalSnowDone    bool
	toRedrawRows     []int // top-to-bottom (descending)
}

func NewVHSTape(input string, cfg config.TerminalConfig) engine.Effect {
	base := NewBaseEffect(input, cfg)

	// Configuration (matches Python defaults)
	glitchLineColors := mustColors("#ffffff", "#ff0000", "#00ff00", "#0000ff", "#ffffff")
	noiseColors := mustColors("#1e1e1f", "#3c3b3d", "#6d6c70", "#a2a1a6", "#cbc9cf", "#ffffff")
	snowChars := []string{"#", "*", ".", ":"}

	finalGradientStops := mustColors("#ab48ff", "#e7b2b2", "#fffebd")
	finalGradient, _ := utils.NewGradient(finalGradientStops, []int{12}, false)
	finalMapping, _ := finalGradient.BuildCoordinateColorMapping(
		base.Canvas.TextBottom, base.Canvas.TextTop,
		base.Canvas.TextLeft, base.Canvas.TextRight,
		utils.GradientVertical,
	)

	v := &VHSTape{
		base:                 base,
		finalGradientMapping: finalMapping,
		linesByRow:           map[int]*vhsLine{},
		phase:                "glitching",
		totalGlitchTime:      600,
		glitchLineChance:     0.05,
		noiseChance:          0.004,
		glitchLineColors:     glitchLineColors,
		noiseColors:          noiseColors,
		snowChars:            snowChars,
	}

	// Group characters by row coordinate.
	charsByRow := map[int][]*engine.EffectCharacter{}
	for _, ch := range base.Characters {
		row := ch.InputCoord.Row
		charsByRow[row] = append(charsByRow[row], ch)
	}

	// Build deterministic row order.
	for row := range charsByRow {
		v.rows = append(v.rows, row)
	}
	sort.Ints(v.rows) // bottom-to-top (ascending row)

	// Create per-line constants (offset/direction) and build per-character paths/scenes.
	for _, row := range v.rows {
		chars := charsByRow[row]
		// Python: offset = randint(4,25); direction = choice(-1,1)
		offset := randIntInclusive(4, 25)
		direction := 1
		if utils.RandIntn(2) == 0 {
			direction = -1
		}
		line := &vhsLine{chars: chars, row: row, glitchOffset: offset * direction, phase: lineIdle}
		v.linesByRow[row] = line

		for _, ch := range chars {
			ch.Visible = true
			v.buildCharacterPathsAndScenes(ch, line.glitchOffset)
			ch.Animation.ActivateScene("base")
		}
	}

	// Build redraw order: top-to-bottom (descending row).
	v.toRedrawRows = append([]int{}, v.rows...)
	sort.Sort(sort.Reverse(sort.IntSlice(v.toRedrawRows)))

	return v
}

func (v *VHSTape) buildCharacterPathsAndScenes(ch *engine.EffectCharacter, glitchOffset int) {
	finalColor := v.finalGradientMapping[ch.InputCoord]

	// --- Motion paths ---
	// Glitch: move horizontally to offset.
	glitchPath, _ := ch.Motion.NewPath(2, nil, nil, 0, false, "glitch")
	glitchPath.AddWaypoint(utils.Coord{Row: ch.InputCoord.Row, Col: ch.InputCoord.Col + glitchOffset})

	// Restore: return to input coordinate.
	restorePath, _ := ch.Motion.NewPath(2, nil, nil, 0, false, "restore")
	restorePath.AddWaypoint(ch.InputCoord)

	// Glitch wave paths: fixed offsets +8 and +14.
	waveMid, _ := ch.Motion.NewPath(2, nil, nil, 0, false, "wave_mid")
	waveMid.AddWaypoint(utils.Coord{Row: ch.InputCoord.Row, Col: ch.InputCoord.Col + 8})

	waveEnd, _ := ch.Motion.NewPath(2, nil, nil, 0, false, "wave_end")
	waveEnd.AddWaypoint(utils.Coord{Row: ch.InputCoord.Row, Col: ch.InputCoord.Col + 14})

	// Prevent unused warnings (paths referenced via QueryPath later).
	_, _, _, _ = glitchPath, restorePath, waveMid, waveEnd

	// --- Scenes ---
	// Base scene: looping so it's stable and doesn't complete.
	baseScene := ch.Animation.NewScene("base")
	_ = baseScene.AddFrame(ch.Symbol, 1, &utils.ColorPair{FG: &finalColor})
	baseScene.IsLooping = true

	// RGB glitch forward.
	glitchFwd := ch.Animation.NewScene("rgb_glitch_fwd")
	for _, color := range v.glitchLineColors {
		c := color
		_ = glitchFwd.AddFrame(ch.Symbol, 1, &utils.ColorPair{FG: &c})
	}

	// RGB glitch backward.
	glitchBwd := ch.Animation.NewScene("rgb_glitch_bwd")
	for i := len(v.glitchLineColors) - 1; i >= 0; i-- {
		c := v.glitchLineColors[i]
		_ = glitchBwd.AddFrame(ch.Symbol, 1, &utils.ColorPair{FG: &c})
	}

	// Snow/noise.
	snow := ch.Animation.NewScene("snow")
	for i := 0; i < 25; i++ {
		sym := v.snowChars[utils.RandIntn(len(v.snowChars))]
		col := v.noiseColors[utils.RandIntn(len(v.noiseColors))]
		c := col
		_ = snow.AddFrame(sym, 2, &utils.ColorPair{FG: &c})
	}
	_ = snow.AddFrame(ch.Symbol, 1, &utils.ColorPair{FG: &finalColor})

	// Final snow (no final symbol frame; matches Python).
	finalSnow := ch.Animation.NewScene("final_snow")
	for i := 0; i < 30; i++ {
		sym := v.snowChars[utils.RandIntn(len(v.snowChars))]
		col := v.noiseColors[utils.RandIntn(len(v.noiseColors))]
		c := col
		_ = finalSnow.AddFrame(sym, 2, &utils.ColorPair{FG: &c})
	}

	// Final redraw.
	white := utils.Color{R: 255, G: 255, B: 255}
	finalRedraw := ch.Animation.NewScene("final_redraw")
	_ = finalRedraw.AddFrame("█", 6, &utils.ColorPair{FG: &white})
	_ = finalRedraw.AddFrame(ch.Symbol, 1, &utils.ColorPair{FG: &finalColor})

	// Scene completion: when rgb_glitch_bwd completes, go back to base.
	// (In Python this is EventHandler.Event.SCENE_COMPLETE -> activate base.)
	if ch.EventHandler != nil {
		_ = ch.EventHandler.RegisterEvent(engine.EventSceneComplete, glitchBwd, engine.ActionActivateScene, baseScene)
	}
}

func (v *VHSTape) startSingleLineGlitch(line *vhsLine, holdFrames int) {
	if line == nil || line.phase != lineIdle || line.inWave {
		return
	}

	line.phase = lineGlitchMoving
	line.holdRemaining = holdFrames

	// Python: glitch_path.speed = 40 / randint(20,40)
	line.setPathSpeed("glitch", randVhsSpeed())
	line.activateScene("rgb_glitch_fwd", true)
	line.activatePath("glitch")
}

func (v *VHSTape) startRestore(line *vhsLine) {
	if line == nil {
		return
	}
	line.phase = lineRestoring
	line.holdRemaining = 0

	// Python: restore_path.speed = 40 / randint(20,40)
	line.setPathSpeed("restore", randVhsSpeed())
	line.activateScene("rgb_glitch_bwd", true)
	line.activatePath("restore")
}

func (v *VHSTape) startWaveMove(line *vhsLine, pathID string) {
	if line == nil {
		return
	}
	// Wave moves do not hold.
	line.phase = lineWaveMoving
	line.holdRemaining = 0

	line.activateScene("rgb_glitch_fwd", true)
	line.activatePath(pathID)
}

func (v *VHSTape) updateSingleLineState(line *vhsLine) {
	if line == nil {
		return
	}

	switch line.phase {
	case lineGlitchMoving:
		if line.movementComplete() {
			if line.holdRemaining > 0 {
				line.phase = lineHolding
			} else {
				v.startRestore(line)
			}
		}
	case lineHolding:
		if line.holdRemaining > 0 {
			line.holdRemaining--
		}
		if line.holdRemaining <= 0 {
			v.startRestore(line)
		}
	case lineRestoring:
		// When motion completes, mark idle. Base scene activation is handled by the
		// rgb_glitch_bwd SceneComplete event.
		if line.movementComplete() {
			line.phase = lineIdle
		}
	}
}

func (v *VHSTape) glitchWaveStep() {
	// Python: wave only runs if text_height >= 3
	textHeight := v.base.Canvas.TextTop - v.base.Canvas.TextBottom + 1
	if textHeight < 3 {
		return
	}

	if v.glitchWaveTop == 0 {
		// Python: choose a wave top row in the top half (or at least 3 rows up)
		minOffset := max(3, int(math.Round(float64(textHeight)*0.5)))
		maxOffset := textHeight
		v.glitchWaveTop = v.base.Canvas.TextBottom + randIntInclusive(minOffset, maxOffset)
		// Clamp like Python: max(2, min(wave_top, text_top))
		v.glitchWaveTop = clampInt(v.glitchWaveTop, 2, v.base.Canvas.TextTop)
	}

	// Only progress wave when current wave lines have finished their movement.
	if len(v.glitchWaveLines) > 0 {
		for _, ln := range v.glitchWaveLines {
			if !ln.movementComplete() {
				return
			}
		}
	}

	// Possibly adjust wave top (mostly stays or moves down, sometimes up).
	if len(v.glitchWaveLines) > 0 {
		waveTopDelta := 0
		if utils.RandFloat64() < 0.3 {
			if utils.RandFloat64() < 0.3 {
				waveTopDelta = 1
			} else {
				waveTopDelta = -1
			}
		}
		v.glitchWaveTop += waveTopDelta
		v.glitchWaveTop = clampInt(v.glitchWaveTop, 2, v.base.Canvas.TextTop)
	}

	// Compute new wave lines: rows [waveTop-2, waveTop-1, waveTop] (bottom-to-top).
	newWave := make([]*vhsLine, 0, 3)
	newSet := map[*vhsLine]struct{}{}
	for r := v.glitchWaveTop - 2; r <= v.glitchWaveTop; r++ {
		if ln, ok := v.linesByRow[r]; ok {
			newWave = append(newWave, ln)
			newSet[ln] = struct{}{}
		}
	}

	// Restore any lines no longer part of the wave.
	for _, old := range v.glitchWaveLines {
		if _, ok := newSet[old]; !ok {
			old.inWave = false
			v.startRestore(old)
		}
	}

	v.glitchWaveLines = newWave
	for _, ln := range v.glitchWaveLines {
		ln.inWave = true
	}

	// If wave is at bottom, restore lines and reset.
	if v.glitchWaveTop < v.base.Canvas.TextBottom+2 {
		for _, ln := range v.glitchWaveLines {
			ln.inWave = false
			v.startRestore(ln)
		}
		v.glitchWaveTop = 0
		v.glitchWaveLines = nil
		return
	}

	// Activate movement for the wave lines using the pattern (mid, end, mid).
	pathPattern := []string{"wave_mid", "wave_end", "wave_mid"}
	for i := 0; i < len(v.glitchWaveLines) && i < len(pathPattern); i++ {
		v.startWaveMove(v.glitchWaveLines[i], pathPattern[i])
	}
}

func (v *VHSTape) maybeStartRandomGlitchLine() {
	if utils.RandFloat64() >= v.glitchLineChance {
		return
	}
	if len(v.activeGlitchLines) >= 3 {
		return
	}

	// Choose a random eligible line.
	candidates := make([]*vhsLine, 0, len(v.linesByRow))
	for _, row := range v.rows {
		ln := v.linesByRow[row]
		if ln == nil {
			continue
		}
		if ln.inWave {
			continue
		}
		if ln.phase != lineIdle {
			continue
		}
		candidates = append(candidates, ln)
	}
	if len(candidates) == 0 {
		return
	}

	ln := candidates[utils.RandIntn(len(candidates))]
	hold := randIntInclusive(20, 75)
	v.startSingleLineGlitch(ln, hold)
	v.activeGlitchLines = append(v.activeGlitchLines, ln)
}

func (v *VHSTape) maybeSnowAllLines() {
	if utils.RandFloat64() >= v.noiseChance {
		return
	}
	for _, ln := range v.linesByRow {
		if ln == nil {
			continue
		}
		ln.activateScene("snow", true)
	}
}

func (v *VHSTape) anyBusyNonBase() bool {
	for _, ch := range v.base.Characters {
		if ch.Motion != nil && !ch.Motion.MovementComplete() {
			return true
		}
		if ch.Animation != nil && ch.Animation.ActiveScene != nil {
			id := ch.Animation.ActiveScene.ID
			if id != "base" && !ch.Animation.ActiveScene.IsComplete() {
				return true
			}
		}
	}
	return false
}

func (v *VHSTape) updateAllLineStates() {
	// Update single-line state machines (including restores for wave exits).
	for _, ln := range v.linesByRow {
		v.updateSingleLineState(ln)
	}

	// Drop completed single-line glitches from the active list.
	filtered := v.activeGlitchLines[:0]
	for _, ln := range v.activeGlitchLines {
		if ln != nil && ln.phase != lineIdle {
			filtered = append(filtered, ln)
		}
	}
	v.activeGlitchLines = filtered
}

func (v *VHSTape) Next() (string, bool) {
	switch v.phase {
	case "glitching":
		v.glitchingSteps++

		// Keep the glitch wave moving.
		v.glitchWaveStep()

		// Update all line state machines (holds/restores).
		v.updateAllLineStates()

		// Random extra glitches.
		v.maybeStartRandomGlitchLine()

		// Rare global snow.
		v.maybeSnowAllLines()

		// End glitching phase.
		if v.glitchingSteps >= v.totalGlitchTime {
			// Restore wave lines.
			for _, ln := range v.glitchWaveLines {
				if ln != nil {
					ln.inWave = false
					v.startRestore(ln)
				}
			}
			v.glitchWaveTop = 0
			v.glitchWaveLines = nil

			// Restore any additional glitch lines.
			for _, ln := range v.activeGlitchLines {
				if ln != nil {
					v.startRestore(ln)
				}
			}
			v.activeGlitchLines = nil

			v.phase = "noise"
		}

	case "noise":
		// Wait for motion + non-base animations to settle.
		v.updateAllLineStates()
		if !v.anyBusyNonBase() {
			// Activate final snow once.
			if !v.finalSnowStarted {
				for _, ch := range v.base.Characters {
					if scn := ch.Animation.QueryScene("final_snow"); scn != nil {
						scn.ResetScene()
					}
					ch.Animation.ActivateScene("final_snow")
				}
				v.finalSnowStarted = true
				v.phase = "redraw"
			}
		}

	case "redraw":
		// Wait for final snow to finish before redrawing.
		if !v.finalSnowDone {
			done := true
			for _, ch := range v.base.Characters {
				if ch.Animation != nil && ch.Animation.ActiveScene != nil && ch.Animation.ActiveScene.ID == "final_snow" {
					if !ch.Animation.ActiveScene.IsComplete() {
						done = false
						break
					}
				}
			}
			v.finalSnowDone = done
		}

		if v.finalSnowDone {
			// If no active redraw animation is running, start the next line.
			redrawActive := false
			for _, ch := range v.base.Characters {
				if ch.Animation != nil && ch.Animation.ActiveScene != nil && ch.Animation.ActiveScene.ID == "final_redraw" {
					if !ch.Animation.ActiveScene.IsComplete() {
						redrawActive = true
						break
					}
				}
			}

			if !redrawActive {
				if len(v.toRedrawRows) > 0 {
					row := v.toRedrawRows[0]
					v.toRedrawRows = v.toRedrawRows[1:]
					if ln, ok := v.linesByRow[row]; ok && ln != nil {
						ln.activateScene("final_redraw", true)
					}
				} else {
					v.phase = "complete"
				}
			}
		}

	case "complete":
		return "", false
	}

	// Tick all characters (motion + animation).
	for _, ch := range v.base.Characters {
		ch.Tick()
	}

	frame := engine.RenderFrame(v.base.Canvas, v.base.Characters)
	return frame, true
}

func (v *VHSTape) CanvasHeight() int {
	return v.base.CanvasHeight()
}

func (v *VHSTape) CanvasWidth() int {
	return v.base.CanvasWidth()
}

func randIntInclusive(minVal, maxVal int) int {
	if maxVal < minVal {
		minVal, maxVal = maxVal, minVal
	}
	return minVal + utils.RandIntn(maxVal-minVal+1)
}

func clampInt(value, minVal, maxVal int) int {
	if value < minVal {
		return minVal
	}
	if value > maxVal {
		return maxVal
	}
	return value
}

func randVhsSpeed() float64 {
	// Python: 40 / randint(20, 40)
	denom := float64(randIntInclusive(20, 40))
	return 40.0 / denom
}
