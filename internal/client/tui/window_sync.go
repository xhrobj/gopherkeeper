package tui

import (
	"strconv"
	"strings"

	"charm.land/lipgloss/v2"
)

const (
	syncLabelWidth      = 9
	syncPasswordRow     = 4
	syncButtonRow       = 9
	syncResultButtonRow = 7
)

func syncWindowWidth(screenWidth int) int {
	return clamp(screenWidth-24, 50, 62)
}

func syncResultWindowWidth(screenWidth int) int {
	return clamp(screenWidth-36, 36, 44)
}

func renderSyncWindow(t theme, width int, login string, form syncForm, pending, blocked bool, spinnerFrame string) string {
	contentWidth := max(1, width-4)
	inputWidth := max(16, contentWidth-syncLabelWidth-2)
	labelGap := t.recordFormBody.Width(2).Render("")

	userLine := t.recordFormLabel.Width(syncLabelWidth).Render("User") + labelGap +
		t.recordFormReadOnly.Width(inputWidth).Render(fitSingleLine(login, inputWidth))
	passwordLine := t.recordFormLabel.Width(syncLabelWidth).Render("Password") + labelGap +
		renderTextField(t, form.password, inputWidth, form.focus == syncPassword && !blocked)

	rows := []string{
		userLine,
		t.recordFormBody.Width(contentWidth).Render(""),
		passwordLine,
		t.recordFormBody.Width(contentWidth).Render(""),
		t.recordFormInfo.Width(contentWidth).AlignHorizontal(lipgloss.Center).Render("Updates the encrypted local cache"),
		t.recordFormInfo.Width(contentWidth).AlignHorizontal(lipgloss.Center).Render("with the current Server data."),
		t.recordFormBody.Width(contentWidth).Render(""),
		renderSyncButtons(t, contentWidth, form.focus, !form.canSubmit(), blocked),
	}

	title := renderWindowTitle(t.windowTitle, width, "Synchronize Local Cache", spinnerFrame, pending)
	body := t.recordFormBody.Width(width).Padding(1, 2).Render(strings.Join(rows, "\n"))

	return lipgloss.JoinVertical(lipgloss.Left, title, body)
}

func syncButtonsLayout(t theme, width int, focus syncFocus, submitDisabled, blocked bool) buttonRowLayout {
	buttons := []struct {
		label    string
		active   bool
		disabled bool
	}{
		{label: "< Sync >", active: focus == syncSubmit, disabled: blocked || submitDisabled},
		{label: "< Cancel >", active: focus == syncCancel, disabled: blocked},
	}

	styled := make([]styledButton, 0, len(buttons))
	for _, button := range buttons {
		style := t.recordFormButton
		switch {
		case button.disabled && button.active:
			style = t.recordFormButtonDisabledActive
		case button.disabled:
			style = t.recordFormButtonDisabled
		case button.active:
			style = t.recordFormButtonActive
		}
		styled = append(styled, styledButton{label: button.label, style: style})
	}

	return centeredButtonRowLayout(t.recordFormBody, width, 3, styled)
}

func renderSyncButtons(t theme, width int, focus syncFocus, submitDisabled, blocked bool) string {
	return syncButtonsLayout(t, width, focus, submitDisabled, blocked).content
}

func renderSyncResultWindow(t theme, width int, result SyncSummary, blocked bool) string {
	contentWidth := max(1, width-4)
	rows := []string{
		renderSyncResultRow(t, contentWidth, "Added", result.Added),
		renderSyncResultRow(t, contentWidth, "Updated", result.Updated),
		renderSyncResultRow(t, contentWidth, "Removed", result.Removed),
		renderSyncResultRow(t, contentWidth, "Unchanged", result.Unchanged),
		t.recordFormBody.Width(contentWidth).Render(""),
		renderSyncResultButton(t, contentWidth, blocked),
	}

	title := renderWindowTitle(t.windowTitle, width, "Synchronization Complete", "", false)
	body := t.recordFormBody.Width(width).Padding(1, 2).Render(strings.Join(rows, "\n"))

	return lipgloss.JoinVertical(lipgloss.Left, title, body)
}

func renderSyncResultRow(t theme, width int, label string, value int) string {
	labelPart := t.recordFormInfo.Width(12).Render(label + ":")
	valuePart := t.recordFormBody.Width(max(1, width-12)).Render(strconv.Itoa(value))

	return labelPart + valuePart
}

func syncResultButtonLayout(t theme, width int, blocked bool) buttonRowLayout {
	style := t.recordFormButtonActive
	if blocked {
		style = t.recordFormButtonDisabledActive
	}

	return singleStyledButtonLayout(t.recordFormBody, style, width, "< OK >")
}

func renderSyncResultButton(t theme, width int, blocked bool) string {
	return syncResultButtonLayout(t, width, blocked).content
}
