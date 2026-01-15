package utils

import "math"

func PyRound(value float64) int {
	floor := math.Floor(value)
	frac := value - floor
	if frac > 0.5 {
		return int(math.Ceil(value))
	}
	if frac < 0.5 {
		return int(floor)
	}
	if int(floor)%2 == 0 {
		return int(floor)
	}
	return int(math.Ceil(value))
}
