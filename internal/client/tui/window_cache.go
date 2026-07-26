package tui

import (
	"strings"

	"charm.land/lipgloss/v2"
)

const (
	cacheBrowseLabelWidth    = 8
	cacheBrowseFirstFieldRow = 2
	cacheBrowseFieldRowStep  = 2
	cacheBrowseButtonGap     = 3
	cacheBrowseButtonRow     = 6
)

func cacheBrowseWindowWidth(screenWidth int) int {
	return clamp(screenWidth-18, 48, 62)
}

func renderCacheBrowseWindow(
	t theme,
	width int,
	form cacheBrowseForm,
	pending bool,
	blocked bool,
	spinnerFrame string,
) string {
	layout := newLabeledFieldColumnLayout(
		width,
		cacheBrowseLabelWidth,
		16,
		2,
		cacheBrowseFirstFieldRow,
		cacheBrowseFieldRowStep,
	)

	rows := []string{
		renderCacheBrowseField(t, "Login", form.login, layout.inputWidth, form.focus == cacheBrowseLogin && !blocked),
		t.recordFormBody.Width(layout.contentWidth).Render(""),
		renderCacheBrowseField(t, "Password", form.password, layout.inputWidth, form.focus == cacheBrowsePassword && !blocked),
		t.recordFormBody.Width(layout.contentWidth).Render(""),
		renderCacheBrowseButtons(t, layout.contentWidth, form.focus, !form.canSubmit(), blocked),
	}

	title := renderWindowTitle(t.windowTitle, width, "Open Local Cache", spinnerFrame, pending)
	body := t.recordFormBody.Width(width).Padding(1, 2).Render(strings.Join(rows, "\n"))

	return lipgloss.JoinVertical(lipgloss.Left, title, body)
}

func renderCacheBrowseField(t theme, label string, field textField, inputWidth int, active bool) string {
	labelPart := t.recordFormLabel.Width(cacheBrowseLabelWidth).Render(label)
	gap := t.recordFormBody.Width(2).Render("")

	return labelPart + gap + renderTextField(t, field, inputWidth, active)
}

func cacheBrowseButtonsLayout(
	t theme,
	width int,
	focus cacheBrowseFocus,
	submitDisabled bool,
	blocked bool,
) buttonRowLayout {
	return formButtonsLayout(theme{
		button:               t.recordFormButton,
		buttonActive:         t.recordFormButtonActive,
		buttonDisabled:       t.recordFormButtonDisabled,
		buttonDisabledActive: t.recordFormButtonDisabledActive,
		windowBody:           t.recordFormBody,
	}, width, cacheBrowseButtonGap, []formButton{
		{label: "< Browse >", active: focus == cacheBrowseSubmit, disabled: blocked || submitDisabled},
		{label: "< Cancel >", active: focus == cacheBrowseCancel, disabled: blocked},
	})
}

func renderCacheBrowseButtons(
	t theme,
	width int,
	focus cacheBrowseFocus,
	submitDisabled bool,
	blocked bool,
) string {
	return cacheBrowseButtonsLayout(t, width, focus, submitDisabled, blocked).content
}
