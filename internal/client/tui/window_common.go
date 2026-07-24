package tui

import (
	"strings"

	"charm.land/lipgloss/v2"
)

const currentUserButtonRow = 4

func renderWindowTitle(style lipgloss.Style, width int, title, spinnerFrame string, pending bool) string {
	title = fitSingleLine(title, max(1, width))
	titleWidth := lipgloss.Width(title)
	left := max(0, (width-titleWidth)/2)

	suffix := ""
	if pending && spinnerFrame != "" {
		suffix = " " + spinnerFrame
	}
	right := max(0, width-left-titleWidth-lipgloss.Width(suffix))
	line := strings.Repeat(" ", left) + title + suffix + strings.Repeat(" ", right)

	return style.Width(width).AlignHorizontal(lipgloss.Left).Render(line)
}

func renderShadow(t theme, width, height int) string {
	return t.shadow.Width(max(1, width)).Height(max(1, height)).Render("")
}

func renderCurrentUserWindow(
	t theme,
	width int,
	login string,
	pending bool,
	blocked bool,
	spinnerFrame string,
) string {
	contentWidth := max(1, width-4)
	message := t.aboutText.Render("Logged in as ") + t.aboutTitle.Render(login)
	buttonStyle := t.aboutButtonActive

	if pending {
		message = ""
	}

	if blocked {
		buttonStyle = t.aboutButtonDisabledActive
	}

	rows := []string{
		t.aboutBody.Width(contentWidth).AlignHorizontal(lipgloss.Center).Render(message),
		t.aboutBody.Width(contentWidth).Render(""),
		renderSingleStyledButton(t.aboutBody, buttonStyle, contentWidth, "< OK >"),
	}

	title := renderWindowTitle(t.windowTitle, width, "Current User", spinnerFrame, pending)
	body := t.aboutBody.
		Width(width).
		Padding(1, 2).
		Render(lipgloss.JoinVertical(lipgloss.Left, rows...))

	return lipgloss.JoinVertical(lipgloss.Left, title, body)
}

func renderAlertWindow(
	t theme,
	state alertState,
	title string,
	message string,
	highlight string,
) string {
	const width = 54

	bodyStyle := t.aboutBody
	titleStyle := t.aboutTitle
	buttonStyle := t.aboutButtonActive

	if state == alertError {
		bodyStyle = t.errorBody
		titleStyle = t.errorTitle
		buttonStyle = t.errorButton
	}

	contentWidth := width - 4
	rows := []string{
		bodyStyle.Width(width).Render(""),
		titleStyle.Width(width).AlignHorizontal(lipgloss.Center).Render(title),
		bodyStyle.Width(width).Render(""),
	}

	for _, messageRow := range renderAlertMessageRows(bodyStyle, titleStyle, contentWidth, message, highlight) {
		rows = append(rows,
			bodyStyle.Width(2).Render("")+messageRow+bodyStyle.Width(2).Render(""),
		)
	}

	rows = append(rows,
		bodyStyle.Width(width).Render(""),
		bodyStyle.Width(2).Render("")+
			renderSingleStyledButton(bodyStyle, buttonStyle, contentWidth, "< OK >")+
			bodyStyle.Width(2).Render(""),
		bodyStyle.Width(width).Render(""),
	)

	return lipgloss.JoinVertical(lipgloss.Left, rows...)
}

func renderAlertMessageRows(
	bodyStyle lipgloss.Style,
	highlightStyle lipgloss.Style,
	width int,
	message string,
	highlight string,
) []string {
	if highlight != "" {
		return []string{renderAlertMessage(bodyStyle, highlightStyle, width, message, highlight)}
	}

	lines := wrapAlertText(message, width)
	if len(lines) == 0 {
		lines = []string{""}
	}

	rows := make([]string, 0, len(lines))
	for _, line := range lines {
		rows = append(rows, bodyStyle.Width(width).AlignHorizontal(lipgloss.Center).Render(line))
	}

	return rows
}

func wrapAlertText(value string, width int) []string {
	width = max(1, width)
	value = strings.ReplaceAll(value, "\r\n", "\n")
	value = strings.ReplaceAll(value, "\r", "\n")

	paragraphs := strings.Split(value, "\n")
	lines := make([]string, 0, len(paragraphs))

	for _, paragraph := range paragraphs {
		words := strings.Fields(paragraph)
		if len(words) == 0 {
			lines = append(lines, "")
			continue
		}

		line := ""
		for _, word := range words {
			candidate := word
			if line != "" {
				candidate = line + " " + word
			}
			if lipgloss.Width(candidate) <= width {
				line = candidate
				continue
			}

			if line != "" {
				lines = append(lines, line)
				line = ""
			}

			if lipgloss.Width(word) <= width {
				line = word
				continue
			}

			parts := wrapRecordViewText(word, width)
			if len(parts) == 0 {
				continue
			}
			lines = append(lines, parts[:len(parts)-1]...)
			line = parts[len(parts)-1]
		}

		if line != "" {
			lines = append(lines, line)
		}
	}

	return lines
}

func alertButtonRow(message, highlight string) int {
	if highlight != "" {
		return 5
	}

	lineCount := len(wrapAlertText(message, 50))
	if lineCount == 0 {
		lineCount = 1
	}

	return 4 + lineCount
}

func renderAlertMessage(
	bodyStyle lipgloss.Style,
	highlightStyle lipgloss.Style,
	width int,
	message string,
	highlight string,
) string {
	value := fitSingleLine(message, width)
	if highlight == "" {
		return bodyStyle.Width(width).AlignHorizontal(lipgloss.Center).Render(value)
	}

	index := strings.Index(value, highlight)
	if index < 0 {
		return bodyStyle.Width(width).AlignHorizontal(lipgloss.Center).Render(value)
	}

	before := bodyStyle.Render(value[:index])
	highlighted := highlightStyle.Render(highlight)
	after := bodyStyle.Render(value[index+len(highlight):])
	contentWidth := lipgloss.Width(before) + lipgloss.Width(highlighted) + lipgloss.Width(after)
	left := max(0, (width-contentWidth)/2)
	right := max(0, width-contentWidth-left)

	return bodyStyle.Width(left).Render("") +
		before +
		highlighted +
		after +
		bodyStyle.Width(right).Render("")
}

func singleStyledButtonLayout(
	background lipgloss.Style,
	style lipgloss.Style,
	width int,
	label string,
) buttonRowLayout {
	return centeredButtonRowLayout(background, width, 0, []styledButton{{label: label, style: style}})
}

func renderSingleStyledButton(
	background lipgloss.Style,
	style lipgloss.Style,
	width int,
	label string,
) string {
	return singleStyledButtonLayout(background, style, width, label).content
}

func fitSingleLine(value string, width int) string {
	value = strings.Join(strings.Fields(value), " ")

	if width <= 0 {
		return ""
	}

	if lipgloss.Width(value) <= width {
		return value
	}

	const truncationMarker = "…"
	markerWidth := lipgloss.Width(truncationMarker)
	if width <= markerWidth {
		return truncationMarker
	}

	runes := []rune(value)
	limit := width - markerWidth

	for len(runes) > 0 && lipgloss.Width(string(runes)) > limit {
		runes = runes[:len(runes)-1]
	}

	return string(runes) + truncationMarker
}

type formButton struct {
	label    string
	active   bool
	disabled bool
}

func formButtonsLayout(t theme, width, gap int, buttons []formButton) buttonRowLayout {
	styled := make([]styledButton, 0, len(buttons))

	for _, button := range buttons {
		style := t.button
		switch {
		case button.disabled && button.active:
			style = t.buttonDisabledActive
		case button.disabled:
			style = t.buttonDisabled
		case button.active:
			style = t.buttonActive
		}
		styled = append(styled, styledButton{label: button.label, style: style})
	}

	return centeredButtonRowLayout(t.windowBody, width, gap, styled)
}
