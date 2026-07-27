package tui

import (
	"strings"

	"charm.land/lipgloss/v2"
	recordmodel "github.com/xhrobj/gopherkeeper/internal/model"
)

const recordViewButtonGap = 3

type recordViewLine struct {
	message  string
	label    string
	value    string
	rendered string
	textArea bool
}

type recordViewWindowOptions struct {
	width        int
	height       int
	state        recordViewState
	activeButton int
	pending      bool
	blocked      bool
	spinnerFrame string
}

func newRecordViewWindowOptions(
	width, height int,
	state recordViewState,
	activeButton int,
	pending, blocked bool,
	spinnerFrame string,
) recordViewWindowOptions {
	return recordViewWindowOptions{
		width:        width,
		height:       height,
		state:        state,
		activeButton: activeButton,
		pending:      pending,
		blocked:      blocked,
		spinnerFrame: spinnerFrame,
	}
}

func renderRecordViewWindow(t theme, options recordViewWindowOptions) string {
	layout := newRecordViewContentLayout(t, options.width, options.height, options.state)
	contentWidth := layout.window.contentWidth
	bodyHeight := layout.window.bodyHeight

	rows := make([]string, 0, layout.window.viewportHeight+1)
	for _, line := range layout.visible {
		rows = append(rows, renderRecordViewLine(t, contentWidth, line))
	}
	rows = append(rows, renderRecordViewButtons(t, contentWidth, options.state, options.activeButton, options.blocked))

	title := "Record"
	if options.state.record.Metadata.Type != "" {
		title = "Record " + recordTypeTitle(options.state.record.Metadata.Type)
	}

	if options.state.source == recordSourceCache {
		title += " from Cache"
		if options.state.login != "" {
			title += ": " + options.state.login
		}
	} else {
		title += " Online"
	}

	titleLine := renderWindowTitle(
		t.windowTitle,
		options.width,
		fitSingleLine(title, options.width),
		options.spinnerFrame,
		options.pending,
	)
	body := t.windowBody.Width(options.width).Height(bodyHeight).Padding(1, 2).Render(strings.Join(rows, "\n"))

	return lipgloss.JoinVertical(lipgloss.Left, titleLine, body)
}

func renderRecordViewLine(t theme, width int, line recordViewLine) string {
	return renderRecordViewLineWithStyles(t.windowBody, t.recordInfo, width, line)
}

func renderRecordViewLineWithStyles(
	bodyStyle lipgloss.Style,
	labelStyle lipgloss.Style,
	width int,
	line recordViewLine,
) string {
	if line.rendered != "" {
		used := lipgloss.Width(line.rendered)
		if used >= width {
			return line.rendered
		}
		return line.rendered + bodyStyle.Width(width-used).Render("")
	}

	if line.message != "" {
		return labelStyle.Width(width).Render(fitSingleLinePreserve(line.message, width))
	}

	if line.label == "" && line.value == "" {
		return bodyStyle.Width(width).Render("")
	}

	if line.label == "" {
		return bodyStyle.Width(width).Render(fitSingleLinePreserve(line.value, width))
	}

	label := labelStyle.Render(line.label)
	value := bodyStyle.Render(line.value)
	used := lipgloss.Width(label) + lipgloss.Width(value)
	padding := max(0, width-used)

	return label + value + bodyStyle.Width(padding).Render("")
}

func recordViewButtonsLayout(
	t theme,
	width int,
	state recordViewState,
	activeButton int,
	blocked bool,
) buttonRowLayout {
	labels := recordViewButtonLabels(state)
	if len(labels) == 0 {
		return buttonRowLayout{content: t.windowBody.Width(width).Render("")}
	}

	if len(labels) == 1 {
		style := t.buttonActive
		if blocked {
			style = t.buttonDisabledActive
		}
		return centeredButtonRowLayout(t.windowBody, width, 0, []styledButton{{label: labels[0], style: style}})
	}

	styles := make([]lipgloss.Style, len(labels))
	for index := range styles {
		styles[index] = t.button
	}

	if blocked {
		for index := range styles {
			styles[index] = t.buttonDisabled
		}
		if activeButton >= 0 && activeButton < len(styles) {
			styles[activeButton] = t.buttonDisabledActive
		}
	} else if activeButton >= 0 && activeButton < len(styles) {
		styles[activeButton] = t.buttonActive
	}

	buttons := make([]styledButton, len(labels))
	for index, label := range labels {
		buttons[index] = styledButton{label: label, style: styles[index]}
	}

	return centeredButtonRowLayout(t.windowBody, width, recordViewButtonGap, buttons)
}

func renderRecordViewButtons(
	t theme,
	width int,
	state recordViewState,
	activeButton int,
	blocked bool,
) string {
	return recordViewButtonsLayout(t, width, state, activeButton, blocked).content
}

func recordViewButtonLabels(state recordViewState) []string {
	if state.status == recordViewIdle {
		return nil
	}

	if state.record.Metadata.Type == recordmodel.RecordTypeBinary {
		return []string{"< Save As... >", closeButtonLabel}
	}

	if state.record.Metadata.Type == recordmodel.RecordTypeCredentials ||
		state.record.Metadata.Type == recordmodel.RecordTypeCard ||
		recordViewHasSensitiveFields(state.record) {
		label := "< Reveal >"
		if state.revealed {
			label = "< Hide >"
		}
		return []string{label, closeButtonLabel}
	}

	return []string{closeButtonLabel}
}
