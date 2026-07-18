package tui

import (
	tea "charm.land/bubbletea/v2"
	"charm.land/lipgloss/v2"
)

const (
	menuBarY  = 0
	dropdownY = 1
	buttonGap = 3
)

type mouseBounds struct {
	x      int
	y      int
	width  int
	height int
}

func (bounds mouseBounds) contains(x, y int) bool {
	return x >= bounds.x && x < bounds.x+bounds.width &&
		y >= bounds.y && y < bounds.y+bounds.height
}

func (m model) updateMouse(msg tea.MouseClickMsg) (tea.Model, tea.Cmd) {
	if msg.Button != tea.MouseLeft || m.width < minimumWidth || m.height < minimumHeight {
		return m, nil
	}

	if m.alert != alertNone {
		buttons := m.dialogButtonBounds()
		if len(buttons) == 1 && buttons[0].contains(msg.X, msg.Y) {
			m.dismissAlert()
		}
		return m, nil
	}

	definitions := menuDefinitions(m.dialog, m.auth.authenticated())

	if msg.Y == menuBarY {
		index, ok := menuIndexAtX(definitions, msg.X)
		if !ok || definitions[index].disabled {
			m.closeMenu()
			return m, nil
		}
		if m.dropdownOpen && m.activeMenu == index {
			m.closeMenu()
			return m, nil
		}
		return m.openMenu(index)
	}

	if m.dropdownOpen {
		dropdownX := dropdownScreenX(m.width, m.activeMenu, definitions[m.activeMenu])
		bounds := dropdownScreenBounds(dropdownX, definitions[m.activeMenu])
		if bounds.contains(msg.X, msg.Y) {
			selected, item, ok := dropdownItemAt(definitions[m.activeMenu], dropdownX, msg.X, msg.Y)
			if !ok || item.disabled {
				return m, nil
			}
			m.selectedItem = selected
			return m.activate(item.action)
		}

		m.closeMenu()
	}

	if m.dialog == dialogLogin {
		if !m.loginRequest.pending {
			for index, bounds := range m.loginFieldBounds() {
				if bounds.contains(msg.X, msg.Y) {
					m.loginForm.setFocus(loginFocus(index), false)
					return m, nil
				}
			}
		}

		for index, bounds := range m.dialogButtonBounds() {
			if !bounds.contains(msg.X, msg.Y) {
				continue
			}
			focus := loginFocus(int(loginSubmit) + index)
			submitDisabled := m.loginRequest.pending || !m.loginForm.canSubmit()
			if submitDisabled && focus == loginSubmit {
				return m, nil
			}
			m.loginForm.setFocus(focus, submitDisabled)
			return m.activateLogin()
		}
		return m, nil
	}

	if m.dialog == dialogRegister {
		if !m.registerRequest.pending {
			for index, bounds := range m.registerFieldBounds() {
				if bounds.contains(msg.X, msg.Y) {
					m.registerForm.setFocus(registerFocus(index), !m.registerForm.canSubmit())
					return m, nil
				}
			}
		}

		for index, bounds := range m.dialogButtonBounds() {
			if !bounds.contains(msg.X, msg.Y) {
				continue
			}
			focus := registerFocus(int(registerSubmit) + index)
			submitDisabled := m.registerRequest.pending || !m.registerForm.canSubmit()
			if submitDisabled && focus == registerSubmit {
				return m, nil
			}
			m.registerForm.setFocus(focus, submitDisabled)
			return m.activateRegister()
		}
		return m, nil
	}

	if m.dialog == dialogConfig {
		for index, bounds := range m.configFieldBounds() {
			if bounds.contains(msg.X, msg.Y) {
				m.configForm.setFocus(configFocus(index))
				return m, nil
			}
		}

		for index, bounds := range m.dialogButtonBounds() {
			if bounds.contains(msg.X, msg.Y) {
				m.configForm.setFocus(configFocus(int(configSave) + index))
				return m.activateConfig()
			}
		}
		return m, nil
	}

	for index, bounds := range m.dialogButtonBounds() {
		if !bounds.contains(msg.X, msg.Y) {
			continue
		}
		if m.dialog == dialogServerStatus && m.statusState == serverStatusChecking && index == 0 {
			return m, nil
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

func menuIndexAtX(definitions []menuDefinition, x int) (int, bool) {
	offset := 0
	for index, definition := range definitions {
		width := lipgloss.Width(definition.name) + 2
		if x >= offset && x < offset+width {
			return index, true
		}
		offset += width
	}
	return 0, false
}

func dropdownScreenX(screenWidth, activeMenu int, definition menuDefinition) int {
	width := dropdownWidth(definition) + 2
	return min(menuOffset(activeMenu), max(0, screenWidth-width))
}

func dropdownScreenBounds(x int, definition menuDefinition) mouseBounds {
	return mouseBounds{
		x:      x,
		y:      dropdownY,
		width:  dropdownWidth(definition) + 2,
		height: len(definition.items) + 2,
	}
}

func dropdownItemAt(
	definition menuDefinition,
	dropdownX int,
	x int,
	y int,
) (int, menuItem, bool) {
	bounds := dropdownScreenBounds(dropdownX, definition)
	if !bounds.contains(x, y) || x == bounds.x || x == bounds.x+bounds.width-1 {
		return 0, menuItem{}, false
	}

	row := y - bounds.y - 1
	if row < 0 || row >= len(definition.items) {
		return 0, menuItem{}, false
	}

	selected := -1
	for index, item := range definition.items {
		if item.separator {
			if index == row {
				return 0, menuItem{}, false
			}
			continue
		}
		selected++
		if index == row {
			return selected, item, true
		}
	}

	return 0, menuItem{}, false
}

func (m model) dialogButtonBounds() []mouseBounds {
	if m.alert != alertNone {
		window, ok := m.alertPlacement()
		if !ok {
			return nil
		}
		style := m.theme.aboutButtonActive
		if m.alert == alertError {
			style = m.theme.errorButton
		}
		return centeredStyledButtonBounds(
			style,
			window.x+2,
			window.y+alertButtonRow,
			window.width-4,
			[]string{"< OK >"},
			0,
		)
	}

	window, ok := m.dialogPlacement()
	if !ok {
		return nil
	}
	windowX := window.x
	windowY := window.y
	windowWidth := window.width

	switch m.dialog {
	case dialogLogin:
		return centeredStyledButtonBounds(
			m.theme.button,
			windowX+2,
			windowY+loginButtonRow,
			windowWidth-4,
			[]string{"< Login >", "< Close >"},
			loginButtonGap,
		)
	case dialogRegister:
		return centeredStyledButtonBounds(
			m.theme.button,
			windowX+2,
			windowY+registerButtonRow,
			windowWidth-4,
			[]string{"< Register >", "< Close >"},
			registerButtonGap,
		)
	case dialogCurrentUser:
		return centeredStyledButtonBounds(
			m.theme.aboutButtonActive,
			windowX+2,
			windowY+currentUserButtonRow,
			windowWidth-4,
			[]string{"< OK >"},
			0,
		)
	case dialogAbout:
		return centeredStyledButtonBounds(
			m.theme.aboutButton,
			windowX,
			windowY+aboutButtonRow,
			windowWidth,
			[]string{"< Course >", "< OK >"},
			aboutButtonGap,
		)
	case dialogConfig:
		return centeredButtonBounds(
			m.theme,
			windowX+2,
			windowY+configButtonRow,
			windowWidth-4,
			[]string{"< Save >", "< Cancel >"},
		)
	case dialogControls:
		return centeredButtonBounds(
			m.theme,
			windowX+2,
			windowY+controlsButtonRow,
			windowWidth-4,
			[]string{"< OK >"},
		)
	case dialogServerStatus:
		style := m.theme.button
		if m.statusState == serverStatusFailed {
			style = m.theme.errorText.Padding(0, 1)
		}
		return centeredStyledButtonBounds(
			style,
			windowX+2,
			windowY+serverStatusButtonRow,
			windowWidth-4,
			[]string{"< Retry >", "< OK >"},
			serverStatusButtonGap,
		)
	default:
		return nil
	}
}

func (m model) loginFieldBounds() []mouseBounds {
	window, ok := m.dialogPlacement()
	if !ok {
		return nil
	}
	windowWidth := window.width
	windowX := window.x
	windowY := window.y
	contentWidth := max(1, windowWidth-4)
	inputWidth := max(16, contentWidth-loginLabelWidth-2)
	inputX := windowX + 2 + loginLabelWidth + 2

	bounds := make([]mouseBounds, 0, 2)
	for index := range 2 {
		bounds = append(bounds, mouseBounds{
			x: inputX, y: windowY + loginFirstFieldRow + index*loginFieldRowStep,
			width: inputWidth, height: 1,
		})
	}
	return bounds
}

func (m model) registerFieldBounds() []mouseBounds {
	window, ok := m.dialogPlacement()
	if !ok {
		return nil
	}
	windowWidth := window.width
	windowX := window.x
	windowY := window.y
	contentWidth := max(1, windowWidth-4)
	inputWidth := max(16, contentWidth-registerLabelWidth-2)
	inputX := windowX + 2 + registerLabelWidth + 2

	bounds := make([]mouseBounds, 0, 3)
	for index := range 3 {
		bounds = append(bounds, mouseBounds{
			x: inputX, y: windowY + registerFirstFieldRow + index*registerFieldRowStep,
			width: inputWidth, height: 1,
		})
	}
	return bounds
}

func (m model) configFieldBounds() []mouseBounds {
	window, ok := m.dialogPlacement()
	if !ok {
		return nil
	}
	windowX := window.x
	windowY := window.y
	bodyWidth := max(1, configWindowWidth(m.width)-4)
	inputWidth := max(18, bodyWidth-configLabelWidth-2)
	inputX := windowX + 2 + configLabelWidth + 2

	bounds := make([]mouseBounds, 0, 4)
	for index := range 4 {
		bounds = append(bounds, mouseBounds{
			x:      inputX,
			y:      windowY + configFirstFieldRow + index*configFieldRowStep,
			width:  inputWidth,
			height: 1,
		})
	}
	return bounds
}

func centeredButtonBounds(
	t theme,
	x int,
	y int,
	containerWidth int,
	labels []string,
) []mouseBounds {
	return centeredStyledButtonBounds(t.button, x, y, containerWidth, labels, buttonGap)
}

func centeredStyledButtonBounds(
	style lipgloss.Style,
	x int,
	y int,
	containerWidth int,
	labels []string,
	gap int,
) []mouseBounds {
	if len(labels) == 0 {
		return nil
	}

	widths := make([]int, len(labels))
	totalWidth := 0
	for index, label := range labels {
		widths[index] = lipgloss.Width(style.Render(label))
		totalWidth += widths[index]
	}
	totalWidth += gap * (len(labels) - 1)

	currentX := x + max(0, (containerWidth-totalWidth)/2)
	bounds := make([]mouseBounds, 0, len(labels))
	for _, width := range widths {
		bounds = append(bounds, mouseBounds{x: currentX, y: y, width: width, height: 1})
		currentX += width + gap
	}
	return bounds
}
