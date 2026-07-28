package tui

import "charm.land/lipgloss/v2"

const (
	controlsActionWidth             = 17
	controlsSeparatorWidth          = 1
	controlsKeyWidth                = 22
	controlsColumnGap               = 1
	controlsWindowHorizontalPadding = 2
	controlsWindowVerticalPadding   = 1
)

type controlsContentLayout struct {
	content      string
	buttonBounds []layoutBounds
}

type controlsWindowLayout struct {
	content      string
	buttonBounds []layoutBounds
}

func newControlsWindowLayout(t theme, width int) controlsWindowLayout {
	contentWidth := max(1, width-2*controlsWindowHorizontalPadding)
	contentLayout := newControlsContentLayout(t, contentWidth)
	title := t.windowTitle.Width(width).Render("Controls")

	body := t.windowBody.
		Width(width).
		Padding(controlsWindowVerticalPadding, controlsWindowHorizontalPadding).
		Render(contentLayout.content)

	return controlsWindowLayout{
		content: lipgloss.JoinVertical(lipgloss.Left, title, body),
		buttonBounds: translateLayoutBounds(
			contentLayout.buttonBounds,
			controlsWindowHorizontalPadding,
			lipgloss.Height(title)+controlsWindowVerticalPadding,
		),
	}
}

func renderControlsWindow(t theme, width int) string {
	return newControlsWindowLayout(t, width).content
}

func renderControls(t theme, width int) string {
	return newControlsContentLayout(t, width).content
}

func newControlsContentLayout(t theme, width int) controlsContentLayout {
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
	}

	buttonRow := len(rows)
	buttonLayout := controlsButtonLayout(t, width, okButtonLabel)
	rows = append(rows, buttonLayout.content)

	return controlsContentLayout{
		content:      lipgloss.JoinVertical(lipgloss.Left, rows...),
		buttonBounds: translateLayoutBounds(buttonLayout.bounds, 0, buttonRow),
	}
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
