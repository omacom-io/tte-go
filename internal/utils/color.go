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

// AdjustBrightness adjusts the brightness of a color using HSL color space.
// Factor > 1 brightens, factor < 1 darkens. This matches Python's implementation.
func AdjustBrightness(color Color, factor float64) Color {
	if factor < 0 {
		factor = 0
	}

	// Normalize RGB to 0-1 range
	r := float64(color.R) / 255.0
	g := float64(color.G) / 255.0
	b := float64(color.B) / 255.0

	// Convert RGB to HSL
	maxVal := maxFloat(r, maxFloat(g, b))
	minVal := minFloat(r, minFloat(g, b))
	lightness := (maxVal + minVal) / 2.0

	var hue, saturation float64

	if maxVal == minVal {
		// Achromatic (gray)
		hue = 0
		saturation = 0
	} else {
		diff := maxVal - minVal
		if lightness > 0.5 {
			saturation = diff / (2.0 - maxVal - minVal)
		} else {
			saturation = diff / (maxVal + minVal)
		}

		if maxVal == r {
			hue = (g - b) / diff
			if g < b {
				hue += 6
			}
		} else if maxVal == g {
			hue = (b-r)/diff + 2
		} else { // b
			hue = (r-g)/diff + 4
		}
		hue /= 6
	}

	// Adjust lightness
	lightness = lightness * factor
	if lightness < 0 {
		lightness = 0
	}
	if lightness > 1 {
		lightness = 1
	}

	// Convert back to RGB
	var newR, newG, newB float64

	if saturation == 0 {
		// Achromatic
		newR = lightness
		newG = lightness
		newB = lightness
	} else {
		var colorIntensity float64
		if lightness < 0.5 {
			colorIntensity = lightness * (1 + saturation)
		} else {
			colorIntensity = lightness + saturation - lightness*saturation
		}
		lightnessScaled := 2*lightness - colorIntensity

		newR = hueToRGB(lightnessScaled, colorIntensity, hue+1.0/3.0)
		newG = hueToRGB(lightnessScaled, colorIntensity, hue)
		newB = hueToRGB(lightnessScaled, colorIntensity, hue-1.0/3.0)
	}

	return Color{
		R: clampColor(int(newR * 255)),
		G: clampColor(int(newG * 255)),
		B: clampColor(int(newB * 255)),
	}
}

func maxFloat(a, b float64) float64 {
	if a > b {
		return a
	}
	return b
}

func minFloat(a, b float64) float64 {
	if a < b {
		return a
	}
	return b
}

// hueToRGB is a helper function for HSL to RGB conversion
func hueToRGB(lightnessScaled, colorIntensity, hueValue float64) float64 {
	if hueValue < 0 {
		hueValue += 1
	}
	if hueValue > 1 {
		hueValue -= 1
	}
	if hueValue < 1.0/6.0 {
		return lightnessScaled + (colorIntensity-lightnessScaled)*6*hueValue
	}
	if hueValue < 1.0/2.0 {
		return colorIntensity
	}
	if hueValue < 2.0/3.0 {
		return lightnessScaled + (colorIntensity-lightnessScaled)*(2.0/3.0-hueValue)*6
	}
	return lightnessScaled
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
