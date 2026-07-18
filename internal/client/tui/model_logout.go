package tui

import tea "charm.land/bubbletea/v2"

func (m model) startLogout() (tea.Model, tea.Cmd) {
	requestCtx, requestID := m.logoutRequest.begin(m.ctx)
	m.dialog = dialogNone
	return m, logoutCommand(requestCtx, m.backend, requestID)
}
