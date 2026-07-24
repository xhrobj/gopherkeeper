package tui

import (
	"strings"

	tea "charm.land/bubbletea/v2"
)

func (m model) updateConfig(key string) (tea.Model, tea.Cmd) {
	switch key {
	case "esc":
		m.dialog = dialogNone
		m.configForm = newConfigForm(m.config)
	case "tab", "down":
		m.configForm.move(1)
	case "shift+tab", "up":
		m.configForm.move(-1)
	case "left":
		if m.configForm.activeField() != nil {
			m.configForm.moveCursor(-1)
		} else {
			m.configForm.move(-1)
		}
	case "right":
		if m.configForm.activeField() != nil {
			m.configForm.moveCursor(1)
		} else {
			m.configForm.move(1)
		}
	case "home":
		m.configForm.moveCursorToStart()
	case "end":
		m.configForm.moveCursorToEnd()
	case "backspace":
		m.configForm.backspace()
	case "delete":
		m.configForm.delete()
	case "enter":
		if m.configForm.activeField() != nil {
			m.configForm.move(1)
			return m, nil
		}
		if target, ok := configBrowseTarget(m.configForm.focus); ok {
			return m.openConfigPathPicker(target)
		}
		return m.activateConfig()
	default:
		m.configForm.insertKey(key)
	}

	return m, nil
}

func (m model) openConfigPathPicker(target pathPickerTarget) (tea.Model, tea.Cmd) {
	fieldIndex := configTargetFieldIndex(target)
	picker, command := newPathPicker(
		target,
		m.configForm.fields[fieldIndex].value,
		m.height,
	)
	m.pathPicker = picker
	m.dialog = dialogPathPicker

	return m, command
}

func (m model) updatePathPicker(msg tea.Msg) (tea.Model, tea.Cmd) {
	switch msg := msg.(type) {
	case pathPickerReadMsg:
		m.pathPicker.applyReadResult(msg)
		if m.pathPicker.focus == pathPickerSelect && !m.pathPicker.selectEnabled() {
			m.pathPicker.focusTree()
		}
		return m, nil
	case pathPickerMouseOpenMsg:
		if m.menuFocused || m.dropdownOpen || m.alert != alertNone ||
			!m.pathPicker.acceptsMouseOpen(msg) {
			return m, nil
		}
		m.pathPicker.clearMouseClick()
		m.pathPicker.selected = msg.entryIndex
		m.pathPicker.focusTree()
		entry, ok := m.pathPicker.highlighted()
		if !ok || !entry.directory {
			return m, nil
		}
		return m, m.pathPicker.openHighlightedDirectory()
	case tea.KeyPressMsg:
		m.pathPicker.clearMouseClick()
		switch msg.String() {
		case "esc":
			m.closePathPicker()
			return m, nil
		case "tab":
			m.pathPicker.moveFocus(1)
		case "shift+tab":
			m.pathPicker.moveFocus(-1)
		case "up":
			if m.pathPicker.focus == pathPickerTree {
				m.pathPicker.move(-1)
			}
		case "down":
			if m.pathPicker.focus == pathPickerTree {
				m.pathPicker.move(1)
			}
		case "enter":
			return m.activatePathPickerFocus()
		}
	}

	return m, nil
}

func (m model) activatePathPickerFocus() (tea.Model, tea.Cmd) {
	switch m.pathPicker.focus {
	case pathPickerTree:
		entry, ok := m.pathPicker.highlighted()
		if !ok || !entry.directory {
			return m, nil
		}
		return m, m.pathPicker.openHighlightedDirectory()
	case pathPickerSelect:
		path, ok := m.pathPicker.selectedPath()
		if !ok {
			return m, nil
		}
		m.applyPathSelection(path)
	case pathPickerCancel:
		m.closePathPicker()
	}

	return m, nil
}

func (m *model) closePathPicker() {
	target := m.pathPicker.target
	m.pathPicker = pathPicker{}

	switch target {
	case pathPickerBinarySaveDirectory:
		m.dialog = dialogBinarySave
		m.recordFeature.binarySaveForm.setFocus(binarySavePath)
	case pathPickerBinaryCreateFile:
		m.dialog = dialogRecordCreate
		m.recordFeature.createForm.setFocus(recordFormFilePath)
	case pathPickerBinaryEditFile:
		m.dialog = dialogRecordEdit
		m.recordFeature.edit.form.setFocus(recordFormFilePath)
	default:
		m.dialog = dialogConfig
	}
}

func (m *model) applyPathSelection(path string) {
	target := m.pathPicker.target
	rootDirectory := m.pathPicker.rootDirectory
	value := selectedPathValue(rootDirectory, path)

	switch target {
	case pathPickerBinarySaveDirectory:
		m.recordFeature.binarySaveForm.setDirectory(value)
		m.pathPicker = pathPicker{}
		m.dialog = dialogBinarySave
		m.recordFeature.binarySaveForm.setFocus(binarySavePath)
		return
	case pathPickerBinaryCreateFile:
		m.recordFeature.createForm.filePath.setValue(value)
		m.recordFeature.createForm.setFocus(recordFormFilePath)
		m.pathPicker = pathPicker{}
		m.dialog = dialogRecordCreate
		return
	case pathPickerBinaryEditFile:
		m.recordFeature.edit.form.filePath.setValue(value)
		m.recordFeature.edit.form.setFocus(recordFormFilePath)
		m.pathPicker = pathPicker{}
		m.dialog = dialogRecordEdit
		return
	}

	fieldIndex := configTargetFieldIndex(target)
	m.configForm.fields[fieldIndex].setValue(value)
	m.configForm.setFocus(configTargetFieldFocus(target))
	m.configForm.errorMessage = ""
	m.pathPicker = pathPicker{}
	m.dialog = dialogConfig
}

func (m model) activateConfig() (tea.Model, tea.Cmd) {
	switch m.configForm.focus {
	case configSave:
		if !m.configForm.canSave() {
			return m, nil
		}

		candidate := m.configForm.config()
		candidate.Address = strings.TrimSpace(candidate.Address)
		candidate.CACertFile = strings.TrimSpace(candidate.CACertFile)
		changed := candidate != m.config
		nextBackend := m.backend
		if changed {
			var err error
			nextBackend, err = createBackend(m.backendFactory, candidate)
			if err != nil {
				m.configForm.errorMessage = cleanFailureMessage(err, "Unable to apply config")
				return m, nil
			}
		}
		if m.configFile != "" && m.saveConfig != nil {
			if err := m.saveConfig(m.configFile, candidate); err != nil {
				m.configForm.errorMessage = "Unable to save config file"
				return m, nil
			}
		}
		m.config = candidate
		if changed {
			m.cancelAllRequests()
			m.leaveRecordView()
			m.recordFeature.workspace.clear()
			m.backend = nextBackend
		}
		m.dialog = dialogNone
		m.configForm = newConfigForm(m.config)
		if changed {
			m.statusState = serverStatusIdle
			m.statusValue = ""
			m.statusFailure = serverStatusFailure{}
			return m, m.beginCurrentUserCheck(currentUserCheckRestore)
		}
	case configCancel:
		m.dialog = dialogNone
		m.configForm = newConfigForm(m.config)
	}

	return m, nil
}
