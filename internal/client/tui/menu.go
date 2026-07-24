package tui

import (
	"strings"
	"unicode"

	"charm.land/lipgloss/v2"
)

const (
	menuBarY  = 0
	dropdownY = 1
)

type menuID int

const (
	menuSystem menuID = iota
	menuAccount
	menuRecord
	menuSync
	menuWindow
	menuHelp
)

type menuDefinition struct {
	name     string
	mnemonic rune
	disabled bool
	items    []menuItem
}

type menuItem struct {
	label              string
	mnemonic           rune
	mnemonicOccurrence int
	shortcut           string
	action             actionID
	disabled           bool
	separator          bool
}

type actionID int

const (
	actionNone actionID = iota
	actionServerStatus
	actionConfig
	actionQuit
	actionRegister
	actionLogin
	actionCurrentUser
	actionLogout
	actionBrowseRecords
	actionNewRecord
	actionViewRecord
	actionEditRecord
	actionDeleteRecord
	actionSynchronize
	actionCloseWindow
	actionControls
	actionAbout
)

var menus = []menuDefinition{
	{
		name:     "System",
		mnemonic: 'S',
		items: []menuItem{
			{label: "Server Status", mnemonic: 'u', action: actionServerStatus},
			{label: "Config...", mnemonic: 'c', action: actionConfig},
			{separator: true},
			{label: "Quit", mnemonic: 'q', shortcut: "^Q", action: actionQuit},
		},
	},
	{
		name:     "Account",
		mnemonic: 'a',
		items: []menuItem{
			{label: "Login...", mnemonic: 'l', action: actionLogin},
			{label: "Register...", mnemonic: 'r', action: actionRegister},
			{label: "Current User", mnemonic: 'U', action: actionCurrentUser, disabled: true},
			{separator: true},
			{label: "Logout", mnemonic: 'o', mnemonicOccurrence: 1, action: actionLogout, disabled: true},
		},
	},
	{
		name:     "Record",
		mnemonic: 'R',
		disabled: true,
		items: []menuItem{
			{label: "Browse", mnemonic: 'b', action: actionBrowseRecords, disabled: true},
			{separator: true},
			{label: "New...", mnemonic: 'n', action: actionNewRecord, disabled: true},
			{label: "View", mnemonic: 'v', action: actionViewRecord, disabled: true},
			{label: "Edit...", mnemonic: 'e', action: actionEditRecord, disabled: true},
			{label: "Delete...", mnemonic: 'd', action: actionDeleteRecord, disabled: true},
		},
	},
	{
		name:     "Sync",
		mnemonic: 'y',
		disabled: true,
		items: []menuItem{
			{label: "Synchronize...", action: actionSynchronize, disabled: true},
		},
	},
	{
		name:     "Window",
		mnemonic: 'W',
		items: []menuItem{
			{label: "Close Active Window", mnemonic: 'W', action: actionCloseWindow},
		},
	},
	{
		name:     "Help",
		mnemonic: 'h',
		items: []menuItem{
			{label: "Controls", mnemonic: 'c', action: actionControls},
			{label: "About GophKeeper", mnemonic: 'g', action: actionAbout},
		},
	},
}

// menuDefinitions возвращает определения меню для текущего состояния окна.
func menuDefinitions(dialog dialogID, loggedIn bool) []menuDefinition {
	definitions := make([]menuDefinition, len(menus))
	copy(definitions, menus)

	for index := range definitions {
		definitions[index].items = append([]menuItem(nil), menus[index].items...)
	}

	setMenuActionDisabled(&definitions[menuAccount], actionRegister, loggedIn)
	setMenuActionDisabled(&definitions[menuAccount], actionLogin, loggedIn)
	setMenuActionDisabled(&definitions[menuAccount], actionCurrentUser, !loggedIn)
	setMenuActionDisabled(&definitions[menuAccount], actionLogout, !loggedIn)

	if dialog == dialogNone {
		setMenuActionDisabled(&definitions[menuWindow], actionCloseWindow, true)
	}

	for index := range definitions {
		if !hasEnabledMenuItem(definitions[index]) {
			definitions[index].disabled = true
		}
	}

	return definitions
}

func (m model) currentMenuDefinitions() []menuDefinition {
	definitions := menuDefinitions(m.dialog, m.authentication.session.authenticated())
	recordMenu := &definitions[menuRecord]
	loggedIn := m.authentication.session.authenticated()

	setMenuActionDisabled(recordMenu, actionBrowseRecords, !loggedIn || !m.recordsAvailable())
	setMenuActionDisabled(recordMenu, actionNewRecord, !loggedIn || !m.recordCreateAvailable())

	_, viewSelected := m.recordViewTarget()
	setMenuActionDisabled(recordMenu, actionViewRecord, !loggedIn || !m.recordsAvailable() || !viewSelected)

	_, editSelected := m.recordEditTarget()
	setMenuActionDisabled(recordMenu, actionEditRecord, !loggedIn || !m.recordEditAvailable() || !editSelected)

	_, deleteSelected := m.recordDeleteTarget()
	setMenuActionDisabled(recordMenu, actionDeleteRecord, !loggedIn || !m.recordDeleteAvailable() || !deleteSelected)
	recordMenu.disabled = !hasEnabledMenuItem(*recordMenu)

	if m.dialog == dialogNone && m.recordFeature.workspace.open {
		setMenuActionDisabled(&definitions[menuWindow], actionCloseWindow, false)
		definitions[menuWindow].disabled = false
	}

	return definitions
}

func (m model) currentAction() actionID {
	if m.dialog == dialogPathPicker {
		switch m.pathPicker.target {
		case pathPickerBinarySaveDirectory:
			return actionBrowseRecords
		case pathPickerBinaryCreateFile:
			return actionNewRecord
		case pathPickerBinaryEditFile:
			return actionEditRecord
		}
	}

	if m.dialog != dialogNone {
		return currentDialogAction(m.dialog)
	}

	if !m.recordFeature.workspace.open {
		return actionNone
	}

	return actionBrowseRecords
}

func setMenuActionDisabled(definition *menuDefinition, action actionID, disabled bool) bool {
	for index := range definition.items {
		if definition.items[index].action == action {
			definition.items[index].disabled = disabled
			return true
		}
	}

	return false
}

func hasEnabledMenuItem(definition menuDefinition) bool {
	for _, item := range definition.items {
		if !item.separator && !item.disabled {
			return true
		}
	}

	return false
}

type menuBarLayout struct {
	content string
	bounds  []layoutBounds
}

func buildMenuBarLayout(t theme, definitions []menuDefinition, width int, active int, focused, blocked bool) menuBarLayout {
	var content strings.Builder
	bounds := make([]layoutBounds, len(definitions))
	x := 0

	for index, definition := range definitions {
		pad := renderMenuPad(t, definition, focused && index == active, blocked)
		padWidth := lipgloss.Width(pad)
		bounds[index] = layoutBounds{x: x, y: menuBarY, width: padWidth, height: 1}
		x += padWidth
		content.WriteString(pad)
	}

	return menuBarLayout{
		content: t.menuBar.Width(width).Render(content.String()),
		bounds:  bounds,
	}
}

func renderMenuBar(t theme, definitions []menuDefinition, width int, active int, focused, blocked bool) string {
	return buildMenuBarLayout(t, definitions, width, active, focused, blocked).content
}

func renderMenuPad(t theme, definition menuDefinition, active, blocked bool) string {
	if definition.disabled {
		return t.menuDisabled.Render(" " + definition.name + " ")
	}
	if blocked {
		return t.menuBlocked.Render(" " + definition.name + " ")
	}

	if active {
		return t.menuActive.Render(" " + definition.name + " ")
	}

	before, mnemonic, after := splitMnemonic(definition.name, definition.mnemonic)

	return t.menuItem.Render(" "+before) +
		t.menuMnemonic.Render(mnemonic) +
		t.menuItem.Render(after+" ")
}

func splitMnemonic(name string, mnemonic rune) (string, string, string) {
	return splitMnemonicOccurrence(name, mnemonic, 0)
}

func splitMnemonicOccurrence(name string, mnemonic rune, occurrence int) (string, string, string) {
	runes := []rune(name)
	wanted := unicode.ToLower(mnemonic)
	matched := 0

	for index, value := range runes {
		if unicode.ToLower(value) != wanted {
			continue
		}
		if matched == occurrence {
			return string(runes[:index]), string(value), string(runes[index+1:])
		}
		matched++
	}
	return name, "", ""
}

func renderDropdown(t theme, definition menuDefinition, selected int, current actionID) string {
	contentWidth := dropdownWidth(definition)
	rows := make([]string, 0, len(definition.items)+2)
	selectable := -1

	rows = append(rows, t.dropdownFrame.Render(
		"┌"+strings.Repeat("─", contentWidth)+"┐",
	))

	for _, item := range definition.items {
		if item.separator {
			rows = append(rows, t.dropdownFrame.Render(
				"├"+strings.Repeat("─", contentWidth)+"┤",
			))
			continue
		}

		selectable++
		line := renderMenuItemLine(t, item, contentWidth, selectable == selected, item.action == current)
		rows = append(
			rows,
			t.dropdownFrame.Render("│")+line+t.dropdownFrame.Render("│"),
		)
	}

	rows = append(rows, t.dropdownFrame.Render(
		"└"+strings.Repeat("─", contentWidth)+"┘",
	))

	return t.dropdown.Render(strings.Join(rows, "\n"))
}

func renderMenuItemLine(t theme, item menuItem, contentWidth int, selected, current bool) string {
	left := " " + item.label
	right := ""

	if item.shortcut != "" {
		right = item.shortcut + " "
	}

	gap := max(0, contentWidth-lipgloss.Width(left)-lipgloss.Width(right))
	if item.shortcut != "" {
		gap = max(1, gap)
	}

	line := left + strings.Repeat(" ", gap) + right

	if item.disabled {
		return t.dropdownDisabled.Render(line)
	}

	if current {
		if selected {
			return t.dropdownCurrentSelected.Render(line)
		}
		return t.dropdownCurrent.Render(line)
	}

	baseStyle := t.dropdownRow
	mnemonicStyle := t.dropdownMnemonic
	if selected {
		baseStyle = t.dropdownPick
		mnemonicStyle = t.dropdownPickMnemonic
	}

	if item.mnemonic == 0 {
		return baseStyle.Render(line)
	}

	before, mnemonic, after := splitMnemonicOccurrence(item.label, item.mnemonic, item.mnemonicOccurrence)
	if mnemonic == "" {
		return baseStyle.Render(line)
	}

	return baseStyle.Render(" "+before) +
		mnemonicStyle.Render(mnemonic) +
		baseStyle.Render(after+strings.Repeat(" ", gap)+right)
}

func menuItemByMnemonic(definition menuDefinition, value rune) (int, menuItem, bool) {
	value = unicode.ToLower(value)
	selectable := -1

	for _, item := range definition.items {
		if item.separator {
			continue
		}
		selectable++
		if !item.disabled && item.mnemonic != 0 && unicode.ToLower(item.mnemonic) == value {
			return selectable, item, true
		}
	}

	return 0, menuItem{}, false
}

// dropdownWidth возвращает ширину содержимого между боковыми границами.
func dropdownWidth(definition menuDefinition) int {
	width := 0

	for _, item := range definition.items {
		if item.separator {
			continue
		}

		itemWidth := lipgloss.Width(item.label) + 2
		if item.shortcut != "" {
			itemWidth += lipgloss.Width(item.shortcut) + 2
		}
		width = max(width, itemWidth)
	}

	return width
}

func selectableItems(definition menuDefinition) []menuItem {
	items := make([]menuItem, 0, len(definition.items))

	for _, item := range definition.items {
		if !item.separator {
			items = append(items, item)
		}
	}

	return items
}

func menuOffset(active int) int {
	offset := 0

	for index := range active {
		offset += len(menus[index].name) + 2
	}

	return offset
}

func firstEnabledMenuIndex(definitions []menuDefinition) int {
	for index, definition := range definitions {
		if !definition.disabled {
			return index
		}
	}

	return 0
}

func nextEnabledMenuIndex(definitions []menuDefinition, current, step int) int {
	if len(definitions) == 0 {
		return 0
	}

	for range len(definitions) {
		current = (current + step + len(definitions)) % len(definitions)
		if !definitions[current].disabled {
			return current
		}
	}

	return current
}

func menuIndexByMnemonic(definitions []menuDefinition, value rune) (int, bool) {
	value = unicode.ToLower(value)

	for index, definition := range definitions {
		if !definition.disabled && unicode.ToLower(definition.mnemonic) == value {
			return index, true
		}
	}

	return 0, false
}

func menuIndexByAltKey(definitions []menuDefinition, key string) (int, bool) {
	const prefix = "alt+"

	if !strings.HasPrefix(key, prefix) {
		return 0, false
	}

	value := []rune(strings.TrimPrefix(key, prefix))

	if len(value) != 1 {
		return 0, false
	}

	return menuIndexByMnemonic(definitions, value[0])
}

func menuIndexAtX(bounds []layoutBounds, x int) (int, bool) {
	for index, bound := range bounds {
		if bound.contains(x, menuBarY) {
			return index, true
		}
	}

	return 0, false
}

type dropdownItemLayout struct {
	selected int
	item     menuItem
	bounds   layoutBounds
}

type dropdownMenuLayout struct {
	bounds layoutBounds
	items  []dropdownItemLayout
}

func buildDropdownMenuLayout(screenWidth, activeMenu int, definition menuDefinition) dropdownMenuLayout {
	width := dropdownWidth(definition) + 2
	x := min(menuOffset(activeMenu), max(0, screenWidth-width))
	layout := dropdownMenuLayout{
		bounds: layoutBounds{
			x:      x,
			y:      dropdownY,
			width:  width,
			height: len(definition.items) + 2,
		},
	}
	selected := -1

	for index, item := range definition.items {
		if item.separator {
			continue
		}
		selected++
		layout.items = append(layout.items, dropdownItemLayout{
			selected: selected,
			item:     item,
			bounds: layoutBounds{
				x:      x + 1,
				y:      dropdownY + 1 + index,
				width:  max(0, width-2),
				height: 1,
			},
		})
	}

	return layout
}

func (layout dropdownMenuLayout) itemAt(x, y int) (int, menuItem, bool) {
	if !layout.bounds.contains(x, y) {
		return 0, menuItem{}, false
	}

	for _, item := range layout.items {
		if item.bounds.contains(x, y) {
			return item.selected, item.item, true
		}
	}

	return 0, menuItem{}, false
}
