package tui

import (
	"time"

	tea "charm.land/bubbletea/v2"
	"github.com/xhrobj/gopherkeeper/internal/client/config"
)

func (m model) updateMouse(msg tea.MouseClickMsg) (tea.Model, tea.Cmd) {
	if msg.Button != tea.MouseLeft || m.width < minimumWidth || m.height < minimumHeight {
		return m, nil
	}

	if m.interactionBlocked() {
		return m, nil
	}

	if m.alert != alertNone {
		return m.updateAlertMouse(msg)
	}

	if updated, command, handled := m.updateMenuMouse(msg); handled {
		return updated, command
	}

	switch m.dialog {
	case dialogLogin:
		return m.updateLoginMouse(msg)
	case dialogRegister:
		return m.updateRegisterMouse(msg)
	case dialogPathPicker:
		return m.updatePathPickerMouse(msg)
	case dialogConfig:
		return m.updateConfigMouse(msg)
	case dialogCacheBrowse:
		return m.updateCacheBrowseMouse(msg)
	case dialogSync:
		return m.updateSyncMouse(msg)
	}

	if updated, command, handled := m.updateRecordMouse(msg); handled {
		return updated, command
	}

	return m.updateDialogButtonsMouse(msg)
}

func (m model) updateAlertMouse(msg tea.MouseClickMsg) (tea.Model, tea.Cmd) {
	buttons := m.dialogButtonBounds()

	if len(buttons) == 1 && buttons[0].contains(msg.X, msg.Y) {
		m.dismissAlert()
	}

	return m, nil
}

func (m *model) updateMenuMouse(msg tea.MouseClickMsg) (tea.Model, tea.Cmd, bool) {
	definitions := m.currentMenuDefinitions()

	if msg.Y == menuBarY {
		menuLayout := buildMenuBarLayout(
			m.theme,
			definitions,
			m.width,
			m.activeMenu,
			m.menuFocused || m.dropdownOpen,
			m.interactionBlocked(),
		)
		index, ok := menuIndexAtX(menuLayout.bounds, msg.X)
		if !ok || definitions[index].disabled {
			m.closeMenu()
			return *m, nil, true
		}
		if m.dropdownOpen && m.activeMenu == index {
			m.closeMenu()
			return *m, nil, true
		}
		updated, command := m.openMenu(index)
		return updated, command, true
	}

	if !m.dropdownOpen {
		return *m, nil, false
	}

	dropdownLayout := buildDropdownMenuLayout(m.width, m.activeMenu, definitions[m.activeMenu])

	if dropdownLayout.bounds.contains(msg.X, msg.Y) {
		selected, item, ok := dropdownLayout.itemAt(msg.X, msg.Y)
		if !ok || item.disabled {
			return *m, nil, true
		}
		m.selectedItem = selected
		updated, command := m.activate(item.action)
		return updated, command, true
	}

	m.closeMenu()

	return *m, nil, false
}

func (m model) updateLoginMouse(msg tea.MouseClickMsg) (tea.Model, tea.Cmd) {
	if !m.operations.pending(operationLogin) {
		for index, bounds := range m.loginFieldBounds() {
			if bounds.contains(msg.X, msg.Y) {
				m.authentication.loginForm.setFocus(loginFocus(index), false)
				return m, nil
			}
		}
	}

	for index, bounds := range m.dialogButtonBounds() {
		if !bounds.contains(msg.X, msg.Y) {
			continue
		}
		focus := loginFocus(int(loginSubmit) + index)
		submitDisabled := m.operations.pending(operationLogin) || !m.authentication.loginForm.canSubmit()
		if submitDisabled && focus == loginSubmit {
			return m, nil
		}
		m.authentication.loginForm.setFocus(focus, submitDisabled)
		return m.activateLogin()
	}

	return m, nil
}

func (m model) updateRegisterMouse(msg tea.MouseClickMsg) (tea.Model, tea.Cmd) {
	if !m.operations.pending(operationRegister) {
		for index, bounds := range m.registerFieldBounds() {
			if bounds.contains(msg.X, msg.Y) {
				m.authentication.registerForm.setFocus(registerFocus(index), !m.authentication.registerForm.canSubmit())
				return m, nil
			}
		}
	}

	for index, bounds := range m.dialogButtonBounds() {
		if !bounds.contains(msg.X, msg.Y) {
			continue
		}

		focus := registerFocus(int(registerSubmit) + index)
		submitDisabled := m.operations.pending(operationRegister) || !m.authentication.registerForm.canSubmit()

		if submitDisabled && focus == registerSubmit {
			return m, nil
		}

		m.authentication.registerForm.setFocus(focus, submitDisabled)

		return m.activateRegister()
	}

	return m, nil
}

func (m model) updateCacheBrowseMouse(msg tea.MouseClickMsg) (tea.Model, tea.Cmd) {
	if !m.operations.pending(operationOpenCache) {
		for index, bounds := range m.cacheBrowseFieldBounds() {
			if bounds.contains(msg.X, msg.Y) {
				m.cacheFeature.form.setFocus(cacheBrowseFocus(index), false)
				return m, nil
			}
		}
	}

	for index, bounds := range m.dialogButtonBounds() {
		if !bounds.contains(msg.X, msg.Y) {
			continue
		}
		focus := cacheBrowseFocus(int(cacheBrowseSubmit) + index)
		submitDisabled := !m.cacheFeature.form.canSubmit()
		if submitDisabled && focus == cacheBrowseSubmit {
			return m, nil
		}
		m.cacheFeature.form.setFocus(focus, submitDisabled)
		return m.activateCacheBrowse()
	}

	return m, nil
}

func (m model) updateSyncMouse(msg tea.MouseClickMsg) (tea.Model, tea.Cmd) {
	for _, bounds := range m.syncFieldBounds() {
		if bounds.contains(msg.X, msg.Y) {
			m.syncFeature.form.setFocus(syncPassword, false)
			return m, nil
		}
	}

	for index, bounds := range m.dialogButtonBounds() {
		if !bounds.contains(msg.X, msg.Y) {
			continue
		}
		focus := syncFocus(int(syncSubmit) + index)
		submitDisabled := !m.syncFeature.form.canSubmit()
		if submitDisabled && focus == syncSubmit {
			return m, nil
		}
		m.syncFeature.form.setFocus(focus, submitDisabled)
		return m.activateSync()
	}

	return m, nil
}

func (m model) updatePathPickerMouse(msg tea.MouseClickMsg) (tea.Model, tea.Cmd) {
	for index, bounds := range m.pathPickerButtonBounds() {
		if !bounds.contains(msg.X, msg.Y) {
			continue
		}

		m.pathPicker.clearMouseClick()

		if index == 0 {
			if !m.pathPicker.selectEnabled() {
				return m, nil
			}
			m.pathPicker.focus = pathPickerSelect
			path, _ := m.pathPicker.selectedPath()
			m.applyPathSelection(path)
			return m, nil

		}
		m.pathPicker.focus = pathPickerCancel
		m.closePathPicker()

		return m, nil
	}

	index, ok := m.pathPickerEntryAt(msg.X, msg.Y)

	if !ok {
		m.pathPicker.clearMouseClick()
		return m, nil
	}

	m.pathPicker.focusTree()
	doubleClick, command := m.pathPicker.registerMouseClick(index, time.Now())

	if !doubleClick {
		return m, command
	}

	entry, ok := m.pathPicker.highlighted()
	if !ok {
		return m, nil
	}

	if m.pathPicker.canSelect(entry) {
		path, _ := m.pathPicker.selectedPath()
		m.applyPathSelection(path)
		return m, nil
	}

	if entry.directory {
		return m, m.pathPicker.openHighlightedDirectory()
	}

	return m, nil
}

func (m model) updateConfigMouse(msg tea.MouseClickMsg) (tea.Model, tea.Cmd) {
	if updated, command, ok := m.selectConfigTransportMouse(msg); ok {
		return updated, command
	}
	if updated, command, ok := m.focusConfigFieldMouse(msg); ok {
		return updated, command
	}
	if updated, command, ok := m.openConfigBrowseMouse(msg); ok {
		return updated, command
	}
	if updated, command, ok := m.activateConfigButtonMouse(msg); ok {
		return updated, command
	}

	return m, nil
}

func (m model) selectConfigTransportMouse(msg tea.MouseClickMsg) (tea.Model, tea.Cmd, bool) {
	for index, bounds := range m.configTransportBounds() {
		if !bounds.contains(msg.X, msg.Y) {
			continue
		}

		m.configForm.setFocus(configTransport)
		m.configForm.selectTransport(configTransportAt(index))

		return m, nil, true
	}

	return m, nil, false
}

func configTransportAt(index int) config.Transport {
	if index == 0 {
		return config.TransportHTTPS
	}
	return config.TransportGRPC
}

func (m model) focusConfigFieldMouse(msg tea.MouseClickMsg) (tea.Model, tea.Cmd, bool) {
	layout := m.configLayout()
	for index, bounds := range m.configFieldBounds() {
		if !bounds.contains(msg.X, msg.Y) {
			continue
		}

		m.configForm.setFocus(layout.fieldFocus[index])

		return m, nil, true
	}

	return m, nil, false
}

func (m model) openConfigBrowseMouse(msg tea.MouseClickMsg) (tea.Model, tea.Cmd, bool) {
	browseFocus := []configFocus{configCACertBrowse, configSessionBrowse, configCacheBrowse}
	for index, bounds := range m.configBrowseButtonBounds() {
		if !bounds.contains(msg.X, msg.Y) {
			continue
		}

		m.configForm.setFocus(browseFocus[index])
		target, _ := configBrowseTarget(browseFocus[index])
		updated, command := m.openConfigPathPicker(target)

		return updated, command, true
	}

	return m, nil, false
}

func (m model) activateConfigButtonMouse(msg tea.MouseClickMsg) (tea.Model, tea.Cmd, bool) {
	for index, bounds := range m.dialogButtonBounds() {
		if !bounds.contains(msg.X, msg.Y) {
			continue
		}

		if index == 0 && !m.configForm.canSave() {
			return m, nil, true
		}

		m.configForm.setFocus(configFocus(int(configSave) + index))
		updated, command := m.activateConfig()

		return updated, command, true
	}

	return m, nil, false
}

func (m model) updateDialogButtonsMouse(msg tea.MouseClickMsg) (tea.Model, tea.Cmd) {
	for index, bounds := range m.dialogButtonBounds() {
		if !bounds.contains(msg.X, msg.Y) {
			continue
		}

		m.activeButton = index

		return m.activateDialogButton()
	}

	return m, nil
}

func (m *model) closeMenu() {
	m.menuFocused = false
	m.dropdownOpen = false
	m.selectedItem = 0
}

func (m model) dialogButtonBounds() []layoutBounds {
	if m.alert != alertNone {
		layout := m.alertWindowLayout()

		window, ok := centeredWindowPlacement(layout.content, m.width, m.height)
		if !ok {
			return nil
		}

		return window.screenBounds(layout.buttonBounds)
	}

	window, ok := m.dialogPlacement()
	if !ok {
		return nil
	}

	contentWidth := max(1, window.width-4)
	blocked := m.interactionBlocked()
	var layout buttonRowLayout

	switch m.dialog {
	case dialogLogin:
		loginLayout := m.loginWindowLayout()
		return window.screenBounds(loginLayout.buttonBounds)
	case dialogRecordView:
		viewLayout := newRecordViewWindowLayout(window.width, window.height)
		layout = recordViewButtonsLayout(
			m.theme,
			contentWidth,
			m.recordFeature.view,
			m.activeButton,
			blocked,
		).positioned(2, viewLayout.buttonRow)
	case dialogBinarySave:
		binarySaveLayout := newBinarySaveWindowLayout(
			m.theme,
			window.width,
			m.recordFeature.binarySaveForm,
			m.operations.pending(operationBinarySave),
		)
		return window.screenBounds(binarySaveLayout.buttonBounds)
	case dialogRecordDelete:
		recordDeleteLayout := newRecordDeleteWindowLayout(
			m.theme,
			window.width,
			m.recordFeature.deletion,
			m.operations.pending(operationDeleteRecord),
			blocked,
			m.spinnerFrameValue(),
			m.activeButton,
		)
		return window.screenBounds(recordDeleteLayout.buttonBounds)
	case dialogRegister:
		registerLayout := m.registerWindowLayout()
		return window.screenBounds(registerLayout.buttonBounds)
	case dialogCurrentUser:
		currentUserLayout := newCurrentUserWindowLayout(
			m.theme,
			window.width,
			m.authentication.session.login,
			m.operations.pending(operationCurrentUser),
			blocked,
			m.spinnerFrameValue(),
		)
		return window.screenBounds(currentUserLayout.buttonBounds)
	case dialogAbout:
		aboutLayout := m.aboutWindowLayout()
		return window.screenBounds(aboutLayout.buttonBounds)
	case dialogPathPicker:
		pathPickerLayout := newPathPickerWindowLayout(m.theme, window.width, m.pathPicker)
		return window.screenBounds(pathPickerLayout.buttonBounds)
	case dialogConfig:
		return window.screenBounds(m.configLayout().buttonBounds)
	case dialogControls:
		controlsLayout := newControlsWindowLayout(m.theme, window.width)
		return window.screenBounds(controlsLayout.buttonBounds)
	case dialogCacheBrowse:
		cacheLayout := newCacheBrowseWindowLayout(
			m.theme,
			window.width,
			m.cacheFeature.form,
			m.operations.pending(operationOpenCache),
			blocked,
			m.spinnerFrameValue(),
		)
		return window.screenBounds(cacheLayout.buttonBounds)
	case dialogSync:
		syncLayout := newSyncWindowLayout(
			m.theme,
			window.width,
			m.authentication.session.login,
			m.syncFeature.form,
			m.operations.pending(operationSync),
			blocked,
			m.spinnerFrameValue(),
		)
		return window.screenBounds(syncLayout.buttonBounds)
	case dialogSyncResult:
		syncResultLayout := newSyncResultWindowLayout(
			m.theme,
			window.width,
			m.syncFeature.result,
			blocked,
		)
		return window.screenBounds(syncResultLayout.buttonBounds)
	case dialogServerStatus:
		statusLayout := m.serverStatusWindowLayout()
		return window.screenBounds(statusLayout.buttonBounds)
	default:
		return nil
	}

	return window.screenBounds(layout.bounds)
}

func (m model) loginFieldBounds() []layoutBounds {
	layout := m.loginWindowLayout()
	window, ok := centeredWindowPlacement(layout.content, m.width, m.height)
	if !ok {
		return nil
	}

	return window.screenBounds(layout.fieldBounds)
}

func (m model) cacheBrowseFieldBounds() []layoutBounds {
	if m.dialog != dialogCacheBrowse {
		return nil
	}

	layout := newCacheBrowseWindowLayout(
		m.theme,
		cacheBrowseWindowWidth(m.width),
		m.cacheFeature.form,
		m.operations.pending(operationOpenCache),
		m.interactionBlocked(),
		m.spinnerFrameValue(),
	)

	window, ok := centeredWindowPlacement(layout.content, m.width, m.height)
	if !ok {
		return nil
	}

	return window.screenBounds(layout.fieldBounds)
}

func (m model) syncFieldBounds() []layoutBounds {
	if m.dialog != dialogSync {
		return nil
	}

	layout := newSyncWindowLayout(
		m.theme,
		syncWindowWidth(m.width),
		m.authentication.session.login,
		m.syncFeature.form,
		m.operations.pending(operationSync),
		m.interactionBlocked(),
		m.spinnerFrameValue(),
	)

	window, ok := centeredWindowPlacement(layout.content, m.width, m.height)
	if !ok {
		return nil
	}

	return window.screenBounds(layout.fieldBounds)
}

func (m model) registerFieldBounds() []layoutBounds {
	layout := m.registerWindowLayout()
	window, ok := centeredWindowPlacement(layout.content, m.width, m.height)
	if !ok {
		return nil
	}

	return window.screenBounds(layout.fieldBounds)
}

func (m model) configLayout() configWindowLayout {
	if m.dialog != dialogConfig {
		return configWindowLayout{}
	}

	return m.configWindowLayout()
}

func (m model) configLayoutPlacement() (configWindowLayout, windowPlacement, bool) {
	layout := m.configLayout()
	window, ok := centeredWindowPlacement(layout.content, m.width, m.height)

	return layout, window, ok
}

func (m model) configTransportBounds() []layoutBounds {
	layout, window, ok := m.configLayoutPlacement()
	if !ok {
		return nil
	}

	return window.screenBounds(layout.transportBounds)
}

func (m model) configFieldBounds() []layoutBounds {
	layout, window, ok := m.configLayoutPlacement()
	if !ok {
		return nil
	}

	return window.screenBounds(layout.fieldBounds)
}

func (m model) configBrowseButtonBounds() []layoutBounds {
	layout, window, ok := m.configLayoutPlacement()
	if !ok {
		return nil
	}

	return window.screenBounds(layout.browseBounds)
}

func (m model) pathPickerButtonBounds() []layoutBounds {
	if m.dialog != dialogPathPicker {
		return nil
	}

	return m.dialogButtonBounds()
}

func (m model) pathPickerEntryAt(x, y int) (int, bool) {
	window, ok := m.dialogPlacement()
	if !ok || m.dialog != dialogPathPicker {
		return 0, false
	}

	listBounds := newPathPickerWindowLayout(m.theme, window.width, m.pathPicker).listBounds.
		translated(window.x, window.y)
	if !listBounds.contains(x, y) {
		return 0, false
	}

	index := m.pathPicker.offset + y - listBounds.y
	if index < 0 || index >= len(m.pathPicker.entries) {
		return 0, false
	}

	return index, true
}

func (m model) binarySaveFieldBounds() []layoutBounds {
	if m.dialog != dialogBinarySave {
		return nil
	}

	layout := newBinarySaveWindowLayout(
		m.theme,
		binarySaveWindowWidth(m.width),
		m.recordFeature.binarySaveForm,
		m.operations.pending(operationBinarySave),
	)

	window, ok := centeredWindowPlacement(layout.content, m.width, m.height)
	if !ok {
		return nil
	}

	return []layoutBounds{layout.browseBounds.translated(window.x, window.y)}
}

func (m model) updateMouseWheel(msg tea.MouseWheelMsg) (tea.Model, tea.Cmd) {
	if m.interactionBlocked() || m.alert != alertNone || m.menuFocused || m.dropdownOpen || m.dialog != dialogRecordView {
		return m, nil
	}

	mouse := msg.Mouse()
	bounds, ok := m.recordViewTextAreaBounds()

	if !ok || !bounds.contains(mouse.X, mouse.Y) {
		return m, nil
	}

	switch mouse.Button {
	case tea.MouseWheelUp:
		m.recordFeature.view.textArea.scroll(-readOnlyTextAreaWheelStep)
	case tea.MouseWheelDown:
		m.recordFeature.view.textArea.scroll(readOnlyTextAreaWheelStep)
	}

	return m, nil
}
