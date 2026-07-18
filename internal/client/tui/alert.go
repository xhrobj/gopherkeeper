package tui

func (m *model) showAlert(state alertState, title, message string, returnDialog dialogID) {
	m.showAlertWithHighlight(state, title, message, "", returnDialog)
}

func (m *model) showAlertWithHighlight(
	state alertState,
	title string,
	message string,
	highlight string,
	returnDialog dialogID,
) {
	m.alert = state
	m.alertTitle = title
	m.alertMessage = message
	m.alertHighlight = highlight
	m.alertReturnDialog = returnDialog
}

func (m *model) dismissAlert() {
	m.alert = alertNone
	m.alertTitle = ""
	m.alertMessage = ""
	m.alertHighlight = ""
	m.dialog = m.alertReturnDialog
	m.alertReturnDialog = dialogNone
}
