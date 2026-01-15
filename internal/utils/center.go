package utils

import "math"

func FindNormalizedDistanceFromCenter(minRow, maxRow, minCol, maxCol int, coord Coord) float64 {
	centerRow := float64(minRow+maxRow) / 2
	centerCol := float64(minCol+maxCol) / 2
	dr := float64(coord.Row) - centerRow
	dc := float64(coord.Col) - centerCol
	maxDist := math.Sqrt(math.Pow(float64(maxRow-minRow)/2, 2) + math.Pow(float64(maxCol-minCol)/2, 2))
	if maxDist == 0 {
		return 0
	}
	return math.Sqrt(dr*dr+dc*dc) / maxDist
}
