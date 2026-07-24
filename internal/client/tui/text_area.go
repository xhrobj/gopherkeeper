package tui

import (
	"strings"
	"unicode"
	"unicode/utf8"

	"charm.land/lipgloss/v2"
	recordmodel "github.com/xhrobj/gopherkeeper/internal/model"
)

const recordTextAreaHeight = 5

type textArea struct {
	value  string
	cursor int
	scroll int
}

func newTextArea(value string) textArea {
	area := textArea{}
	area.setValue(value)
	return area
}

func (area *textArea) setValue(value string) {
	area.value = normalizeTextAreaValue(value, recordmodel.TextPayloadMaxSize)
	area.cursor = utf8.RuneCountInString(area.value)
	area.scroll = 0
}

func normalizeTextAreaValue(value string, maxBytes int) string {
	value = strings.ReplaceAll(value, "\r\n", "\n")
	value = strings.ReplaceAll(value, "\r", "\n")

	var result strings.Builder
	result.Grow(min(len(value), maxBytes))

	for _, valueRune := range value {
		switch {
		case valueRune == '\n':
			// Разрешаем перевод строки как часть произвольного text payload.
		case valueRune == '\t':
			valueRune = ' '
		case !unicode.IsPrint(valueRune):
			continue
		}

		encodedLength := utf8.RuneLen(valueRune)
		if encodedLength < 0 || result.Len()+encodedLength > maxBytes {
			break
		}
		result.WriteRune(valueRune)
	}

	return result.String()
}

func (area *textArea) insert(value string) bool {
	inserted := normalizeTextAreaValue(value, recordmodel.TextPayloadMaxSize)
	if inserted == "" {
		return false
	}

	current := []rune(area.value)
	area.cursor = clamp(area.cursor, 0, len(current))
	prefix := string(current[:area.cursor])
	suffix := string(current[area.cursor:])

	available := recordmodel.TextPayloadMaxSize - len(prefix) - len(suffix)
	if available <= 0 {
		return false
	}

	inserted = normalizeTextAreaValue(inserted, available)
	if inserted == "" {
		return false
	}

	area.value = prefix + inserted + suffix
	area.cursor += len([]rune(inserted))

	return true
}

func (area *textArea) insertKey(key string) bool {
	if key == "space" {
		return area.insert(" ")
	}

	runes := []rune(key)
	if len(runes) != 1 || !unicode.IsPrint(runes[0]) || runes[0] == '\t' {
		return false
	}

	return area.insert(key)
}

func (area *textArea) insertNewline() bool {
	return area.insert("\n")
}

func (area *textArea) backspace() bool {
	runes := []rune(area.value)
	area.cursor = clamp(area.cursor, 0, len(runes))

	if area.cursor == 0 {
		return false
	}

	area.value = string(append(runes[:area.cursor-1], runes[area.cursor:]...))
	area.cursor--

	return true
}

func (area *textArea) delete() bool {
	runes := []rune(area.value)
	area.cursor = clamp(area.cursor, 0, len(runes))

	if area.cursor >= len(runes) {
		return false
	}

	area.value = string(append(runes[:area.cursor], runes[area.cursor+1:]...))

	return true
}

func (area *textArea) moveCursor(step int) {
	area.cursor = clamp(area.cursor+step, 0, utf8.RuneCountInString(area.value))
}

func (area *textArea) moveCursorToLineStart() {
	runes := []rune(area.value)
	area.cursor = clamp(area.cursor, 0, len(runes))

	for area.cursor > 0 && runes[area.cursor-1] != '\n' {
		area.cursor--
	}
}

func (area *textArea) moveCursorToLineEnd() {
	runes := []rune(area.value)
	area.cursor = clamp(area.cursor, 0, len(runes))

	for area.cursor < len(runes) && runes[area.cursor] != '\n' {
		area.cursor++
	}
}

func (area *textArea) moveLine(step int) {
	lines, row, column := area.linesAndCursor()
	if len(lines) == 0 {
		return
	}

	targetRow := clamp(row+step, 0, len(lines)-1)
	targetColumn := min(column, len([]rune(lines[targetRow])))

	area.cursor = textAreaCursorIndex(lines, targetRow, targetColumn)
}

func (area *textArea) movePage(step, height int) {
	area.moveLine(step * max(1, height))
}

func (area *textArea) ensureVisible(height int) {
	lines, cursorRow, _ := area.linesAndCursor()
	area.ensureVisibleFor(height, len(lines), cursorRow)
}

func (area *textArea) ensureVisibleFor(height, lineCount, cursorRow int) {
	height = max(1, height)

	if cursorRow < area.scroll {
		area.scroll = cursorRow
	}

	if cursorRow >= area.scroll+height {
		area.scroll = cursorRow - height + 1
	}

	area.scroll = clamp(area.scroll, 0, max(0, lineCount-height))
}

func (area textArea) linesAndCursor() ([]string, int, int) {
	lines := strings.Split(area.value, "\n")
	if len(lines) == 0 {
		lines = []string{""}
	}

	runes := []rune(area.value)
	cursor := clamp(area.cursor, 0, len(runes))

	row := 0
	column := 0

	for _, valueRune := range runes[:cursor] {
		if valueRune == '\n' {
			row++
			column = 0
			continue
		}
		column++
	}

	row = clamp(row, 0, len(lines)-1)
	column = min(column, len([]rune(lines[row])))

	return lines, row, column
}

func textAreaCursorIndex(lines []string, row, column int) int {
	row = clamp(row, 0, max(0, len(lines)-1))
	index := 0

	for lineIndex := 0; lineIndex < row; lineIndex++ {
		index += len([]rune(lines[lineIndex])) + 1
	}

	return index + clamp(column, 0, len([]rune(lines[row])))
}

func renderTextArea(t theme, area textArea, width, height int, active bool) string {
	width = max(1, width)
	height = max(1, height)

	lines, cursorRow, cursorColumn := area.linesAndCursor()
	area.ensureVisibleFor(height, len(lines), cursorRow)

	rows := make([]string, 0, height)
	for index := 0; index < height; index++ {
		lineIndex := area.scroll + index
		if lineIndex >= len(lines) {
			rows = append(rows, t.input.Width(width).Render(""))
			continue
		}

		lineRunes := []rune(lines[lineIndex])
		if active && lineIndex == cursorRow {
			rows = append(rows, renderTextAreaCursorLine(t, lineRunes, cursorColumn, width))
			continue
		}

		end := visiblePrefixEnd(lineRunes, width)
		text := string(lineRunes[:end])
		rows = append(rows, t.input.Width(width).Render(text))
	}

	return strings.Join(rows, "\n")
}

func renderTextAreaCursorLine(t theme, runes []rune, cursor, width int) string {
	cursor = clamp(cursor, 0, len(runes))
	cursorText := " "
	afterStart := cursor

	if cursor < len(runes) {
		if value := string(runes[cursor]); lipgloss.Width(value) > 0 {
			cursorText = value
		}
		afterStart++
	}

	cursorWidth := lipgloss.Width(cursorText)
	start := visibleStartBeforeCursor(runes, cursor, max(0, width-cursorWidth))
	before := string(runes[start:cursor])
	beforeWidth := lipgloss.Width(before)
	remainingWidth := max(0, width-beforeWidth-cursorWidth)
	end := afterStart + visiblePrefixEnd(runes[afterStart:], remainingWidth)
	after := string(runes[afterStart:end])
	used := beforeWidth + cursorWidth + lipgloss.Width(after)
	padding := max(0, width-used)

	return t.input.Render(before) +
		t.inputCursor.Render(cursorText) +
		t.input.Render(after+strings.Repeat(" ", padding))
}
