package utils

import (
	"fmt"
)

type GradientDirection int

const (
	GradientVertical GradientDirection = iota + 1
	GradientHorizontal
	GradientRadial
	GradientDiagonal
)

type Gradient struct {
	Stops    []Color
	Steps    []int
	Loop     bool
	Spectrum []Color
	Index    int
}

func NewGradient(stops []Color, steps []int, loop bool) (*Gradient, error) {
	if len(stops) == 0 {
		return nil, fmt.Errorf("at least one stop required")
	}
	if len(steps) == 0 {
		steps = []int{1}
	}
	gradient := &Gradient{Stops: stops, Steps: steps, Loop: loop}
	spectrum, err := gradient.generate()
	if err != nil {
		return nil, err
	}
	gradient.Spectrum = spectrum
	return gradient, nil
}

func (g *Gradient) generate() ([]Color, error) {
	steps := g.Steps
	for _, step := range steps {
		if step < 1 {
			return nil, fmt.Errorf("steps must be > 0")
		}
	}
	stops := g.Stops
	if len(stops) == 1 {
		colors := make([]Color, steps[0])
		for i := range colors {
			colors[i] = stops[0]
		}
		return colors, nil
	}
	if g.Loop {
		stops = append(stops, stops[0])
	}
	pairs := make([][2]Color, 0, len(stops)-1)
	for i := 1; i < len(stops); i++ {
		pairs = append(pairs, [2]Color{stops[i-1], stops[i]})
	}
	if len(steps) < len(pairs) {
		last := steps[len(steps)-1]
		for len(steps) < len(pairs) {
			steps = append(steps, last)
		}
	}
	colors := []Color{}
	for i, pair := range pairs {
		stepCount := steps[i]
		start := pair[0]
		end := pair[1]
		startInts := []int{start.R, start.G, start.B}
		endInts := []int{end.R, end.G, end.B}
		redDelta := (endInts[0] - startInts[0]) / stepCount
		greenDelta := (endInts[1] - startInts[1]) / stepCount
		blueDelta := (endInts[2] - startInts[2]) / stepCount
		rangeStart := 0
		if len(colors) > 0 {
			rangeStart = 1
		}
		for j := rangeStart; j < max(1, stepCount); j++ {
			red := clampColor(startInts[0] + redDelta*j)
			green := clampColor(startInts[1] + greenDelta*j)
			blue := clampColor(startInts[2] + blueDelta*j)
			colors = append(colors, Color{R: red, G: green, B: blue})
		}
		colors = append(colors, end)
	}
	return colors, nil
}

func (g *Gradient) GetColorAtFraction(fraction float64) (Color, error) {
	if fraction < 0 || fraction > 1 {
		return Color{}, fmt.Errorf("fraction must be between 0 and 1")
	}
	if len(g.Spectrum) == 0 {
		return Color{}, nil
	}
	for i := 1; i <= len(g.Spectrum); i++ {
		if fraction <= float64(i)/float64(len(g.Spectrum)) {
			return g.Spectrum[i-1], nil
		}
	}
	return g.Spectrum[len(g.Spectrum)-1], nil
}

func (g *Gradient) BuildCoordinateColorMapping(minRow, maxRow, minCol, maxCol int, direction GradientDirection) (map[Coord]Color, error) {
	if minRow < 1 || minCol < 1 || maxRow < 1 || maxCol < 1 {
		return nil, fmt.Errorf("min/max must be > 0")
	}
	if minRow > maxRow || minCol > maxCol {
		return nil, fmt.Errorf("min values must be <= max values")
	}
	rowOffset := minRow - 1
	colOffset := minCol - 1
	mapping := map[Coord]Color{}
	switch direction {
	case GradientVertical:
		for row := minRow; row <= maxRow; row++ {
			fraction := float64(row-rowOffset) / float64(maxRow-rowOffset)
			color, _ := g.GetColorAtFraction(fraction)
			for col := minCol; col <= maxCol; col++ {
				mapping[Coord{Row: row, Col: col}] = color
			}
		}
	case GradientHorizontal:
		for col := minCol; col <= maxCol; col++ {
			fraction := float64(col-colOffset) / float64(maxCol-colOffset)
			color, _ := g.GetColorAtFraction(fraction)
			for row := minRow; row <= maxRow; row++ {
				mapping[Coord{Row: row, Col: col}] = color
			}
		}
	case GradientRadial:
		for row := minRow; row <= maxRow; row++ {
			for col := minCol; col <= maxCol; col++ {
				distance := FindNormalizedDistanceFromCenter(minRow, maxRow, minCol, maxCol, Coord{Row: row, Col: col})
				color, _ := g.GetColorAtFraction(distance)
				mapping[Coord{Row: row, Col: col}] = color
			}
		}
	case GradientDiagonal:
		for row := minRow; row <= maxRow; row++ {
			for col := minCol; col <= maxCol; col++ {
				fraction := (float64(row-rowOffset)*2 + float64(col-colOffset)) / (float64(maxRow-rowOffset)*2 + float64(maxCol-colOffset))
				color, _ := g.GetColorAtFraction(fraction)
				mapping[Coord{Row: row, Col: col}] = color
			}
		}
	}
	return mapping, nil
}

func RandomColor() Color {
	value := RandIntn(0xFFFFFF + 1)
	return Color{R: (value >> 16) & 0xFF, G: (value >> 8) & 0xFF, B: value & 0xFF}
}

func ShiftColorTowards(color Color, target Color, factor float64) Color {
	if factor < 0 {
		factor = 0
	}
	if factor > 1 {
		factor = 1
	}
	interpolate := func(start, end float64, factor float64) float64 {
		return start + (end-start)*factor
	}
	colorRed := float64(color.R) / 255
	colorGreen := float64(color.G) / 255
	colorBlue := float64(color.B) / 255
	targetRed := float64(target.R) / 255
	targetGreen := float64(target.G) / 255
	targetBlue := float64(target.B) / 255
	newRed := interpolate(colorRed, targetRed, factor)
	newGreen := interpolate(colorGreen, targetGreen, factor)
	newBlue := interpolate(colorBlue, targetBlue, factor)
	return Color{R: clampColor(int(newRed * 255)), G: clampColor(int(newGreen * 255)), B: clampColor(int(newBlue * 255))}
}

func max(a, b int) int {
	if a > b {
		return a
	}
	return b
}
