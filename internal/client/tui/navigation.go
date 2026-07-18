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
			if m.statusState == serverStatusChecking {
				return m, nil
			}
			return m.startServerStatusCheck()
		}
		m.cancelStatusRequest()
		m.dialog = dialogNone
		m.activeButton = 0
	}

	return m, nil
}

func (m model) updateMenu(key string) (tea.Model, tea.Cmd) {
	definitions := menuDefinitions(m.dialog, m.auth.authenticated())

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
		m.selectedItem = selectedItemForMenu(definitions[m.activeMenu], currentDialogAction(m.dialog))
	case "right":
		m.activeMenu = nextEnabledMenuIndex(definitions, m.activeMenu, 1)
		m.dropdownOpen = true
		m.selectedItem = selectedItemForMenu(definitions[m.activeMenu], currentDialogAction(m.dialog))
	case "down":
		if !m.dropdownOpen {
			m.dropdownOpen = true
			m.selectedItem = selectedItemForMenu(definitions[m.activeMenu], currentDialogAction(m.dialog))
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
			m.selectedItem = selectedItemForMenu(definitions[m.activeMenu], currentDialogAction(m.dialog))
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
	definitions := menuDefinitions(m.dialog, m.auth.authenticated())

	if index < 0 || index >= len(definitions) || definitions[index].disabled {
		return m, nil
	}

	m.activeMenu = index
	m.menuFocused = true
	m.dropdownOpen = true
	m.selectedItem = selectedItemForMenu(definitions[index], currentDialogAction(m.dialog))

	return m, nil
}

func (m *model) openCurrentMenu(definitions []menuDefinition) {
	current := currentDialogAction(m.dialog)

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

	if action == currentDialogAction(m.dialog) {
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
		m.loginForm = newLoginForm()
	case actionRegister:
		m.cancelSessionCheckForManualAuth()
		m.dialog = dialogRegister
		m.registerForm = newRegisterForm()
	case actionCurrentUser:
		if m.auth.authenticated() {
			m.dialog = dialogCurrentUser
			m.activeButton = 0
		}
	case actionLogout:
		if m.auth.authenticated() {
			return m.startLogout()
		}
	case actionControls:
		m.dialog = dialogControls
	case actionAbout:
		m.dialog = dialogAbout
		m.activeButton = 1
	case actionServerStatus:
		m.dialog = dialogServerStatus
		m.activeButton = 1
		return m.startServerStatusCheck()
	case actionConfig:
		m.dialog = dialogConfig
		m.configForm = newConfigForm(m.config)
	case actionCloseWindow:
		m.closeActiveDialog()
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
		m.cancelStatusRequest()
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
		m.cancelStatusRequest()
	}
}

func (m *model) cancelSessionCheckForManualAuth() {
	m.cancelAuthRequest()
	if m.auth.state == authUnknown {
		m.auth = authSession{state: authGuest}
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
	case dialogConfig:
		return actionConfig
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
