package tui

import "charm.land/lipgloss/v2"

type windowPlacement struct {
	content string
	x       int
	y       int
	width   int
	height  int
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

func (m model) dialogPlacement() (windowPlacement, bool) {
	return centeredWindowPlacement(m.renderDialog(), m.width, m.height)
}

func (m model) alertPlacement() (windowPlacement, bool) {
	return centeredWindowPlacement(m.renderAlert(), m.width, m.height)
}
