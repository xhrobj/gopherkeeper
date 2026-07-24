package tui

import (
	"context"
	"errors"
	"path/filepath"
	"strings"

	tea "charm.land/bubbletea/v2"
	"charm.land/lipgloss/v2"
	recordmodel "github.com/xhrobj/gopherkeeper/internal/model"
)

const binarySaveErrorMaxRunes = 512

type binarySaveResultMsg struct {
	requestID uint64
	path      string
	err       error
}

type binaryFileWriter func(string, []byte) error

func binarySaveCommand(
	ctx context.Context,
	writer binaryFileWriter,
	requestID uint64,
	path string,
	data []byte,
) tea.Cmd {
	return func() tea.Msg {
		select {
		case <-ctx.Done():
			return binarySaveResultMsg{requestID: requestID, path: path, err: ctx.Err()}
		default:
		}
		err := writer(path, data)
		return binarySaveResultMsg{requestID: requestID, path: path, err: err}
	}
}

func cleanBinarySaveError(err error) string {
	if err == nil {
		return "Unknown binary save error"
	}

	if errors.Is(err, context.Canceled) {
		return "Binary save canceled"
	}

	message := strings.TrimSpace(err.Error())
	if message == "" {
		return "Unable to save binary file"
	}

	message = capitalizeFirst(message)
	runes := []rune(message)

	if len(runes) > binarySaveErrorMaxRunes {
		message = string(runes[:binarySaveErrorMaxRunes]) + "..."
	}

	return message
}

type binarySaveFocus int

const (
	binarySavePath binarySaveFocus = iota
	binarySaveSubmit
	binarySaveCancel
	binarySaveFocusCount
)

const (
	binarySaveLabelWidth    = 6
	binarySaveFirstFieldRow = 2
	binarySaveBrowseGap     = 2
	binarySaveBrowseLabel   = "<...>"
	binarySaveButtonGap     = 3
	binarySaveButtonRow     = 5
)

type binarySaveForm struct {
	fileName string
	path     string
	focus    binarySaveFocus
}

func newBinarySaveForm(fileName string) binarySaveForm {
	fileName = filepath.Base(strings.TrimSpace(fileName))

	if fileName == "." {
		fileName = ""
	}

	return binarySaveForm{
		fileName: fileName,
		path:     fileName,
		focus:    binarySavePath,
	}
}

func (form binarySaveForm) canSubmit() bool {
	return strings.TrimSpace(form.path) != ""
}

func (form *binarySaveForm) setDirectory(directory string) {
	directory = strings.TrimSpace(directory)

	if directory == "" || directory == "." {
		form.path = form.fileName
		return
	}

	form.path = filepath.Join(directory, form.fileName)
}

func (form *binarySaveForm) move(step int) {
	count := int(binarySaveFocusCount)

	for range count {
		form.focus = binarySaveFocus((int(form.focus) + step + count) % count)
		if form.canSubmit() || form.focus != binarySaveSubmit {
			return
		}
	}
}

func (form *binarySaveForm) setFocus(focus binarySaveFocus) {
	if focus < 0 || focus >= binarySaveFocusCount {
		return
	}

	if focus == binarySaveSubmit && !form.canSubmit() {
		return
	}

	form.focus = focus
}

func (m *model) openBinarySave() {
	payload, ok := m.recordFeature.view.record.Payload.(*recordmodel.BinaryPayload)
	if !ok || payload == nil {
		return
	}

	m.operations.cancel(operationBinarySave)
	m.recordFeature.binarySaveForm = newBinarySaveForm(payload.Filename)
	m.dialog = dialogBinarySave
	m.activeButton = 0
}

func (m model) openBinarySavePathPicker() (tea.Model, tea.Cmd) {
	picker, command := newPathPicker(
		pathPickerBinarySaveDirectory,
		m.recordFeature.binarySaveForm.path,
		m.height,
	)

	m.pathPicker = picker
	m.dialog = dialogPathPicker

	return m, command
}

func (m model) handleBinarySaveResult(msg binarySaveResultMsg) (tea.Model, tea.Cmd) {
	if !m.operations.accepts(operationBinarySave, msg.requestID) {
		return m, nil
	}

	m.operations.finish(operationBinarySave)

	if msg.err != nil {
		m.showAlert(alertError, "Unable to save binary", cleanBinarySaveError(msg.err), dialogBinarySave)
		return m, nil
	}

	m.recordFeature.binarySaveForm = newBinarySaveForm("")
	m.returnToRecordView()
	m.showAlert(alertNotice, "Binary saved", "Saved to "+msg.path, dialogRecordView)

	return m, nil
}

func (m model) updateBinarySave(key string) (tea.Model, tea.Cmd) {
	if m.operations.pending(operationBinarySave) {
		return m, nil
	}

	switch key {
	case "esc":
		m.closeBinarySave()
	case "tab", "down", "right":
		m.recordFeature.binarySaveForm.move(1)
	case "shift+tab", "up", "left":
		m.recordFeature.binarySaveForm.move(-1)
	case "enter":
		return m.activateBinarySave()
	}

	return m, nil
}

func (m model) activateBinarySave() (tea.Model, tea.Cmd) {
	switch m.recordFeature.binarySaveForm.focus {
	case binarySavePath:
		return m.openBinarySavePathPicker()
	case binarySaveSubmit:
		if m.operations.pending(operationBinarySave) || !m.recordFeature.binarySaveForm.canSubmit() {
			return m, nil
		}
		payload, ok := m.recordFeature.view.record.Payload.(*recordmodel.BinaryPayload)
		if !ok || payload == nil {
			m.closeBinarySave()
			return m, nil
		}
		path := filepath.Clean(strings.TrimSpace(m.recordFeature.binarySaveForm.path))
		if path == "." || path == "" {
			m.recordFeature.binarySaveForm.focus = binarySavePath
			return m, nil
		}
		requestCtx, requestID := m.operations.begin(m.ctx, operationBinarySave)
		return m, m.operationCommand(operationBinarySave, binarySaveCommand(requestCtx, m.writeBinaryFile, requestID, path, payload.Data))
	case binarySaveCancel:
		m.closeBinarySave()
	}

	return m, nil
}

func (m *model) closeBinarySave() {
	m.operations.cancel(operationBinarySave)
	m.recordFeature.binarySaveForm = newBinarySaveForm("")
	m.returnToRecordView()
}

func binarySaveWindowWidth(screenWidth int) int {
	return clamp(screenWidth-18, 54, 82)
}

type binarySaveWindowLayout struct {
	contentWidth int
	pathWidth    int
	browseBounds layoutBounds
}

func newBinarySaveWindowLayout(t theme, windowWidth int) binarySaveWindowLayout {
	contentWidth := max(1, windowWidth-4)
	browseWidth := lipgloss.Width(t.button.Render(binarySaveBrowseLabel))
	pathWidth := max(18, contentWidth-binarySaveLabelWidth-2-binarySaveBrowseGap-browseWidth)
	return binarySaveWindowLayout{
		contentWidth: contentWidth,
		pathWidth:    pathWidth,
		browseBounds: layoutBounds{
			x:      2 + binarySaveLabelWidth + 2 + pathWidth + binarySaveBrowseGap,
			y:      binarySaveFirstFieldRow,
			width:  browseWidth,
			height: 1,
		},
	}
}

func renderBinarySaveWindow(t theme, width int, form binarySaveForm, pending bool) string {
	layout := newBinarySaveWindowLayout(t, width)
	contentWidth := layout.contentWidth
	pathWidth := layout.pathWidth

	label := t.label.Width(binarySaveLabelWidth).Render("Path")
	gap := t.windowBody.Width(2).Render("")
	field := renderConfigReadOnlyInput(t, t.readOnly, form.path, pathWidth, true)
	browseStyle := t.button

	if pending {
		browseStyle = t.buttonDisabled
	} else if form.focus == binarySavePath {
		browseStyle = t.buttonActive
	}

	browse := browseStyle.Render(binarySaveBrowseLabel)
	pathRow := label + gap + field + t.windowBody.Width(binarySaveBrowseGap).Render("") + browse

	status := t.windowBody.Width(contentWidth).Render("")
	if pending {
		status = t.controlsKey.Width(contentWidth).AlignHorizontal(lipgloss.Center).Render("Saving file...")
	}

	rows := []string{
		pathRow,
		t.windowBody.Width(contentWidth).Render(""),
		status,
		renderBinarySaveButtons(t, contentWidth, form.focus, pending || !form.canSubmit(), pending),
	}

	title := t.windowTitle.Width(width).Render("Save Binary As")
	body := t.windowBody.Width(width).Padding(1, 2).Render(strings.Join(rows, "\n"))

	return lipgloss.JoinVertical(lipgloss.Left, title, body)
}

func binarySaveButtonsLayout(t theme, width int, focus binarySaveFocus, submitDisabled, blocked bool) buttonRowLayout {
	saveStyle := t.button
	cancelStyle := t.button

	if blocked {
		saveStyle = t.buttonDisabled
		cancelStyle = t.buttonDisabled
		if focus == binarySaveSubmit {
			saveStyle = t.buttonDisabledActive
		}
		if focus == binarySaveCancel {
			cancelStyle = t.buttonDisabledActive
		}
	} else {
		if submitDisabled {
			saveStyle = t.buttonDisabled
		} else if focus == binarySaveSubmit {
			saveStyle = t.buttonActive
		}
		if focus == binarySaveCancel {
			cancelStyle = t.buttonActive
		}
	}

	return centeredButtonRowLayout(t.windowBody, width, binarySaveButtonGap, []styledButton{
		{label: "< Save >", style: saveStyle},
		{label: "< Cancel >", style: cancelStyle},
	})
}

func renderBinarySaveButtons(t theme, width int, focus binarySaveFocus, submitDisabled, blocked bool) string {
	return binarySaveButtonsLayout(t, width, focus, submitDisabled, blocked).content
}
