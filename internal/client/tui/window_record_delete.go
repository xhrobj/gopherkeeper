package tui

import (
	"strings"

	"charm.land/lipgloss/v2"
)

const recordDeleteButtonGap = 3

type recordDeleteWindowLayout struct {
	content      string
	buttonBounds []layoutBounds
}

func recordDeleteWindowWidth(screenWidth int) int {
	return clamp(screenWidth-28, 44, 62)
}

func newRecordDeleteWindowLayout(
	t theme,
	width int,
	state recordDeleteState,
	pending bool,
	blocked bool,
	spinnerFrame string,
	activeButton int,
) recordDeleteWindowLayout {
	const (
		horizontalPadding = 2
		verticalPadding   = 1
	)

	contentWidth := max(1, width-2*horizontalPadding)
	preview := recordViewMetadataLines(state.metadata, contentWidth)
	if len(preview) > 5 {
		preview = preview[:5]
	}
	preview = append(preview, recordViewLine{})
	preview = appendRecordViewField(preview, "Title", state.metadata.Title, contentWidth)

	title := "Delete Record"
	if state.metadata.Type != "" {
		title += " " + recordTypeTitle(state.metadata.Type)
	}
	titleLine := renderWindowTitle(t.windowTitle, width, title, spinnerFrame, pending)

	rows := make([]string, 0, len(preview)+2)
	for _, line := range preview {
		rows = append(rows, renderRecordViewLineWithStyles(t.errorBody, t.errorLabel, contentWidth, line))
	}
	rows = append(rows, t.errorBody.Width(contentWidth).Render(""))
	buttonLayout := recordDeleteButtonsLayout(t, contentWidth, blocked, activeButton).
		positioned(horizontalPadding, lipgloss.Height(titleLine)+verticalPadding+len(rows))
	rows = append(rows, buttonLayout.content)

	body := t.errorBody.Width(width).Padding(verticalPadding, horizontalPadding).Render(strings.Join(rows, "\n"))

	return recordDeleteWindowLayout{
		content:      lipgloss.JoinVertical(lipgloss.Left, titleLine, body),
		buttonBounds: buttonLayout.bounds,
	}
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
	return newRecordDeleteWindowLayout(t, width, state, pending, blocked, spinnerFrame, activeButton).content
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
