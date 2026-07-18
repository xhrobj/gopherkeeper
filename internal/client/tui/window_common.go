package tui

import (
	"strings"

	"charm.land/lipgloss/v2"
)

const (
	currentUserButtonRow = 4
	alertButtonRow       = 5
)

func renderShadow(t theme, width, height int) string {
	return t.shadow.Width(max(1, width)).Height(max(1, height)).Render("")
}

func renderCurrentUserWindow(t theme, width int, login string) string {
	contentWidth := max(1, width-4)
	message := t.aboutText.Render("Logged in as ") + t.aboutTitle.Render(login)
	rows := []string{
		t.aboutBody.Width(contentWidth).AlignHorizontal(lipgloss.Center).Render(message),
		t.aboutBody.Width(contentWidth).Render(""),
		renderSingleStyledButton(t.aboutBody, t.aboutButtonActive, contentWidth, "< OK >"),
	}

	title := t.windowTitle.Width(width).Render("Current User")
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
		bodyStyle.Width(2).Render("") +
			renderAlertMessage(bodyStyle, titleStyle, contentWidth, message, highlight) +
			bodyStyle.Width(2).Render(""),
		bodyStyle.Width(width).Render(""),
		bodyStyle.Width(2).Render("") +
			renderSingleStyledButton(bodyStyle, buttonStyle, contentWidth, "< OK >") +
			bodyStyle.Width(2).Render(""),
		bodyStyle.Width(width).Render(""),
	}

	return lipgloss.JoinVertical(lipgloss.Left, rows...)
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

func renderSingleStyledButton(
	background lipgloss.Style,
	style lipgloss.Style,
	width int,
	label string,
) string {
	button := style.Render(label)
	left := max(0, (width-lipgloss.Width(button))/2)
	right := max(0, width-lipgloss.Width(button)-left)
	return background.Width(left).Render("") + button + background.Width(right).Render("")
}

func fitSingleLine(value string, width int) string {
	value = strings.Join(strings.Fields(value), " ")

	if width <= 0 {
		return ""
	}

	if lipgloss.Width(value) <= width {
		return value
	}

	if width <= 3 {
		runes := []rune(value)
		for len(runes) > 0 && lipgloss.Width(string(runes)) > width {
			runes = runes[:len(runes)-1]
		}
		return string(runes)
	}

	runes := []rune(value)
	limit := width - 3

	for len(runes) > 0 && lipgloss.Width(string(runes)) > limit {
		runes = runes[:len(runes)-1]
	}

	return string(runes) + "..."
}
