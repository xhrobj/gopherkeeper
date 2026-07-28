package tui

import (
	"strings"

	"charm.land/lipgloss/v2"
)

type layoutBounds struct {
	x      int
	y      int
	width  int
	height int
}

type styledButton struct {
	label string
	style lipgloss.Style
}

type buttonRowLayout struct {
	content string
	bounds  []layoutBounds
}

type windowPlacement struct {
	content string
	x       int
	y       int
	width   int
	height  int
}

type labeledFieldColumnLayout struct {
	contentWidth int
	inputWidth   int
	bounds       []layoutBounds
}

func (bounds layoutBounds) contains(x, y int) bool {
	return x >= bounds.x && x < bounds.x+bounds.width &&
		y >= bounds.y && y < bounds.y+bounds.height
}

func (bounds layoutBounds) translated(x, y int) layoutBounds {
	bounds.x += x
	bounds.y += y

	return bounds
}

func translateLayoutBounds(bounds []layoutBounds, x, y int) []layoutBounds {
	translated := make([]layoutBounds, len(bounds))

	for index, bound := range bounds {
		translated[index] = bound.translated(x, y)
	}

	return translated
}

func centeredButtonRowLayout(
	background lipgloss.Style,
	width int,
	gap int,
	buttons []styledButton,
) buttonRowLayout {
	if len(buttons) == 0 {
		return buttonRowLayout{content: background.Width(max(1, width)).Render("")}
	}

	rendered := make([]string, len(buttons))
	buttonWidths := make([]int, len(buttons))
	totalWidth := gap * (len(buttons) - 1)
	for index, button := range buttons {
		rendered[index] = button.style.Render(button.label)
		buttonWidths[index] = lipgloss.Width(rendered[index])
		totalWidth += buttonWidths[index]
	}

	left := max(0, (width-totalWidth)/2)
	right := max(0, width-totalWidth-left)
	bounds := make([]layoutBounds, len(buttons))
	x := left
	for index, buttonWidth := range buttonWidths {
		bounds[index] = layoutBounds{x: x, width: buttonWidth, height: 1}
		x += buttonWidth + gap
	}

	return buttonRowLayout{
		content: background.Width(left).Render("") +
			strings.Join(rendered, background.Width(gap).Render("")) +
			background.Width(right).Render(""),
		bounds: bounds,
	}
}

func (layout buttonRowLayout) positioned(x, y int) buttonRowLayout {
	layout.bounds = translateLayoutBounds(layout.bounds, x, y)
	return layout
}

func centeredWindowPlacement(content string, screenWidth, screenHeight int) (windowPlacement, bool) {
	if content == "" {
		return windowPlacement{}, false
	}

	width := lipgloss.Width(content)
	height := lipgloss.Height(content)

	return windowPlacement{
		content: content,
		x:       max(0, (screenWidth-width)/2),
		y:       max(2, (screenHeight-height)/2),
		width:   width,
		height:  height,
	}, true
}

func (placement windowPlacement) screenBounds(bounds []layoutBounds) []layoutBounds {
	return translateLayoutBounds(bounds, placement.x, placement.y)
}

func (m model) dialogPlacement() (windowPlacement, bool) {
	return centeredWindowPlacement(m.renderDialog(), m.width, m.height)
}

func (m model) alertWindowLayout() alertWindowLayout {
	if m.alert == alertNone {
		return alertWindowLayout{}
	}

	return newAlertWindowLayout(m.theme, m.alert, m.alertTitle, m.alertMessage, m.alertHighlight)
}

func (m model) alertPlacement() (windowPlacement, bool) {
	return centeredWindowPlacement(m.alertWindowLayout().content, m.width, m.height)
}

func (m model) workspacePlacement() (windowPlacement, bool) {
	if !m.recordFeature.workspace.open {
		return windowPlacement{}, false
	}

	content := renderRecordWorkspace(
		m.theme,
		recordWorkspaceWidth(m.width),
		recordWorkspaceHeight(m.height),
		m.recordFeature.workspace,
		m.dialog == dialogNone && m.alert == alertNone,
		m.operations.pending(operationListRecords),
		m.spinnerFrameValue(),
	)
	return centeredWindowPlacement(content, m.width, m.height)
}

func newLabeledFieldColumnLayout(
	windowWidth, labelWidth, minimumInputWidth, fieldCount, firstRow, rowStep int,
) labeledFieldColumnLayout {
	contentWidth := max(1, windowWidth-4)
	inputWidth := max(minimumInputWidth, contentWidth-labelWidth-2)
	inputX := 2 + labelWidth + 2
	bounds := make([]layoutBounds, fieldCount)

	for index := range fieldCount {
		bounds[index] = layoutBounds{
			x:      inputX,
			y:      firstRow + index*rowStep,
			width:  inputWidth,
			height: 1,
		}
	}

	return labeledFieldColumnLayout{
		contentWidth: contentWidth,
		inputWidth:   inputWidth,
		bounds:       bounds,
	}
}
