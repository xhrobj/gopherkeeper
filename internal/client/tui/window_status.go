package tui

import "charm.land/lipgloss/v2"

const (
	serverStatusButtonRow = 6
	serverStatusButtonGap = 3
)

func serverStatusWindowWidth(screenWidth int) int {
	return clamp(screenWidth-18, 48, 62)
}

func renderServerStatusWindow(
	t theme,
	width int,
	address string,
	state serverStatusState,
	health string,
	failure serverStatusFailure,
	activeButton int,
) string {
	statusValue := "Checking..."
	detailLabel := "Health"
	detailValue := "pending"

	bodyStyle := t.windowBody
	labelStyle := t.label
	statusStyle := t.controlsKey
	detailStyle := t.label
	buttonStyle := t.button
	buttonActiveStyle := t.buttonActive
	buttonDisabledStyle := t.buttonDisabled
	buttonDisabledActiveStyle := t.buttonDisabledActive

	switch state {
	case serverStatusReady:
		statusValue = "Available"
		detailValue = health
	case serverStatusFailed:
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
		labelStyle = t.errorText
		statusStyle = t.errorTitle
		detailStyle = t.errorText
		buttonStyle = t.errorText.Padding(0, 1)
		buttonActiveStyle = t.errorButton
		buttonDisabledStyle = t.errorText.Foreground(lipgloss.Color("#AAAAAA")).Padding(0, 1)
		buttonDisabledActiveStyle = t.errorText.
			Foreground(lipgloss.Color("#AA0000")).
			Background(lipgloss.Color("#AAAAAA")).
			Padding(0, 1)
	}

	contentWidth := max(1, width-4)
	rows := []string{
		renderServerStatusRow(bodyStyle, labelStyle, contentWidth, "Address", address, detailStyle),
		renderServerStatusRow(bodyStyle, labelStyle, contentWidth, "Status", statusValue, statusStyle),
		renderServerStatusRow(bodyStyle, labelStyle, contentWidth, detailLabel, detailValue, detailStyle),
		bodyStyle.Width(contentWidth).Render(""),
		renderStatusButtons(
			bodyStyle,
			buttonStyle,
			buttonActiveStyle,
			buttonDisabledStyle,
			buttonDisabledActiveStyle,
			contentWidth,
			activeButton,
			state == serverStatusChecking,
		),
	}

	title := t.windowTitle.Width(width).Render("Server Status")
	body := bodyStyle.
		Width(width).
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

func renderStatusButtons(
	background lipgloss.Style,
	buttonStyle lipgloss.Style,
	activeStyle lipgloss.Style,
	disabledStyle lipgloss.Style,
	disabledActiveStyle lipgloss.Style,
	width int,
	activeButton int,
	retryDisabled bool,
) string {
	labels := []string{"< Retry >", "< OK >"}
	rendered := make([]string, 0, len(labels))
	for index, label := range labels {
		style := buttonStyle
		if index == 0 && retryDisabled {
			style = disabledStyle
			if activeButton == index {
				style = disabledActiveStyle
			}
		} else if index == activeButton {
			style = activeStyle
		}
		rendered = append(rendered, style.Render(label))
	}

	buttonsWidth := lipgloss.Width(rendered[0]) + serverStatusButtonGap + lipgloss.Width(rendered[1])
	left := max(0, (width-buttonsWidth)/2)
	right := max(0, width-buttonsWidth-left)

	return background.Width(left).Render("") +
		rendered[0] +
		background.Width(serverStatusButtonGap).Render("") +
		rendered[1] +
		background.Width(right).Render("")
}
