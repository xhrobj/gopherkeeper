package tui

import (
	"strings"

	"charm.land/lipgloss/v2"
)

type loginFocus int

const (
	loginName loginFocus = iota
	loginPassword
	loginSubmit
	loginClose
	loginFocusCount
)

const (
	loginLabelWidth    = 8
	loginFirstFieldRow = 2
	loginFieldRowStep  = 2
	loginButtonGap     = 3
	loginButtonRow     = 6
)

type loginForm struct {
	login    textField
	password textField
	focus    loginFocus
}

func newLoginForm() loginForm {
	return loginForm{
		login:    newASCIITextField("", false),
		password: newASCIITextField("", true),
		focus:    loginName,
	}
}

func (form loginForm) canSubmit() bool {
	return strings.TrimSpace(form.login.value) != "" && form.password.value != ""
}

func (form *loginForm) move(step int, submitDisabled bool) {
	count := int(loginFocusCount)
	for range count {
		form.focus = loginFocus((int(form.focus) + step + count) % count)
		if !submitDisabled || form.focus != loginSubmit {
			return
		}
	}
}

func (form *loginForm) setFocus(focus loginFocus, submitDisabled bool) {
	if focus < 0 || focus >= loginFocusCount {
		return
	}
	if submitDisabled && focus == loginSubmit {
		return
	}

	form.focus = focus
	if field := form.activeField(); field != nil {
		field.moveCursorToEnd()
	}
}

func (form *loginForm) activeField() *textField {
	switch form.focus {
	case loginName:
		return &form.login
	case loginPassword:
		return &form.password
	default:
		return nil
	}
}

func (form *loginForm) moveCursor(step int) {
	if field := form.activeField(); field != nil {
		field.moveCursor(step)
	}
}

func (form *loginForm) moveCursorToStart() {
	if field := form.activeField(); field != nil {
		field.moveCursorToStart()
	}
}

func (form *loginForm) moveCursorToEnd() {
	if field := form.activeField(); field != nil {
		field.moveCursorToEnd()
	}
}

func (form *loginForm) insert(value string) {
	if field := form.activeField(); field != nil {
		field.insert(value)
	}
}

func (form *loginForm) insertKey(key string) bool {
	if field := form.activeField(); field != nil {
		return field.insertKey(key)
	}
	return false
}

func (form *loginForm) backspace() {
	if field := form.activeField(); field != nil {
		field.backspace()
	}
}

func (form *loginForm) delete() {
	if field := form.activeField(); field != nil {
		field.delete()
	}
}

func renderLoginWindow(t theme, width int, form loginForm, pending bool) string {
	contentWidth := max(1, width-4)
	inputWidth := max(16, contentWidth-loginLabelWidth-2)

	rows := []string{
		renderLoginField(t, "Login", form.login, inputWidth, form.focus == loginName && !pending),
		t.windowBody.Width(contentWidth).Render(""),
		renderLoginField(t, "Password", form.password, inputWidth, form.focus == loginPassword && !pending),
		t.windowBody.Width(contentWidth).Render(""),
		renderLoginButtons(t, contentWidth, form.focus, pending || !form.canSubmit()),
	}

	title := t.windowTitle.Width(width).Render("Login")
	body := t.windowBody.
		Width(width).
		Padding(1, 2).
		Render(strings.Join(rows, "\n"))

	return lipgloss.JoinVertical(lipgloss.Left, title, body)
}

func renderLoginField(t theme, label string, field textField, inputWidth int, active bool) string {
	labelPart := t.label.Width(loginLabelWidth).Render(label)
	gap := t.windowBody.Width(2).Render("")

	return labelPart + gap + renderTextField(t, field, inputWidth, active)
}

func renderLoginButtons(t theme, width int, focus loginFocus, submitDisabled bool) string {
	loginButton := t.button.Render("< Login >")
	closeButton := t.button.Render("< Close >")

	if submitDisabled {
		loginButton = t.buttonDisabled.Render("< Login >")
		if focus == loginSubmit {
			loginButton = t.buttonDisabledActive.Render("< Login >")
		}
	} else if focus == loginSubmit {
		loginButton = t.buttonActive.Render("< Login >")
	}

	if focus == loginClose {
		closeButton = t.buttonActive.Render("< Close >")
	}

	buttonsWidth := lipgloss.Width(loginButton) + loginButtonGap + lipgloss.Width(closeButton)
	left := max(0, (width-buttonsWidth)/2)
	right := max(0, width-buttonsWidth-left)

	return t.windowBody.Width(left).Render("") +
		loginButton +
		t.windowBody.Width(loginButtonGap).Render("") +
		closeButton +
		t.windowBody.Width(right).Render("")
}
