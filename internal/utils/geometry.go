package utils

import "math"

type Coord struct {
	Row int
	Col int
}

func Distance(a, b Coord, doubleRowDiff bool) float64 {
	dr := float64(a.Row - b.Row)
	dc := float64(a.Col - b.Col)
	if doubleRowDiff {
		dr *= 2
	}
	return math.Sqrt(dr*dr + dc*dc)
}

func Lerp(a, b Coord, t float64) Coord {
	row := float64(a.Row) + (float64(b.Row)-float64(a.Row))*t
	col := float64(a.Col) + (float64(b.Col)-float64(a.Col))*t
	return Coord{Row: PyRound(row), Col: PyRound(col)}
}

func BezierPoint(p0, p1, p2 Coord, t float64) Coord {
	u := 1 - t
	x := u*u*float64(p0.Col) + 2*u*t*float64(p1.Col) + t*t*float64(p2.Col)
	y := u*u*float64(p0.Row) + 2*u*t*float64(p1.Row) + t*t*float64(p2.Row)
	return Coord{Row: PyRound(y), Col: PyRound(x)}
}

func BezierLength(p0, p1, p2 Coord, samples int) float64 {
	if samples <= 0 {
		samples = 20
	}
	length := 0.0
	prev := p0
	for i := 1; i <= samples; i++ {
		t := float64(i) / float64(samples)
		curr := BezierPoint(p0, p1, p2, t)
		length += Distance(prev, curr, true)
		prev = curr
	}
	return length
}
