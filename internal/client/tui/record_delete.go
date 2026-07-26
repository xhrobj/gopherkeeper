package tui

import (
	"context"
	"errors"

	tea "charm.land/bubbletea/v2"
	recordmodel "github.com/xhrobj/gopherkeeper/internal/model"
)

type recordDeleteState struct {
	metadata recordmodel.RecordMetadata
}

type recordDeleteResultMsg struct {
	requestID uint64
	metadata  recordmodel.RecordMetadata
	err       error
}

func (m model) recordDeleteAvailable() bool {
	return m.backend != nil
}

func (m model) recordDeleteTarget() (recordmodel.RecordMetadata, bool) {
	if m.dialog == dialogRecordEdit {
		return recordmodel.RecordMetadata{}, false
	}

	if m.dialog == dialogRecordView && m.recordFeature.view.status == recordViewReady {
		if m.recordFeature.view.source != recordSourceServer {
			return recordmodel.RecordMetadata{}, false
		}
		return m.recordFeature.view.record.Metadata, true
	}

	if m.recordFeature.workspace.source != recordSourceServer {
		return recordmodel.RecordMetadata{}, false
	}

	return m.recordFeature.workspace.selectedRecord()
}

func (m *model) openRecordDelete(metadata recordmodel.RecordMetadata) {
	m.recordFeature.deletion = recordDeleteState{metadata: metadata}
	m.dialog = dialogRecordDelete
	m.activeButton = 1
}

func (m *model) closeRecordDelete() {
	m.operations.cancel(operationDeleteRecord)
	m.recordFeature.deletion = recordDeleteState{}
	m.dialog = dialogNone
	m.activeButton = 0
}

func (m model) updateRecordDelete(key string) (tea.Model, tea.Cmd) {
	if m.operations.pending(operationDeleteRecord) {
		return m, nil
	}

	switch key {
	case "esc":
		m.closeRecordDelete()
	case "tab", "shift+tab", "left", "right":
		m.activeButton = 1 - m.activeButton
	case "enter":
		return m.activateRecordDelete()
	}

	return m, nil
}

func (m model) activateRecordDelete() (tea.Model, tea.Cmd) {
	if m.activeButton != 0 {
		m.closeRecordDelete()
		return m, nil
	}
	if m.backend == nil || !m.authentication.session.authenticated() || m.recordFeature.deletion.metadata.ID == "" {
		return m, nil
	}

	requestCtx, requestID := m.operations.begin(m.operationDone, operationDeleteRecord)
	metadata := m.recordFeature.deletion.metadata

	return m, m.operationCommand(operationDeleteRecord, recordDeleteCommand(requestCtx, m.backend, requestID, metadata))
}

func recordDeleteCommand(
	ctx context.Context,
	backend Backend,
	requestID uint64,
	metadata recordmodel.RecordMetadata,
) tea.Cmd {
	return func() tea.Msg {
		err := backend.DeleteRecord(ctx, metadata.ID, metadata.Revision)
		return recordDeleteResultMsg{requestID: requestID, metadata: metadata, err: err}
	}
}

func (m model) handleRecordDeleteResult(msg recordDeleteResultMsg) (tea.Model, tea.Cmd) {
	if !m.operations.accepts(operationDeleteRecord, msg.requestID) {
		return m, nil
	}

	m.operations.finish(operationDeleteRecord)
	if isNotLoggedIn(msg.err) {
		m.handleSessionExpired()
		return m, nil
	}
	if msg.err != nil {
		m.activeButton = 1
		m.showAlert(alertError, "Unable to delete record", cleanRecordDeleteError(msg.err), dialogRecordDelete)
		return m, nil
	}

	m.recordFeature.workspace.remove(msg.metadata.ID, recordWorkspacePageSize(m.height))
	m.recordFeature.deletion = recordDeleteState{}
	m.dialog = dialogNone
	m.activeButton = 0

	m.showAlertWithHighlight(
		alertNotice,
		"Record deleted",
		"Deleted record "+msg.metadata.Title,
		msg.metadata.Title,
		dialogNone,
	)

	return m, nil
}

func cleanRecordDeleteError(err error) string {
	if err == nil {
		return "Unknown record deletion error"
	}

	if errors.Is(err, context.Canceled) {
		return "Record deletion canceled"
	}

	return cleanFailureMessage(err, "Unable to delete record from Server")
}
