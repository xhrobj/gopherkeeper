package tui

import (
	"strings"

	"charm.land/lipgloss/v2"
)

func wrapRecordViewText(value string, width int) []string {
	width = max(1, width)
	paragraphs := strings.Split(normalizeRecordViewText(value), "\n")
	lines := make([]string, 0, len(paragraphs))

	for _, paragraph := range paragraphs {
		lines = append(lines, wrapRecordViewParagraph(paragraph, width)...)
	}

	return lines
}

func normalizeRecordViewText(value string) string {
	value = strings.ReplaceAll(value, "\r\n", "\n")
	value = strings.ReplaceAll(value, "\r", "\n")

	return strings.ReplaceAll(value, "\t", "    ")
}

func wrapRecordViewParagraph(paragraph string, width int) []string {
	if paragraph == "" {
		return []string{""}
	}

	runes := []rune(paragraph)
	lines := make([]string, 0, (len(runes)+width-1)/width)

	for len(runes) > 0 {
		end := recordViewTextChunkEnd(runes, width)
		lines = append(lines, string(runes[:end]))
		runes = runes[end:]
	}

	return lines
}

func recordViewTextChunkEnd(runes []rune, width int) int {
	used := 0
	for index, value := range runes {
		runeWidth := lipgloss.Width(string(value))
		if index > 0 && used+runeWidth > width {
			return index
		}
		used += runeWidth
		if used >= width {
			return index + 1
		}
	}

	return len(runes)
}

func fitSingleLinePreserve(value string, width int) string {
	if width <= 0 {
		return ""
	}

	if lipgloss.Width(value) <= width {
		return value
	}

	return fitSingleLine(value, width)
}
