package tui

import (
	"context"
	"errors"
	"strings"

	tea "charm.land/bubbletea/v2"
	"github.com/xhrobj/gopherkeeper/internal/client/usecase"
	domainmodel "github.com/xhrobj/gopherkeeper/internal/model"
)

type authState int

const (
	authUnknown authState = iota
	authGuest
	authAuthenticated
)

type authSession struct {
	state authState
	login string
}

func (session authSession) authenticated() bool {
	return session.state == authAuthenticated
}

type currentUserCheckMode int

const (
	currentUserCheckRestore currentUserCheckMode = iota
	currentUserCheckManual
)

type currentUserResultMsg struct {
	requestID uint64
	login     string
	err       error
}

func currentUserCommand(
	ctx context.Context,
	backend Backend,
	requestID uint64,
) tea.Cmd {
	return func() tea.Msg {
		login, err := backend.CurrentUser(ctx)
		return currentUserResultMsg{requestID: requestID, login: login, err: err}
	}
}

func isNotLoggedIn(err error) bool {
	return errors.Is(err, usecase.ErrNotLoggedIn)
}

func cleanCurrentUserError(err error) string {
	if err == nil {
		return "Unknown session error"
	}

	if errors.Is(err, context.Canceled) {
		return "Session check canceled"
	}

	return cleanFailureMessage(err, "Unable to check the current session")
}

type loginResultMsg struct {
	requestID uint64
	login     string
	err       error
}

func loginCommand(
	ctx context.Context,
	backend Backend,
	requestID uint64,
	userName string,
	password string,
) tea.Cmd {
	return func() tea.Msg {
		canonicalLogin, err := backend.Login(ctx, userName, password)
		return loginResultMsg{requestID: requestID, login: canonicalLogin, err: err}
	}
}

func cleanLoginError(err error) string {
	if err == nil {
		return "Unknown login error"
	}

	if errors.Is(err, context.Canceled) {
		return "Login canceled"
	}

	if errors.Is(err, domainmodel.ErrInvalidCredentials) {
		return "Invalid login or password"
	}

	return cleanFailureMessage(err, "Unable to log in")
}

type registerResultMsg struct {
	requestID uint64
	login     string
	err       error
}

func registerCommand(
	ctx context.Context,
	backend Backend,
	requestID uint64,
	login string,
	password string,
) tea.Cmd {
	return func() tea.Msg {
		canonicalLogin, err := backend.Register(ctx, login, password)
		return registerResultMsg{requestID: requestID, login: canonicalLogin, err: err}
	}
}

func cleanRegisterError(err error) string {
	if err == nil {
		return "Unknown registration error"
	}

	if errors.Is(err, context.Canceled) {
		return "Registration canceled"
	}

	if errors.Is(err, domainmodel.ErrLoginAlreadyExists) {
		return strings.TrimSpace(err.Error())
	}

	return cleanFailureMessage(err, "Unable to register")
}

type logoutResultMsg struct {
	requestID uint64
	err       error
}

func logoutCommand(
	ctx context.Context,
	backend Backend,
	requestID uint64,
) tea.Cmd {
	return func() tea.Msg {
		return logoutResultMsg{requestID: requestID, err: backend.Logout(ctx)}
	}
}

func cleanLogoutError(err error) string {
	if err == nil {
		return "Unknown logout error"
	}

	if errors.Is(err, context.Canceled) {
		return "Logout canceled"
	}

	return cleanFailureMessage(err, "Unable to log out")
}

func (m model) updateLogin(key string) (tea.Model, tea.Cmd) {
	if m.operations.pending(operationLogin) {
		return m, nil
	}

	submitDisabled := !m.authentication.loginForm.canSubmit()

	switch key {
	case "esc":
		m.clearLoginForm()
		m.dialog = dialogNone
	case "tab", "down":
		m.authentication.loginForm.move(1, submitDisabled)
	case "shift+tab", "up":
		m.authentication.loginForm.move(-1, submitDisabled)
	case "left":
		if m.authentication.loginForm.focus >= loginSubmit {
			m.authentication.loginForm.move(-1, submitDisabled)
		} else if !m.operations.pending(operationLogin) {
			m.authentication.loginForm.moveCursor(-1)
		}
	case "right":
		if m.authentication.loginForm.focus >= loginSubmit {
			m.authentication.loginForm.move(1, submitDisabled)
		} else if !m.operations.pending(operationLogin) {
			m.authentication.loginForm.moveCursor(1)
		}
	case "home":
		if !m.operations.pending(operationLogin) {
			m.authentication.loginForm.moveCursorToStart()
		}
	case "end":
		if !m.operations.pending(operationLogin) {
			m.authentication.loginForm.moveCursorToEnd()
		}
	case "backspace":
		if !m.operations.pending(operationLogin) {
			m.authentication.loginForm.backspace()
		}
	case "delete":
		if !m.operations.pending(operationLogin) {
			m.authentication.loginForm.delete()
		}
	case "enter":
		return m.activateLogin()
	default:
		if !m.operations.pending(operationLogin) {
			m.authentication.loginForm.insertKey(key)
		}
	}

	return m, nil
}

func (m model) activateLogin() (tea.Model, tea.Cmd) {
	switch m.authentication.loginForm.focus {
	case loginSubmit:
		if m.operations.pending(operationLogin) || !m.authentication.loginForm.canSubmit() {
			return m, nil
		}
		return m.startLogin(strings.TrimSpace(m.authentication.loginForm.login.value), m.authentication.loginForm.password.value)
	case loginClose:
		m.clearLoginForm()
		m.dialog = dialogNone
	default:
		if !m.operations.pending(operationLogin) {
			m.authentication.loginForm.move(1, !m.authentication.loginForm.canSubmit())
		}
	}

	return m, nil
}

func (m model) startLogin(userName, password string) (tea.Model, tea.Cmd) {
	requestCtx, requestID := m.operations.begin(m.ctx, operationLogin)
	return m, m.operationCommand(operationLogin, loginCommand(requestCtx, m.backend, requestID, userName, password))
}

func (m *model) clearLoginForm() {
	m.operations.cancel(operationLogin)
	m.authentication.loginForm = newLoginForm()
}

func (m model) updateRegister(key string) (tea.Model, tea.Cmd) {
	if m.operations.pending(operationRegister) {
		return m, nil
	}

	submitDisabled := !m.authentication.registerForm.canSubmit()

	switch key {
	case "esc":
		m.clearRegisterForm()
		m.dialog = dialogNone
	case "tab", "down":
		m.authentication.registerForm.move(1, submitDisabled)
	case "shift+tab", "up":
		m.authentication.registerForm.move(-1, submitDisabled)
	case "left":
		if m.authentication.registerForm.focus >= registerSubmit {
			m.authentication.registerForm.move(-1, submitDisabled)
		} else if !m.operations.pending(operationRegister) {
			m.authentication.registerForm.moveCursor(-1)
		}
	case "right":
		if m.authentication.registerForm.focus >= registerSubmit {
			m.authentication.registerForm.move(1, submitDisabled)
		} else if !m.operations.pending(operationRegister) {
			m.authentication.registerForm.moveCursor(1)
		}
	case "home":
		if !m.operations.pending(operationRegister) {
			m.authentication.registerForm.moveCursorToStart()
		}
	case "end":
		if !m.operations.pending(operationRegister) {
			m.authentication.registerForm.moveCursorToEnd()
		}
	case "backspace":
		if !m.operations.pending(operationRegister) {
			m.authentication.registerForm.backspace()
		}
	case "delete":
		if !m.operations.pending(operationRegister) {
			m.authentication.registerForm.delete()
		}
	case "enter":
		return m.activateRegister()
	default:
		if !m.operations.pending(operationRegister) {
			m.authentication.registerForm.insertKey(key)
		}
	}

	return m, nil
}

func (m model) activateRegister() (tea.Model, tea.Cmd) {
	switch m.authentication.registerForm.focus {
	case registerSubmit:
		if m.operations.pending(operationRegister) || !m.authentication.registerForm.canSubmit() {
			return m, nil
		}
		return m.startRegister(strings.TrimSpace(m.authentication.registerForm.login.value), m.authentication.registerForm.password.value)
	case registerClose:
		m.clearRegisterForm()
		m.dialog = dialogNone
	default:
		if !m.operations.pending(operationRegister) {
			m.authentication.registerForm.move(1, !m.authentication.registerForm.canSubmit())
		}
	}

	return m, nil
}

func (m model) startRegister(userName, password string) (tea.Model, tea.Cmd) {
	requestCtx, requestID := m.operations.begin(m.ctx, operationRegister)
	return m, m.operationCommand(operationRegister, registerCommand(requestCtx, m.backend, requestID, userName, password))
}

func (m *model) clearRegisterForm() {
	m.operations.cancel(operationRegister)
	m.authentication.registerForm = newRegisterForm()
}

func (m model) startLogout() (tea.Model, tea.Cmd) {
	m.operations.cancel(operationCurrentUser)
	requestCtx, requestID := m.operations.begin(m.ctx, operationLogout)

	m.closeMenu()

	m.dialog = dialogNone
	m.activeButton = 0

	return m, m.operationCommand(operationLogout, logoutCommand(requestCtx, m.backend, requestID))
}

const (
	sessionExpiredTitle   = "(+_+)~ Session Expired"
	sessionExpiredMessage = "Please login again"
)

func (m *model) handleSessionExpired() {
	m.cancelAllRequests()
	m.closeMenu()
	m.clearRecordState()

	m.dialog = dialogNone
	m.authentication.session = authSession{state: authGuest}
	m.activeMenu = int(menuSystem)
	m.selectedItem = 0
	m.activeButton = 0

	m.showAlert(alertError, sessionExpiredTitle, sessionExpiredMessage, dialogNone)
}
