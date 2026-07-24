package tui

import (
	"fmt"
	"strings"
	"unicode"

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

func recordViewWindowWidth(screenWidth int) int   { return clamp(screenWidth-14, 56, 92) }
func recordViewWindowHeight(screenHeight int) int { return clamp(screenHeight-6, 16, 36) }
func recordViewContentWidth(screenWidth int) int  { return max(1, recordViewWindowWidth(screenWidth)-4) }

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

type recordViewWindowLayout struct {
	contentWidth   int
	bodyHeight     int
	viewportHeight int
	buttonRow      int
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

type recordViewContentLayout struct {
	window          recordViewWindowLayout
	visible         []recordViewLine
	textAreaBounds  layoutBounds
	textAreaVisible bool
	textAreaUp      layoutBounds
	textAreaDown    layoutBounds
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

func renderRecordViewWindow(
	t theme,
	width, height int,
	state recordViewState,
	activeButton int,
	pending bool,
	blocked bool,
	spinnerFrame string,
) string {
	layout := newRecordViewContentLayout(t, width, height, state)
	contentWidth := layout.window.contentWidth
	bodyHeight := layout.window.bodyHeight

	rows := make([]string, 0, layout.window.viewportHeight+1)
	for _, line := range layout.visible {
		rows = append(rows, renderRecordViewLine(t, contentWidth, line))
	}
	rows = append(rows, renderRecordViewButtons(t, contentWidth, state, activeButton, blocked))

	title := "Record"
	if state.record.Metadata.Type != "" {
		title = "Record " + recordTypeTitle(state.record.Metadata.Type)
	}

	titleLine := renderWindowTitle(t.windowTitle, width, fitSingleLine(title, width), spinnerFrame, pending)
	body := t.windowBody.Width(width).Height(bodyHeight).Padding(1, 2).Render(strings.Join(rows, "\n"))

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

func recordViewLines(t theme, state recordViewState, width int) []recordViewLine {
	record := state.record
	reveal := state.revealed
	metadata := record.Metadata

	lines := make([]recordViewLine, 0, 16)
	lines = append(lines, recordViewMetadataLines(metadata, width)...)
	lines = append(lines, recordViewLine{})
	lines = appendRecordViewField(lines, "Title", metadata.Title, width)
	lines = append(lines, recordViewLine{})

	switch payload := record.Payload.(type) {
	case *recordmodel.TextPayload:
		if payload != nil {
			if state.textArea.active() {
				lines = append(lines, recordViewLine{label: "Text:", textArea: true})
				for _, row := range renderReadOnlyTextArea(t, state.textArea, width) {
					lines = append(lines, recordViewLine{rendered: row, textArea: true})
				}
			} else {
				lines = appendRecordViewField(lines, "Text", payload.Text, width)
			}
			lines = appendRecordViewNotes(lines, payload.Metadata, width)
		}
	case *recordmodel.CredentialsPayload:
		if payload != nil {
			lines = appendRecordViewField(lines, "Login", payload.Login, width)
			lines = appendRecordViewField(lines, "Password", visibleSecret(payload.Password, reveal), width)
			if payload.URL != "" {
				lines = appendRecordViewField(lines, "URL", payload.URL, width)
			}
			lines = appendRecordViewNotes(lines, payload.Metadata, width)
		}
	case *recordmodel.CardPayload:
		if payload != nil {
			lines = append(lines, buildCardPreviewLines(t, payload, reveal, width)...)
			lines = appendRecordViewNotes(lines, payload.Metadata, width)
		}
	case *recordmodel.BinaryPayload:
		if payload != nil {
			lines = appendRecordViewField(lines, "Filename", payload.Filename, width)
			lines = appendRecordViewField(lines, "Size", fmt.Sprintf("%d bytes", len(payload.Data)), width)
			lines = appendRecordViewNotes(lines, payload.Metadata, width)
		}
	default:
		lines = append(lines, recordViewLine{message: "Unsupported record payload"})
	}

	return lines
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

func buildCardPreviewLines(t theme, payload *recordmodel.CardPayload, reveal bool, width int) []recordViewLine {
	cardholder := strings.ToUpper(strings.TrimSpace(payload.Cardholder))
	expiry := ""
	if payload.ExpiryMonth != nil && payload.ExpiryYear != nil {
		expiry = fmt.Sprintf("%02d/%02d", *payload.ExpiryMonth, *payload.ExpiryYear%100)
	}
	cvv := ""
	if payload.CVV != "" {
		cvv = visibleCVV(payload.CVV, reveal)
	}
	number := formattedVisibleCardNumber(payload.Number, reveal)

	cardWidth := clamp(width*2/5, 32, 34)
	stripeFillWidth := max(1, cardWidth-2)
	cardTextWidth := max(1, cardWidth-4)

	cardFace := t.recordCard
	cardLabel := t.recordCardMuted
	cardStripe := t.recordCardStripe

	padToWindow := func(card string) string {
		used := lipgloss.Width(card)
		if used >= width {
			return card
		}
		return card + t.windowBody.Width(width-used).Render("")
	}

	renderCardLine := func(content string) string {
		content = fitSingleLinePreserve(content, cardTextWidth)
		padding := max(0, cardTextWidth-lipgloss.Width(content))
		card := cardFace.Render("  " + content + strings.Repeat(" ", padding) + "  ")
		return padToWindow(card)
	}

	renderBlank := func() string {
		return renderCardLine("")
	}

	renderStripe := func() string {
		content := " " + strings.Repeat("▒", stripeFillWidth) + " "
		card := cardStripe.Render(content)
		return padToWindow(card)
	}

	renderFooter := func() string {
		left := cardLabel.Render("exp") + cardFace.Render(" ") + cardFace.Render(expiry)
		cvvText := fitSingleLinePreserve(cvv, recordmodel.CardCVVSize)
		right := cardFace.Render("[") + cardLabel.Render("cvv") + cardFace.Render(" "+cvvText) + cardFace.Render("]")
		used := lipgloss.Width(left) + lipgloss.Width(right)
		gap := max(1, cardTextWidth-used)
		card := cardFace.Render("  ") + left + cardFace.Render(strings.Repeat(" ", gap)) + right + cardFace.Render("  ")
		return padToWindow(card)
	}

	return []recordViewLine{
		{rendered: renderBlank()},
		{rendered: renderStripe()},
		{rendered: renderBlank()},
		{rendered: renderCardLine(number)},
		{rendered: renderBlank()},
		{rendered: renderCardLine(cardholder)},
		{rendered: renderBlank()},
		{rendered: renderFooter()},
		{rendered: renderBlank()},
		{rendered: renderBlank()},
	}
}

func appendRecordViewNotes(lines []recordViewLine, notes string, width int) []recordViewLine {
	lines = append(lines, recordViewLine{})

	return appendRecordViewField(lines, "Notes", notes, width)
}

func recordViewMetadataLines(metadata recordmodel.RecordMetadata, width int) []recordViewLine {
	lines := make([]recordViewLine, 0, 5)
	lines = appendRecordViewField(lines, "ID", metadata.ID, width)
	lines = appendRecordViewField(lines, "Revision", fmt.Sprintf("%d", metadata.Revision), width)
	lines = append(lines, recordViewLine{})
	lines = appendRecordViewField(lines, "Created at", formatRecordTime(metadata.CreatedAt), width)
	lines = appendRecordViewField(lines, "Updated at", formatRecordTime(metadata.UpdatedAt), width)

	return lines
}

func appendRecordViewField(lines []recordViewLine, label, value string, width int) []recordViewLine {
	prefix := label + ": "
	prefixWidth := lipgloss.Width(prefix)
	valueWidth := max(1, width-prefixWidth)
	wrapped := wrapRecordViewText(value, valueWidth)

	if len(wrapped) == 0 {
		wrapped = []string{""}
	}

	lines = append(lines, recordViewLine{label: prefix, value: wrapped[0]})
	indent := strings.Repeat(" ", prefixWidth)

	for _, continuation := range wrapped[1:] {
		lines = append(lines, recordViewLine{label: indent, value: continuation})
	}

	return lines
}

func wrapRecordViewText(value string, width int) []string {
	width = max(1, width)
	value = strings.ReplaceAll(value, "\r\n", "\n")
	value = strings.ReplaceAll(value, "\r", "\n")
	value = strings.ReplaceAll(value, "\t", "    ")

	paragraphs := strings.Split(value, "\n")
	lines := make([]string, 0, len(paragraphs))

	for _, paragraph := range paragraphs {
		if paragraph == "" {
			lines = append(lines, "")
			continue
		}
		runes := []rune(paragraph)
		for len(runes) > 0 {
			used := 0
			end := 0
			for end < len(runes) {
				runeWidth := lipgloss.Width(string(runes[end]))
				if end > 0 && used+runeWidth > width {
					break
				}
				used += runeWidth
				end++
				if used >= width {
					break
				}
			}
			if end == 0 {
				end = 1
			}
			lines = append(lines, string(runes[:end]))
			runes = runes[end:]
		}
	}

	return lines
}

func recordTypeTitle(recordType recordmodel.RecordType) string {
	value := string(recordType)
	if value == "" {
		return ""
	}

	runes := []rune(value)
	runes[0] = []rune(strings.ToUpper(string(runes[0])))[0]

	return string(runes)
}

func maskSecret(value string) string {
	if value == "" {
		return ""
	}
	return strings.Repeat("•", max(4, len([]rune(value))))
}

func visibleSecret(value string, reveal bool) string {
	if reveal {
		return value
	}
	return maskSecret(value)
}

func visibleCVV(value string, reveal bool) string {
	if reveal {
		return value
	}

	if value == "" {
		return ""
	}

	return strings.Repeat("•", len([]rune(value)))
}

func formattedVisibleCardNumber(value string, reveal bool) string {
	clean := cardDigits(value)
	if clean == "" {
		return ""
	}

	if !reveal {
		clean = maskCardNumber(clean)
	}

	return groupCardDigits(clean)
}

func cardDigits(value string) string {
	var b strings.Builder

	for _, r := range value {
		if unicode.IsDigit(r) {
			b.WriteRune(r)
		}
	}

	return b.String()
}

func groupCardDigits(value string) string {
	if value == "" {
		return ""
	}

	runes := []rune(value)
	parts := make([]string, 0, (len(runes)+3)/4)

	for len(runes) > 0 {
		chunk := min(4, len(runes))
		parts = append(parts, string(runes[:chunk]))
		runes = runes[chunk:]
	}

	return strings.Join(parts, " ")
}

func maskCardNumber(value string) string {
	r := []rune(value)
	digits := 0

	for _, c := range r {
		if unicode.IsDigit(c) {
			digits++
		}
	}

	keep := 4
	seen := 0
	out := make([]rune, len(r))

	for i, c := range r {
		if unicode.IsDigit(c) {
			seen++
			if seen <= digits-keep {
				out[i] = '•'
			} else {
				out[i] = c
			}
		} else {
			out[i] = c
		}
	}

	return string(out)
}

func fitSingleLinePreserve(value string, width int) string {
	if width <= 0 {
		return ""
	}

	if lipgloss.Width(value) <= width {
		return value
	}

	return fitSingleLine(value, width)
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
		return []string{"< Save As... >", "< Close >"}
	}

	if state.record.Metadata.Type == recordmodel.RecordTypeCredentials ||
		state.record.Metadata.Type == recordmodel.RecordTypeCard ||
		recordViewHasSensitiveFields(state.record) {
		label := "< Reveal >"
		if state.revealed {
			label = "< Hide >"
		}
		return []string{label, "< Close >"}
	}

	return []string{"< Close >"}
}
