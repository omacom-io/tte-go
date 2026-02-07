package effects

import (
	"math"

	"tte-go/internal/config"
	"tte-go/internal/engine"
	"tte-go/internal/utils"
)

// Spotlights creates an effect where spotlights search the text area,
// illuminating characters, before converging in the center and expanding.
type Spotlights struct {
	base *BaseEffect

	spotlights          []spotlight
	activeCharacters    map[*engine.EffectCharacter]struct{}
	phase               string // "search", "converge", "expand", "complete"
	frameCount          int
	beamWidth           float64
	characterFinalColor map[*engine.EffectCharacter]utils.Color

	// Config
	beamWidthRatio         float64
	beamFalloff            float64
	searchDuration         int
	searchSpeedRange       [2]float64
	spotlightCount         int
	finalGradientStops     []utils.Color
	finalGradientSteps     []int
	finalGradientDirection utils.GradientDirection
}

type spotlight struct {
	x, y   float64
	vx, vy float64
}

func NewSpotlights(input string, cfg config.TerminalConfig) engine.Effect {
	base := NewBaseEffect(input, cfg)

	// Config defaults
	finalGradientStops := mustColors("ab48ff", "e7b2b2", "fffebd")
	finalGradientSteps := []int{12}
	finalGradientDirection := utils.GradientVertical

	s := &Spotlights{
		base:                   base,
		spotlights:             make([]spotlight, 0),
		activeCharacters:       make(map[*engine.EffectCharacter]struct{}),
		phase:                  "search",
		frameCount:             0,
		characterFinalColor:    make(map[*engine.EffectCharacter]utils.Color),
		beamWidthRatio:         2.0,
		beamFalloff:            0.3,
		searchDuration:         550,
		searchSpeedRange:       [2]float64{0.35, 0.75},
		spotlightCount:         3,
		finalGradientStops:     finalGradientStops,
		finalGradientSteps:     finalGradientSteps,
		finalGradientDirection: finalGradientDirection,
	}

	s.build()
	return s
}

func (s *Spotlights) build() {
	// Build final gradient mapping
	finalGradient, _ := utils.NewGradient(s.finalGradientStops, s.finalGradientSteps, false)
	finalGradientMapping, _ := finalGradient.BuildCoordinateColorMapping(
		s.base.Canvas.TextBottom,
		s.base.Canvas.TextTop,
		s.base.Canvas.TextLeft,
		s.base.Canvas.TextRight,
		s.finalGradientDirection,
	)

	// Calculate beam width
	minDim := min(s.base.Canvas.Width, s.base.Canvas.Height)
	beamRatio := s.beamWidthRatio
	if beamRatio < 1 {
		beamRatio = 1
	}
	s.beamWidth = float64(minDim) / beamRatio

	// Store final colors
	for _, char := range s.base.Terminal.GetCharacters(true, false, false, false, engine.TopToBottomLeftToRight) {
		s.characterFinalColor[char] = finalGradientMapping[char.InputCoord]
		// Start all characters as invisible (dark)
		dark := mustColors("111111")[0]
		char.SetVisual(engine.CharacterVisual{
			Symbol: char.Symbol,
			Colors: &utils.ColorPair{FG: &dark},
		})
		s.base.Terminal.SetCharacterVisibility(char, true)
	}

	// Create spotlights at random positions
	for i := 0; i < s.spotlightCount; i++ {
		speed := s.searchSpeedRange[0] + utils.RandFloat64()*(s.searchSpeedRange[1]-s.searchSpeedRange[0])
		angle := utils.RandFloat64() * 2 * math.Pi
		sp := spotlight{
			x:  float64(utils.RandIntn(s.base.Canvas.Width)),
			y:  float64(utils.RandIntn(s.base.Canvas.Height)),
			vx: math.Cos(angle) * speed,
			vy: math.Sin(angle) * speed,
		}
		s.spotlights = append(s.spotlights, sp)
	}
}

func (s *Spotlights) Next() (string, bool) {
	switch s.phase {
	case "search":
		// Move spotlights
		for i := range s.spotlights {
			sp := &s.spotlights[i]
			sp.x += sp.vx
			sp.y += sp.vy

			// Bounce off walls
			if sp.x < 0 || sp.x >= float64(s.base.Canvas.Width) {
				sp.vx = -sp.vx
				sp.x += sp.vx * 2
			}
			if sp.y < 0 || sp.y >= float64(s.base.Canvas.Height) {
				sp.vy = -sp.vy
				sp.y += sp.vy * 2
			}
		}

		// Update character illumination
		s.updateIllumination()

		s.frameCount++
		if s.frameCount >= s.searchDuration {
			s.phase = "converge"
			s.frameCount = 0
			// Set spotlight velocities toward center
			centerX := float64(s.base.Canvas.Width) / 2
			centerY := float64(s.base.Canvas.Height) / 2
			for i := range s.spotlights {
				sp := &s.spotlights[i]
				dx := centerX - sp.x
				dy := centerY - sp.y
				dist := math.Sqrt(dx*dx + dy*dy)
				if dist > 0 {
					sp.vx = dx / dist * 0.8
					sp.vy = dy / dist * 0.8
				}
			}
		}

	case "converge":
		// Move spotlights toward center
		centerX := float64(s.base.Canvas.Width) / 2
		centerY := float64(s.base.Canvas.Height) / 2
		allAtCenter := true

		for i := range s.spotlights {
			sp := &s.spotlights[i]
			dx := centerX - sp.x
			dy := centerY - sp.y
			dist := math.Sqrt(dx*dx + dy*dy)

			if dist > 2 {
				allAtCenter = false
				sp.x += sp.vx
				sp.y += sp.vy
			} else {
				sp.x = centerX
				sp.y = centerY
			}
		}

		s.updateIllumination()

		if allAtCenter {
			s.phase = "expand"
			s.frameCount = 0
		}

	case "expand":
		// Expand illumination from center
		s.frameCount++
		expandRadius := float64(s.frameCount) * 1.5

		for _, char := range s.base.Terminal.GetCharacters(true, false, false, false, engine.TopToBottomLeftToRight) {
			centerX := float64(s.base.Canvas.Width) / 2
			centerY := float64(s.base.Canvas.Height) / 2
			dx := float64(char.InputCoord.Col) - centerX
			dy := float64(char.InputCoord.Row) - centerY
			dist := math.Sqrt(dx*dx + dy*dy)

			if dist <= expandRadius {
				finalColor := s.characterFinalColor[char]
				colorCopy := finalColor
				char.SetVisual(engine.CharacterVisual{
					Symbol: char.Symbol,
					Colors: &utils.ColorPair{FG: &colorCopy},
				})
			}
		}

		maxDist := math.Sqrt(float64(s.base.Canvas.Width*s.base.Canvas.Width + s.base.Canvas.Height*s.base.Canvas.Height))
		if expandRadius > maxDist {
			s.phase = "complete"
		}

	case "complete":
		return s.base.Terminal.GetFormattedOutputString(), false
	}

	return s.base.Terminal.GetFormattedOutputString(), true
}

func (s *Spotlights) updateIllumination() {
	for _, char := range s.base.Terminal.GetCharacters(true, false, false, false, engine.TopToBottomLeftToRight) {
		maxBrightness := 0.0
		charX := float64(char.InputCoord.Col)
		charY := float64(char.InputCoord.Row)

		// Check distance to each spotlight
		for _, sp := range s.spotlights {
			dist := math.Sqrt((charX-sp.x)*(charX-sp.x) + (charY-sp.y)*(charY-sp.y))
			if dist < s.beamWidth {
				brightness := 1.0
				falloffStart := s.beamWidth * (1 - s.beamFalloff)
				if dist > falloffStart {
					brightness = 1.0 - (dist-falloffStart)/(s.beamWidth-falloffStart)
				}
				if brightness > maxBrightness {
					maxBrightness = brightness
				}
			}
		}

		// Set character color based on brightness
		finalColor := s.characterFinalColor[char]
		adjustedColor := utils.AdjustBrightness(finalColor, 0.1+maxBrightness*0.9)
		char.SetVisual(engine.CharacterVisual{
			Symbol: char.Symbol,
			Colors: &utils.ColorPair{FG: &adjustedColor},
		})
	}
}

func (s *Spotlights) CanvasHeight() int {
	return s.base.CanvasHeight()
}

func (s *Spotlights) CanvasWidth() int {
	return s.base.CanvasWidth()
}
