package tui

import (
	tea "charm.land/bubbletea/v2"
	recordmodel "github.com/xhrobj/gopherkeeper/internal/model"
)

type actionTargets struct {
	viewID        string
	hasView       bool
	cachedViewID  string
	hasCachedView bool
	deleteRecord  recordmodel.RecordMetadata
	hasDelete     bool
}

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
	case dialogSync:
		return m.activateSync()
	case dialogSyncResult:
		m.closeSync()
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
	if updated, command, handled := m.activateMenuMnemonic(definitions, key); handled {
		return updated, command
	}

	return m.updateMenuKey(definitions, key)
}

func (m model) activateMenuMnemonic(
	definitions []menuDefinition,
	key string,
) (tea.Model, tea.Cmd, bool) {
	value := []rune(key)
	if len(value) != 1 {
		return m, nil, false
	}

	if m.dropdownOpen {
		if selected, item, ok := menuItemByMnemonic(definitions[m.activeMenu], value[0]); ok {
			m.selectedItem = selected
			updated, command := m.activate(item.action)
			return updated, command, true
		}
	}

	index, ok := menuIndexByMnemonic(definitions, value[0])
	if !ok {
		return m, nil, false
	}
	updated, command := m.openMenu(index)
	return updated, command, true
}

func (m model) updateMenuKey(definitions []menuDefinition, key string) (tea.Model, tea.Cmd) {
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

	if action == m.currentAction() && action != actionBrowseRecords && action != actionBrowseCache {
		return m, nil
	}

	targets := m.captureActionTargets()
	if !targets.supports(action) {
		return m, nil
	}

	m.prepareDialogChange(action)

	switch action {
	case actionQuit, actionServerStatus, actionConfig:
		return m.activateSystemAction(action)
	case actionLogin, actionRegister, actionCurrentUser, actionLogout:
		return m.activateAccountAction(action)
	case actionBrowseRecords, actionNewRecord, actionViewRecord, actionEditRecord, actionDeleteRecord:
		return m.activateRecordAction(action, targets)
	case actionBrowseCache, actionViewCachedRecord, actionSynchronize:
		return m.activateCacheAction(action, targets)
	case actionCloseWindow, actionControls, actionAbout:
		return m.activateWindowAction(action)
	default:
		return m, nil
	}
}

func (m model) captureActionTargets() actionTargets {
	viewID, hasView := m.recordViewTarget()
	cachedViewID, hasCachedView := m.cachedRecordViewTarget()
	deleteRecord, hasDelete := m.recordDeleteTarget()

	return actionTargets{
		viewID:        viewID,
		hasView:       hasView,
		cachedViewID:  cachedViewID,
		hasCachedView: hasCachedView,
		deleteRecord:  deleteRecord,
		hasDelete:     hasDelete,
	}
}

func (targets actionTargets) supports(action actionID) bool {
	switch action {
	case actionViewRecord:
		return targets.hasView
	case actionViewCachedRecord:
		return targets.hasCachedView
	case actionDeleteRecord:
		return targets.hasDelete
	default:
		return true
	}
}

func (m model) activateSystemAction(action actionID) (tea.Model, tea.Cmd) {
	switch action {
	case actionQuit:
		m.cancelAllRequests()
		return m, tea.Quit
	case actionServerStatus:
		m.dialog = dialogServerStatus
		m.activeButton = 0
		return m.startServerStatusCheck()
	case actionConfig:
		m.dialog = dialogConfig
		m.configForm = newConfigForm(m.config)
	}

	return m, nil
}

func (m model) activateAccountAction(action actionID) (tea.Model, tea.Cmd) {
	switch action {
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
	}

	return m, nil
}

func (m model) activateRecordAction(action actionID, targets actionTargets) (tea.Model, tea.Cmd) {
	switch action {
	case actionBrowseRecords:
		return m.activateBrowseRecords()
	case actionNewRecord:
		return m.activateNewRecord()
	case actionViewRecord:
		if m.authentication.session.authenticated() && m.recordsAvailable() {
			return m, m.beginRecordView(targets.viewID)
		}
	case actionEditRecord:
		if m.authentication.session.authenticated() && m.recordEditAvailable() {
			return m, m.openRecordEdit()
		}
	case actionDeleteRecord:
		if m.authentication.session.authenticated() && m.recordDeleteAvailable() {
			m.openRecordDelete(targets.deleteRecord)
		}
	}

	return m, nil
}

func (m model) activateBrowseRecords() (tea.Model, tea.Cmd) {
	if !m.authentication.session.authenticated() || !m.recordsAvailable() {
		return m, nil
	}

	m.dialog = dialogNone

	return m, m.beginOnlineRecordList()
}

func (m model) activateNewRecord() (tea.Model, tea.Cmd) {
	if !m.authentication.session.authenticated() || !m.recordCreateAvailable() {
		return m, nil
	}

	if m.recordFeature.workspace.source == recordSourceCache {
		m.closeRecordWorkspace()
	}

	return m, m.openRecordTypePicker()
}

func (m model) activateCacheAction(action actionID, targets actionTargets) (tea.Model, tea.Cmd) {
	switch action {
	case actionBrowseCache:
		m.openCacheBrowser()
	case actionViewCachedRecord:
		if m.backend != nil {
			return m, m.beginRecordView(targets.cachedViewID)
		}
	case actionSynchronize:
		m.openSyncDialog()
	}

	return m, nil
}

func (m *model) openCacheBrowser() {
	if m.backend == nil {
		return
	}

	login := m.cacheFeature.form.login.value

	if m.authentication.session.authenticated() {
		login = m.authentication.session.login
	}

	m.dialog = dialogCacheBrowse
	m.cacheFeature.form = newCacheBrowseForm(login)
	m.activeButton = 0
}

func (m *model) openSyncDialog() {
	if !m.authentication.session.authenticated() {
		return
	}

	if m.recordFeature.workspace.source == recordSourceCache {
		m.closeRecordWorkspace()
	}

	m.dialog = dialogSync
	m.syncFeature = syncFeatureState{form: newSyncForm()}
	m.activeButton = 0
}

func (m model) activateWindowAction(action actionID) (tea.Model, tea.Cmd) {
	switch action {
	case actionControls:
		m.dialog = dialogControls
	case actionAbout:
		m.dialog = dialogAbout
		m.activeButton = 1
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
	case dialogCacheBrowse:
		m.closeCacheBrowse()
		return
	case dialogSync, dialogSyncResult:
		m.closeSync()
		return
	}
	m.dialog = dialogNone
	m.activeButton = 0
}

type dialogChangePathPickerState struct {
	binarySave   bool
	binaryCreate bool
	binaryEdit   bool
	action       actionID
}

func (m *model) prepareDialogChange(action actionID) {
	if action == actionCloseWindow {
		return
	}

	pathPickerState := m.dialogChangePathPickerState()

	m.prepareAuthDialogChange(action)
	m.prepareServerStatusDialogChange(action)
	m.preparePathPickerDialogChange(action, pathPickerState)
	m.prepareRecordDialogChange(action, pathPickerState)
	m.prepareCacheAndSyncDialogChange(action)
}

func (m model) dialogChangePathPickerState() dialogChangePathPickerState {
	state := dialogChangePathPickerState{action: actionConfig}
	if m.dialog != dialogPathPicker {
		return state
	}

	switch m.pathPicker.target {
	case pathPickerBinarySaveDirectory:
		state.binarySave = true
		state.action = actionBrowseRecords
	case pathPickerBinaryCreateFile:
		state.binaryCreate = true
		state.action = actionNewRecord
	case pathPickerBinaryEditFile:
		state.binaryEdit = true
		state.action = actionEditRecord
	}

	return state
}

func (m *model) prepareAuthDialogChange(action actionID) {
	if m.dialog == dialogLogin && action != actionLogin {
		m.clearLoginForm()
	}
	if m.dialog == dialogRegister && action != actionRegister {
		m.clearRegisterForm()
	}
}

func (m *model) prepareServerStatusDialogChange(action actionID) {
	if m.dialog == dialogServerStatus && action != actionServerStatus {
		m.operations.cancel(operationServerStatus)
	}
}

func (m *model) preparePathPickerDialogChange(action actionID, state dialogChangePathPickerState) {
	if m.dialog == dialogPathPicker && action != state.action {
		m.pathPicker = pathPicker{}
	}
}

func (m *model) prepareRecordDialogChange(action actionID, state dialogChangePathPickerState) {
	if (m.dialog == dialogRecordView || m.dialog == dialogBinarySave || state.binarySave) && action != actionEditRecord {
		m.leaveRecordView()
	}
	if (m.dialog == dialogRecordType || m.dialog == dialogRecordCreate || state.binaryCreate) && action != actionNewRecord {
		m.closeRecordCreate()
	}
	if (m.dialog == dialogRecordEdit || state.binaryEdit) && action != actionEditRecord {
		m.closeRecordEdit()
	}
	if m.dialog == dialogRecordDelete && action != actionDeleteRecord {
		m.closeRecordDelete()
	}
}

func (m *model) prepareCacheAndSyncDialogChange(action actionID) {
	if m.dialog == dialogCacheBrowse && action != actionBrowseCache {
		m.closeCacheBrowse()
	}
	if (m.dialog == dialogSync || m.dialog == dialogSyncResult) && action != actionSynchronize {
		m.closeSync()
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
	case dialogCacheBrowse:
		return actionBrowseCache
	case dialogSync, dialogSyncResult:
		return actionSynchronize
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
