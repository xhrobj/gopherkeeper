package tui

import (
	"unicode"

	"github.com/xhrobj/gopherkeeper/internal/client/failure"
)

const (
	okButtonLabel    = "< OK >"
	closeButtonLabel = "< Close >"
	menuHintText     = "F10 = Menu"
)

func clamp(value, minimum, maximum int) int {
	return min(max(value, minimum), maximum)
}

func capitalizeFirst(value string) string {
	runes := []rune(value)
	if len(runes) == 0 {
		return value
	}

	runes[0] = unicode.ToUpper(runes[0])
	return string(runes)
}

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

func networkFailureReason(err error) (string, bool) {
	kind := failure.KindOf(err)

	switch kind {
	case failure.Unavailable,
		failure.HostNotFound,
		failure.NetworkUnreachable,
		failure.Timeout,
		failure.TLSCertificate,
		failure.TLSHandshake,
		failure.HTTPSRequired:
		return failure.Reason(kind), true
	default:
		return "", false
	}
}

func cleanFailureMessage(err error, fallback string) string {
	if reason, ok := networkFailureReason(err); ok {
		return reason
	}

	message := failure.Message(err)
	if message == "" ||
		(failure.KindOf(err) == failure.Unknown && message == failure.Reason(failure.Unknown)) {
		return fallback
	}

	return capitalizeFirst(message)
}
