package engine

import (
	"fmt"
	"math"

	"tte-go/internal/utils"
)

type GradientDirection string

const (
	GradientHorizontal GradientDirection = "horizontal"
	GradientVertical   GradientDirection = "vertical"
	GradientDiagonal   GradientDirection = "diagonal"
	GradientRadial     GradientDirection = "radial"
)

type Gradient struct {
	Stops []utils.Color
	Steps int
}

func NewGradient(stops []utils.Color, steps int) *Gradient {
	if steps <= 0 {
		steps = 1
	}
	return &Gradient{Stops: stops, Steps: steps}
}

func (g *Gradient) Colors() []utils.Color {
	if len(g.Stops) == 0 {
		return []utils.Color{}
	}
	if len(g.Stops) == 1 {
		return []utils.Color{g.Stops[0]}
	}
	colors := make([]utils.Color, 0, g.Steps)
	segments := len(g.Stops) - 1
	if segments <= 0 {
		return []utils.Color{g.Stops[0]}
	}
	stepsPerSegment := int(math.Max(1, float64(g.Steps)/float64(segments)))
	for i := 0; i < segments; i++ {
		start := g.Stops[i]
		end := g.Stops[i+1]
		for step := 0; step < stepsPerSegment; step++ {
			t := float64(step) / float64(stepsPerSegment)
			colors = append(colors, lerpColor(start, end, t))
		}
	}
	if len(colors) == 0 {
		colors = append(colors, g.Stops[len(g.Stops)-1])
	}
	return colors
}

func (g *Gradient) ColorAt(index int) utils.Color {
	colors := g.Colors()
	if len(colors) == 0 {
		return utils.Color{R: 255, G: 255, B: 255}
	}
	if index < 0 {
		index = 0
	}
	if index >= len(colors) {
		index = len(colors) - 1
	}
	return colors[index]
}

func (g *Gradient) BuildCoordinateColorMapping(top, bottom, left, right int, direction GradientDirection) map[utils.Coord]utils.Color {
	mapping := make(map[utils.Coord]utils.Color)
	colors := g.Colors()
	if len(colors) == 0 {
		return mapping
	}
	width := max(1, right-left+1)
	height := max(1, bottom-top+1)
	for row := top; row <= bottom; row++ {
		for col := left; col <= right; col++ {
			var index int
			switch direction {
			case GradientVertical:
				index = int(float64(row-top) / float64(height-1) * float64(len(colors)-1))
			case GradientHorizontal:
				index = int(float64(col-left) / float64(width-1) * float64(len(colors)-1))
			case GradientDiagonal:
				norm := (float64(row-top)/float64(height-1) + float64(col-left)/float64(width-1)) / 2
				index = int(norm * float64(len(colors)-1))
			case GradientRadial:
				centerRow := float64(top+bottom) / 2
				centerCol := float64(left+right) / 2
				dr := float64(row) - centerRow
				dc := float64(col) - centerCol
				dist := math.Sqrt(dr*dr + dc*dc)
				maxDist := math.Sqrt(math.Pow(float64(height)/2, 2) + math.Pow(float64(width)/2, 2))
				index = int(dist / maxDist * float64(len(colors)-1))
			default:
				return mapping
			}
			mapping[utils.Coord{Row: row, Col: col}] = colors[clamp(index, 0, len(colors)-1)]
		}
	}
	return mapping
}

func lerpColor(a, b utils.Color, t float64) utils.Color {
	return utils.Color{
		R: int(float64(a.R) + (float64(b.R)-float64(a.R))*t),
		G: int(float64(a.G) + (float64(b.G)-float64(a.G))*t),
		B: int(float64(a.B) + (float64(b.B)-float64(a.B))*t),
	}
}

func clamp(value, minValue, maxValue int) int {
	if value < minValue {
		return minValue
	}
	if value > maxValue {
		return maxValue
	}
	return value
}

func max(a, b int) int {
	if a > b {
		return a
	}
	return b
}

func ParseGradientDirection(value string) (GradientDirection, error) {
	switch value {
	case "horizontal", "vertical", "diagonal", "radial":
		return GradientDirection(value), nil
	default:
		return "", fmt.Errorf("invalid gradient direction: %s", value)
	}
}
