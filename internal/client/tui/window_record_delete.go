package tui

import (
	"strings"

	"charm.land/lipgloss/v2"
)

const (
	recordDeleteButtonRow = 10
	recordDeleteButtonGap = 3
)

func recordDeleteWindowWidth(screenWidth int) int {
	return clamp(screenWidth-28, 44, 62)
}

func renderRecordDeleteWindow(
	t theme,
	width int,
	state recordDeleteState,
	pending bool,
	blocked bool,
	spinnerFrame string,
	activeButton int,
) string {
	contentWidth := max(1, width-4)
	preview := recordViewMetadataLines(state.metadata, contentWidth)
	if len(preview) > 5 {
		preview = preview[:5]
	}
	preview = append(preview, recordViewLine{})
	preview = appendRecordViewField(preview, "Title", state.metadata.Title, contentWidth)

	rows := make([]string, 0, len(preview)+2)
	for _, line := range preview {
		rows = append(rows, renderRecordViewLineWithStyles(t.errorBody, t.errorLabel, contentWidth, line))
	}
	rows = append(rows,
		t.errorBody.Width(contentWidth).Render(""),
		renderRecordDeleteButtons(t, contentWidth, blocked, activeButton),
	)

	title := "Delete Record"
	if state.metadata.Type != "" {
		title += " " + recordTypeTitle(state.metadata.Type)
	}

	titleLine := renderWindowTitle(t.windowTitle, width, title, spinnerFrame, pending)
	body := t.errorBody.Width(width).Padding(1, 2).Render(strings.Join(rows, "\n"))

	return lipgloss.JoinVertical(lipgloss.Left, titleLine, body)
}

func recordDeleteButtonsLayout(t theme, width int, blocked bool, activeButton int) buttonRowLayout {
	labels := []string{"< Delete >", "< Cancel >"}
	buttons := make([]styledButton, len(labels))

	for index, label := range labels {
		style := t.errorText.Padding(0, 1)
		if blocked {
			style = t.errorButtonDisabled
			if index == activeButton {
				style = t.errorButtonDisabledActive
			}
		} else if index == activeButton {
			style = t.errorButton
		}
		buttons[index] = styledButton{label: label, style: style}
	}

	return centeredButtonRowLayout(t.errorBody, width, recordDeleteButtonGap, buttons)
}

func renderRecordDeleteButtons(t theme, width int, blocked bool, activeButton int) string {
	return recordDeleteButtonsLayout(t, width, blocked, activeButton).content
}
