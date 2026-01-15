package utils

import "fmt"

func FgXterm(code int) string {
	return fmt.Sprintf("\x1b[38;5;%dm", code)
}

func BgXterm(code int) string {
	return fmt.Sprintf("\x1b[48;5;%dm", code)
}

func FgRGB(hex string) string {
	color, err := ParseHexColor(hex)
	if err != nil {
		return ""
	}
	return fmt.Sprintf("\x1b[38;2;%d;%d;%dm", color.R, color.G, color.B)
}

func BgRGB(hex string) string {
	color, err := ParseHexColor(hex)
	if err != nil {
		return ""
	}
	return fmt.Sprintf("\x1b[48;2;%d;%d;%dm", color.R, color.G, color.B)
}
