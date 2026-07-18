package tui

import "charm.land/lipgloss/v2"

const (
	controlsActionWidth = 17
	controlsKeyWidth    = 18
	controlsColumnGap   = 2
	controlsButtonRow   = 12
)

func renderControlsWindow(t theme, width int) string {
	contentWidth := max(1, width-4)
	title := t.windowTitle.Width(width).Render("Controls")
	body := t.windowBody.
		Width(width).
		Padding(1, 2).
		Render(renderControls(t, contentWidth))

	return lipgloss.JoinVertical(lipgloss.Left, title, body)
}

func renderControls(t theme, width int) string {
	rows := []string{
		renderControlRow(t, width, "Menu", "F10"),
		renderControlRow(t, width, "Choose menu item", "Highlighted letter"),
		renderControlRow(t, width, "Navigate", "Arrow keys"),
		renderControlRow(t, width, "Choose", "Enter"),
		renderControlRow(t, width, "Mouse", "Click"),
		renderControlRow(t, width, "Back / Close", "Esc"),
		renderControlRow(t, width, "Next control", "Tab"),
		renderControlRow(t, width, "Previous control", "Shift+Tab"),
		renderControlRow(t, width, "Quit", "Ctrl+Q"),
		t.windowBody.Width(width).Render(""),
		renderControlsButton(t, width, "< OK >"),
	}

	return lipgloss.JoinVertical(lipgloss.Left, rows...)
}

func renderControlRow(t theme, width int, action, key string) string {
	tableWidth := controlsActionWidth + controlsColumnGap + controlsKeyWidth
	leftPad := max(0, (width-tableWidth)/2)

	if width > tableWidth {
		leftPad++
	}

	rightPad := max(0, width-tableWidth-leftPad)

	return t.windowBody.Width(leftPad).Render("") +
		t.controlsText.Width(controlsActionWidth).AlignHorizontal(lipgloss.Right).Render(action) +
		t.windowBody.Width(controlsColumnGap).Render("") +
		t.controlsKey.Width(controlsKeyWidth).AlignHorizontal(lipgloss.Left).Render(key) +
		t.windowBody.Width(rightPad).Render("")
}

func renderControlsButton(t theme, width int, label string) string {
	button := t.buttonActive.Render(label)
	buttonWidth := lipgloss.Width(button)
	leftWidth := max(0, (width-buttonWidth)/2)
	rightWidth := max(0, width-buttonWidth-leftWidth)

	return t.windowBody.Width(leftWidth).Render("") +
		button +
		t.windowBody.Width(rightWidth).Render("")
}
