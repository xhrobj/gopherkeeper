package tui

import (
	"strings"

	"charm.land/lipgloss/v2"
)

type textFieldPolicy int

const (
	textFieldASCII textFieldPolicy = iota
	textFieldUnicode
)

type textField struct {
	value  string
	cursor int
	policy textFieldPolicy
	masked bool
}

func newASCIITextField(value string, masked bool) textField {
	field := textField{policy: textFieldASCII, masked: masked}
	field.setValue(value)
	return field
}

func newUnicodeTextField(value string) textField {
	field := textField{policy: textFieldUnicode}
	field.setValue(value)
	return field
}

func (field *textField) setValue(value string) {
	field.value = field.normalize(value)
	field.cursor = len([]rune(field.value))
}

func (field textField) normalize(value string) string {
	if field.policy == textFieldUnicode {
		return normalizeUnicodeTextFieldValue(value)
	}
	return normalizeTextFieldValue(value)
}

func (field textField) keyValue(key string) (string, bool) {
	if field.policy == textFieldUnicode {
		return printableUnicodeTextKey(key)
	}
	return printableTextKey(key)
}

func (field *textField) insertKey(key string) bool {
	value, ok := field.keyValue(key)
	if !ok {
		return false
	}
	return field.insert(value)
}

func (field *textField) insert(value string) bool {
	current := []rune(field.normalize(field.value))
	inserted := []rune(field.normalize(value))
	field.cursor = clamp(field.cursor, 0, len(current))

	if len(inserted) == 0 {
		field.value = string(current)
		return false
	}

	available := max(0, maxTextFieldLength-len(current))
	if len(inserted) > available {
		inserted = inserted[:available]
	}
	if len(inserted) == 0 {
		field.value = string(current)
		return false
	}

	result := make([]rune, 0, len(current)+len(inserted))
	result = append(result, current[:field.cursor]...)
	result = append(result, inserted...)
	result = append(result, current[field.cursor:]...)
	field.value = string(result)
	field.cursor += len(inserted)
	return true
}

func (field *textField) backspace() bool {
	runes := []rune(field.value)
	field.cursor = clamp(field.cursor, 0, len(runes))
	if field.cursor == 0 {
		return false
	}

	field.value = string(append(runes[:field.cursor-1], runes[field.cursor:]...))
	field.cursor--
	return true
}

func (field *textField) delete() bool {
	runes := []rune(field.value)
	field.cursor = clamp(field.cursor, 0, len(runes))
	if field.cursor >= len(runes) {
		return false
	}

	field.value = string(append(runes[:field.cursor], runes[field.cursor+1:]...))
	return true
}

func (field *textField) moveCursor(step int) {
	field.cursor = clamp(field.cursor+step, 0, len([]rune(field.value)))
}

func (field *textField) moveCursorToStart() {
	field.cursor = 0
}

func (field *textField) moveCursorToEnd() {
	field.cursor = len([]rune(field.value))
}

func (field textField) displayValue() string {
	if !field.masked {
		return field.value
	}
	return strings.Repeat("*", len([]rune(field.value)))
}

func renderTextField(t theme, field textField, width int, active bool) string {
	width = max(1, width)
	runes := []rune(field.displayValue())
	cursor := clamp(field.cursor, 0, len(runes))

	if !active {
		start := visibleSuffixStart(runes, width)
		visible := runes[start:]
		text := string(visible)
		padding := max(0, width-lipgloss.Width(text))
		return t.input.Render(text + strings.Repeat(" ", padding))
	}

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

func visiblePrefixEnd(runes []rune, width int) int {
	used := 0

	for index, value := range runes {
		valueWidth := lipgloss.Width(string(value))
		if used+valueWidth > width {
			return index
		}
		used += valueWidth
	}

	return len(runes)
}

func visibleSuffixStart(runes []rune, width int) int {
	used := 0

	for index := len(runes) - 1; index >= 0; index-- {
		valueWidth := lipgloss.Width(string(runes[index]))
		if used+valueWidth > width {
			return index + 1
		}
		used += valueWidth
	}

	return 0
}

func visibleStartBeforeCursor(runes []rune, cursor, width int) int {
	used := 0

	for index := cursor - 1; index >= 0; index-- {
		valueWidth := lipgloss.Width(string(runes[index]))
		if used+valueWidth > width {
			return index + 1
		}
		used += valueWidth
	}

	return 0
}
