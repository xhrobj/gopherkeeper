package tui

import (
	"strings"
	"unicode"

	"charm.land/lipgloss/v2"
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
	label     string
	mnemonic  rune
	shortcut  string
	action    actionID
	disabled  bool
	separator bool
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
		mnemonic: 's',
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
			{label: "Register...", mnemonic: 'r', action: actionRegister},
			{label: "Login...", mnemonic: 'l', action: actionLogin},
			{label: "Current User", mnemonic: 'U', action: actionCurrentUser, disabled: true},
			{separator: true},
			{label: "Logout", mnemonic: 'o', action: actionLogout, disabled: true},
		},
	},
	{
		name:     "Record",
		mnemonic: 'r',
		disabled: true,
		items: []menuItem{
			{label: "Browse", action: actionBrowseRecords, disabled: true},
			{label: "New...", action: actionNewRecord, disabled: true},
			{label: "View", action: actionViewRecord, disabled: true},
			{label: "Edit", action: actionEditRecord, disabled: true},
			{label: "Delete...", action: actionDeleteRecord, disabled: true},
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
		mnemonic: 'w',
		items: []menuItem{
			{label: "Close Active Window", action: actionCloseWindow},
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

	definitions[menuAccount].items[0].disabled = loggedIn
	definitions[menuAccount].items[1].disabled = loggedIn
	definitions[menuAccount].items[2].disabled = !loggedIn
	definitions[menuAccount].items[4].disabled = !loggedIn

	if dialog == dialogNone {
		definitions[menuWindow].items[0].disabled = true
	}

	for index := range definitions {
		if !hasEnabledMenuItem(definitions[index]) {
			definitions[index].disabled = true
		}
	}

	return definitions
}

func hasEnabledMenuItem(definition menuDefinition) bool {
	for _, item := range definition.items {
		if !item.separator && !item.disabled {
			return true
		}
	}

	return false
}

func renderMenuBar(t theme, definitions []menuDefinition, width int, active int, focused bool) string {
	var content strings.Builder

	for index, definition := range definitions {
		content.WriteString(renderMenuPad(t, definition, focused && index == active))
	}

	return t.menuBar.Width(width).Render(content.String())
}

func renderMenuPad(t theme, definition menuDefinition, active bool) string {
	if definition.disabled {
		return t.menuDisabled.Render(" " + definition.name + " ")
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
	runes := []rune(name)

	for index, value := range runes {
		if value == mnemonic {
			return string(runes[:index]), string(value), string(runes[index+1:])
		}
	}

	wanted := unicode.ToLower(mnemonic)
	for index, value := range runes {
		if unicode.ToLower(value) == wanted {
			return string(runes[:index]), string(value), string(runes[index+1:])
		}
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

	before, mnemonic, after := splitMnemonic(item.label, item.mnemonic)
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
