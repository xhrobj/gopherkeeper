package tui

import (
	"time"

	tea "charm.land/bubbletea/v2"
)

func (m model) updateRecordMouse(msg tea.MouseClickMsg) (tea.Model, tea.Cmd, bool) {
	switch m.dialog {
	case dialogRecordType:
		updated, command := m.updateRecordTypeMouse(msg)
		return updated, command, true
	case dialogRecordCreate:
		updated, command := m.updateRecordCreateMouse(msg)
		return updated, command, true
	case dialogRecordEdit:
		updated, command := m.updateRecordEditMouse(msg)
		return updated, command, true
	case dialogRecordView:
		updated, command := m.updateRecordViewMouse(msg)
		return updated, command, true
	case dialogRecordDelete:
		updated, command := m.updateRecordDeleteMouse(msg)
		return updated, command, true
	case dialogBinarySave:
		updated, command := m.updateBinarySaveMouse(msg)
		return updated, command, true
	case dialogNone:
		if m.recordFeature.workspace.open {
			updated, command := m.updateRecordWorkspaceMouse(msg)
			return updated, command, true
		}
	}

	return m, nil, false
}

func (m model) updateRecordTypeMouse(msg tea.MouseClickMsg) (tea.Model, tea.Cmd) {
	window, ok := m.dialogPlacement()
	if !ok {
		return m, nil
	}

	layout := newRecordTypePickerLayout()

	for index, bounds := range window.screenBounds(layout.typeBounds) {
		if bounds.contains(msg.X, msg.Y) {
			m.recordFeature.typePicker.selected = index
			m.openRecordCreate(index)
			return m, nil
		}
	}

	cancel := layout.cancelBounds.translated(window.x, window.y)

	if cancel.contains(msg.X, msg.Y) {
		m.recordFeature.typePicker.selected = len(recordCreateTypes)
		m.closeRecordCreate()
	}

	return m, nil
}

func (m model) updateRecordCreateMouse(msg tea.MouseClickMsg) (tea.Model, tea.Cmd) {
	if m.operations.pending(operationCreateRecord) {
		return m, nil
	}

	for _, field := range m.recordCreateFieldBounds() {
		if field.bounds.contains(msg.X, msg.Y) {
			m.recordFeature.createForm.setFocus(field.control)
			if field.control == recordFormFilePath {
				return m.activateRecordCreate()
			}
			return m, nil
		}
	}

	for _, button := range m.recordCreateButtonBounds() {
		if !button.bounds.contains(msg.X, msg.Y) {
			continue
		}
		if button.control == recordFormSubmit && !m.recordFeature.createForm.canSubmit() {
			return m, nil
		}
		m.recordFeature.createForm.setFocus(button.control)
		return m.activateRecordCreate()
	}

	return m, nil
}

func (m model) updateRecordEditMouse(msg tea.MouseClickMsg) (tea.Model, tea.Cmd) {
	if m.recordFeature.edit.status != recordEditReady || m.operations.pending(operationEditRecord) {
		return m, nil
	}

	for _, field := range m.recordEditFieldBounds() {
		if field.bounds.contains(msg.X, msg.Y) {
			m.recordFeature.edit.form.setFocus(field.control)
			if field.control == recordFormFilePath {
				return m.activateRecordEdit()
			}
			return m, nil
		}
	}

	for _, button := range m.recordEditButtonBounds() {
		if !button.bounds.contains(msg.X, msg.Y) {
			continue
		}

		if button.control == recordFormSubmit && !m.recordFeature.edit.form.canSubmit() {
			return m, nil
		}

		m.recordFeature.edit.form.setFocus(button.control)

		return m.activateRecordEdit()
	}

	return m, nil
}

func (m model) updateRecordViewMouse(msg tea.MouseClickMsg) (tea.Model, tea.Cmd) {
	if step := m.recordViewTextAreaArrowAt(msg.X, msg.Y); step != 0 {
		m.recordFeature.view.textArea.scroll(step)
		return m, nil
	}

	for index, bounds := range m.dialogButtonBounds() {
		if !bounds.contains(msg.X, msg.Y) {
			continue
		}

		m.activeButton = index

		return m.activateDialogButton()
	}

	return m, nil
}

func (m model) updateRecordDeleteMouse(msg tea.MouseClickMsg) (tea.Model, tea.Cmd) {
	if m.operations.pending(operationDeleteRecord) {
		return m, nil
	}

	for index, bounds := range m.dialogButtonBounds() {
		if !bounds.contains(msg.X, msg.Y) {
			continue
		}

		m.activeButton = index

		return m.activateRecordDelete()
	}

	return m, nil
}

func (m model) updateBinarySaveMouse(msg tea.MouseClickMsg) (tea.Model, tea.Cmd) {
	if m.operations.pending(operationBinarySave) {
		return m, nil
	}

	for _, bounds := range m.binarySaveFieldBounds() {
		if bounds.contains(msg.X, msg.Y) {
			m.recordFeature.binarySaveForm.setFocus(binarySavePath)
			return m.activateBinarySave()
		}
	}

	for index, bounds := range m.dialogButtonBounds() {
		if !bounds.contains(msg.X, msg.Y) {
			continue
		}

		m.recordFeature.binarySaveForm.setFocus(binarySaveFocus(int(binarySaveSubmit) + index))

		return m.activateBinarySave()
	}

	return m, nil
}

func (m model) updateRecordWorkspaceMouse(msg tea.MouseClickMsg) (tea.Model, tea.Cmd) {
	index, ok := m.recordRowAt(msg.X, msg.Y)
	if !ok {
		return m, nil
	}

	openRequested := m.recordFeature.workspace.registerClick(index, time.Now())
	m.recordFeature.workspace.moveTo(index, recordWorkspacePageSize(m.height))
	if !openRequested {
		return m, nil
	}

	metadata, selected := m.recordFeature.workspace.selectedRecord()
	if !selected {
		return m, nil
	}

	return m, m.beginRecordView(metadata.ID)
}

func (m model) recordRowAt(x, y int) (int, bool) {
	window, ok := m.workspacePlacement()
	if !ok || m.recordFeature.workspace.state != recordListReady {
		return 0, false
	}

	layout := newRecordWorkspaceLayout(window.width, window.height)
	rowBounds := layout.rowBounds.translated(window.x, window.y)

	if !rowBounds.contains(x, y) {
		return 0, false
	}

	row := y - rowBounds.y
	index := m.recordFeature.workspace.offset + row

	if index < 0 || index >= len(m.recordFeature.workspace.records) {
		return 0, false
	}

	return index, true
}

type recordFormControlBound struct {
	control recordFormControl
	bounds  layoutBounds
}

func (m model) recordCreateFieldBounds() []recordFormControlBound {
	return m.recordFormFieldBounds(dialogRecordCreate, &m.recordFeature.createForm)
}

func (m model) recordEditFieldBounds() []recordFormControlBound {
	return m.recordFormFieldBounds(dialogRecordEdit, &m.recordFeature.edit.form)
}

func (m model) recordFormFieldBounds(dialog dialogID, form *recordForm) []recordFormControlBound {
	if m.dialog != dialog {
		return nil
	}

	window, ok := m.dialogPlacement()
	if !ok {
		return nil
	}

	contentWidth := max(1, window.width-4)
	layout := recordFormLayout(form, contentWidth)
	result := make([]recordFormControlBound, 0, len(layout.controls))
	for _, control := range layout.controls {
		result = append(result, recordFormControlBound{
			control: control.control,
			bounds:  control.bounds.translated(window.x+2, window.y+2),
		})
	}

	return result
}

func (m model) recordCreateButtonBounds() []recordFormControlBound {
	return m.recordFormButtonBounds(dialogRecordCreate, &m.recordFeature.createForm, recordCreateWindowMode.submitLabel)
}

func (m model) recordEditButtonBounds() []recordFormControlBound {
	return m.recordFormButtonBounds(dialogRecordEdit, &m.recordFeature.edit.form, recordEditWindowMode.submitLabel)
}

func (m model) recordFormButtonBounds(
	dialog dialogID,
	form *recordForm,
	submitLabel string,
) []recordFormControlBound {
	if m.dialog != dialog {
		return nil
	}

	window, ok := m.dialogPlacement()
	if !ok {
		return nil
	}

	definitions := recordFormButtonDefinitions(*form, submitLabel)
	contentWidth := max(1, window.width-4)
	formLayout := recordFormLayout(form, contentWidth)
	buttonLayout := recordFormButtonsLayout(
		m.theme,
		contentWidth,
		*form,
		m.interactionBlocked(),
		submitLabel,
	).positioned(2, 2+formLayout.buttonRow)

	bounds := window.screenBounds(buttonLayout.bounds)
	result := make([]recordFormControlBound, 0, len(bounds))

	for index, bound := range bounds {
		result = append(result, recordFormControlBound{control: definitions[index].control, bounds: bound})
	}

	return result
}

func (m model) recordViewTextAreaBounds() (layoutBounds, bool) {
	if m.dialog != dialogRecordView || !m.recordFeature.view.textArea.active() {
		return layoutBounds{}, false
	}

	window, ok := m.dialogPlacement()
	if !ok {
		return layoutBounds{}, false
	}

	layout := newRecordViewContentLayout(m.theme, window.width, window.height, m.recordFeature.view)
	if !layout.textAreaVisible {
		return layoutBounds{}, false
	}

	return layout.textAreaBounds.translated(window.x, window.y), true
}

func (m model) recordViewTextAreaArrowAt(x, y int) int {
	if m.dialog != dialogRecordView || !m.recordFeature.view.textArea.active() {
		return 0
	}

	window, ok := m.dialogPlacement()
	if !ok {
		return 0
	}

	layout := newRecordViewContentLayout(m.theme, window.width, window.height, m.recordFeature.view)
	if !layout.textAreaVisible {
		return 0
	}

	if layout.textAreaUp.translated(window.x, window.y).contains(x, y) {
		return -1
	}

	if layout.textAreaDown.translated(window.x, window.y).contains(x, y) {
		return 1
	}

	return 0
}
