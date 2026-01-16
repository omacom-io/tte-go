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

func FindCoordsOnCircle(origin Coord, radius int, coordsLimit int, unique bool) []Coord {
	points := []Coord{}
	if radius == 0 {
		return points
	}
	if coordsLimit == 0 {
		coordsLimit = PyRound(2 * math.Pi * float64(radius))
	}
	if coordsLimit <= 0 {
		coordsLimit = 1
	}
	angleStep := 2 * math.Pi / float64(coordsLimit)
	seen := map[Coord]struct{}{}
	for i := 0; i < coordsLimit; i++ {
		angle := angleStep * float64(i)
		x := float64(origin.Col) + float64(radius)*math.Cos(angle)
		xDiff := x - float64(origin.Col)
		x += xDiff
		y := float64(origin.Row) + float64(radius)*math.Sin(angle)
		point := Coord{Row: PyRound(y), Col: PyRound(x)}
		if unique {
			if _, ok := seen[point]; ok {
				continue
			}
			seen[point] = struct{}{}
		}
		points = append(points, point)
	}
	return points
}

func FindCoordsInCircle(center Coord, diameter int) []Coord {
	coords := []Coord{}
	if diameter == 0 {
		return coords
	}
	h := center.Col
	k := center.Row
	aSquared := float64(diameter * diameter)
	bSquared := math.Pow(float64(diameter)/2, 2)
	for x := h - diameter; x <= h+diameter; x++ {
		xComponent := float64((x-h)*(x-h)) / aSquared
		maxYOffset := int(math.Sqrt(math.Max(bSquared*(1-xComponent), 0)))
		for y := k - maxYOffset; y <= k+maxYOffset; y++ {
			coords = append(coords, Coord{Row: y, Col: x})
		}
	}
	return coords
}
