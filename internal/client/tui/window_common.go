package tui

import (
	"strings"

	"charm.land/lipgloss/v2"
)

type currentUserWindowLayout struct {
	content      string
	buttonBounds []layoutBounds
}

type alertWindowLayout struct {
	content      string
	buttonBounds []layoutBounds
}

type formButton struct {
	label    string
	active   bool
	disabled bool
}

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

func newCurrentUserWindowLayout(
	t theme,
	width int,
	login string,
	pending bool,
	blocked bool,
	spinnerFrame string,
) currentUserWindowLayout {
	const (
		horizontalPadding = 2
		verticalPadding   = 1
	)

	contentWidth := max(1, width-2*horizontalPadding)
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
	}
	buttonRow := len(rows)
	buttonLayout := singleStyledButtonLayout(t.aboutBody, buttonStyle, contentWidth, okButtonLabel).
		positioned(0, buttonRow)
	rows = append(rows, buttonLayout.content)

	title := renderWindowTitle(t.windowTitle, width, "Current User", spinnerFrame, pending)
	body := t.aboutBody.
		Width(width).
		Padding(verticalPadding, horizontalPadding).
		Render(lipgloss.JoinVertical(lipgloss.Left, rows...))

	return currentUserWindowLayout{
		content: lipgloss.JoinVertical(lipgloss.Left, title, body),
		buttonBounds: translateLayoutBounds(
			buttonLayout.bounds,
			horizontalPadding,
			lipgloss.Height(title)+verticalPadding,
		),
	}
}

func renderCurrentUserWindow(
	t theme,
	width int,
	login string,
	pending bool,
	blocked bool,
	spinnerFrame string,
) string {
	return newCurrentUserWindowLayout(t, width, login, pending, blocked, spinnerFrame).content
}

func newAlertWindowLayout(
	t theme,
	state alertState,
	title string,
	message string,
	highlight string,
) alertWindowLayout {
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

	buttonRow := len(rows) + 1
	buttons := singleStyledButtonLayout(bodyStyle, buttonStyle, contentWidth, okButtonLabel).
		positioned(2, buttonRow)
	rows = append(rows,
		bodyStyle.Width(width).Render(""),
		bodyStyle.Width(2).Render("")+buttons.content+bodyStyle.Width(2).Render(""),
		bodyStyle.Width(width).Render(""),
	)

	return alertWindowLayout{
		content:      lipgloss.JoinVertical(lipgloss.Left, rows...),
		buttonBounds: buttons.bounds,
	}
}

func renderAlertWindow(
	t theme,
	state alertState,
	title string,
	message string,
	highlight string,
) string {
	return newAlertWindowLayout(t, state, title, message, highlight).content
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
		lines = append(lines, wrapAlertParagraph(paragraph, width)...)
	}

	return lines
}

func wrapAlertParagraph(paragraph string, width int) []string {
	words := strings.Fields(paragraph)
	if len(words) == 0 {
		return []string{""}
	}

	lines := make([]string, 0, len(words))
	line := ""

	for _, word := range words {
		lines, line = appendAlertWord(lines, line, word, width)
	}
	if line != "" {
		lines = append(lines, line)
	}

	return lines
}

func appendAlertWord(lines []string, line, word string, width int) ([]string, string) {
	candidate := word
	if line != "" {
		candidate = line + " " + word
	}
	if lipgloss.Width(candidate) <= width {
		return lines, candidate
	}
	if line != "" {
		lines = append(lines, line)
	}
	if lipgloss.Width(word) <= width {
		return lines, word
	}

	parts := wrapRecordViewText(word, width)
	if len(parts) == 0 {
		return lines, ""
	}
	lines = append(lines, parts[:len(parts)-1]...)

	return lines, parts[len(parts)-1]
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
