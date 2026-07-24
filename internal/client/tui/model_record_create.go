package tui

import tea "charm.land/bubbletea/v2"

func (m model) recordCreateAvailable() bool {
	return m.backend != nil
}

func (m *model) openRecordTypePicker() tea.Cmd {
	m.operations.cancel(operationCreateRecord)
	m.recordFeature.typePicker = recordTypePicker{}
	m.recordFeature.createForm = recordForm{}
	m.dialog = dialogRecordType
	m.activeButton = 0

	return nil
}

func (m *model) openRecordCreate(recordTypeIndex int) {
	if recordTypeIndex < 0 || recordTypeIndex >= len(recordCreateTypes) {
		return
	}

	m.operations.cancel(operationCreateRecord)
	m.recordFeature.createForm = newRecordCreateForm(recordCreateTypes[recordTypeIndex])
	m.dialog = dialogRecordCreate
	m.activeButton = 0
}

func (m model) openRecordCreateBinaryPathPicker() (tea.Model, tea.Cmd) {
	picker, command := newPathPicker(
		pathPickerBinaryCreateFile,
		m.recordFeature.createForm.filePath.value,
		m.height,
	)

	m.pathPicker = picker
	m.dialog = dialogPathPicker

	return m, command
}

func (m model) updateRecordTypePicker(key string) (tea.Model, tea.Cmd) {
	switch key {
	case "esc":
		m.closeRecordCreate()
	case "up", "left", "shift+tab":
		m.recordFeature.typePicker.move(-1)
	case "down", "right", "tab":
		m.recordFeature.typePicker.move(1)
	case "home":
		m.recordFeature.typePicker.selected = 0
	case "end":
		m.recordFeature.typePicker.selected = len(recordCreateTypes)
	case "enter":
		if recordType, ok := m.recordFeature.typePicker.selectedType(); ok {
			for index, candidate := range recordCreateTypes {
				if candidate == recordType {
					m.openRecordCreate(index)
					break
				}
			}
		} else {
			m.closeRecordCreate()
		}
	}

	return m, nil
}

func (m model) updateRecordCreate(key string) (tea.Model, tea.Cmd) {
	if m.operations.pending(operationCreateRecord) {
		return m, nil
	}

	if key == "esc" {
		m.closeRecordCreate()
		return m, nil
	}

	if updateRecordForm(&m.recordFeature.createForm, key) {
		return m.activateRecordCreate()
	}

	return m, nil
}

func (m model) activateRecordCreate() (tea.Model, tea.Cmd) {
	if m.operations.pending(operationCreateRecord) {
		return m, nil
	}

	switch m.recordFeature.createForm.activeControl() {
	case recordFormFilePath:
		return m.openRecordCreateBinaryPathPicker()
	case recordFormSubmit:
		if !m.recordFeature.createForm.canSubmit() {
			return m, nil
		}
		input := recordFormInputFrom(m.recordFeature.createForm)
		if err := input.validate(); err != nil {
			m.showAlert(alertError, "Invalid record", cleanRecordCreateError(err), dialogRecordCreate)
			return m, nil
		}
		if m.backend == nil {
			return m, nil
		}
		requestCtx, requestID := m.operations.begin(m.ctx, operationCreateRecord)
		return m, m.operationCommand(operationCreateRecord, recordCreateCommand(
			requestCtx,
			m.backend,
			m.readBinaryFile,
			requestID,
			input,
		))
	case recordFormReveal:
		m.recordFeature.createForm.toggleReveal()
	case recordFormCancel:
		m.closeRecordCreate()
	default:
		m.recordFeature.createForm.move(1)
	}

	return m, nil
}

func (m model) handleRecordCreateResult(msg recordCreateResultMsg) (tea.Model, tea.Cmd) {
	if !m.operations.accepts(operationCreateRecord, msg.requestID) {
		return m, nil
	}

	m.operations.finish(operationCreateRecord)

	if isNotLoggedIn(msg.err) {
		m.handleSessionExpired()
		return m, nil
	}

	if msg.err != nil {
		m.showAlert(alertError, "Unable to create record", cleanRecordCreateError(msg.err), dialogRecordCreate)
		return m, nil
	}

	m.dialog = dialogNone
	m.recordFeature.createForm = recordForm{}
	m.recordFeature.typePicker = recordTypePicker{}
	m.operations.cancel(operationListRecords)

	if m.recordFeature.workspace.open && m.recordFeature.workspace.state == recordListReady {
		m.recordFeature.workspace.prepend(msg.record.Metadata, recordWorkspacePageSize(m.height))
	} else {
		m.recordFeature.workspace.clear()
	}

	m.showAlertWithHighlight(
		alertNotice,
		"Record created",
		recordCreateNotice(msg.record),
		msg.record.Metadata.Title,
		dialogNone,
	)

	return m, nil
}

func (m *model) closeRecordCreate() {
	m.operations.cancel(operationCreateRecord)
	m.recordFeature.typePicker = recordTypePicker{}
	m.recordFeature.createForm = recordForm{}
	m.dialog = dialogNone
	m.activeButton = 0
}
