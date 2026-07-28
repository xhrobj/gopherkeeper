package tui

import (
	"strings"

	"charm.land/lipgloss/v2"
)

const (
	textFieldASCII textFieldPolicy = iota
	textFieldUnicode
	textFieldDigits
)

type textFieldPolicy int

type textField struct {
	value     string
	cursor    int
	policy    textFieldPolicy
	masked    bool
	maxLength int
}

func newASCIITextField(value string, masked bool) textField {
	return newASCIITextFieldWithLimit(value, masked, maxTextFieldLength)
}

func newASCIITextFieldWithLimit(value string, masked bool, maxLength int) textField {
	field := textField{
		policy:    textFieldASCII,
		masked:    masked,
		maxLength: clamp(maxLength, 1, maxTextFieldLength),
	}
	field.setValue(value)

	return field
}

func newUnicodeTextField(value string) textField {
	return newUnicodeTextFieldWithLimit(value, maxTextFieldLength)
}

func newUnicodeTextFieldWithLimit(value string, maxLength int) textField {
	field := textField{
		policy:    textFieldUnicode,
		maxLength: clamp(maxLength, 1, maxTextFieldLength),
	}
	field.setValue(value)

	return field
}

func newDigitsTextFieldWithLimit(value string, maxLength int) textField {
	field := textField{
		policy:    textFieldDigits,
		maxLength: clamp(maxLength, 1, maxTextFieldLength),
	}
	field.setValue(value)

	return field
}

func (field *textField) setValue(value string) {
	field.value = field.normalize(value)
	field.cursor = len([]rune(field.value))
}

func (field textField) normalize(value string) string {
	switch field.policy {
	case textFieldUnicode:
		value = normalizeUnicodeTextFieldValue(value)
	case textFieldDigits:
		value = normalizeDigitsTextFieldValue(value)
	default:
		value = normalizeTextFieldValue(value)
	}

	limit := field.maxLength
	if limit <= 0 || limit > maxTextFieldLength {
		limit = maxTextFieldLength
	}

	runes := []rune(value)
	if len(runes) > limit {
		runes = runes[:limit]
	}

	return string(runes)
}

func (field textField) keyValue(key string) (string, bool) {
	switch field.policy {
	case textFieldUnicode:
		return printableUnicodeTextKey(key)
	case textFieldDigits:
		return printableDigitsTextKey(key)
	default:
		return printableTextKey(key)
	}
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

	limit := field.maxLength
	if limit <= 0 || limit > maxTextFieldLength {
		limit = maxTextFieldLength
	}

	available := max(0, limit-len(current))
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
	return renderStyledTextField(field, width, active, false, t.input, t.inputCursor)
}

func renderStyledTextField(
	field textField,
	width int,
	active bool,
	keepEndCursorInside bool,
	bodyStyle lipgloss.Style,
	cursorStyle lipgloss.Style,
) string {
	width = max(1, width)
	runes := []rune(field.displayValue())
	cursor := clamp(field.cursor, 0, len(runes))

	if active && keepEndCursorInside && len(runes) == width && cursor == len(runes) {
		cursor--
	}

	if !active {
		start := visibleSuffixStart(runes, width)
		text := string(runes[start:])
		return bodyStyle.Render(text + strings.Repeat(" ", max(0, width-lipgloss.Width(text))))
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

	return bodyStyle.Render(before) +
		cursorStyle.Render(cursorText) +
		bodyStyle.Render(after+strings.Repeat(" ", max(0, width-used)))
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
