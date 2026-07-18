package tui

import (
	"strings"

	tea "charm.land/bubbletea/v2"
)

func (m model) updateLogin(key string) (tea.Model, tea.Cmd) {
	if m.loginRequest.pending {
		switch key {
		case "esc":
			m.clearLoginForm()
			m.dialog = dialogNone
		case "tab", "shift+tab", "up", "down", "left", "right":
			m.loginForm.focus = loginClose
		case "enter":
			if m.loginForm.focus == loginClose {
				m.clearLoginForm()
				m.dialog = dialogNone
			}
		}
		return m, nil
	}

	submitDisabled := !m.loginForm.canSubmit()
	switch key {
	case "esc":
		m.clearLoginForm()
		m.dialog = dialogNone
	case "tab", "down":
		m.loginForm.move(1, submitDisabled)
	case "shift+tab", "up":
		m.loginForm.move(-1, submitDisabled)
	case "left":
		if m.loginForm.focus >= loginSubmit {
			m.loginForm.move(-1, submitDisabled)
		} else if !m.loginRequest.pending {
			m.loginForm.moveCursor(-1)
		}
	case "right":
		if m.loginForm.focus >= loginSubmit {
			m.loginForm.move(1, submitDisabled)
		} else if !m.loginRequest.pending {
			m.loginForm.moveCursor(1)
		}
	case "home":
		if !m.loginRequest.pending {
			m.loginForm.moveCursorToStart()
		}
	case "end":
		if !m.loginRequest.pending {
			m.loginForm.moveCursorToEnd()
		}
	case "backspace":
		if !m.loginRequest.pending {
			m.loginForm.backspace()
		}
	case "delete":
		if !m.loginRequest.pending {
			m.loginForm.delete()
		}
	case "enter":
		return m.activateLogin()
	default:
		if !m.loginRequest.pending {
			m.loginForm.insertKey(key)
		}
	}

	return m, nil
}

func (m model) activateLogin() (tea.Model, tea.Cmd) {
	switch m.loginForm.focus {
	case loginSubmit:
		if m.loginRequest.pending || !m.loginForm.canSubmit() {
			return m, nil
		}
		return m.startLogin(strings.TrimSpace(m.loginForm.login.value), m.loginForm.password.value)
	case loginClose:
		m.clearLoginForm()
		m.dialog = dialogNone
	default:
		if !m.loginRequest.pending {
			m.loginForm.move(1, !m.loginForm.canSubmit())
		}
	}

	return m, nil
}

func (m model) startLogin(userName, password string) (tea.Model, tea.Cmd) {
	requestCtx, requestID := m.loginRequest.begin(m.ctx)
	return m, loginCommand(requestCtx, m.backend, requestID, userName, password)
}

func (m *model) clearLoginForm() {
	m.cancelLoginRequest()
	m.loginForm = newLoginForm()
}
