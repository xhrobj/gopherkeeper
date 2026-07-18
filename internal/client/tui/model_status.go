package tui

import tea "charm.land/bubbletea/v2"

func (m model) startServerStatusCheck() (tea.Model, tea.Cmd) {
	m.statusState = serverStatusChecking
	m.statusValue = ""
	m.statusFailure = serverStatusFailure{}

	requestCtx, requestID := m.statusRequest.begin(m.ctx)

	return m, serverStatusCmd(
		requestCtx,
		m.backend,
		requestID,
		m.statusMinDuration,
	)
}

func (m *model) moveServerStatusButton() {
	if m.statusState == serverStatusChecking {
		if m.activeButton == 0 {
			m.activeButton = 1
		}
		return
	}
	m.activeButton = 1 - m.activeButton
}
