package utils

import "math"

var xtermPalette = buildXtermPalette()

func buildXtermPalette() []Color {
	palette := make([]Color, 0, 256)
	base := []Color{
		{0, 0, 0}, {128, 0, 0}, {0, 128, 0}, {128, 128, 0}, {0, 0, 128}, {128, 0, 128}, {0, 128, 128}, {192, 192, 192},
		{128, 128, 128}, {255, 0, 0}, {0, 255, 0}, {255, 255, 0}, {0, 0, 255}, {255, 0, 255}, {0, 255, 255}, {255, 255, 255},
	}
	palette = append(palette, base...)

	steps := []int{0x00, 0x5f, 0x87, 0xaf, 0xd7, 0xff}
	for _, r := range steps {
		for _, g := range steps {
			for _, b := range steps {
				palette = append(palette, Color{R: r, G: g, B: b})
			}
		}
	}

	for i := 0; i < 24; i++ {
		value := 8 + i*10
		palette = append(palette, Color{R: value, G: value, B: value})
	}
	return palette
}

func HexToXterm(hex string) int {
	color, err := ParseHexColor(hex)
	if err != nil {
		return 0
	}
	return NearestXtermColor(*color)
}

func NearestXtermColor(color Color) int {
	closest := 0
	minDiff := math.MaxFloat64
	for i, candidate := range xtermPalette {
		diff := colorDistance(color, candidate)
		if diff < minDiff {
			minDiff = diff
			closest = i
		}
	}
	return closest
}

func XtermToHex(code int) string {
	if code < 0 || code >= len(xtermPalette) {
		return "000000"
	}
	return xtermPalette[code].Hex()
}

func colorDistance(a, b Color) float64 {
	dr := float64(a.R - b.R)
	dg := float64(a.G - b.G)
	db := float64(a.B - b.B)
	return (math.Abs(dr) + math.Abs(dg) + math.Abs(db)) / 3
}
