package tui

import "unicode"

func clamp(value, minimum, maximum int) int {
	return min(max(value, minimum), maximum)
}

func capitalizeFirst(value string) string {
	runes := []rune(value)
	if len(runes) == 0 {
		return value
	}

	runes[0] = unicode.ToUpper(runes[0])
	return string(runes)
}
