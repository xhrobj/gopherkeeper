package tui

import (
	"context"
	"time"

	tea "charm.land/bubbletea/v2"
	"charm.land/lipgloss/v2"
	"github.com/xhrobj/gopherkeeper/internal/client/failure"
)

type serverStatusResultMsg struct {
	requestID uint64
	status    string
	err       error
}

type serverStatusFailure struct {
	status string
	reason string
}

type serverStatusWindowOptions struct {
	width        int
	address      string
	state        serverStatusState
	health       string
	failure      serverStatusFailure
	activeButton int
	pending      bool
	blocked      bool
	spinnerFrame string
}

type statusButtonStyles struct {
	background     lipgloss.Style
	button         lipgloss.Style
	active         lipgloss.Style
	disabled       lipgloss.Style
	disabledActive lipgloss.Style
}

func serverStatusCmd(
	ctx context.Context,
	backend Backend,
	requestID uint64,
	minimumDuration time.Duration,
) tea.Cmd {
	return func() tea.Msg {
		startedAt := time.Now()
		status, err := backend.Health(ctx)
		if remaining := minimumDuration - time.Since(startedAt); remaining > 0 {
			timer := time.NewTimer(remaining)
			defer timer.Stop()
			select {
			case <-timer.C:
			case <-ctx.Done():
			}
		}
		return serverStatusResultMsg{requestID: requestID, status: status, err: err}
	}
}

func describeServerStatusError(err error) serverStatusFailure {
	if err == nil {
		return serverStatusFailure{}
	}

	kind := failure.KindOf(err)
	switch kind {
	case failure.Unavailable, failure.HostNotFound, failure.NetworkUnreachable, failure.Timeout:
		return serverStatusFailure{status: "Unreachable", reason: serverStatusReason(err, kind)}
	case failure.TLSCertificate, failure.TLSHandshake, failure.HTTPSRequired:
		return serverStatusFailure{status: "TLS error", reason: failure.Reason(kind)}
	default:
		return serverStatusFailure{status: "Connection error", reason: cleanServerStatusReason(err)}
	}
}

func serverStatusReason(err error, kind failure.Kind) string {
	message := failure.Message(err)

	if message == "" || message == failure.Reason(failure.Unknown) {
		return failure.Reason(kind)
	}

	return message
}

func cleanServerStatusReason(err error) string {
	return cleanFailureMessage(err, "Unknown connection error")
}

func (m model) startServerStatusCheck() (tea.Model, tea.Cmd) {
	m.activeButton = 0

	requestCtx, requestID := m.operations.begin(m.operationDone, operationServerStatus)

	return m, m.operationCommand(operationServerStatus, serverStatusCmd(
		requestCtx,
		m.backend,
		requestID,
		m.statusMinDuration,
	))
}

func (m *model) moveServerStatusButton() {
	m.activeButton = 1 - m.activeButton
}

const (
	serverStatusButtonRow = 6
	serverStatusButtonGap = 3
)

func serverStatusWindowWidth(screenWidth int) int {
	return clamp(screenWidth-18, 48, 62)
}

func renderServerStatusWindow(t theme, options serverStatusWindowOptions) string {
	statusValue := ""
	detailLabel := "Health"
	detailValue := ""
	failure := options.failure

	bodyStyle := t.aboutBody
	labelStyle := t.aboutLabel
	statusStyle := t.aboutTitle
	detailStyle := t.aboutValue

	switch {
	case options.pending:
		statusValue = "Checking..."
		detailValue = "pending"
	case options.state == serverStatusReady:
		statusValue = "Available"
		detailValue = options.health
	case options.state == serverStatusFailed:
		if failure.status == "" {
			failure = serverStatusFailure{
				status: "Connection error",
				reason: "Unknown connection error",
			}
		}
		statusValue = failure.status
		detailLabel = "Reason"
		detailValue = failure.reason
		bodyStyle = t.errorBody
		labelStyle = t.errorLabel
		statusStyle = t.errorText.Bold(true)
		detailStyle = t.errorText
	}

	contentWidth := max(1, options.width-4)
	rows := []string{
		renderServerStatusRow(bodyStyle, labelStyle, contentWidth, "Address", options.address, detailStyle),
		renderServerStatusRow(bodyStyle, labelStyle, contentWidth, "Status", statusValue, statusStyle),
		renderServerStatusRow(bodyStyle, labelStyle, contentWidth, detailLabel, detailValue, detailStyle),
		bodyStyle.Width(contentWidth).Render(""),
		serverStatusButtonsLayout(
			t,
			contentWidth,
			options.state,
			options.pending,
			options.activeButton,
			options.blocked,
		).content,
	}

	title := renderWindowTitle(t.windowTitle, options.width, "Server Status", options.spinnerFrame, options.pending)
	body := bodyStyle.
		Width(options.width).
		Padding(1, 2).
		Render(lipgloss.JoinVertical(lipgloss.Left, rows...))

	return lipgloss.JoinVertical(lipgloss.Left, title, body)
}

func renderServerStatusRow(
	background lipgloss.Style,
	labelStyle lipgloss.Style,
	width int,
	label string,
	value string,
	valueStyle lipgloss.Style,
) string {
	const labelWidth = 8

	labelPart := labelStyle.Width(labelWidth).Render(label)
	gap := background.Render("  ")
	remaining := max(1, width-labelWidth-2)
	valuePart := valueStyle.Width(remaining).Render(fitSingleLine(value, remaining))

	return labelPart + gap + valuePart
}

func serverStatusButtonsLayout(
	t theme,
	width int,
	state serverStatusState,
	pending bool,
	activeButton int,
	blocked bool,
) buttonRowLayout {
	background := t.aboutBody
	buttonStyle := t.aboutButton
	activeStyle := t.aboutButtonActive
	disabledStyle := t.aboutButtonDisabled
	disabledActiveStyle := t.aboutButtonDisabledActive

	if state == serverStatusFailed && !pending {
		background = t.errorBody
		buttonStyle = t.errorText.Padding(0, 1)
		activeStyle = t.errorButton
		disabledStyle = t.errorButtonDisabled
		disabledActiveStyle = t.errorButtonDisabledActive
	}

	return statusButtonsLayout(
		statusButtonStyles{
			background:     background,
			button:         buttonStyle,
			active:         activeStyle,
			disabled:       disabledStyle,
			disabledActive: disabledActiveStyle,
		},
		width,
		activeButton,
		blocked,
	)
}

func statusButtonsLayout(
	styles statusButtonStyles,
	width int,
	activeButton int,
	blocked bool,
) buttonRowLayout {
	labels := []string{"< Check >", okButtonLabel}
	buttons := make([]styledButton, len(labels))

	for index, label := range labels {
		style := styles.button
		if blocked {
			style = styles.disabled
			if index == activeButton {
				style = styles.disabledActive
			}
		} else if index == activeButton {
			style = styles.active
		}
		buttons[index] = styledButton{label: label, style: style}
	}

	return centeredButtonRowLayout(styles.background, width, serverStatusButtonGap, buttons)
}
