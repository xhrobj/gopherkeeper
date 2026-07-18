package tui

import (
	"fmt"

	tea "charm.land/bubbletea/v2"
	"charm.land/lipgloss/v2"
	"github.com/xhrobj/gopherkeeper/internal/buildinfo"
)

func (m model) View() tea.View {
	content := m.render()
	view := tea.NewView(content)
	view.AltScreen = true
	view.MouseMode = tea.MouseModeCellMotion
	view.WindowTitle = "(^-^)/"

	return view
}

func (m model) render() string {
	if m.width < minimumWidth || m.height < minimumHeight {
		return m.renderTooSmall()
	}

	definitions := menuDefinitions(m.dialog, m.auth.authenticated())
	background := m.theme.desktop.Width(m.width).Height(m.height).Render("")
	layers := []*lipgloss.Layer{
		lipgloss.NewLayer(background).X(0).Y(0).Z(0),
		lipgloss.NewLayer(renderMenuBar(
			m.theme,
			definitions,
			m.width,
			m.activeMenu,
			m.menuFocused || m.dropdownOpen,
		)).X(0).Y(0).Z(10),
		lipgloss.NewLayer(m.theme.menuHint.Render("F10 = Menu")).
			X(max(0, m.width-len("F10 = Menu")-1)).Y(0).Z(11),
	}

	if window, ok := m.dialogPlacement(); ok {
		layers = append(
			layers,
			lipgloss.NewLayer(renderShadow(m.theme, window.width, window.height)).
				X(window.x+2).Y(window.y+1).Z(19),
			lipgloss.NewLayer(window.content).X(window.x).Y(window.y).Z(20),
		)
	}

	if m.dropdownOpen {
		dropdown := renderDropdown(
			m.theme,
			definitions[m.activeMenu],
			m.selectedItem,
			currentDialogAction(m.dialog),
		)
		dropdownWidth := lipgloss.Width(dropdown)
		dropdownHeight := lipgloss.Height(dropdown)
		dropdownX := min(menuOffset(m.activeMenu), max(0, m.width-dropdownWidth))

		layers = append(
			layers,
			lipgloss.NewLayer(renderShadow(m.theme, dropdownWidth, dropdownHeight)).X(dropdownX+1).Y(2).Z(29),
			lipgloss.NewLayer(dropdown).X(dropdownX).Y(1).Z(30),
		)
	}

	if overlay, ok := m.alertPlacement(); ok {
		layers = append(
			layers,
			lipgloss.NewLayer(renderShadow(m.theme, overlay.width, overlay.height)).
				X(overlay.x+2).Y(overlay.y+1).Z(39),
			lipgloss.NewLayer(overlay.content).X(overlay.x).Y(overlay.y).Z(40),
		)
	}

	return lipgloss.NewCompositor(layers...).Render()
}

func (m model) renderDialog() string {
	switch m.dialog {
	case dialogNone:
		return ""
	case dialogAbout:
		return renderAboutWindow(
			m.theme,
			aboutWindowWidth(m.width),
			buildinfo.Value(m.info.Version),
			buildinfo.Value(m.info.Date),
			buildinfo.Value(m.info.Commit),
			m.activeButton,
		)
	case dialogControls:
		width := clamp(m.width-18, 44, 58)
		return renderControlsWindow(m.theme, width)
	case dialogConfig:
		return renderConfigWindow(m.theme, configWindowWidth(m.width), m.configForm, m.configFile)
	case dialogServerStatus:
		return renderServerStatusWindow(
			m.theme,
			serverStatusWindowWidth(m.width),
			m.config.Address,
			m.statusState,
			m.statusValue,
			m.statusFailure,
			m.activeButton,
		)
	case dialogLogin:
		width := clamp(m.width-18, 44, 58)
		return renderLoginWindow(m.theme, width, m.loginForm, m.loginRequest.pending)
	case dialogRegister:
		width := clamp(m.width-18, 48, 62)
		return renderRegisterWindow(m.theme, width, m.registerForm, m.registerRequest.pending)
	case dialogCurrentUser:
		width := clamp(m.width-24, 44, 58)
		return renderCurrentUserWindow(m.theme, width, m.auth.login)
	default:
		return ""
	}
}

func (m model) renderAlert() string {
	if m.alert == alertNone {
		return ""
	}

	return renderAlertWindow(m.theme, m.alert, m.alertTitle, m.alertMessage, m.alertHighlight)
}

func (m model) renderTooSmall() string {
	width := max(1, m.width)
	height := max(1, m.height)
	message := fmt.Sprintf(
		"Terminal window is too small.\n\nRequired: %d×%d\nCurrent:  %d×%d\n\nResize the terminal or press ^Q to exit.",
		minimumWidth,
		minimumHeight,
		m.width,
		m.height,
	)

	return m.theme.desktop.
		Width(width).
		Height(height).
		AlignHorizontal(lipgloss.Center).
		AlignVertical(lipgloss.Center).
		Render(m.theme.warning.Render(message))
}
