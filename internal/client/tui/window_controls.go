package tui

import "charm.land/lipgloss/v2"

const (
	controlsActionWidth    = 17
	controlsSeparatorWidth = 1
	controlsKeyWidth       = 22
	controlsColumnGap      = 1
	controlsButtonRow      = 21
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
	spacerRow := renderControlSpacerRow(t, width)

	rows := []string{
		renderControlRow(t, width, "Menu", "F10"),
		spacerRow,
		renderControlRow(t, width, "Choose menu item", "Highlighted letter"),
		renderControlRow(t, width, "Navigate menu", "Arrow keys"),
		spacerRow,
		renderControlRow(t, width, "Move in list", "Up Arrow / Down Arrow"),
		renderControlRow(t, width, "Page list", "PgUp / PgDn"),
		renderControlRow(t, width, "First / last", "Home / End"),
		spacerRow,
		renderControlRow(t, width, "Choose", "Enter"),
		renderControlRow(t, width, "Delete record", "Del"),
		renderControlRow(t, width, "Mouse", "Click"),
		spacerRow,
		renderControlRow(t, width, "Back / Close", "Esc"),
		renderControlRow(t, width, "Next control", "Tab / Right Arrow"),
		renderControlRow(t, width, "Previous control", "Shift+Tab / Left Arrow"),
		spacerRow,
		renderControlRow(t, width, "Quit", "Ctrl+Q"),
		spacerRow,
		renderControlsButton(t, width, "< OK >"),
	}

	return lipgloss.JoinVertical(lipgloss.Left, rows...)
}

func renderControlSpacerRow(t theme, width int) string {
	return t.windowBody.Width(width).Render("")
}

func renderControlRow(t theme, width int, action, key string) string {
	tableWidth := controlsActionWidth + controlsSeparatorWidth + controlsColumnGap + controlsKeyWidth
	leftPad := max(0, (width-tableWidth)/2)

	if width > tableWidth {
		leftPad++
	}

	rightPad := max(0, width-tableWidth-leftPad)

	return t.windowBody.Width(leftPad).Render("") +
		t.controlsText.Width(controlsActionWidth).AlignHorizontal(lipgloss.Right).Render(action) +
		t.controlsSeparator.Width(controlsSeparatorWidth).Render(":") +
		t.windowBody.Width(controlsColumnGap).Render("") +
		t.controlsKey.Width(controlsKeyWidth).AlignHorizontal(lipgloss.Left).Render(key) +
		t.windowBody.Width(rightPad).Render("")
}

func controlsButtonLayout(t theme, width int, label string) buttonRowLayout {
	return centeredButtonRowLayout(t.windowBody, width, 0, []styledButton{
		{label: label, style: t.buttonActive},
	})
}

func renderControlsButton(t theme, width int, label string) string {
	return controlsButtonLayout(t, width, label).content
}
