package terminal

import "strings"

func clipToWidth(frame string, width int) string {
	if width <= 0 {
		return frame
	}
	lines := strings.Split(frame, "\n")
	for i, line := range lines {
		if len(line) > width {
			lines[i] = line[:width]
		}
	}
	return strings.Join(lines, "\n")
}

func lenLongestLine(frame string) int {
	max := 0
	for _, line := range strings.Split(frame, "\n") {
		if len(line) > max {
			max = len(line)
		}
	}
	return max
}
