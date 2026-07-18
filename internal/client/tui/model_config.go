package tui

import (
	"strings"

	tea "charm.land/bubbletea/v2"
)

func (m model) updateConfig(key string) (tea.Model, tea.Cmd) {
	switch key {
	case "esc":
		m.dialog = dialogNone
		m.configForm = newConfigForm(m.config)
	case "tab", "down":
		m.configForm.move(1)
	case "shift+tab", "up":
		m.configForm.move(-1)
	case "left":
		if m.configForm.focus >= configSave {
			m.configForm.move(-1)
		} else {
			m.configForm.moveCursor(-1)
		}
	case "right":
		if m.configForm.focus >= configSave {
			m.configForm.move(1)
		} else {
			m.configForm.moveCursor(1)
		}
	case "home":
		m.configForm.moveCursorToStart()
	case "end":
		m.configForm.moveCursorToEnd()
	case "backspace":
		m.configForm.backspace()
	case "delete":
		m.configForm.delete()
	case "enter":
		if m.configForm.focus < configSave {
			m.configForm.move(1)
			return m, nil
		}
		return m.activateConfig()
	default:
		m.configForm.insertKey(key)
	}

	return m, nil
}

func (m model) activateConfig() (tea.Model, tea.Cmd) {
	switch m.configForm.focus {
	case configSave:
		candidate := m.configForm.config()
		candidate.Address = strings.TrimSpace(candidate.Address)
		if candidate.Address == "" {
			m.configForm.focus = configAddress
			m.configForm.errorMessage = "Server address is required"
			return m, nil
		}
		if m.configFile != "" && m.saveConfig != nil {
			if err := m.saveConfig(m.configFile, candidate); err != nil {
				m.configForm.errorMessage = "Unable to save config file"
				return m, nil
			}
		}
		changed := candidate != m.config
		m.config = candidate
		if changed {
			m.cancelAllRequests()
			m.backend = m.backendFactory(m.config)
		}
		m.dialog = dialogNone
		m.configForm = newConfigForm(m.config)
		if changed {
			m.statusState = serverStatusIdle
			m.statusValue = ""
			m.statusFailure = serverStatusFailure{}
			return m, m.beginCurrentUserCheck()
		}
	case configCancel:
		m.dialog = dialogNone
		m.configForm = newConfigForm(m.config)
	}

	return m, nil
}
