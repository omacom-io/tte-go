package utils

import (
	"fmt"
	"strconv"
	"strings"
)

type Color struct {
	R int
	G int
	B int
}

type ColorPair struct {
	FG *Color
	BG *Color
}

func ParseHexColor(value string) (*Color, error) {
	clean := strings.TrimPrefix(strings.TrimSpace(value), "#")
	if len(clean) != 6 {
		return nil, fmt.Errorf("invalid color: %s", value)
	}
	r, err := strconv.ParseInt(clean[0:2], 16, 0)
	if err != nil {
		return nil, err
	}
	g, err := strconv.ParseInt(clean[2:4], 16, 0)
	if err != nil {
		return nil, err
	}
	b, err := strconv.ParseInt(clean[4:6], 16, 0)
	if err != nil {
		return nil, err
	}
	return &Color{R: int(r), G: int(g), B: int(b)}, nil
}

func (c Color) Hex() string {
	return fmt.Sprintf("%02x%02x%02x", c.R, c.G, c.B)
}

func (c Color) HexWithHash() string {
	return "#" + c.Hex()
}

func AdjustBrightness(color Color, factor float64) Color {
	if factor < 0 {
		factor = 0
	}
	return Color{
		R: clampColor(int(float64(color.R) * factor)),
		G: clampColor(int(float64(color.G) * factor)),
		B: clampColor(int(float64(color.B) * factor)),
	}
}

func clampColor(value int) int {
	if value < 0 {
		return 0
	}
	if value > 255 {
		return 255
	}
	return value
}
