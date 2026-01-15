package utils

import "time"

var globalRNG = NewMT19937(uint32(time.Now().UnixNano()))

func SeedRNG(seed uint32) {
	globalRNG = NewMT19937(seed)
}

func RandFloat64() float64 {
	return float64(globalRNG.Uint32()) / float64(uint64(1)<<32)
}

func RandIntn(n int) int {
	if n <= 0 {
		return 0
	}
	return int(globalRNG.Uint32() % uint32(n))
}

func Shuffle(n int, swap func(i, j int)) {
	for i := n - 1; i > 0; i-- {
		j := RandIntn(i + 1)
		swap(i, j)
	}
}
