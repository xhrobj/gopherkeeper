package tui

import tea "charm.land/bubbletea/v2"

func (m model) activateDialogButton() (tea.Model, tea.Cmd) {
	switch m.dialog {
	case dialogLogin:
		return m.activateLogin()
	case dialogRegister:
		return m.activateRegister()
	case dialogCurrentUser:
		m.dialog = dialogNone
	case dialogRecordView:
		if m.activeButton == 0 && recordViewIsBinary(m.recordFeature.view.record) {
			m.openBinarySave()
		} else if m.activeButton == 0 && recordViewHasSensitiveFields(m.recordFeature.view.record) {
			m.recordFeature.view.revealed = !m.recordFeature.view.revealed
		} else {
			m.closeRecordView()
		}
	case dialogBinarySave:
		return m.activateBinarySave()
	case dialogRecordDelete:
		return m.activateRecordDelete()
	case dialogAbout:
		if m.activeButton == 0 {
			return m, openURLCommand(m.openURL, aboutURL)
		}
		m.dialog = dialogNone
		m.activeButton = 0
	case dialogConfig:
		return m.activateConfig()
	case dialogControls:
		m.dialog = dialogNone
		m.activeButton = 0
	case dialogServerStatus:
		if m.activeButton == 0 {
			return m.startServerStatusCheck()
		}
		m.operations.cancel(operationServerStatus)
		m.dialog = dialogNone
		m.activeButton = 0
	}

	return m, nil
}

func (m model) updateMenu(key string) (tea.Model, tea.Cmd) {
	definitions := m.currentMenuDefinitions()

	if value := []rune(key); len(value) == 1 {
		if m.dropdownOpen {
			if selected, item, ok := menuItemByMnemonic(definitions[m.activeMenu], value[0]); ok {
				m.selectedItem = selected
				return m.activate(item.action)
			}
		}
		if index, ok := menuIndexByMnemonic(definitions, value[0]); ok {
			return m.openMenu(index)
		}
	}

	switch key {
	case "esc":
		m.closeMenu()
	case "left":
		m.activeMenu = nextEnabledMenuIndex(definitions, m.activeMenu, -1)
		m.dropdownOpen = true
		m.selectedItem = selectedItemForMenu(definitions[m.activeMenu], m.currentAction())
	case "right":
		m.activeMenu = nextEnabledMenuIndex(definitions, m.activeMenu, 1)
		m.dropdownOpen = true
		m.selectedItem = selectedItemForMenu(definitions[m.activeMenu], m.currentAction())
	case "down":
		if !m.dropdownOpen {
			m.dropdownOpen = true
			m.selectedItem = selectedItemForMenu(definitions[m.activeMenu], m.currentAction())
		} else {
			m.selectedItem = nextEnabledIndex(definitions[m.activeMenu], m.selectedItem, 1)
		}
	case "up":
		if m.dropdownOpen {
			m.selectedItem = nextEnabledIndex(definitions[m.activeMenu], m.selectedItem, -1)
		}
	case "enter":
		if !m.dropdownOpen {
			m.dropdownOpen = true
			m.selectedItem = selectedItemForMenu(definitions[m.activeMenu], m.currentAction())
			break
		}
		items := selectableItems(definitions[m.activeMenu])
		if m.selectedItem < 0 || m.selectedItem >= len(items) || items[m.selectedItem].disabled {
			break
		}
		return m.activate(items[m.selectedItem].action)
	}

	return m, nil
}

func (m model) openMenu(index int) (tea.Model, tea.Cmd) {
	definitions := m.currentMenuDefinitions()

	if index < 0 || index >= len(definitions) || definitions[index].disabled {
		return m, nil
	}

	m.activeMenu = index
	m.menuFocused = true
	m.dropdownOpen = true
	m.selectedItem = selectedItemForMenu(definitions[index], m.currentAction())

	return m, nil
}

func (m *model) openCurrentMenu(definitions []menuDefinition) {
	current := m.currentAction()

	if menuIndex, itemIndex, ok := menuSelectionByAction(definitions, current); ok {
		m.activeMenu = menuIndex
		m.selectedItem = itemIndex
	} else {
		m.activeMenu = firstEnabledMenuIndex(definitions)
		m.selectedItem = firstEnabledIndex(definitions[m.activeMenu])
	}

	m.menuFocused = true
	m.dropdownOpen = true
}

func (m model) activate(action actionID) (tea.Model, tea.Cmd) {
	m.closeMenu()

	if action == m.currentAction() && action != actionBrowseRecords {
		return m, nil
	}
	viewTarget, hasViewTarget := m.recordViewTarget()
	if action == actionViewRecord && !hasViewTarget {
		return m, nil
	}
	deleteTarget, hasDeleteTarget := m.recordDeleteTarget()
	if action == actionDeleteRecord && !hasDeleteTarget {
		return m, nil
	}
	m.prepareDialogChange(action)

	switch action {
	case actionQuit:
		m.cancelAllRequests()
		return m, tea.Quit
	case actionLogin:
		m.cancelSessionCheckForManualAuth()
		m.dialog = dialogLogin
		m.authentication.loginForm = newLoginForm()
	case actionRegister:
		m.cancelSessionCheckForManualAuth()
		m.dialog = dialogRegister
		m.authentication.registerForm = newRegisterForm()
	case actionCurrentUser:
		if m.authentication.session.authenticated() {
			m.dialog = dialogCurrentUser
			m.activeButton = 0
			return m, m.beginCurrentUserCheck(currentUserCheckManual)
		}
	case actionLogout:
		if m.authentication.session.authenticated() {
			return m.startLogout()
		}
	case actionBrowseRecords:
		if m.authentication.session.authenticated() && m.recordsAvailable() {
			m.dialog = dialogNone
			return m, m.beginOnlineRecordList()
		}
	case actionNewRecord:
		if m.authentication.session.authenticated() && m.recordCreateAvailable() {
			return m, m.openRecordTypePicker()
		}
	case actionViewRecord:
		if hasViewTarget && m.authentication.session.authenticated() && m.recordsAvailable() {
			return m, m.beginRecordView(viewTarget)
		}
	case actionEditRecord:
		if m.authentication.session.authenticated() && m.recordEditAvailable() {
			return m, m.openRecordEdit()
		}
	case actionDeleteRecord:
		if hasDeleteTarget && m.authentication.session.authenticated() && m.recordDeleteAvailable() {
			m.openRecordDelete(deleteTarget)
		}
	case actionControls:
		m.dialog = dialogControls
	case actionAbout:
		m.dialog = dialogAbout
		m.activeButton = 1
	case actionServerStatus:
		m.dialog = dialogServerStatus
		m.activeButton = 0
		return m.startServerStatusCheck()
	case actionConfig:
		m.dialog = dialogConfig
		m.configForm = newConfigForm(m.config)
	case actionCloseWindow:
		if m.dialog != dialogNone {
			m.closeActiveDialog()
		} else {
			m.closeRecordWorkspace()
		}
	}

	return m, nil
}

func (m *model) closeActiveDialog() {
	switch m.dialog {
	case dialogLogin:
		m.clearLoginForm()
	case dialogRegister:
		m.clearRegisterForm()
	case dialogServerStatus:
		m.operations.cancel(operationServerStatus)
	case dialogPathPicker:
		m.closePathPicker()
		return
	case dialogRecordView:
		m.leaveRecordView()
	case dialogBinarySave:
		m.closeBinarySave()
		return
	case dialogRecordType, dialogRecordCreate:
		m.closeRecordCreate()
		return
	case dialogRecordEdit:
		m.closeRecordEdit()
		return
	case dialogRecordDelete:
		m.closeRecordDelete()
		return
	}
	m.dialog = dialogNone
	m.activeButton = 0
}

func (m *model) prepareDialogChange(action actionID) {
	if action == actionCloseWindow {
		return
	}

	if m.dialog == dialogLogin && action != actionLogin {
		m.clearLoginForm()
	}
	if m.dialog == dialogRegister && action != actionRegister {
		m.clearRegisterForm()
	}
	if m.dialog == dialogServerStatus && action != actionServerStatus {
		m.operations.cancel(operationServerStatus)
	}
	binarySavePathPicker := m.dialog == dialogPathPicker &&
		m.pathPicker.target == pathPickerBinarySaveDirectory
	binaryCreatePathPicker := m.dialog == dialogPathPicker &&
		m.pathPicker.target == pathPickerBinaryCreateFile
	binaryEditPathPicker := m.dialog == dialogPathPicker &&
		m.pathPicker.target == pathPickerBinaryEditFile
	pathPickerAction := actionConfig
	switch {
	case binarySavePathPicker:
		pathPickerAction = actionBrowseRecords
	case binaryCreatePathPicker:
		pathPickerAction = actionNewRecord
	case binaryEditPathPicker:
		pathPickerAction = actionEditRecord
	}
	if m.dialog == dialogPathPicker && action != pathPickerAction {
		m.pathPicker = pathPicker{}
	}
	if (m.dialog == dialogRecordView || m.dialog == dialogBinarySave || binarySavePathPicker) && action != actionEditRecord {
		m.leaveRecordView()
	}
	if (m.dialog == dialogRecordType || m.dialog == dialogRecordCreate || binaryCreatePathPicker) && action != actionNewRecord {
		m.closeRecordCreate()
	}
	if (m.dialog == dialogRecordEdit || binaryEditPathPicker) && action != actionEditRecord {
		m.closeRecordEdit()
	}
	if m.dialog == dialogRecordDelete && action != actionDeleteRecord {
		m.closeRecordDelete()
	}
}

func (m *model) cancelSessionCheckForManualAuth() {
	m.operations.cancel(operationCurrentUser)
	if m.authentication.session.state == authUnknown {
		m.authentication.session = authSession{state: authGuest}
	}
}

func firstEnabledIndex(definition menuDefinition) int {
	items := selectableItems(definition)
	for index, item := range items {
		if !item.disabled {
			return index
		}
	}
	return 0
}

func nextEnabledIndex(definition menuDefinition, current, step int) int {
	items := selectableItems(definition)
	if len(items) == 0 {
		return 0
	}
	for range len(items) {
		current = (current + step + len(items)) % len(items)
		if !items[current].disabled {
			return current
		}
	}
	return current
}

func currentDialogAction(dialog dialogID) actionID {
	switch dialog {
	case dialogLogin:
		return actionLogin
	case dialogRegister:
		return actionRegister
	case dialogCurrentUser:
		return actionCurrentUser
	case dialogControls:
		return actionControls
	case dialogAbout:
		return actionAbout
	case dialogServerStatus:
		return actionServerStatus
	case dialogConfig, dialogPathPicker:
		return actionConfig
	case dialogRecordView, dialogBinarySave:
		return actionBrowseRecords
	case dialogRecordType, dialogRecordCreate:
		return actionNewRecord
	case dialogRecordEdit:
		return actionEditRecord
	case dialogRecordDelete:
		return actionDeleteRecord
	default:
		return actionNone
	}
}

func menuSelectionByAction(
	definitions []menuDefinition,
	action actionID,
) (menuIndex int, itemIndex int, ok bool) {
	if action == actionNone {
		return 0, 0, false
	}

	for menuIndex, definition := range definitions {
		selectable := -1
		for _, item := range definition.items {
			if item.separator {
				continue
			}
			selectable++
			if item.action == action && !item.disabled && !definition.disabled {
				return menuIndex, selectable, true
			}
		}
	}

	return 0, 0, false
}

func selectedItemForMenu(definition menuDefinition, action actionID) int {
	selectable := -1

	for _, item := range definition.items {
		if item.separator {
			continue
		}
		selectable++
		if item.action == action && !item.disabled {
			return selectable
		}
	}

	return firstEnabledIndex(definition)
}
