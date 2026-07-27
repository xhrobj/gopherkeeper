package tui

import (
	"strings"

	tea "charm.land/bubbletea/v2"
	"github.com/xhrobj/gopherkeeper/internal/client/config"
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
		m.moveConfigHorizontal(-1)
	case "right":
		m.moveConfigHorizontal(1)
	case " ", "space":
		if m.configForm.focus == configTransport {
			m.configForm.toggleTransport()
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
		return m.activateConfigFocus()
	default:
		m.configForm.insertKey(key)
	}

	return m, nil
}

func (m *model) moveConfigHorizontal(step int) {
	switch {
	case m.configForm.focus == configTransport:
		m.configForm.moveTransport(step)
	case m.configForm.activeField() != nil:
		m.configForm.moveCursor(step)
	default:
		m.configForm.move(step)
	}
}

func (m model) activateConfigFocus() (tea.Model, tea.Cmd) {
	if m.configForm.focus == configTransport {
		m.configForm.toggleTransport()
		return m, nil
	}

	if m.configForm.activeField() != nil {
		m.configForm.move(1)
		return m, nil
	}

	if target, ok := configBrowseTarget(m.configForm.focus); ok {
		return m.openConfigPathPicker(target)
	}

	return m.activateConfig()
}

func (m model) openConfigPathPicker(target pathPickerTarget) (tea.Model, tea.Cmd) {
	fieldFocus, ok := configTargetFieldFocus(target)
	if !ok {
		return m, nil
	}

	picker, command := newPathPicker(
		target,
		m.configForm.fields[fieldFocus].value,
		m.height,
	)

	m.pathPicker = picker
	m.dialog = dialogPathPicker

	return m, command
}

func (m model) updatePathPicker(msg tea.Msg) (tea.Model, tea.Cmd) {
	switch msg := msg.(type) {
	case pathPickerReadMsg:
		return m.handlePathPickerRead(msg)
	case pathPickerMouseOpenMsg:
		return m.handlePathPickerMouseOpen(msg)
	case tea.KeyPressMsg:
		return m.updatePathPickerKey(msg.String())
	}

	return m, nil
}

func (m model) handlePathPickerRead(msg pathPickerReadMsg) (tea.Model, tea.Cmd) {
	m.pathPicker.applyReadResult(msg)
	if m.pathPicker.focus == pathPickerSelect && !m.pathPicker.selectEnabled() {
		m.pathPicker.focusTree()
	}
	return m, nil
}

func (m model) handlePathPickerMouseOpen(msg pathPickerMouseOpenMsg) (tea.Model, tea.Cmd) {
	if m.menuFocused || m.dropdownOpen || m.alert != alertNone || !m.pathPicker.acceptsMouseOpen(msg) {
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
}

func (m model) updatePathPickerKey(key string) (tea.Model, tea.Cmd) {
	m.pathPicker.clearMouseClick()
	switch key {
	case "esc":
		m.closePathPicker()
	case "tab":
		m.pathPicker.moveFocus(1)
	case "shift+tab":
		m.pathPicker.moveFocus(-1)
	case "up":
		m.movePathPickerSelection(-1)
	case "down":
		m.movePathPickerSelection(1)
	case "enter":
		return m.activatePathPickerFocus()
	}
	return m, nil
}

func (m *model) movePathPickerSelection(step int) {
	if m.pathPicker.focus == pathPickerTree {
		m.pathPicker.move(step)
	}
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

	fieldFocus, ok := configTargetFieldFocus(target)
	if !ok {
		m.closePathPicker()
		return
	}

	m.configForm.fields[fieldFocus].setValue(value)
	m.configForm.setFocus(fieldFocus)
	m.configForm.errorMessage = ""
	m.pathPicker = pathPicker{}
	m.dialog = dialogConfig
}

func (m model) activateConfig() (tea.Model, tea.Cmd) {
	switch m.configForm.focus {
	case configSave:
		return m.applyConfigForm()
	case configCancel:
		m.closeConfigForm()
	}

	return m, nil
}

func (m model) applyConfigForm() (tea.Model, tea.Cmd) {
	if !m.configForm.canSave() {
		return m, nil
	}

	candidate := normalizedClientConfig(m.configForm.config())
	configChanged := candidate != m.config
	runtimeChanged := runtimeConfigChanged(m.config, candidate)
	sessionChanged := candidate.SessionDir != m.config.SessionDir

	nextBackend, err := m.backendForConfig(candidate, runtimeChanged)
	if err != nil {
		m.configForm.errorMessage = cleanFailureMessage(err, "Unable to apply config")
		return m, nil
	}

	if configChanged {
		if err := m.persistConfig(candidate); err != nil {
			if runtimeChanged {
				closeBackendTransport(nextBackend)
			}
			m.configForm.errorMessage = "Unable to save config file"
			return m, nil
		}
	}

	m.applyRuntimeConfig(candidate, nextBackend, runtimeChanged)
	if runtimeChanged {
		checkMode := currentUserCheckReconfigure
		if sessionChanged {
			checkMode = currentUserCheckRestore
		}
		return m, m.beginCurrentUserCheck(checkMode)
	}

	return m, nil
}

func normalizedClientConfig(candidate config.Config) config.Config {
	candidate.Address = strings.TrimSpace(candidate.Address)
	candidate.GRPCAddress = strings.TrimSpace(candidate.GRPCAddress)
	candidate.CACertFile = strings.TrimSpace(candidate.CACertFile)

	return candidate
}

func runtimeConfigChanged(previous, candidate config.Config) bool {
	if previous.Transport != candidate.Transport ||
		previous.CACertFile != candidate.CACertFile ||
		previous.SessionDir != candidate.SessionDir ||
		previous.CacheDir != candidate.CacheDir {
		return true
	}

	if candidate.Transport == config.TransportGRPC {
		return previous.GRPCAddress != candidate.GRPCAddress
	}

	return previous.Address != candidate.Address
}

func (m model) backendForConfig(candidate config.Config, runtimeChanged bool) (Backend, error) {
	if !runtimeChanged {
		return m.backend, nil
	}

	return createBackend(m.backendFactory, candidate)
}

func (m model) persistConfig(candidate config.Config) error {
	if m.configFile == "" || m.saveConfig == nil {
		return nil
	}

	return m.saveConfig(m.configFile, candidate)
}

func (m *model) applyRuntimeConfig(candidate config.Config, nextBackend Backend, runtimeChanged bool) {
	previous := m.config
	m.config = candidate

	if runtimeChanged {
		oldBackend := m.backend
		cacheChanged := candidate.CacheDir != previous.CacheDir

		m.cancelAllRequests()
		if cacheChanged || m.recordFeature.workspace.source == recordSourceServer {
			m.clearRecordState()
		}

		if cacheChanged {
			m.clearCacheState()
		}

		m.clearSyncState()

		m.backend = nextBackend
		m.statusState = serverStatusIdle
		m.statusValue = ""
		m.statusFailure = serverStatusFailure{}

		closeBackendTransport(oldBackend)
	}

	m.closeConfigForm()
}

func (m *model) closeConfigForm() {
	m.dialog = dialogNone
	m.configForm = newConfigForm(m.config)
}
