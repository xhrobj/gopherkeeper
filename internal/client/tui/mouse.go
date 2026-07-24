package tui

import (
	"time"

	tea "charm.land/bubbletea/v2"
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
	} else {
		m = updated.(model)
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

func (m model) updateMenuMouse(msg tea.MouseClickMsg) (tea.Model, tea.Cmd, bool) {
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
			return m, nil, true
		}
		if m.dropdownOpen && m.activeMenu == index {
			m.closeMenu()
			return m, nil, true
		}
		updated, command := m.openMenu(index)
		return updated, command, true
	}

	if !m.dropdownOpen {
		return m, nil, false
	}

	dropdownLayout := buildDropdownMenuLayout(m.width, m.activeMenu, definitions[m.activeMenu])

	if dropdownLayout.bounds.contains(msg.X, msg.Y) {
		selected, item, ok := dropdownLayout.itemAt(msg.X, msg.Y)
		if !ok || item.disabled {
			return m, nil, true
		}
		m.selectedItem = selected
		updated, command := m.activate(item.action)
		return updated, command, true
	}

	m.closeMenu()

	return m, nil, false
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
	for index, bounds := range m.configFieldBounds() {
		if bounds.contains(msg.X, msg.Y) {
			m.configForm.setFocus(configFieldFocus(index))
			return m, nil
		}
	}

	browseFocus := []configFocus{configCACertBrowse, configSessionBrowse, configCacheBrowse}
	for index, bounds := range m.configBrowseButtonBounds() {
		if bounds.contains(msg.X, msg.Y) {
			m.configForm.setFocus(browseFocus[index])
			target, _ := configBrowseTarget(browseFocus[index])
			return m.openConfigPathPicker(target)
		}
	}

	for index, bounds := range m.dialogButtonBounds() {
		if !bounds.contains(msg.X, msg.Y) {
			continue
		}
		if index == 0 && !m.configForm.canSave() {
			return m, nil
		}
		m.configForm.setFocus(configFocus(int(configSave) + index))
		return m.activateConfig()
	}

	return m, nil
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
		window, ok := m.alertPlacement()
		if !ok {
			return nil
		}
		bodyStyle := m.theme.aboutBody
		buttonStyle := m.theme.aboutButtonActive
		if m.alert == alertError {
			bodyStyle = m.theme.errorBody
			buttonStyle = m.theme.errorButton
		}
		layout := singleStyledButtonLayout(bodyStyle, buttonStyle, window.width-4, "< OK >").
			positioned(2, alertButtonRow(m.alertMessage, m.alertHighlight))
		return window.screenBounds(layout.bounds)
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
		layout = loginButtonsLayout(
			m.theme,
			contentWidth,
			m.authentication.loginForm.focus,
			!m.authentication.loginForm.canSubmit(),
			blocked,
		).positioned(2, loginButtonRow)
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
		layout = binarySaveButtonsLayout(
			m.theme,
			contentWidth,
			m.recordFeature.binarySaveForm.focus,
			m.operations.pending(operationBinarySave) || !m.recordFeature.binarySaveForm.canSubmit(),
			m.operations.pending(operationBinarySave),
		).positioned(2, binarySaveButtonRow)
	case dialogRecordDelete:
		layout = recordDeleteButtonsLayout(
			m.theme,
			contentWidth,
			blocked,
			m.activeButton,
		).positioned(2, recordDeleteButtonRow)
	case dialogRegister:
		layout = registerButtonsLayout(
			m.theme,
			contentWidth,
			m.authentication.registerForm.focus,
			!m.authentication.registerForm.canSubmit(),
			blocked,
		).positioned(2, registerButtonRow)
	case dialogCurrentUser:
		style := m.theme.aboutButtonActive
		if blocked {
			style = m.theme.aboutButtonDisabledActive
		}
		layout = singleStyledButtonLayout(m.theme.aboutBody, style, contentWidth, "< OK >").
			positioned(2, currentUserButtonRow)
	case dialogAbout:
		layout = aboutButtonsLayout(m.theme, window.width, m.activeButton).
			positioned(0, aboutButtonRow)
	case dialogPathPicker:
		layout = pathPickerButtonsLayout(m.theme, contentWidth, m.pathPicker).
			positioned(2, pathPickerButtonRow(m.pathPicker.height))
	case dialogConfig:
		layout = configButtonsLayout(m.theme, contentWidth, m.configForm.focus, m.configForm.canSave()).
			positioned(2, configButtonRow)
	case dialogControls:
		layout = controlsButtonLayout(m.theme, contentWidth, "< OK >").
			positioned(2, controlsButtonRow)
	case dialogServerStatus:
		layout = serverStatusButtonsLayout(
			m.theme,
			contentWidth,
			m.statusState,
			m.operations.pending(operationServerStatus),
			m.activeButton,
			blocked,
		).positioned(2, serverStatusButtonRow)
	default:
		return nil
	}

	return window.screenBounds(layout.bounds)
}

func (m model) loginFieldBounds() []layoutBounds {
	window, ok := m.dialogPlacement()
	if !ok {
		return nil
	}

	layout := newLabeledFieldColumnLayout(
		window.width,
		loginLabelWidth,
		16,
		2,
		loginFirstFieldRow,
		loginFieldRowStep,
	)

	return window.screenBounds(layout.bounds)
}

func (m model) registerFieldBounds() []layoutBounds {
	window, ok := m.dialogPlacement()
	if !ok {
		return nil
	}

	layout := newLabeledFieldColumnLayout(
		window.width,
		registerLabelWidth,
		16,
		3,
		registerFirstFieldRow,
		registerFieldRowStep,
	)

	return window.screenBounds(layout.bounds)
}

func (m model) configFieldBounds() []layoutBounds {
	window, ok := m.dialogPlacement()
	if !ok {
		return nil
	}

	return window.screenBounds(newConfigWindowLayout(m.theme, window.width).fieldBounds)
}

func (m model) configBrowseButtonBounds() []layoutBounds {
	window, ok := m.dialogPlacement()
	if !ok {
		return nil
	}

	return window.screenBounds(newConfigWindowLayout(m.theme, window.width).browseBounds)
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

	listBounds := newPathPickerWindowLayout(window.width, m.pathPicker.height).listBounds.
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
	window, ok := m.dialogPlacement()
	if !ok {
		return nil
	}

	bounds := newBinarySaveWindowLayout(m.theme, window.width).browseBounds

	return []layoutBounds{bounds.translated(window.x, window.y)}
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
