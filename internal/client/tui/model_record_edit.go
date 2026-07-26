package tui

import tea "charm.land/bubbletea/v2"

func (m model) recordEditAvailable() bool {
	return m.backend != nil
}

func (m model) recordEditTarget() (string, bool) {
	if m.dialog == dialogRecordView && m.recordFeature.view.status == recordViewReady {
		if m.recordFeature.view.source != recordSourceServer {
			return "", false
		}
		return m.recordFeature.view.record.Metadata.ID, true
	}

	if m.dialog == dialogRecordEdit && m.recordFeature.edit.status == recordEditReady {
		return m.recordFeature.edit.record.Metadata.ID, true
	}

	if m.recordFeature.workspace.source != recordSourceServer {
		return "", false
	}

	metadata, ok := m.recordFeature.workspace.selectedRecord()
	if !ok {
		return "", false
	}

	return metadata.ID, true
}

func (m *model) openRecordEdit() tea.Cmd {
	if !m.authentication.session.authenticated() || !m.recordEditAvailable() {
		return nil
	}

	if m.dialog == dialogRecordView && m.recordFeature.view.status == recordViewReady {
		m.operations.cancel(operationLoadRecordForEdit)
		m.operations.cancel(operationEditRecord)
		m.recordFeature.edit.apply(m.recordFeature.view.record, dialogRecordView)
		m.dialog = dialogRecordEdit
		m.activeButton = 0
		return nil
	}

	recordID, ok := m.recordEditTarget()
	if !ok {
		return nil
	}

	if m.backend == nil {
		return nil
	}

	m.operations.cancel(operationEditRecord)
	requestCtx, requestID := m.operations.begin(m.operationDone, operationLoadRecordForEdit)
	m.recordFeature.edit.begin(m.recordMetadataByID(recordID), dialogNone)
	m.dialog = dialogRecordEdit
	m.activeButton = 0

	return m.operationCommand(operationLoadRecordForEdit, recordEditLoadCommand(requestCtx, m.backend, requestID, recordID))
}

func (m model) handleRecordEditLoadResult(msg recordEditLoadResultMsg) (tea.Model, tea.Cmd) {
	if !m.operations.accepts(operationLoadRecordForEdit, msg.requestID) {
		return m, nil
	}

	m.operations.finish(operationLoadRecordForEdit)
	if isNotLoggedIn(msg.err) {
		m.handleSessionExpired()
		return m, nil
	}

	if msg.err != nil {
		m.recordFeature.edit.clear()
		m.dialog = dialogNone
		m.showAlert(alertError, "Unable to load record", cleanRecordViewError(msg.err), dialogNone)
		return m, nil
	}

	if m.dialog != dialogRecordEdit || !m.authentication.session.authenticated() {
		m.recordFeature.edit.clear()
		return m, nil
	}

	m.recordFeature.edit.apply(msg.record, dialogNone)

	return m, nil
}

func (m model) openRecordEditBinaryPathPicker() (tea.Model, tea.Cmd) {
	picker, command := newPathPicker(
		pathPickerBinaryEditFile,
		m.recordFeature.edit.form.filePath.value,
		m.height,
	)

	m.pathPicker = picker
	m.dialog = dialogPathPicker

	return m, command
}

func (m model) updateRecordEdit(key string) (tea.Model, tea.Cmd) {
	if m.recordFeature.edit.status == recordEditLoading {
		if key == "esc" {
			m.closeRecordEdit()
		}
		return m, nil
	}

	if m.recordFeature.edit.status != recordEditReady || m.operations.pending(operationEditRecord) {
		return m, nil
	}

	if key == "esc" {
		m.closeRecordEdit()
		return m, nil
	}

	if updateRecordForm(&m.recordFeature.edit.form, key) {
		return m.activateRecordEdit()
	}

	return m, nil
}

func (m model) activateRecordEdit() (tea.Model, tea.Cmd) {
	if m.recordFeature.edit.status != recordEditReady || m.operations.pending(operationEditRecord) {
		return m, nil
	}

	switch m.recordFeature.edit.form.activeControl() {
	case recordFormFilePath:
		return m.openRecordEditBinaryPathPicker()
	case recordFormSubmit:
		if !m.recordFeature.edit.form.canSubmit() {
			return m, nil
		}
		input := recordFormInputFrom(m.recordFeature.edit.form)
		if m.backend == nil {
			return m, nil
		}
		requestCtx, requestID := m.operations.begin(m.operationDone, operationEditRecord)
		return m, m.operationCommand(operationEditRecord, recordEditCommand(
			requestCtx,
			m.backend,
			m.readBinaryFile,
			requestID,
			m.recordFeature.edit.record,
			input,
		))
	case recordFormReveal:
		m.recordFeature.edit.form.toggleReveal()
	case recordFormCancel:
		m.closeRecordEdit()
	default:
		m.recordFeature.edit.form.move(1)
	}

	return m, nil
}

func (m model) handleRecordEditResult(msg recordEditResultMsg) (tea.Model, tea.Cmd) {
	if !m.operations.accepts(operationEditRecord, msg.requestID) {
		return m, nil
	}

	m.operations.finish(operationEditRecord)

	if isNotLoggedIn(msg.err) {
		m.handleSessionExpired()
		return m, nil
	}

	if msg.err != nil {
		m.showAlert(alertError, "Unable to update record", cleanRecordEditError(msg.err), dialogRecordEdit)
		return m, nil
	}

	m.recordFeature.workspace.prepend(msg.record.Metadata, recordWorkspacePageSize(m.height))
	m.recordFeature.view.apply(msg.record, m.width, m.height)
	m.recordFeature.edit.clear()
	m.dialog = dialogRecordView
	m.activeButton = recordViewDefaultButton(msg.record)

	m.showAlertWithHighlight(
		alertNotice,
		"Record updated",
		recordEditNotice(msg.record),
		msg.record.Metadata.Title,
		dialogRecordView,
	)

	return m, nil
}

func (m *model) closeRecordEdit() {
	m.operations.cancel(operationLoadRecordForEdit)
	m.operations.cancel(operationEditRecord)

	returnDialog := m.recordFeature.edit.returnDialog

	m.recordFeature.edit.clear()

	if returnDialog == dialogRecordView && m.recordFeature.view.status == recordViewReady {
		m.returnToRecordView()
		return
	}

	m.dialog = dialogNone
	m.activeButton = 0
}
