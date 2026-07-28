package tui

import (
	"strconv"
	"strings"

	"charm.land/lipgloss/v2"
)

const syncLabelWidth = 9

type syncWindowLayout struct {
	content      string
	fieldBounds  []layoutBounds
	buttonBounds []layoutBounds
}

type syncResultWindowLayout struct {
	content      string
	buttonBounds []layoutBounds
}

func syncWindowWidth(screenWidth int) int {
	return clamp(screenWidth-24, 50, 62)
}

func syncResultWindowWidth(screenWidth int) int {
	return clamp(screenWidth-36, 36, 44)
}

func newSyncWindowLayout(
	t theme,
	width int,
	login string,
	form syncForm,
	pending, blocked bool,
	spinnerFrame string,
) syncWindowLayout {
	const (
		horizontalPadding = 2
		verticalPadding   = 1
	)

	title := renderWindowTitle(t.windowTitle, width, "Synchronize Local Cache", spinnerFrame, pending)
	contentWidth := max(1, width-2*horizontalPadding)
	inputWidth := max(16, contentWidth-syncLabelWidth-2)
	labelGap := t.recordFormBody.Width(2).Render("")

	userLine := t.recordFormLabel.Width(syncLabelWidth).Render("User") + labelGap +
		t.recordFormReadOnly.Width(inputWidth).Render(fitSingleLine(login, inputWidth))
	passwordLine := t.recordFormLabel.Width(syncLabelWidth).Render("Password") + labelGap +
		renderTextField(t, form.password, inputWidth, form.focus == syncPassword && !blocked)

	rows := []string{
		userLine,
		t.recordFormBody.Width(contentWidth).Render(""),
	}

	passwordRow := lipgloss.Height(title) + verticalPadding + len(rows)
	rows = append(rows,
		passwordLine,
		t.recordFormBody.Width(contentWidth).Render(""),
		t.recordFormInfo.Width(contentWidth).AlignHorizontal(lipgloss.Center).Render("Updates the encrypted local cache"),
		t.recordFormInfo.Width(contentWidth).AlignHorizontal(lipgloss.Center).Render("with the current Server data."),
		t.recordFormBody.Width(contentWidth).Render(""),
	)

	buttonRow := len(rows)
	buttonLayout := syncButtonsLayout(t, contentWidth, form.focus, !form.canSubmit(), blocked).
		positioned(0, buttonRow)
	rows = append(rows, buttonLayout.content)

	body := t.recordFormBody.
		Width(width).
		Padding(verticalPadding, horizontalPadding).
		Render(strings.Join(rows, "\n"))

	return syncWindowLayout{
		content: lipgloss.JoinVertical(lipgloss.Left, title, body),
		fieldBounds: []layoutBounds{{
			x:      horizontalPadding + syncLabelWidth + 2,
			y:      passwordRow,
			width:  inputWidth,
			height: 1,
		}},
		buttonBounds: translateLayoutBounds(
			buttonLayout.bounds,
			horizontalPadding,
			lipgloss.Height(title)+verticalPadding,
		),
	}
}

func renderSyncWindow(t theme, width int, login string, form syncForm, pending, blocked bool, spinnerFrame string) string {
	return newSyncWindowLayout(t, width, login, form, pending, blocked, spinnerFrame).content
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

func newSyncResultWindowLayout(t theme, width int, result SyncSummary, blocked bool) syncResultWindowLayout {
	const (
		horizontalPadding = 2
		verticalPadding   = 1
	)

	contentWidth := max(1, width-2*horizontalPadding)
	rows := []string{
		renderSyncResultRow(t, contentWidth, "Added", result.Added),
		renderSyncResultRow(t, contentWidth, "Updated", result.Updated),
		renderSyncResultRow(t, contentWidth, "Removed", result.Removed),
		renderSyncResultRow(t, contentWidth, "Unchanged", result.Unchanged),
		t.recordFormBody.Width(contentWidth).Render(""),
	}
	buttonRow := len(rows)
	buttonLayout := syncResultButtonLayout(t, contentWidth, blocked).positioned(0, buttonRow)
	rows = append(rows, buttonLayout.content)

	title := renderWindowTitle(t.windowTitle, width, "Synchronization Complete", "", false)
	body := t.recordFormBody.
		Width(width).
		Padding(verticalPadding, horizontalPadding).
		Render(strings.Join(rows, "\n"))

	return syncResultWindowLayout{
		content: lipgloss.JoinVertical(lipgloss.Left, title, body),
		buttonBounds: translateLayoutBounds(
			buttonLayout.bounds,
			horizontalPadding,
			lipgloss.Height(title)+verticalPadding,
		),
	}
}

func renderSyncResultWindow(t theme, width int, result SyncSummary, blocked bool) string {
	return newSyncResultWindowLayout(t, width, result, blocked).content
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

	return singleStyledButtonLayout(t.recordFormBody, style, width, okButtonLabel)
}
