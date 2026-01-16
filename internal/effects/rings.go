package effects

import (
	"math"

	"tte-go/internal/config"
	"tte-go/internal/engine"
	"tte-go/internal/utils"
)

// Rings creates an effect where characters are dispersed and form into spinning rings.
type Rings struct {
	base *BaseEffect

	rings            []*ring
	activeCharacters map[*engine.EffectCharacter]struct{}
	phase            string // "disperse", "spinning", "gather", "complete"
	cyclesRemaining  int
	frameCount       int
	currentDuration  int

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

type ring struct {
	radius        int
	origin        utils.Coord
	coords        []utils.Coord
	color         utils.Color
	characters    []*engine.EffectCharacter
	rotationSpeed float64
	angle         float64
	clockwise     bool
}

func NewRings(input string, cfg config.TerminalConfig) engine.Effect {
	base := NewBaseEffect(input, cfg)

	// Config defaults
	ringColors := mustColors("ab48ff", "e7b2b2", "fffebd")
	finalGradientStops := mustColors("ab48ff", "e7b2b2", "fffebd")
	finalGradientSteps := []int{12}
	finalGradientDirection := utils.GradientVertical

	r := &Rings{
		base:                   base,
		rings:                  make([]*ring, 0),
		activeCharacters:       make(map[*engine.EffectCharacter]struct{}),
		phase:                  "disperse",
		cyclesRemaining:        3,
		frameCount:             0,
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

func (r *Rings) build() {
	// Build final gradient mapping
	finalGradient, _ := utils.NewGradient(r.finalGradientStops, r.finalGradientSteps, false)
	finalGradientMapping, _ := finalGradient.BuildCoordinateColorMapping(
		r.base.Canvas.TextBottom,
		r.base.Canvas.TextTop,
		r.base.Canvas.TextLeft,
		r.base.Canvas.TextRight,
		r.finalGradientDirection,
	)

	// Calculate ring parameters
	origin := r.base.Canvas.Center
	smallestDim := min(r.base.Canvas.Width, r.base.Canvas.Height)
	ringGapPixels := int(float64(smallestDim) * r.ringGap)
	if ringGapPixels < 1 {
		ringGapPixels = 1
	}
	maxRadius := smallestDim / 2

	// Create rings from center outward
	allChars := r.base.Terminal.GetCharacters(true, false, false, false, engine.TopToBottomLeftToRight)

	// Shuffle characters
	for i := len(allChars) - 1; i > 0; i-- {
		j := utils.RandIntn(i + 1)
		allChars[i], allChars[j] = allChars[j], allChars[i]
	}

	charIndex := 0
	radius := ringGapPixels
	colorIndex := 0

	for radius <= maxRadius && charIndex < len(allChars) {
		// Calculate ring coordinates (circle)
		circumference := int(2 * math.Pi * float64(radius))
		numPoints := max(circumference, 4)

		ringCoords := make([]utils.Coord, 0, numPoints)
		for i := 0; i < numPoints; i++ {
			angle := 2 * math.Pi * float64(i) / float64(numPoints)
			col := origin.Col + int(math.Cos(angle)*float64(radius))
			row := origin.Row + int(math.Sin(angle)*float64(radius)/2) // /2 for terminal aspect ratio
			ringCoords = append(ringCoords, utils.Coord{Col: col, Row: row})
		}

		// Create ring
		rng := &ring{
			radius:        radius,
			origin:        origin,
			coords:        ringCoords,
			color:         r.ringColors[colorIndex%len(r.ringColors)],
			characters:    make([]*engine.EffectCharacter, 0),
			rotationSpeed: r.spinSpeedRange[0] + utils.RandFloat64()*(r.spinSpeedRange[1]-r.spinSpeedRange[0]),
			angle:         0,
			clockwise:     colorIndex%2 == 0,
		}

		// Assign characters to this ring
		charsForRing := min(len(ringCoords), len(allChars)-charIndex)
		for i := 0; i < charsForRing; i++ {
			char := allChars[charIndex]
			charIndex++

			// Set initial color from final gradient
			charFinalColor := finalGradientMapping[char.InputCoord]
			colorCopy := charFinalColor
			char.SetVisual(engine.CharacterVisual{
				Symbol: char.Symbol,
				Colors: &utils.ColorPair{FG: &colorCopy},
			})

			rng.characters = append(rng.characters, char)
			r.activeCharacters[char] = struct{}{}
		}

		r.rings = append(r.rings, rng)
		radius += ringGapPixels
		colorIndex++
	}

	// Make all characters visible in dispersed state
	for _, char := range allChars {
		r.base.Terminal.SetCharacterVisibility(char, true)
		// Disperse to random positions
		char.Coord = utils.Coord{
			Col: utils.RandIntn(r.base.Canvas.Width),
			Row: utils.RandIntn(r.base.Canvas.Height),
		}
		char.Motion.SetCoordinate(char.Coord)
	}

	r.currentDuration = r.disperseDuration
}

func (r *Rings) Next() (string, bool) {
	switch r.phase {
	case "disperse":
		r.frameCount++
		if r.frameCount >= r.currentDuration {
			r.phase = "spinning"
			r.frameCount = 0
			r.currentDuration = r.spinDuration
			// Move characters to ring positions
			for _, rng := range r.rings {
				for i, char := range rng.characters {
					if i < len(rng.coords) {
						coord := rng.coords[i]
						char.Coord = coord
						char.Motion.SetCoordinate(coord)
						// Set ring color
						colorCopy := rng.color
						char.SetVisual(engine.CharacterVisual{
							Symbol: char.Symbol,
							Colors: &utils.ColorPair{FG: &colorCopy},
						})
					}
				}
			}
		}

	case "spinning":
		// Rotate characters around rings
		for _, rng := range r.rings {
			// Update angle
			if rng.clockwise {
				rng.angle += rng.rotationSpeed * 0.1
			} else {
				rng.angle -= rng.rotationSpeed * 0.1
			}

			// Update character positions
			for i, char := range rng.characters {
				baseAngle := 2 * math.Pi * float64(i) / float64(len(rng.characters))
				angle := baseAngle + rng.angle
				col := rng.origin.Col + int(math.Cos(angle)*float64(rng.radius))
				row := rng.origin.Row + int(math.Sin(angle)*float64(rng.radius)/2)
				char.Coord = utils.Coord{Col: col, Row: row}
				char.Motion.SetCoordinate(char.Coord)
			}
		}

		r.frameCount++
		if r.frameCount >= r.currentDuration {
			r.cyclesRemaining--
			if r.cyclesRemaining > 0 {
				r.phase = "disperse"
				r.frameCount = 0
				r.currentDuration = r.disperseDuration
				// Disperse characters again
				for _, rng := range r.rings {
					for _, char := range rng.characters {
						char.Coord = utils.Coord{
							Col: utils.RandIntn(r.base.Canvas.Width),
							Row: utils.RandIntn(r.base.Canvas.Height),
						}
						char.Motion.SetCoordinate(char.Coord)
					}
				}
			} else {
				r.phase = "gather"
				r.frameCount = 0
			}
		}

	case "gather":
		// Move characters to final positions
		allDone := true
		for _, rng := range r.rings {
			for _, char := range rng.characters {
				// Move toward input coord
				dx := char.InputCoord.Col - char.Coord.Col
				dy := char.InputCoord.Row - char.Coord.Row
				if dx != 0 || dy != 0 {
					allDone = false
					// Move a fraction of the way
					if dx != 0 {
						if dx > 0 {
							char.Coord.Col++
						} else {
							char.Coord.Col--
						}
					}
					if dy != 0 {
						if dy > 0 {
							char.Coord.Row++
						} else {
							char.Coord.Row--
						}
					}
					char.Motion.SetCoordinate(char.Coord)
				}
			}
		}

		if allDone {
			r.phase = "complete"
			// Set final colors
			finalGradient, _ := utils.NewGradient(r.finalGradientStops, r.finalGradientSteps, false)
			finalGradientMapping, _ := finalGradient.BuildCoordinateColorMapping(
				r.base.Canvas.TextBottom,
				r.base.Canvas.TextTop,
				r.base.Canvas.TextLeft,
				r.base.Canvas.TextRight,
				r.finalGradientDirection,
			)
			for _, rng := range r.rings {
				for _, char := range rng.characters {
					charFinalColor := finalGradientMapping[char.InputCoord]
					colorCopy := charFinalColor
					char.SetVisual(engine.CharacterVisual{
						Symbol: char.Symbol,
						Colors: &utils.ColorPair{FG: &colorCopy},
					})
				}
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
