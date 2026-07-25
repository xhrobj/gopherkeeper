package tui

import (
	"fmt"

	tea "charm.land/bubbletea/v2"
	"charm.land/lipgloss/v2"
	"github.com/xhrobj/gopherkeeper/internal/buildinfo"
)

func renderMenuHint(t theme, blocked bool) string {
	if blocked {
		return t.menuHintBlocked.Render(menuHintText)
	}

	return t.menuHint.Render(menuHintText)
}

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

	definitions := m.currentMenuDefinitions()
	blocked := m.interactionBlocked()
	background := m.theme.desktop.Width(m.width).Height(m.height).Render("")
	layers := []*lipgloss.Layer{
		lipgloss.NewLayer(background).X(0).Y(0).Z(0),
		lipgloss.NewLayer(renderMenuBar(
			m.theme,
			definitions,
			m.width,
			m.activeMenu,
			m.menuFocused || m.dropdownOpen,
			blocked,
		)).X(0).Y(0).Z(10),
		lipgloss.NewLayer(renderMenuHint(m.theme, blocked)).
			X(max(0, m.width-len(menuHintText)-1)).Y(0).Z(11),
	}

	if window, ok := m.workspacePlacement(); ok {
		layers = append(
			layers,
			lipgloss.NewLayer(renderShadow(m.theme, window.width, window.height)).
				X(window.x+2).Y(window.y+1).Z(14),
			lipgloss.NewLayer(window.content).X(window.x).Y(window.y).Z(15),
		)
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
			m.currentAction(),
		)
		dropdownLayout := buildDropdownMenuLayout(m.width, m.activeMenu, definitions[m.activeMenu])
		bounds := dropdownLayout.bounds

		layers = append(
			layers,
			lipgloss.NewLayer(renderShadow(m.theme, bounds.width, bounds.height)).X(bounds.x+1).Y(bounds.y+1).Z(29),
			lipgloss.NewLayer(dropdown).X(bounds.x).Y(bounds.y).Z(30),
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

	if overlay, ok := m.networkBusyPlacement(); ok {
		layers = append(
			layers,
			lipgloss.NewLayer(renderShadow(m.theme, overlay.width, overlay.height)).
				X(overlay.x+2).Y(overlay.y+1).Z(49),
			lipgloss.NewLayer(overlay.content).X(overlay.x).Y(overlay.y).Z(50),
		)
	}

	return lipgloss.NewCompositor(layers...).Render()
}

func (m model) renderDialog() string {
	blocked := m.interactionBlocked()
	spinnerFrame := m.spinnerFrameValue()

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
		width := clamp(m.width-18, 50, 58)
		return renderControlsWindow(m.theme, width)
	case dialogConfig:
		return renderConfigWindow(m.theme, configWindowWidth(m.width), m.configForm, m.configFile)
	case dialogPathPicker:
		return renderPathPickerWindow(
			m.theme,
			pathPickerWindowWidth(m.width),
			m.pathPicker,
		)
	case dialogServerStatus:
		return renderServerStatusWindow(m.theme, serverStatusWindowOptions{
			width:        serverStatusWindowWidth(m.width),
			address:      m.config.Address,
			state:        m.statusState,
			health:       m.statusValue,
			failure:      m.statusFailure,
			activeButton: m.activeButton,
			pending:      m.operations.pending(operationServerStatus),
			blocked:      blocked,
			spinnerFrame: spinnerFrame,
		})
	case dialogLogin:
		width := clamp(m.width-18, 44, 58)
		return renderLoginWindow(m.theme, width, m.authentication.loginForm, m.operations.pending(operationLogin), blocked, spinnerFrame)
	case dialogRegister:
		width := clamp(m.width-18, 48, 62)
		return renderRegisterWindow(m.theme, width, m.authentication.registerForm, m.operations.pending(operationRegister), blocked, spinnerFrame)
	case dialogCurrentUser:
		width := clamp(m.width-24, 44, 58)
		return renderCurrentUserWindow(m.theme, width, m.authentication.session.login, m.operations.pending(operationCurrentUser), blocked, spinnerFrame)
	case dialogRecordView:
		return renderRecordViewWindow(m.theme, recordViewWindowOptions{
			width:        recordViewWindowWidth(m.width),
			height:       recordViewWindowHeightForState(m.theme, m.width, m.height, m.recordFeature.view),
			state:        m.recordFeature.view,
			activeButton: m.activeButton,
			pending:      m.operations.pending(operationViewRecord) || m.operations.pending(operationViewCachedRecord),
			blocked:      blocked,
			spinnerFrame: spinnerFrame,
		})
	case dialogBinarySave:
		return renderBinarySaveWindow(m.theme, binarySaveWindowWidth(m.width), m.recordFeature.binarySaveForm, m.operations.pending(operationBinarySave))
	case dialogRecordType:
		return renderRecordTypePickerWindow(m.theme, m.recordFeature.typePicker)
	case dialogRecordCreate:
		return renderRecordCreateWindow(
			m.theme,
			recordCreateWindowWidth(m.width),
			m.recordFeature.createForm,
			m.operations.pending(operationCreateRecord),
			blocked,
			spinnerFrame,
		)
	case dialogRecordEdit:
		if m.recordFeature.edit.status == recordEditLoading {
			return renderRecordEditLoadingWindow(
				m.theme,
				recordCreateWindowWidth(m.width),
				m.recordFeature.edit.form,
				spinnerFrame,
			)
		}
		return renderRecordEditWindow(
			m.theme,
			recordCreateWindowWidth(m.width),
			m.recordFeature.edit.form,
			m.operations.pending(operationEditRecord),
			blocked,
			spinnerFrame,
		)
	case dialogCacheBrowse:
		return renderCacheBrowseWindow(
			m.theme,
			cacheBrowseWindowWidth(m.width),
			m.cacheFeature.form,
			m.operations.pending(operationOpenCache),
			blocked,
			spinnerFrame,
		)
	case dialogSync:
		return renderSyncWindow(
			m.theme,
			syncWindowWidth(m.width),
			m.authentication.session.login,
			m.syncFeature.form,
			m.operations.pending(operationSync),
			blocked,
			spinnerFrame,
		)
	case dialogSyncResult:
		return renderSyncResultWindow(m.theme, syncResultWindowWidth(m.width), m.syncFeature.result, blocked)
	case dialogRecordDelete:
		return renderRecordDeleteWindow(
			m.theme,
			recordDeleteWindowWidth(m.width),
			m.recordFeature.deletion,
			m.operations.pending(operationDeleteRecord),
			blocked,
			spinnerFrame,
			m.activeButton,
		)
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
