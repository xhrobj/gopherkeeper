package tui

import (
	"strings"

	"charm.land/lipgloss/v2"
	recordmodel "github.com/xhrobj/gopherkeeper/internal/model"
)

const (
	recordTypePickerWidth     = 50
	recordTypePickerFirstRow  = 4
	recordTypePickerCancelRow = 12
)

type recordTypePickerLayout struct {
	contentWidth int
	typeBounds   []layoutBounds
	cancelBounds layoutBounds
}

func newRecordTypePickerLayout() recordTypePickerLayout {
	const contentWidth = recordTypePickerWidth - 4

	typeBounds := make([]layoutBounds, len(recordCreateTypes))

	for index := range typeBounds {
		typeBounds[index] = layoutBounds{
			x:      2,
			y:      recordTypePickerFirstRow + index*2,
			width:  contentWidth,
			height: 1,
		}
	}

	return recordTypePickerLayout{
		contentWidth: contentWidth,
		typeBounds:   typeBounds,
		cancelBounds: layoutBounds{x: 2, y: recordTypePickerCancelRow, width: contentWidth, height: 1},
	}
}

func renderRecordTypePickerWindow(t theme, picker recordTypePicker) string {
	layout := newRecordTypePickerLayout()
	contentWidth := layout.contentWidth

	rows := make([]string, 0, 11)
	rows = append(rows,
		t.wizardPrompt.Width(contentWidth).Render("Select record type:"),
		t.wizardBody.Width(contentWidth).Render(""),
	)
	for index, recordType := range recordCreateTypes {
		rows = append(rows, renderRecordTypePickerRow(t, contentWidth, recordType, index == picker.selected))
		if index < len(recordCreateTypes)-1 {
			rows = append(rows, t.wizardBody.Width(contentWidth).Render(""))
		}
	}
	rows = append(rows, t.wizardBody.Width(contentWidth).Render(""))

	cancelStyle := t.aboutButton
	if picker.selected == len(recordCreateTypes) {
		cancelStyle = t.aboutButtonActive
	}
	rows = append(rows, renderSingleStyledButton(t.wizardBody, cancelStyle, contentWidth, "< Cancel >"))

	title := t.windowTitle.Width(recordTypePickerWidth).Render("New Record Wizard")
	body := t.wizardBody.
		Width(recordTypePickerWidth).
		Padding(1, 2).
		Render(strings.Join(rows, "\n"))

	return lipgloss.JoinVertical(lipgloss.Left, title, body)
}

func renderRecordTypePickerRow(t theme, width int, recordType recordmodel.RecordType, selected bool) string {
	rowStyle := t.wizardBody
	typeStyle := t.wizardType
	descriptionStyle := t.wizardDescription

	if selected {
		rowStyle = t.wizardSelected
		typeStyle = t.wizardSelectedType
		descriptionStyle = t.wizardSelectedDescription
	}

	name := strings.ToUpper(string(recordType))
	description := recordTypeDescription(recordType)

	const typeWidth = 12

	left := typeStyle.Width(typeWidth).Render(name)
	right := descriptionStyle.Width(max(1, width-typeWidth)).Render(description)
	row := left + right
	used := lipgloss.Width(row)

	if used < width {
		row += rowStyle.Width(width - used).Render("")
	}

	return row
}

func recordTypeDescription(recordType recordmodel.RecordType) string {
	switch recordType {
	case recordmodel.RecordTypeCredentials:
		return "Login, password and URL"
	case recordmodel.RecordTypeCard:
		return "Payment card details"
	case recordmodel.RecordTypeText:
		return "Arbitrary private text"
	case recordmodel.RecordTypeBinary:
		return "File up to 2 MiB"
	default:
		return ""
	}
}
