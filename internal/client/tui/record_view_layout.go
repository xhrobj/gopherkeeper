package tui

type recordViewWindowLayout struct {
	contentWidth   int
	bodyHeight     int
	viewportHeight int
	buttonRow      int
}

type recordViewContentLayout struct {
	window          recordViewWindowLayout
	visible         []recordViewLine
	textAreaBounds  layoutBounds
	textAreaVisible bool
	textAreaUp      layoutBounds
	textAreaDown    layoutBounds
}

func recordViewWindowWidth(screenWidth int) int { return clamp(screenWidth-14, 56, 92) }

func recordViewWindowHeight(screenHeight int) int { return clamp(screenHeight-6, 16, 36) }

func recordViewContentWidth(screenWidth int) int { return max(1, recordViewWindowWidth(screenWidth)-4) }

func recordViewWindowHeightForState(t theme, screenWidth, screenHeight int, state recordViewState) int {
	height := recordViewWindowHeight(screenHeight)

	if state.status != recordViewReady {
		return min(height, 16)
	}

	lineCount := len(recordViewLines(t, state, recordViewContentWidth(screenWidth)))

	return clamp(lineCount+5, 16, height)
}

func recordViewPageSizeForState(t theme, screenWidth, screenHeight int, state recordViewState) int {
	width := recordViewWindowWidth(screenWidth)
	height := recordViewWindowHeightForState(t, screenWidth, screenHeight, state)

	return newRecordViewWindowLayout(width, height).viewportHeight
}

func newRecordViewWindowLayout(width, height int) recordViewWindowLayout {
	bodyHeight := max(1, height-1)
	viewportHeight := max(1, bodyHeight-3)

	return recordViewWindowLayout{
		contentWidth:   max(1, width-4),
		bodyHeight:     bodyHeight,
		viewportHeight: viewportHeight,
		buttonRow:      2 + viewportHeight,
	}
}

func newRecordViewContentLayout(t theme, width, height int, state recordViewState) recordViewContentLayout {
	window := newRecordViewWindowLayout(width, height)

	var lines []recordViewLine

	if state.status == recordViewReady {
		lines = recordViewLines(t, state, window.contentWidth)
	}

	maxOffset := max(0, len(lines)-window.viewportHeight)
	offset := clamp(state.offset, 0, maxOffset)
	end := min(len(lines), offset+window.viewportHeight)
	visible := append([]recordViewLine(nil), lines[offset:end]...)

	for len(visible) < window.viewportHeight {
		visible = append(visible, recordViewLine{})
	}

	layout := recordViewContentLayout{
		window:  window,
		visible: visible,
	}

	start, textEnd, ok := recordViewTextAreaRange(lines)
	if !ok || textEnd-start <= 1 {
		return layout
	}

	visibleStart := max(start+1, offset)
	visibleEnd := min(textEnd, offset+window.viewportHeight)

	if visibleStart >= visibleEnd {
		return layout
	}

	layout.textAreaVisible = true
	layout.textAreaBounds = layoutBounds{
		x:      2,
		y:      2 + visibleStart - offset,
		width:  window.contentWidth,
		height: visibleEnd - visibleStart,
	}
	scrollX := 2 + window.contentWidth - 1
	firstContentY := 2 + start + 1 - offset
	layout.textAreaUp = layoutBounds{x: scrollX, y: firstContentY, width: 1, height: 1}
	layout.textAreaDown = layoutBounds{
		x:      scrollX,
		y:      firstContentY + state.textArea.height - 1,
		width:  1,
		height: 1,
	}

	return layout
}

func recordViewTextAreaRange(lines []recordViewLine) (int, int, bool) {
	start := -1
	end := -1

	for index, line := range lines {
		if !line.textArea {
			continue
		}
		if start < 0 {
			start = index
		}
		end = index + 1
	}

	return start, end, start >= 0
}
