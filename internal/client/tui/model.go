package tui

import (
	"context"
	"strings"
	"time"

	tea "charm.land/bubbletea/v2"
	"github.com/xhrobj/gopherkeeper/internal/buildinfo"
	"github.com/xhrobj/gopherkeeper/internal/client/config"
)

const (
	minimumWidth  = 62
	minimumHeight = 23
	defaultWidth  = 80
	defaultHeight = 25
)

type dialogID int

const (
	dialogNone dialogID = iota
	dialogLogin
	dialogRegister
	dialogCurrentUser
	dialogControls
	dialogAbout
	dialogServerStatus
	dialogConfig
)

type alertState int

const (
	alertNone alertState = iota
	alertError
	alertNotice
)

type configSaver func(string, config.Config) error

type serverStatusState int

const (
	serverStatusIdle serverStatusState = iota
	serverStatusChecking
	serverStatusReady
	serverStatusFailed
)

type model struct {
	width             int
	height            int
	config            config.Config
	configFile        string
	saveConfig        configSaver
	info              buildinfo.Info
	theme             theme
	menuFocused       bool
	dropdownOpen      bool
	activeMenu        int
	selectedItem      int
	dialog            dialogID
	activeButton      int
	configForm        configForm
	backend           Backend
	backendFactory    BackendFactory
	startupCmd        tea.Cmd
	auth              authSession
	authRequest       requestState
	loginForm         loginForm
	loginRequest      requestState
	registerForm      registerForm
	registerRequest   requestState
	logoutRequest     requestState
	alert             alertState
	alertTitle        string
	alertMessage      string
	alertHighlight    string
	alertReturnDialog dialogID
	openURL           openURLFunc
	ctx               context.Context
	statusState       serverStatusState
	statusValue       string
	statusFailure     serverStatusFailure
	statusRequest     requestState
	statusMinDuration time.Duration
}

func newModel(
	ctx context.Context,
	cfg config.Config,
	configFile string,
	info buildinfo.Info,
	backendFactory BackendFactory,
) model {
	if ctx == nil {
		ctx = context.Background()
	}

	backend := backendFactory(cfg)

	m := model{
		width:             defaultWidth,
		height:            defaultHeight,
		config:            cfg,
		configFile:        configFile,
		saveConfig:        config.Save,
		info:              info,
		theme:             newTheme(),
		dialog:            dialogNone,
		activeButton:      0,
		backend:           backend,
		backendFactory:    backendFactory,
		auth:              authSession{state: authUnknown},
		configForm:        newConfigForm(cfg),
		loginForm:         newLoginForm(),
		registerForm:      newRegisterForm(),
		openURL:           openExternalURL,
		ctx:               ctx,
		statusState:       serverStatusIdle,
		statusMinDuration: 500 * time.Millisecond,
	}
	m.startupCmd = m.beginCurrentUserCheck()

	return m
}

func (m model) Init() tea.Cmd {
	return m.startupCmd
}

func (m model) Update(msg tea.Msg) (tea.Model, tea.Cmd) {
	switch msg := msg.(type) {
	case tea.WindowSizeMsg:
		m.width = msg.Width
		m.height = msg.Height
		return m, nil
	case tea.MouseClickMsg:
		return m.updateMouse(msg)
	case currentUserResultMsg:
		if !m.authRequest.accepts(msg.requestID) {
			return m, nil
		}
		m.authRequest.finish()
		if msg.err == nil {
			m.auth = authSession{state: authAuthenticated, login: msg.login}
			if m.dialog == dialogLogin {
				m.dialog = dialogNone
			}
			return m, nil
		}

		m.auth = authSession{state: authGuest}
		if isNotLoggedIn(msg.err) {
			if m.dialog == dialogNone {
				m.dialog = dialogLogin
			}
			return m, nil
		}
		returnDialog := m.dialog
		if returnDialog == dialogNone {
			returnDialog = dialogLogin
		}
		m.showAlert(
			alertError,
			"Session check failed",
			cleanCurrentUserError(msg.err),
			returnDialog,
		)
		return m, nil
	case loginResultMsg:
		if !m.loginRequest.accepts(msg.requestID) {
			return m, nil
		}
		m.loginRequest.finish()
		if msg.err != nil {
			m.showAlert(alertError, "Login failed", cleanLoginError(msg.err), dialogLogin)
			m.loginForm.focus = loginSubmit
		} else {
			m.auth = authSession{state: authAuthenticated, login: msg.login}
			m.loginForm = newLoginForm()
			m.dialog = dialogCurrentUser
			m.activeButton = 0
		}
		return m, nil
	case registerResultMsg:
		if !m.registerRequest.accepts(msg.requestID) {
			return m, nil
		}
		m.registerRequest.finish()
		if msg.err != nil {
			message := cleanRegisterError(msg.err)
			m.showAlertWithHighlight(
				alertError,
				"Registration failed",
				message,
				strings.TrimSpace(m.registerForm.login.value),
				dialogRegister,
			)
			m.registerForm.focus = registerSubmit
		} else {
			m.loginForm = newLoginForm()
			m.loginForm.login.setValue(msg.login)
			m.loginForm.focus = loginPassword
			m.registerForm = newRegisterForm()
			m.dialog = dialogNone
			m.showAlertWithHighlight(
				alertNotice,
				"Registration successful",
				"Registered as "+msg.login+". Please log in.",
				msg.login,
				dialogLogin,
			)
		}
		return m, nil
	case logoutResultMsg:
		if !m.logoutRequest.accepts(msg.requestID) {
			return m, nil
		}
		m.logoutRequest.finish()
		if msg.err != nil {
			m.showAlert(alertError, "Logout failed", cleanLogoutError(msg.err), dialogNone)
		} else {
			m.auth = authSession{state: authGuest}
			m.dialog = dialogNone
			m.showAlert(alertNotice, "Logged out", "You are now logged out", dialogNone)
		}
		return m, nil
	case serverStatusResultMsg:
		if !m.statusRequest.accepts(msg.requestID) {
			return m, nil
		}
		m.statusRequest.finish()
		m.activeButton = 1
		if msg.err != nil {
			m.statusState = serverStatusFailed
			m.statusValue = ""
			m.statusFailure = describeServerStatusError(msg.err)
		} else {
			m.statusState = serverStatusReady
			m.statusValue = msg.status
			m.statusFailure = serverStatusFailure{}
		}
		return m, nil
	case openURLResultMsg:
		if msg.err != nil {
			m.showAlert(alertError, "Unable to open link", cleanOpenURLError(msg.err), dialogAbout)
		}
		return m, nil
	case tea.PasteMsg:
		if m.alert != alertNone {
			return m, nil
		}
		switch {
		case m.dialog == dialogConfig && m.configForm.focus <= configCacheDir:
			m.configForm.insert(msg.Content)
		case m.dialog == dialogLogin && !m.loginRequest.pending && m.loginForm.focus <= loginPassword:
			m.loginForm.insert(msg.Content)
		case m.dialog == dialogRegister && !m.registerRequest.pending && m.registerForm.focus <= registerRepeatPassword:
			m.registerForm.insert(msg.Content)
		}
		return m, nil
	case tea.KeyPressMsg:
		key := msg.String()
		if key == "ctrl+c" || key == "ctrl+q" {
			m.cancelAllRequests()
			return m, tea.Quit
		}
		if m.width < minimumWidth || m.height < minimumHeight {
			return m, nil
		}
		if m.alert != alertNone {
			if key == "enter" || key == "esc" {
				m.dismissAlert()
			}
			return m, nil
		}

		definitions := menuDefinitions(m.dialog, m.auth.authenticated())
		if index, ok := menuIndexByAltKey(definitions, key); ok {
			return m.openMenu(index)
		}

		if key == "f10" {
			if m.menuFocused || m.dropdownOpen {
				m.closeMenu()
				return m, nil
			}
			m.openCurrentMenu(definitions)
			return m, nil
		}

		if m.menuFocused || m.dropdownOpen {
			return m.updateMenu(key)
		}

		switch m.dialog {
		case dialogConfig:
			return m.updateConfig(key)
		case dialogLogin:
			return m.updateLogin(key)
		case dialogRegister:
			return m.updateRegister(key)
		}

		switch key {
		case "tab", "shift+tab", "right", "left":
			switch m.dialog {
			case dialogAbout:
				m.activeButton = 1 - m.activeButton
			case dialogServerStatus:
				m.moveServerStatusButton()
			}
		case "enter":
			return m.activateDialogButton()
		case "esc":
			m.closeActiveDialog()
		}
	}

	return m, nil
}
