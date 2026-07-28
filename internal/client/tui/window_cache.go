package tui

import (
	"strings"

	"charm.land/lipgloss/v2"
)

const (
	cacheBrowseLabelWidth   = 8
	cacheBrowseFieldRowStep = 2
	cacheBrowseButtonGap    = 3
)

type cacheBrowseWindowLayout struct {
	content      string
	fieldBounds  []layoutBounds
	buttonBounds []layoutBounds
}

func cacheBrowseWindowWidth(screenWidth int) int {
	return clamp(screenWidth-18, 48, 62)
}

func newCacheBrowseWindowLayout(
	t theme,
	width int,
	form cacheBrowseForm,
	pending bool,
	blocked bool,
	spinnerFrame string,
) cacheBrowseWindowLayout {
	const (
		horizontalPadding = 2
		verticalPadding   = 1
	)

	title := renderWindowTitle(t.windowTitle, width, "Open Local Cache", spinnerFrame, pending)
	layout := newLabeledFieldColumnLayout(
		width,
		cacheBrowseLabelWidth,
		16,
		2,
		lipgloss.Height(title)+verticalPadding,
		cacheBrowseFieldRowStep,
	)

	rows := []string{
		renderCacheBrowseField(t, "Login", form.login, layout.inputWidth, form.focus == cacheBrowseLogin && !blocked),
		t.recordFormBody.Width(layout.contentWidth).Render(""),
		renderCacheBrowseField(t, "Password", form.password, layout.inputWidth, form.focus == cacheBrowsePassword && !blocked),
		t.recordFormBody.Width(layout.contentWidth).Render(""),
	}
	buttonRow := len(rows)
	buttonLayout := cacheBrowseButtonsLayout(t, layout.contentWidth, form.focus, !form.canSubmit(), blocked).
		positioned(0, buttonRow)
	rows = append(rows, buttonLayout.content)

	body := t.recordFormBody.
		Width(width).
		Padding(verticalPadding, horizontalPadding).
		Render(strings.Join(rows, "\n"))

	return cacheBrowseWindowLayout{
		content:     lipgloss.JoinVertical(lipgloss.Left, title, body),
		fieldBounds: layout.bounds,
		buttonBounds: translateLayoutBounds(
			buttonLayout.bounds,
			horizontalPadding,
			lipgloss.Height(title)+verticalPadding,
		),
	}
}

func renderCacheBrowseWindow(
	t theme,
	width int,
	form cacheBrowseForm,
	pending bool,
	blocked bool,
	spinnerFrame string,
) string {
	return newCacheBrowseWindowLayout(t, width, form, pending, blocked, spinnerFrame).content
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
