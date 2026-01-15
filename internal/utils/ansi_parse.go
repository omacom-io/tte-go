package utils

import (
	"fmt"
	"regexp"
	"strconv"
	"strings"
)

var ansiColorRegex = regexp.MustCompile("\x1b\\[[0-9;?]*m")

func ParseAnsiColorSequence(sequence string) (string, error) {
	trimmed := strings.TrimSuffix(sequence, "m")
	trimmed = strings.TrimPrefix(trimmed, "\x1b[")
	trimmed = strings.TrimPrefix(trimmed, "\033[")
	if strings.HasPrefix(trimmed, "38;2") || strings.HasPrefix(trimmed, "48;2") {
		parts := strings.Split(trimmed, ";")
		if len(parts) < 5 {
			return "", fmt.Errorf("invalid 24-bit color sequence: %s", sequence)
		}
		r := parseOrZero(parts[2])
		g := parseOrZero(parts[3])
		b := parseOrZero(parts[4])
		return fmt.Sprintf("%02x%02x%02x", r, g, b), nil
	}
	if strings.HasPrefix(trimmed, "38;5") || strings.HasPrefix(trimmed, "48;5") {
		parts := strings.Split(trimmed, ";")
		if len(parts) < 3 {
			return "", fmt.Errorf("invalid 8-bit color sequence: %s", sequence)
		}
		return parts[2], nil
	}
	return "", fmt.Errorf("invalid ansi color sequence: %s", sequence)
}

func ExtractAnsiSequences(input string) []string {
	return ansiColorRegex.FindAllString(input, -1)
}

func StripAnsiSequences(input string) string {
	return ansiColorRegex.ReplaceAllString(input, "")
}

func parseOrZero(value string) int {
	if value == "" {
		return 0
	}
	parsed, err := strconv.Atoi(value)
	if err != nil {
		return 0
	}
	return parsed
}
