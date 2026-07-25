package tui

import (
	"context"
	"errors"

	tea "charm.land/bubbletea/v2"
	"github.com/xhrobj/gopherkeeper/internal/client/usecase"
	recordmodel "github.com/xhrobj/gopherkeeper/internal/model"
)

type recordViewStatus int

const (
	recordViewIdle recordViewStatus = iota
	recordViewLoading
	recordViewReady
)

type recordViewState struct {
	source   recordSource
	login    string
	status   recordViewStatus
	record   recordmodel.Record
	offset   int
	revealed bool
	textArea readOnlyTextArea
}

func (state *recordViewState) begin(metadata recordmodel.RecordMetadata, sources ...recordSource) {
	source := recordSourceServer
	if len(sources) > 0 {
		source = sources[0]
	}
	state.source = source
	state.login = ""
	state.status = recordViewLoading
	state.record = recordmodel.Record{Metadata: metadata}
	state.offset = 0
	state.revealed = false
	state.textArea.clear()
}

func (state *recordViewState) apply(record recordmodel.Record, screenWidth, screenHeight int) {
	state.status = recordViewReady
	state.record = record
	state.offset = 0
	state.revealed = false
	state.textArea.clear()

	if value, ok := recordViewLongText(record); ok {
		state.textArea.setValue(value)
		state.resize(screenWidth, screenHeight)
	}
}

func (state *recordViewState) clear() {
	*state = recordViewState{}
}

func (state *recordViewState) resize(screenWidth, screenHeight int) {
	if !state.textArea.active() {
		return
	}
	state.textArea.resize(
		max(1, recordViewContentWidth(screenWidth)-1),
		readOnlyTextAreaRows(screenHeight),
	)
}

type recordViewResultMsg struct {
	requestID uint64
	record    recordmodel.Record
	err       error
}

func (m model) recordViewTarget() (string, bool) {
	if m.dialog != dialogNone || !m.recordFeature.workspace.open ||
		m.recordFeature.workspace.source != recordSourceServer {
		return "", false
	}

	metadata, ok := m.recordFeature.workspace.selectedRecord()
	if !ok {
		return "", false
	}

	return metadata.ID, true
}

func (m model) cachedRecordViewTarget() (string, bool) {
	if m.dialog != dialogNone || !m.recordFeature.workspace.open ||
		m.recordFeature.workspace.source != recordSourceCache {
		return "", false
	}

	metadata, ok := m.recordFeature.workspace.selectedRecord()
	if !ok {
		return "", false
	}

	return metadata.ID, true
}

func (m *model) beginRecordView(recordID string) tea.Cmd {
	if m.backend == nil || recordID == "" {
		return nil
	}

	source := m.recordFeature.workspace.source
	metadata := m.recordMetadataByID(recordID)

	m.recordFeature.view.begin(metadata, source)
	if source == recordSourceCache {
		m.recordFeature.view.login = m.recordFeature.workspace.login
	}
	m.dialog = dialogRecordView
	m.activeButton = recordViewDefaultButton(m.recordFeature.view.record)

	if source == recordSourceCache {
		requestCtx, requestID := m.operations.begin(m.ctx, operationViewCachedRecord)

		return m.operationCommand(
			operationViewCachedRecord,
			cachedRecordViewCommand(requestCtx, m.backend, requestID, recordID),
		)
	}

	requestCtx, requestID := m.operations.begin(m.ctx, operationViewRecord)

	return m.operationCommand(operationViewRecord, recordViewCommand(requestCtx, m.backend, requestID, recordID))
}

func (m model) recordMetadataByID(recordID string) recordmodel.RecordMetadata {
	for _, metadata := range m.recordFeature.workspace.records {
		if metadata.ID == recordID {
			return metadata
		}
	}

	return recordmodel.RecordMetadata{ID: recordID}
}

func recordViewCommand(ctx context.Context, backend Backend, requestID uint64, recordID string) tea.Cmd {
	return func() tea.Msg {
		record, err := backend.GetRecord(ctx, recordID)
		return recordViewResultMsg{requestID: requestID, record: record, err: err}
	}
}

type cachedRecordViewResultMsg struct {
	requestID uint64
	record    recordmodel.Record
	err       error
}

func cachedRecordViewCommand(
	ctx context.Context,
	backend Backend,
	requestID uint64,
	recordID string,
) tea.Cmd {
	return func() tea.Msg {
		record, err := backend.GetCachedRecord(ctx, recordID)
		return cachedRecordViewResultMsg{requestID: requestID, record: record, err: err}
	}
}

func (m model) handleRecordViewResult(msg recordViewResultMsg) (tea.Model, tea.Cmd) {
	if !m.operations.accepts(operationViewRecord, msg.requestID) {
		return m, nil
	}

	m.operations.finish(operationViewRecord)
	if isNotLoggedIn(msg.err) {
		m.handleSessionExpired()
		return m, nil
	}
	if msg.err != nil {
		m.leaveRecordView()
		m.dialog = dialogNone
		m.showAlert(alertError, "Unable to load record", cleanRecordViewError(msg.err), dialogNone)
		return m, nil
	}
	if m.dialog != dialogRecordView || !m.authentication.session.authenticated() {
		m.leaveRecordView()
		return m, nil
	}

	m.recordFeature.view.apply(msg.record, m.width, m.height)
	m.returnToRecordView()

	return m, nil
}

func (m model) handleCachedRecordViewResult(msg cachedRecordViewResultMsg) (tea.Model, tea.Cmd) {
	if !m.operations.accepts(operationViewCachedRecord, msg.requestID) {
		return m, nil
	}

	m.operations.finish(operationViewCachedRecord)
	if msg.err != nil {
		m.leaveRecordView()
		m.dialog = dialogNone
		m.showAlert(alertError, "Unable to load cached record", cleanCachedRecordViewError(msg.err), dialogNone)
		return m, nil
	}

	if m.dialog != dialogRecordView || m.recordFeature.workspace.source != recordSourceCache {
		m.leaveRecordView()
		return m, nil
	}

	m.recordFeature.view.apply(msg.record, m.width, m.height)
	m.recordFeature.view.source = recordSourceCache

	m.returnToRecordView()

	return m, nil
}

func recordViewDefaultButton(record recordmodel.Record) int {
	if record.Metadata.Type == recordmodel.RecordTypeBinary || recordViewIsBinary(record) {
		return 1
	}
	return 0
}

func (m *model) returnToRecordView() {
	m.dialog = dialogRecordView
	m.activeButton = recordViewDefaultButton(m.recordFeature.view.record)
}

func (m *model) leaveRecordView() {
	m.operations.cancel(operationViewRecord)
	m.operations.cancel(operationViewCachedRecord)
	m.operations.cancel(operationBinarySave)
	m.recordFeature.binarySaveForm = newBinarySaveForm("")
	m.recordFeature.view.clear()
}

func (m *model) closeRecordView() {
	m.leaveRecordView()
	m.dialog = dialogNone
	m.activeButton = 0
}

func (m model) updateRecordView(key string) (tea.Model, tea.Cmd) {
	if key == "esc" {
		m.closeRecordView()
		return m, nil
	}

	if m.recordFeature.view.status != recordViewReady {
		return m, nil
	}

	hasTwoButtons := len(recordViewButtonLabels(m.recordFeature.view)) == 2
	if key == "tab" || key == "shift+tab" || key == "left" || key == "right" {
		if hasTwoButtons {
			m.activeButton = 1 - m.activeButton
		}
		return m, nil
	}
	if key == "enter" {
		return m.activateDialogButton()
	}

	pageSize := recordViewPageSizeForState(m.theme, m.width, m.height, m.recordFeature.view)
	lines := recordViewLines(m.theme, m.recordFeature.view, recordViewContentWidth(m.width))
	maxOffset := max(0, len(lines)-pageSize)

	switch key {
	case "up":
		if !m.scrollRecordViewTextArea(-1, lines, pageSize) {
			m.recordFeature.view.offset = clamp(m.recordFeature.view.offset-1, 0, maxOffset)
		}
	case "down":
		if !m.scrollRecordViewTextArea(1, lines, pageSize) {
			m.recordFeature.view.offset = clamp(m.recordFeature.view.offset+1, 0, maxOffset)
		}
	case "pgup":
		if !m.pageRecordViewTextArea(-1, lines, pageSize) {
			m.recordFeature.view.offset = clamp(m.recordFeature.view.offset-pageSize, 0, maxOffset)
		}
	case "pgdown":
		if !m.pageRecordViewTextArea(1, lines, pageSize) {
			m.recordFeature.view.offset = clamp(m.recordFeature.view.offset+pageSize, 0, maxOffset)
		}
	case "home":
		m.recordFeature.view.offset = 0
		m.recordFeature.view.textArea.home()
	case "end":
		m.recordFeature.view.offset = maxOffset
		m.recordFeature.view.textArea.end()
	}

	return m, nil
}

func (m *model) scrollRecordViewTextArea(step int, lines []recordViewLine, pageSize int) bool {
	if !m.recordFeature.view.textArea.active() {
		return false
	}

	start, end, ok := recordViewTextAreaRange(lines)
	if !ok {
		return false
	}

	visibleStart := m.recordFeature.view.offset
	visibleEnd := visibleStart + max(1, pageSize)
	if step > 0 && end > visibleEnd {
		m.recordFeature.view.offset++
		return true
	}
	if step < 0 && start < visibleStart {
		m.recordFeature.view.offset--
		return true
	}

	return m.recordFeature.view.textArea.scroll(step)
}

func (m *model) pageRecordViewTextArea(step int, lines []recordViewLine, pageSize int) bool {
	if !m.recordFeature.view.textArea.active() {
		return false
	}

	start, end, ok := recordViewTextAreaRange(lines)
	if !ok {
		return false
	}

	visibleStart := m.recordFeature.view.offset
	visibleEnd := visibleStart + max(1, pageSize)
	if step > 0 && end > visibleEnd {
		m.recordFeature.view.offset = min(end-visibleEnd+m.recordFeature.view.offset, max(0, len(lines)-pageSize))
		return true
	}
	if step < 0 && start < visibleStart {
		m.recordFeature.view.offset = start
		return true
	}

	return m.recordFeature.view.textArea.page(step)
}

func recordViewLongText(record recordmodel.Record) (string, bool) {
	payload, ok := record.Payload.(*recordmodel.TextPayload)
	if !ok || payload == nil {
		return "", false
	}

	return payload.Text, true
}

func recordViewIsBinary(record recordmodel.Record) bool {
	payload, ok := record.Payload.(*recordmodel.BinaryPayload)
	return ok && payload != nil
}

func recordViewHasSensitiveFields(record recordmodel.Record) bool {
	switch payload := record.Payload.(type) {
	case *recordmodel.CredentialsPayload:
		return payload != nil && payload.Password != ""
	case *recordmodel.CardPayload:
		return payload != nil && (payload.Number != "" || payload.CVV != "")
	default:
		return false
	}
}

func cleanRecordViewError(err error) string {
	if err == nil {
		return "Unknown record error"
	}

	if errors.Is(err, context.Canceled) {
		return "Record loading canceled"
	}

	return cleanFailureMessage(err, "Unable to load record from Server")
}

func cleanCachedRecordViewError(err error) string {
	if err == nil {
		return "Unknown cached record error"
	}

	if errors.Is(err, context.Canceled) {
		return "Cached record loading canceled"
	}

	if errors.Is(err, usecase.ErrLocalCacheRecordsUnreadable) {
		return "This cached record is damaged or incompatible.\nRun Cache/Sync... to restore it from the Server."
	}

	return cleanFailureMessage(err, "Unable to load record from Cache")
}
