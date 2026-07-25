package tui

import (
	"strings"

	"charm.land/bubbles/v2/spinner"
	tea "charm.land/bubbletea/v2"
)

func (m model) updateWindowSize(msg tea.WindowSizeMsg) (tea.Model, tea.Cmd) {
	m.width = msg.Width
	m.height = msg.Height
	m.recordFeature.view.resize(m.width, m.height)

	if m.dialog == dialogPathPicker {
		m.pathPicker.resize(m.height)
		return m, nil
	}

	if m.recordFeature.workspace.open {
		m.recordFeature.workspace.ensureVisible(recordWorkspacePageSize(m.height))
	}

	return m, nil
}

func (m model) updateSpinner(msg spinner.TickMsg) (tea.Model, tea.Cmd) {
	if !m.spinnerPending() {
		return m, nil
	}

	var command tea.Cmd
	m.activitySpinner, command = m.activitySpinner.Update(msg)

	return m, command
}

func (m model) updateResultMessage(msg tea.Msg) (tea.Model, tea.Cmd, bool) {
	switch typed := msg.(type) {
	case currentUserResultMsg:
		updated, command := m.handleCurrentUserResult(typed)
		return updated, command, true
	case loginResultMsg:
		updated, command := m.handleLoginResult(typed)
		return updated, command, true
	case registerResultMsg:
		updated, command := m.handleRegisterResult(typed)
		return updated, command, true
	case logoutResultMsg:
		updated, command := m.handleLogoutResult(typed)
		return updated, command, true
	case recordListResultMsg:
		updated, command := m.handleRecordListResult(typed)
		return updated, command, true
	case cacheOpenResultMsg:
		updated, command := m.handleCacheOpenResult(typed)
		return updated, command, true
	case recordViewResultMsg:
		updated, command := m.handleRecordViewResult(typed)
		return updated, command, true
	case cachedRecordViewResultMsg:
		updated, command := m.handleCachedRecordViewResult(typed)
		return updated, command, true
	case binarySaveResultMsg:
		updated, command := m.handleBinarySaveResult(typed)
		return updated, command, true
	case recordCreateResultMsg:
		updated, command := m.handleRecordCreateResult(typed)
		return updated, command, true
	case recordEditLoadResultMsg:
		updated, command := m.handleRecordEditLoadResult(typed)
		return updated, command, true
	case recordEditResultMsg:
		updated, command := m.handleRecordEditResult(typed)
		return updated, command, true
	case recordDeleteResultMsg:
		updated, command := m.handleRecordDeleteResult(typed)
		return updated, command, true
	case syncResultMsg:
		updated, command := m.handleSyncResult(typed)
		return updated, command, true
	case serverStatusResultMsg:
		updated, command := m.handleServerStatusResult(typed)
		return updated, command, true
	case openURLResultMsg:
		if typed.err != nil {
			m.showAlert(alertError, "Unable to open link", cleanOpenURLError(typed.err), dialogAbout)
		}
		return m, nil, true
	default:
		return m, nil, false
	}
}

func (m model) handleCurrentUserResult(msg currentUserResultMsg) (tea.Model, tea.Cmd) {
	if !m.operations.accepts(operationCurrentUser, msg.requestID) {
		return m, nil
	}

	checkMode := m.authentication.currentUserCheck
	previousLogin := m.authentication.session.login
	m.operations.finish(operationCurrentUser)

	if msg.err != nil {
		return m.handleCurrentUserFailure(checkMode, msg.err)
	}
	return m.handleCurrentUserSuccess(checkMode, previousLogin, msg.login)
}

func (m model) handleCurrentUserSuccess(
	checkMode currentUserCheckMode,
	previousLogin,
	login string,
) (tea.Model, tea.Cmd) {
	m.authentication.session = authSession{state: authAuthenticated, login: login}
	if checkMode == currentUserCheckManual {
		if m.recordFeature.workspace.open && previousLogin != login {
			m.closeRecordWorkspace()
		}
		m.dialog = dialogCurrentUser
		m.activeButton = 0
		return m, nil
	}
	if m.dialog == dialogLogin {
		m.dialog = dialogNone
	}
	return m, m.beginOnlineRecordList()
}

func (m model) handleCurrentUserFailure(checkMode currentUserCheckMode, err error) (tea.Model, tea.Cmd) {
	if isNotLoggedIn(err) {
		if checkMode == currentUserCheckManual {
			m.handleSessionExpired()
			return m, nil
		}

		m.authentication.session = authSession{state: authGuest}
		if m.dialog == dialogNone {
			m.dialog = dialogLogin
		}

		return m, nil
	}

	if checkMode == currentUserCheckManual {
		m.showAlert(alertError, "Current user check failed", cleanCurrentUserError(err), dialogNone)
		return m, nil
	}

	m.authentication.session = authSession{state: authGuest}

	returnDialog := m.dialog
	if returnDialog == dialogNone {
		returnDialog = dialogLogin
	}

	m.showAlert(alertError, "Session check failed", cleanCurrentUserError(err), returnDialog)

	return m, nil
}

func (m model) handleLoginResult(msg loginResultMsg) (tea.Model, tea.Cmd) {
	if !m.operations.accepts(operationLogin, msg.requestID) {
		return m, nil
	}

	m.operations.finish(operationLogin)
	if msg.err != nil {
		m.showAlert(alertError, "Login failed", cleanLoginError(msg.err), dialogLogin)
		m.authentication.loginForm.focus = loginSubmit

		return m, nil
	}

	m.authentication.session = authSession{state: authAuthenticated, login: msg.login}
	m.authentication.loginForm = newLoginForm()
	m.dialog = dialogCurrentUser
	m.activeButton = 0

	return m, m.beginOnlineRecordList()
}

func (m model) handleRegisterResult(msg registerResultMsg) (tea.Model, tea.Cmd) {
	if !m.operations.accepts(operationRegister, msg.requestID) {
		return m, nil
	}

	m.operations.finish(operationRegister)
	if msg.err != nil {
		m.showAlertWithHighlight(
			alertError,
			"Registration failed",
			cleanRegisterError(msg.err),
			strings.TrimSpace(m.authentication.registerForm.login.value),
			dialogRegister,
		)

		m.authentication.registerForm.focus = registerSubmit

		return m, nil
	}

	m.authentication.loginForm = newLoginForm()
	m.authentication.loginForm.login.setValue(msg.login)
	m.authentication.loginForm.focus = loginPassword
	m.authentication.registerForm = newRegisterForm()
	m.dialog = dialogNone

	m.showAlertWithHighlight(
		alertNotice,
		"Registration successful",
		"Registered as "+msg.login+". Please log in.",
		msg.login,
		dialogLogin,
	)

	return m, nil
}

func (m model) handleLogoutResult(msg logoutResultMsg) (tea.Model, tea.Cmd) {
	if !m.operations.accepts(operationLogout, msg.requestID) {
		return m, nil
	}

	m.operations.finish(operationLogout)
	m.dialog = dialogNone

	if msg.err != nil {
		m.showAlert(alertError, "Logout failed", cleanLogoutError(msg.err), dialogNone)
		return m, nil
	}

	m.clearRecordState()
	m.clearCacheState()
	m.clearSyncState()
	m.authentication.session = authSession{state: authGuest}
	m.showAlert(alertNotice, "Logged out", "You are now logged out", dialogNone)

	return m, nil
}

func (m model) handleServerStatusResult(msg serverStatusResultMsg) (tea.Model, tea.Cmd) {
	if !m.operations.accepts(operationServerStatus, msg.requestID) {
		return m, nil
	}

	m.operations.finish(operationServerStatus)
	if msg.err != nil {
		m.statusState = serverStatusFailed
		m.statusValue = ""
		m.statusFailure = describeServerStatusError(msg.err)
		return m, nil
	}

	m.statusState = serverStatusReady
	m.statusValue = msg.status
	m.statusFailure = serverStatusFailure{}

	return m, nil
}

func (m model) updatePaste(msg tea.PasteMsg) (tea.Model, tea.Cmd) {
	if m.alert != alertNone || m.interactionBlocked() {
		return m, nil
	}

	switch {
	case m.dialog == dialogConfig && m.configForm.activeField() != nil:
		m.configForm.insert(msg.Content)
	case m.dialog == dialogLogin && !m.operations.pending(operationLogin) && m.authentication.loginForm.focus <= loginPassword:
		m.authentication.loginForm.insert(msg.Content)
	case m.dialog == dialogRegister && !m.operations.pending(operationRegister) && m.authentication.registerForm.focus <= registerRepeatPassword:
		m.authentication.registerForm.insert(msg.Content)
	case m.dialog == dialogRecordCreate && !m.operations.pending(operationCreateRecord):
		m.recordFeature.createForm.insert(msg.Content)
	case m.dialog == dialogRecordEdit && m.recordFeature.edit.status == recordEditReady && !m.operations.pending(operationEditRecord):
		m.recordFeature.edit.form.insert(msg.Content)
	case m.dialog == dialogCacheBrowse && !m.operations.pending(operationOpenCache):
		m.cacheFeature.form.insert(msg.Content)
	case m.dialog == dialogSync && !m.operations.pending(operationSync):
		m.syncFeature.form.insert(msg.Content)
	}

	return m, nil
}

func (m model) updateKeyPress(msg tea.KeyPressMsg) (tea.Model, tea.Cmd) {
	key := msg.String()

	if updated, command, handled := m.handleImmediateKey(key); handled {
		return updated, command
	}

	if updated, command, handled := m.handleAlertKey(key); handled {
		return updated, command
	}

	if updated, command, handled := m.handleMenuKey(key); handled {
		return updated, command
	}

	if updated, command, handled := m.handleDialogOrWorkspaceKey(msg, key); handled {
		return updated, command
	}

	return m.updateWindowKey(key)
}

func (m model) handleImmediateKey(key string) (tea.Model, tea.Cmd, bool) {
	if key == "ctrl+c" || key == "ctrl+q" {
		m.cancelAllRequests()
		return m, tea.Quit, true
	}

	if m.width < minimumWidth || m.height < minimumHeight || m.interactionBlocked() {
		return m, nil, true
	}

	return m, nil, false
}

func (m model) handleAlertKey(key string) (tea.Model, tea.Cmd, bool) {
	if m.alert == alertNone {
		return m, nil, false
	}

	if key == "enter" || key == "esc" {
		m.dismissAlert()
	}

	return m, nil, true
}

func (m model) handleMenuKey(key string) (tea.Model, tea.Cmd, bool) {
	definitions := m.currentMenuDefinitions()
	if index, ok := menuIndexByAltKey(definitions, key); ok {
		updated, command := m.openMenu(index)
		return updated, command, true
	}

	if key == "f10" {
		if m.menuFocused || m.dropdownOpen {
			m.closeMenu()
		} else {
			m.openCurrentMenu(definitions)
		}
		return m, nil, true
	}

	if m.menuFocused || m.dropdownOpen {
		updated, command := m.updateMenu(key)
		return updated, command, true
	}

	return m, nil, false
}

func (m model) handleDialogOrWorkspaceKey(
	msg tea.KeyPressMsg,
	key string,
) (tea.Model, tea.Cmd, bool) {
	if updated, command, handled := m.updateDialogKey(msg, key); handled {
		return updated, command, true
	}

	if m.dialog == dialogNone && m.recordFeature.workspace.open {
		updated, command := m.updateRecordWorkspace(key)
		return updated, command, true
	}

	return m, nil, false
}

func (m model) updateWindowKey(key string) (tea.Model, tea.Cmd) {
	switch key {
	case "tab", "shift+tab", "right", "left":
		m.moveDialogButton()
	case "enter":
		return m.activateDialogButton()
	case "esc":
		m.closeVisibleWindow()
	}

	return m, nil
}

func (m *model) moveDialogButton() {
	switch m.dialog {
	case dialogAbout:
		m.activeButton = 1 - m.activeButton
	case dialogServerStatus:
		m.moveServerStatusButton()
	}
}

func (m *model) closeVisibleWindow() {
	if m.dialog != dialogNone {
		m.closeActiveDialog()
		return
	}

	if m.recordFeature.workspace.open {
		m.closeRecordWorkspace()
	}
}

func (m model) updateDialogKey(msg tea.KeyPressMsg, key string) (tea.Model, tea.Cmd, bool) {
	switch m.dialog {
	case dialogConfig:
		updated, command := m.updateConfig(key)
		return updated, command, true
	case dialogPathPicker:
		updated, command := m.updatePathPicker(msg)
		return updated, command, true
	case dialogLogin:
		updated, command := m.updateLogin(key)
		return updated, command, true
	case dialogRegister:
		updated, command := m.updateRegister(key)
		return updated, command, true
	case dialogRecordView:
		updated, command := m.updateRecordView(key)
		return updated, command, true
	case dialogBinarySave:
		updated, command := m.updateBinarySave(key)
		return updated, command, true
	case dialogRecordType:
		updated, command := m.updateRecordTypePicker(key)
		return updated, command, true
	case dialogRecordCreate:
		updated, command := m.updateRecordCreate(key)
		return updated, command, true
	case dialogRecordEdit:
		updated, command := m.updateRecordEdit(key)
		return updated, command, true
	case dialogRecordDelete:
		updated, command := m.updateRecordDelete(key)
		return updated, command, true
	case dialogCacheBrowse:
		updated, command := m.updateCacheBrowse(key)
		return updated, command, true
	case dialogSync:
		updated, command := m.updateSync(key)
		return updated, command, true
	case dialogSyncResult:
		if key == "enter" || key == "esc" {
			m.closeSync()
		}
		return m, nil, true
	default:
		return m, nil, false
	}
}
