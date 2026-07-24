package tui

import (
	"context"
	"errors"

	tea "charm.land/bubbletea/v2"
	recordmodel "github.com/xhrobj/gopherkeeper/internal/model"
)

type recordListResultMsg struct {
	requestID uint64
	records   []recordmodel.RecordMetadata
	err       error
}

func onlineRecordListCommand(
	ctx context.Context,
	backend Backend,
	requestID uint64,
) tea.Cmd {
	return func() tea.Msg {
		records, err := backend.ListRecords(ctx)
		return recordListResultMsg{
			requestID: requestID,
			records:   records,
			err:       err,
		}
	}
}

func cleanRecordListError(err error) string {
	if err == nil {
		return "Unknown record list error"
	}

	if errors.Is(err, context.Canceled) {
		return "Record loading canceled"
	}

	return cleanFailureMessage(err, "Unable to load records from Server")
}

// clearRecordState отменяет все операции с записями и удаляет приватные данные из модели TUI.
func (m *model) clearRecordState() {
	m.operations.cancel(operationListRecords)
	m.operations.cancel(operationViewRecord)
	m.operations.cancel(operationBinarySave)
	m.operations.cancel(operationCreateRecord)
	m.cancelRecordEditRequests()
	m.operations.cancel(operationDeleteRecord)

	m.recordFeature.workspace.clear()
	m.recordFeature.view.clear()
	m.recordFeature.binarySaveForm = newBinarySaveForm("")
	m.recordFeature.typePicker = recordTypePicker{}
	m.recordFeature.createForm = recordForm{}
	m.recordFeature.edit.clear()
	m.recordFeature.deletion = recordDeleteState{}
}

func (m model) recordsAvailable() bool {
	return m.backend != nil
}

func (m *model) beginOnlineRecordList() tea.Cmd {
	if m.backend == nil || !m.authentication.session.authenticated() {
		return nil
	}

	requestCtx, requestID := m.operations.begin(m.ctx, operationListRecords)
	m.recordFeature.workspace.beginServer()
	return m.operationCommand(operationListRecords, onlineRecordListCommand(requestCtx, m.backend, requestID))
}

func (m *model) closeRecordWorkspace() {
	m.operations.cancel(operationListRecords)
	m.recordFeature.workspace.clear()
}

func (m model) handleRecordListResult(msg recordListResultMsg) (tea.Model, tea.Cmd) {
	if !m.operations.accepts(operationListRecords, msg.requestID) {
		return m, nil
	}
	m.operations.finish(operationListRecords)
	if isNotLoggedIn(msg.err) {
		m.handleSessionExpired()
		return m, nil
	}
	if msg.err != nil {
		message := cleanRecordListError(msg.err)
		if m.recordFeature.workspace.state != recordListReady {
			m.recordFeature.workspace.fail(message)
		}
		m.showAlert(alertError, "Unable to load records", message, m.dialog)
		return m, nil
	}

	m.recordFeature.workspace.apply(msg.records, recordWorkspacePageSize(m.height))

	return m, nil
}

func (m model) updateRecordWorkspace(key string) (tea.Model, tea.Cmd) {
	pageSize := recordWorkspacePageSize(m.height)

	switch key {
	case "esc":
		m.closeRecordWorkspace()
	case "up":
		m.recordFeature.workspace.move(-1, pageSize)
	case "down":
		m.recordFeature.workspace.move(1, pageSize)
	case "pgup":
		m.recordFeature.workspace.move(-pageSize, pageSize)
	case "pgdown":
		m.recordFeature.workspace.move(pageSize, pageSize)
	case "home":
		m.recordFeature.workspace.moveTo(0, pageSize)
	case "end":
		m.recordFeature.workspace.moveTo(len(m.recordFeature.workspace.records)-1, pageSize)
	case "enter":
		metadata, ok := m.recordFeature.workspace.selectedRecord()
		if ok {
			return m, m.beginRecordView(metadata.ID)
		}
	case "delete":
		metadata, ok := m.recordFeature.workspace.selectedRecord()
		if ok && m.authentication.session.authenticated() && m.recordDeleteAvailable() {
			m.openRecordDelete(metadata)
		}
	}

	return m, nil
}
