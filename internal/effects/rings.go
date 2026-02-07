package effects

import (
	"math"

	"tte-go/internal/config"
	"tte-go/internal/engine"
	"tte-go/internal/utils"
)

// Path IDs for the motion system
const (
	pathInitialDisperse = "initial_disperse"
	pathDisperseLoop    = "disperse_loop"
	pathCondense        = "condense"
	pathRingLoop        = "ring_loop"
	pathHome            = "home"
	pathExternal        = "external"
)

// ringsPath represents a waypoint path for character motion
type ringsPath struct {
	ID        string
	Waypoints []utils.Coord
	Speed     float64
	Loop      bool
	NextID    string // path to activate on completion (empty = none)
}

// ringsMotion tracks per-character motion state with float precision
type ringsMotion struct {
	Paths       map[string]*ringsPath
	ActiveID    string
	WaypointIdx int
	X, Y        float64 // float position to avoid rounding stalls
}

func newRingsMotion(startCoord utils.Coord) *ringsMotion {
	return &ringsMotion{
		Paths:       make(map[string]*ringsPath),
		ActiveID:    "",
		WaypointIdx: 0,
		X:           float64(startCoord.Col),
		Y:           float64(startCoord.Row),
	}
}

func (m *ringsMotion) addPath(id string, waypoints []utils.Coord, speed float64, loop bool, nextID string) {
	m.Paths[id] = &ringsPath{
		ID:        id,
		Waypoints: waypoints,
		Speed:     speed,
		Loop:      loop,
		NextID:    nextID,
	}
}

func (m *ringsMotion) activate(id string) {
	m.ActiveID = id
	m.WaypointIdx = 0
}

func (m *ringsMotion) currentCoord() utils.Coord {
	return utils.Coord{Col: int(math.Round(m.X)), Row: int(math.Round(m.Y))}
}

// step moves toward current waypoint, returns (completedPathID, didComplete)
func (m *ringsMotion) step() (string, bool) {
	if m.ActiveID == "" {
		return "", false
	}
	path, ok := m.Paths[m.ActiveID]
	if !ok || len(path.Waypoints) == 0 {
		return "", false
	}

	target := path.Waypoints[m.WaypointIdx]
	dx := float64(target.Col) - m.X
	dy := float64(target.Row) - m.Y
	dist := math.Sqrt(dx*dx + dy*dy)

	// Arrival threshold: within speed distance (prevents stalls)
	if dist <= path.Speed {
		// Snap to waypoint
		m.X = float64(target.Col)
		m.Y = float64(target.Row)

		// Advance to next waypoint
		m.WaypointIdx++
		if m.WaypointIdx >= len(path.Waypoints) {
			if path.Loop {
				m.WaypointIdx = 0
			} else {
				// Path complete
				completedID := m.ActiveID
				if path.NextID != "" {
					m.activate(path.NextID)
				} else {
					m.ActiveID = ""
				}
				return completedID, true
			}
		}
	} else {
		// Move toward target
		ratio := path.Speed / dist
		m.X += dx * ratio
		m.Y += dy * ratio
	}
	return "", false
}

// Rings creates an effect where characters are dispersed and form into spinning rings.
type Rings struct {
	base *BaseEffect

	rings        []*ringsRing
	ringChars    []*engine.EffectCharacter
	nonRingChars []*engine.EffectCharacter

	// Per-character motion controllers
	charMotion map[*engine.EffectCharacter]*ringsMotion

	characterFinalColorMap map[*engine.EffectCharacter]utils.Color
	phase                  string // "start", "disperse", "spin", "final", "complete"
	spinTimeRemaining      int
	disperseTimeRemaining  int
	cyclesRemaining        int
	initialPhaseRemaining  int
	ringGapPixels          int

	// Config
	ringColors             []utils.Color
	ringGap                float64
	spinDuration           int
	spinSpeedRange         [2]float64
	disperseDuration       int
	spinDisperseCycles     int
	finalGradientStops     []utils.Color
	finalGradientSteps     []int
	finalGradientDirection utils.GradientDirection
}

type ringsRing struct {
	radius                 int
	origin                 utils.Coord
	counterClockwiseCoords []utils.Coord
	clockwiseCoords        []utils.Coord
	color                  utils.Color
	characters             []*engine.EffectCharacter
	rotationSpeed          float64
	clockwise              bool
	ringGap                int

	// Per-character data
	assignedRingIndex map[*engine.EffectCharacter]int // character's position on ring
}

func NewRings(input string, cfg config.TerminalConfig) engine.Effect {
	base := NewBaseEffect(input, cfg)

	// Config defaults matching Python
	ringColors := mustColors("ab48ff", "e7b2b2", "fffebd")
	finalGradientStops := mustColors("ab48ff", "e7b2b2", "fffebd")
	finalGradientSteps := []int{12}
	finalGradientDirection := utils.GradientVertical

	r := &Rings{
		base:                   base,
		rings:                  make([]*ringsRing, 0),
		ringChars:              make([]*engine.EffectCharacter, 0),
		nonRingChars:           make([]*engine.EffectCharacter, 0),
		charMotion:             make(map[*engine.EffectCharacter]*ringsMotion),
		characterFinalColorMap: make(map[*engine.EffectCharacter]utils.Color),
		phase:                  "start",
		initialPhaseRemaining:  100,
		spinTimeRemaining:      200,
		disperseTimeRemaining:  200,
		cyclesRemaining:        3,
		ringColors:             ringColors,
		ringGap:                0.1,
		spinDuration:           200,
		spinSpeedRange:         [2]float64{0.25, 1.0},
		disperseDuration:       200,
		spinDisperseCycles:     3,
		finalGradientStops:     finalGradientStops,
		finalGradientSteps:     finalGradientSteps,
		finalGradientDirection: finalGradientDirection,
	}

	r.build()
	return r
}

// ringsFindCoordsOnCircle finds points on a circle, matching Python's implementation
// which doubles x distance for terminal aspect ratio
func ringsFindCoordsOnCircle(origin utils.Coord, radius int, coordsLimit int, unique bool) []utils.Coord {
	if radius == 0 {
		return []utils.Coord{}
	}
	if coordsLimit == 0 {
		coordsLimit = int(math.Round(2 * math.Pi * float64(radius)))
	}

	points := make([]utils.Coord, 0, coordsLimit)
	seen := make(map[utils.Coord]bool)
	angleStep := 2 * math.Pi / float64(coordsLimit)

	for i := 0; i < coordsLimit; i++ {
		angle := angleStep * float64(i)
		x := float64(origin.Col) + float64(radius)*math.Cos(angle)
		// Correct for terminal character height/width ratio by doubling the x distance from origin
		xDiff := x - float64(origin.Col)
		x += xDiff
		y := float64(origin.Row) + float64(radius)*math.Sin(angle)

		point := utils.Coord{Col: int(math.Round(x)), Row: int(math.Round(y))}
		if unique {
			if !seen[point] {
				points = append(points, point)
				seen[point] = true
			}
		} else {
			points = append(points, point)
		}
	}
	return points
}

// findCoordsInRect finds coords within a rectangle around origin
func findCoordsInRect(origin utils.Coord, distance int) []utils.Coord {
	if distance == 0 {
		return []utils.Coord{}
	}
	coords := make([]utils.Coord, 0, (2*distance+1)*(2*distance+1))
	for col := origin.Col - distance; col <= origin.Col+distance; col++ {
		for row := origin.Row - distance; row <= origin.Row+distance; row++ {
			coords = append(coords, utils.Coord{Col: col, Row: row})
		}
	}
	return coords
}

func (r *Rings) build() {
	// Calculate ring gap in pixels - Python uses min(canvas.top, canvas.right) * ring_gap
	r.ringGapPixels = int(math.Max(math.Round(float64(min(r.base.Canvas.Height, r.base.Canvas.Width))*r.ringGap), 1))

	// Build final gradient mapping
	finalGradient, _ := utils.NewGradient(r.finalGradientStops, r.finalGradientSteps, false)
	finalGradientMapping, _ := finalGradient.BuildCoordinateColorMapping(
		r.base.Canvas.TextBottom,
		r.base.Canvas.TextTop,
		r.base.Canvas.TextLeft,
		r.base.Canvas.TextRight,
		r.finalGradientDirection,
	)

	// Get all characters and set up initial state
	allChars := r.base.Terminal.GetCharacters(true, false, false, false, engine.TopToBottomLeftToRight)
	pendingChars := make([]*engine.EffectCharacter, len(allChars))
	copy(pendingChars, allChars)

	for _, char := range allChars {
		r.characterFinalColorMap[char] = finalGradientMapping[char.InputCoord]
		// Set initial color from final gradient
		colorCopy := r.characterFinalColorMap[char]
		char.SetVisual(engine.CharacterVisual{
			Symbol: char.Symbol,
			Colors: &utils.ColorPair{FG: &colorCopy},
		})
		r.base.Terminal.SetCharacterVisibility(char, true)

		// Initialize motion controller at input position
		r.charMotion[char] = newRingsMotion(char.InputCoord)
	}

	// Shuffle pending chars
	for i := len(pendingChars) - 1; i > 0; i-- {
		j := utils.RandIntn(i + 1)
		pendingChars[i], pendingChars[j] = pendingChars[j], pendingChars[i]
	}

	// Create rings - Python iterates from 1 to max(canvas.right, canvas.top) with step ring_gap
	origin := r.base.Canvas.Center
	maxDim := max(r.base.Canvas.Width, r.base.Canvas.Height)

	for radius := 1; radius < maxDim; radius += r.ringGapPixels {
		// Python uses 7 * radius as coords_limit
		ringCoords := ringsFindCoordsOnCircle(origin, radius, 7*radius, true)

		// Check if enough of the ring is in canvas (at least 25%)
		inCanvasCount := 0
		for _, coord := range ringCoords {
			if coord.Col >= 1 && coord.Col <= r.base.Canvas.Width &&
				coord.Row >= 1 && coord.Row <= r.base.Canvas.Height {
				inCanvasCount++
			}
		}
		if len(ringCoords) > 0 && float64(inCanvasCount)/float64(len(ringCoords)) < 0.25 {
			break
		}

		rng := &ringsRing{
			radius:                 radius,
			origin:                 origin,
			counterClockwiseCoords: ringCoords,
			clockwiseCoords:        reverseCoords(ringCoords),
			color:                  r.ringColors[len(r.rings)%len(r.ringColors)],
			characters:             make([]*engine.EffectCharacter, 0),
			rotationSpeed:          r.spinSpeedRange[0] + utils.RandFloat64()*(r.spinSpeedRange[1]-r.spinSpeedRange[0]),
			clockwise:              len(r.rings)%2 == 1, // odd rings go clockwise
			ringGap:                r.ringGapPixels,
			assignedRingIndex:      make(map[*engine.EffectCharacter]int),
		}

		// Assign characters to this ring (one per coord position)
		for i := range ringCoords {
			if len(pendingChars) == 0 {
				break
			}
			char := pendingChars[0]
			pendingChars = pendingChars[1:]
			rng.characters = append(rng.characters, char)
			rng.assignedRingIndex[char] = i
			r.ringChars = append(r.ringChars, char)
		}

		r.rings = append(r.rings, rng)
	}

	// Characters not assigned to rings become non-ring chars
	for _, char := range allChars {
		found := false
		for _, rc := range r.ringChars {
			if rc == char {
				found = true
				break
			}
		}
		if !found {
			r.nonRingChars = append(r.nonRingChars, char)
		}
	}

	// Set up paths for non-ring characters
	for _, char := range r.nonRingChars {
		motion := r.charMotion[char]
		// External path - move off screen
		externalTarget := utils.Coord{
			Col: r.base.Canvas.Width + 1 + utils.RandIntn(10),
			Row: utils.RandIntn(r.base.Canvas.Height) + 1,
		}
		motion.addPath(pathExternal, []utils.Coord{externalTarget}, 0.8, false, "")
		// Home path - back to input
		motion.addPath(pathHome, []utils.Coord{char.InputCoord}, 0.8, false, "")
	}

	// Set up paths for ring characters
	for _, rng := range r.rings {
		coords := rng.counterClockwiseCoords
		if rng.clockwise {
			coords = rng.clockwiseCoords
		}

		for _, char := range rng.characters {
			motion := r.charMotion[char]
			idx := rng.assignedRingIndex[char]
			assignedRingCoord := coords[idx%len(coords)]

			// Generate initial disperse waypoints using ring coord as origin
			disperseWaypoints := rng.makeDisperseWaypoints(assignedRingCoord)

			// Path A: initial_disperse - from input to first disperse waypoint (speed 0.3)
			motion.addPath(pathInitialDisperse, []utils.Coord{disperseWaypoints[0]}, 0.3, false, pathDisperseLoop)

			// Path B: disperse_loop - cycle through disperse waypoints (speed 0.14)
			motion.addPath(pathDisperseLoop, disperseWaypoints, 0.14, true, "")

			// Path C: condense - to ring position (speed 0.1), then ring_loop
			motion.addPath(pathCondense, []utils.Coord{assignedRingCoord}, 0.1, false, pathRingLoop)

			// Path D: ring_loop - rotate through all ring coords starting from assigned position
			// Build coords starting from character's position for smooth entry
			ringLoopCoords := make([]utils.Coord, len(coords))
			for i := 0; i < len(coords); i++ {
				ringLoopCoords[i] = coords[(idx+i)%len(coords)]
			}
			motion.addPath(pathRingLoop, ringLoopCoords, rng.rotationSpeed, true, "")

			// Path E: home - back to input position (speed 0.8)
			motion.addPath(pathHome, []utils.Coord{char.InputCoord}, 0.8, false, "")
		}
	}
}

func reverseCoords(coords []utils.Coord) []utils.Coord {
	result := make([]utils.Coord, len(coords))
	for i, c := range coords {
		result[len(coords)-1-i] = c
	}
	return result
}

func (rng *ringsRing) makeDisperseWaypoints(origin utils.Coord) []utils.Coord {
	disperseCoords := findCoordsInRect(origin, rng.ringGap)
	if len(disperseCoords) == 0 {
		return []utils.Coord{origin}
	}
	// Select 5 random waypoints
	waypoints := make([]utils.Coord, 5)
	for i := 0; i < 5; i++ {
		waypoints[i] = disperseCoords[utils.RandIntn(len(disperseCoords))]
	}
	return waypoints
}

func (r *Rings) Next() (string, bool) {
	switch r.phase {
	case "start":
		// Initial phase - characters visible in starting positions
		if r.initialPhaseRemaining <= 0 {
			r.phase = "disperse"
			// Activate initial disperse paths
			for _, char := range r.ringChars {
				r.charMotion[char].activate(pathInitialDisperse)
			}
			for _, char := range r.nonRingChars {
				r.charMotion[char].activate(pathExternal)
			}
		} else {
			r.initialPhaseRemaining--
		}

	case "disperse":
		// Step all character motions
		for _, char := range r.ringChars {
			motion := r.charMotion[char]
			motion.step()
			char.Motion.SetCoordinate(motion.currentCoord())
		}
		for _, char := range r.nonRingChars {
			motion := r.charMotion[char]
			motion.step()
			coord := motion.currentCoord()
			char.Motion.SetCoordinate(coord)
			// Hide if off-screen
			if coord.Col > r.base.Canvas.Width || coord.Col < 1 {
				r.base.Terminal.SetCharacterVisibility(char, false)
			}
		}

		if r.disperseTimeRemaining <= 0 {
			r.phase = "spin"
			r.cyclesRemaining--
			r.spinTimeRemaining = r.spinDuration

			// Activate condense paths for all ring chars
			for _, rng := range r.rings {
				for _, char := range rng.characters {
					motion := r.charMotion[char]
					// Update condense target to current ring position based on where we are
					// For first spin, use assigned index; for subsequent, use last position
					coords := rng.counterClockwiseCoords
					if rng.clockwise {
						coords = rng.clockwiseCoords
					}
					idx := rng.assignedRingIndex[char]
					condenseTarget := coords[idx%len(coords)]
					motion.Paths[pathCondense].Waypoints = []utils.Coord{condenseTarget}
					motion.activate(pathCondense)

					// Set ring color
					colorCopy := rng.color
					char.SetVisual(engine.CharacterVisual{
						Symbol: char.Symbol,
						Colors: &utils.ColorPair{FG: &colorCopy},
					})
				}
			}
		} else {
			r.disperseTimeRemaining--
		}

	case "spin":
		// Step all character motions
		for _, char := range r.ringChars {
			motion := r.charMotion[char]
			motion.step()
			char.Motion.SetCoordinate(motion.currentCoord())
		}

		if r.spinTimeRemaining <= 0 {
			if r.cyclesRemaining <= 0 {
				r.phase = "final"
				// Activate home paths for all characters
				for _, char := range r.ringChars {
					motion := r.charMotion[char]
					motion.addPath(pathHome, []utils.Coord{char.InputCoord}, 0.8, false, "")
					motion.activate(pathHome)
					// Set final color
					colorCopy := r.characterFinalColorMap[char]
					char.SetVisual(engine.CharacterVisual{
						Symbol: char.Symbol,
						Colors: &utils.ColorPair{FG: &colorCopy},
					})
				}
				for _, char := range r.nonRingChars {
					r.base.Terminal.SetCharacterVisibility(char, true)
					motion := r.charMotion[char]
					motion.addPath(pathHome, []utils.Coord{char.InputCoord}, 0.8, false, "")
					motion.activate(pathHome)
				}
			} else {
				r.disperseTimeRemaining = r.disperseDuration
				r.phase = "disperse"

				// Start new disperse cycle
				for _, rng := range r.rings {
					coords := rng.counterClockwiseCoords
					if rng.clockwise {
						coords = rng.clockwiseCoords
					}

					for _, char := range rng.characters {
						motion := r.charMotion[char]

						// Save current ring position for next condense
						// Find which ring coord we're closest to
						currentCoord := motion.currentCoord()
						closestIdx := 0
						closestDist := math.MaxFloat64
						for i, c := range coords {
							dx := float64(c.Col - currentCoord.Col)
							dy := float64(c.Row - currentCoord.Row)
							dist := dx*dx + dy*dy
							if dist < closestDist {
								closestDist = dist
								closestIdx = i
							}
						}
						rng.assignedRingIndex[char] = closestIdx

						// Generate new disperse waypoints from current position
						disperseWaypoints := rng.makeDisperseWaypoints(currentCoord)
						motion.Paths[pathDisperseLoop].Waypoints = disperseWaypoints
						motion.activate(pathDisperseLoop)

						// Set disperse color
						colorCopy := r.characterFinalColorMap[char]
						char.SetVisual(engine.CharacterVisual{
							Symbol: char.Symbol,
							Colors: &utils.ColorPair{FG: &colorCopy},
						})
					}
				}
			}
		} else {
			r.spinTimeRemaining--
		}

	case "final":
		// Step all character motions toward home
		allDone := true
		for _, char := range r.ringChars {
			motion := r.charMotion[char]
			_, completed := motion.step()
			char.Motion.SetCoordinate(motion.currentCoord())
			if motion.ActiveID != "" && !completed {
				allDone = false
			}
		}
		for _, char := range r.nonRingChars {
			motion := r.charMotion[char]
			_, completed := motion.step()
			char.Motion.SetCoordinate(motion.currentCoord())
			if motion.ActiveID != "" && !completed {
				allDone = false
			}
		}

		if allDone {
			r.phase = "complete"
			// Ensure all characters at final positions with final colors
			for _, char := range r.ringChars {
				char.Motion.SetCoordinate(char.InputCoord)
				colorCopy := r.characterFinalColorMap[char]
				char.SetVisual(engine.CharacterVisual{
					Symbol: char.Symbol,
					Colors: &utils.ColorPair{FG: &colorCopy},
				})
			}
			for _, char := range r.nonRingChars {
				char.Motion.SetCoordinate(char.InputCoord)
				colorCopy := r.characterFinalColorMap[char]
				char.SetVisual(engine.CharacterVisual{
					Symbol: char.Symbol,
					Colors: &utils.ColorPair{FG: &colorCopy},
				})
			}
		}

	case "complete":
		return r.base.Terminal.GetFormattedOutputString(), false
	}

	return r.base.Terminal.GetFormattedOutputString(), true
}

func (r *Rings) CanvasHeight() int {
	return r.base.CanvasHeight()
}

func (r *Rings) CanvasWidth() int {
	return r.base.CanvasWidth()
}
