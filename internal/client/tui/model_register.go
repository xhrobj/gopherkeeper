package tui

import (
	"strings"

	tea "charm.land/bubbletea/v2"
)

func (m model) updateRegister(key string) (tea.Model, tea.Cmd) {
	if m.registerRequest.pending {
		switch key {
		case "esc":
			m.clearRegisterForm()
			m.dialog = dialogNone
		case "tab", "shift+tab", "up", "down", "left", "right":
			m.registerForm.focus = registerClose
		case "enter":
			if m.registerForm.focus == registerClose {
				m.clearRegisterForm()
				m.dialog = dialogNone
			}
		}

		return m, nil
	}

	submitDisabled := !m.registerForm.canSubmit()

	switch key {
	case "esc":
		m.clearRegisterForm()
		m.dialog = dialogNone
	case "tab", "down":
		m.registerForm.move(1, submitDisabled)
	case "shift+tab", "up":
		m.registerForm.move(-1, submitDisabled)
	case "left":
		if m.registerForm.focus >= registerSubmit {
			m.registerForm.move(-1, submitDisabled)
		} else if !m.registerRequest.pending {
			m.registerForm.moveCursor(-1)
		}
	case "right":
		if m.registerForm.focus >= registerSubmit {
			m.registerForm.move(1, submitDisabled)
		} else if !m.registerRequest.pending {
			m.registerForm.moveCursor(1)
		}
	case "home":
		if !m.registerRequest.pending {
			m.registerForm.moveCursorToStart()
		}
	case "end":
		if !m.registerRequest.pending {
			m.registerForm.moveCursorToEnd()
		}
	case "backspace":
		if !m.registerRequest.pending {
			m.registerForm.backspace()
		}
	case "delete":
		if !m.registerRequest.pending {
			m.registerForm.delete()
		}
	case "enter":
		return m.activateRegister()
	default:
		if !m.registerRequest.pending {
			m.registerForm.insertKey(key)
		}
	}

	return m, nil
}

func (m model) activateRegister() (tea.Model, tea.Cmd) {
	switch m.registerForm.focus {
	case registerSubmit:
		if m.registerRequest.pending || !m.registerForm.canSubmit() {
			return m, nil
		}
		return m.startRegister(strings.TrimSpace(m.registerForm.login.value), m.registerForm.password.value)
	case registerClose:
		m.clearRegisterForm()
		m.dialog = dialogNone
	default:
		if !m.registerRequest.pending {
			m.registerForm.move(1, !m.registerForm.canSubmit())
		}
	}

	return m, nil
}

func (m model) startRegister(userName, password string) (tea.Model, tea.Cmd) {
	requestCtx, requestID := m.registerRequest.begin(m.ctx)
	return m, registerCommand(requestCtx, m.backend, requestID, userName, password)
}

func (m *model) clearRegisterForm() {
	m.cancelRegisterRequest()
	m.registerForm = newRegisterForm()
}
