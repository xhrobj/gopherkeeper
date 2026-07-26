package tui

import (
	"math"
	"strings"

	"charm.land/lipgloss/v2"
)

const (
	readOnlyTextAreaMinRows   = 4
	readOnlyTextAreaMaxRows   = 10
	readOnlyTextAreaWheelStep = 3
)

type readOnlyTextArea struct {
	enabled bool
	value   string
	lines   []string
	width   int
	height  int
	offset  int
}

func (area *readOnlyTextArea) clear() {
	*area = readOnlyTextArea{}
}

func (area *readOnlyTextArea) setValue(value string) {
	area.enabled = true
	area.value = value
	area.offset = 0

	if area.width > 0 && area.height > 0 {
		area.rewrap()
	} else {
		area.lines = nil
	}
}

func (area *readOnlyTextArea) resize(width, height int) {
	width = max(1, width)
	height = max(1, height)

	if area.width == width && area.height == height {
		return
	}

	oldMax := area.maxOffset()
	position := 0.0

	if oldMax > 0 {
		position = float64(area.offset) / float64(oldMax)
	}

	area.width = width
	area.height = height
	area.rewrap()
	area.offset = int(math.Round(position * float64(area.maxOffset())))
	area.clampOffset()
}

func (area *readOnlyTextArea) rewrap() {
	if area.value == "" {
		area.lines = nil
		area.offset = 0
		return
	}

	area.lines = wrapRecordViewText(area.value, max(1, area.width))
	area.clampOffset()
}

func (area readOnlyTextArea) active() bool {
	return area.enabled
}

func (area readOnlyTextArea) maxOffset() int {
	return max(0, len(area.lines)-max(1, area.height))
}

func (area *readOnlyTextArea) clampOffset() {
	area.offset = clamp(area.offset, 0, area.maxOffset())
}

func (area *readOnlyTextArea) scroll(lines int) bool {
	before := area.offset
	area.offset = clamp(area.offset+lines, 0, area.maxOffset())

	return area.offset != before
}

func (area *readOnlyTextArea) page(pages int) bool {
	return area.scroll(pages * max(1, area.height-1))
}

func (area *readOnlyTextArea) home() {
	area.offset = 0
}

func (area *readOnlyTextArea) end() {
	area.offset = area.maxOffset()
}

func (area readOnlyTextArea) visibleLines() []string {
	height := max(1, area.height)
	visible := make([]string, 0, height)
	end := min(len(area.lines), area.offset+height)

	if area.offset < len(area.lines) {
		visible = append(visible, area.lines[area.offset:end]...)
	}

	for len(visible) < height {
		visible = append(visible, "")
	}

	return visible
}

func readOnlyTextAreaRows(screenHeight int) int {
	windowHeight := recordViewWindowHeight(screenHeight)
	return clamp((windowHeight-9)/2, readOnlyTextAreaMinRows, readOnlyTextAreaMaxRows)
}

func renderReadOnlyTextArea(t theme, area readOnlyTextArea, width int) []string {
	width = max(4, width)
	textWidth := max(1, width-1)
	area.resize(textWidth, max(1, area.height))
	visible := area.visibleLines()

	rows := make([]string, 0, len(visible))
	for index, line := range visible {
		text := fitSingleLinePreserve(line, textWidth)
		padding := max(0, textWidth-lipgloss.Width(text))
		body := t.recordMemoBody.Render(text + strings.Repeat(" ", padding))
		scroll := t.recordMemoTrack.Render(readOnlyTextAreaScrollGlyph(area, index, len(visible)))
		rows = append(rows, body+scroll)
	}

	return rows
}

func readOnlyTextAreaScrollGlyph(area readOnlyTextArea, row, height int) string {
	if row == 0 {
		return "▲"
	}

	if row == height-1 {
		return "▼"
	}

	trackHeight := max(1, height-2)
	if area.maxOffset() == 0 {
		return " "
	}

	thumb := 0
	if trackHeight > 1 {
		thumb = int(math.Round(float64(area.offset) / float64(area.maxOffset()) * float64(trackHeight-1)))
	}

	if row-1 == thumb {
		return "█"
	}

	return "░"
}
