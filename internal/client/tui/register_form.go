package tui

import (
	"strings"

	"charm.land/lipgloss/v2"
)

type registerFocus int

const (
	registerName registerFocus = iota
	registerPassword
	registerRepeatPassword
	registerSubmit
	registerClose
	registerFocusCount
)

const (
	registerLabelWidth    = 15
	registerFirstFieldRow = 2
	registerFieldRowStep  = 2
	registerButtonGap     = 3
	registerButtonRow     = 8
)

type registerForm struct {
	login          textField
	password       textField
	repeatPassword textField
	focus          registerFocus
}

func newRegisterForm() registerForm {
	return registerForm{
		login:          newASCIITextField("", false),
		password:       newASCIITextField("", true),
		repeatPassword: newASCIITextField("", true),
		focus:          registerName,
	}
}

func (form registerForm) canSubmit() bool {
	return strings.TrimSpace(form.login.value) != "" &&
		form.password.value != "" &&
		form.repeatPassword.value != "" &&
		form.password.value == form.repeatPassword.value
}

func (form *registerForm) move(step int, submitDisabled bool) {
	count := int(registerFocusCount)
	for range count {
		form.focus = registerFocus((int(form.focus) + step + count) % count)
		if !submitDisabled || form.focus != registerSubmit {
			return
		}
	}
}

func (form *registerForm) setFocus(focus registerFocus, submitDisabled bool) {
	if focus < 0 || focus >= registerFocusCount {
		return
	}
	if submitDisabled && focus == registerSubmit {
		return
	}

	form.focus = focus
	if field := form.activeField(); field != nil {
		field.moveCursorToEnd()
	}
}

func (form *registerForm) activeField() *textField {
	switch form.focus {
	case registerName:
		return &form.login
	case registerPassword:
		return &form.password
	case registerRepeatPassword:
		return &form.repeatPassword
	default:
		return nil
	}
}

func (form *registerForm) moveCursor(step int) {
	if field := form.activeField(); field != nil {
		field.moveCursor(step)
	}
}

func (form *registerForm) moveCursorToStart() {
	if field := form.activeField(); field != nil {
		field.moveCursorToStart()
	}
}

func (form *registerForm) moveCursorToEnd() {
	if field := form.activeField(); field != nil {
		field.moveCursorToEnd()
	}
}

func (form *registerForm) insert(value string) {
	if field := form.activeField(); field != nil {
		field.insert(value)
	}
}

func (form *registerForm) insertKey(key string) bool {
	if field := form.activeField(); field != nil {
		return field.insertKey(key)
	}
	return false
}

func (form *registerForm) backspace() {
	if field := form.activeField(); field != nil {
		field.backspace()
	}
}

func (form *registerForm) delete() {
	if field := form.activeField(); field != nil {
		field.delete()
	}
}

func renderRegisterWindow(t theme, width int, form registerForm, pending bool) string {
	contentWidth := max(1, width-4)
	inputWidth := max(16, contentWidth-registerLabelWidth-2)

	rows := []string{
		renderRegisterField(t, "Login", form.login, inputWidth, form.focus == registerName && !pending),
		t.windowBody.Width(contentWidth).Render(""),
		renderRegisterField(t, "Password", form.password, inputWidth, form.focus == registerPassword && !pending),
		t.windowBody.Width(contentWidth).Render(""),
		renderRegisterField(t, "Repeat password", form.repeatPassword, inputWidth, form.focus == registerRepeatPassword && !pending),
		t.windowBody.Width(contentWidth).Render(""),
		renderRegisterButtons(t, contentWidth, form.focus, pending || !form.canSubmit()),
	}

	title := t.windowTitle.Width(width).Render("Register")
	body := t.windowBody.
		Width(width).
		Padding(1, 2).
		Render(strings.Join(rows, "\n"))
	return lipgloss.JoinVertical(lipgloss.Left, title, body)
}

func renderRegisterField(t theme, label string, field textField, inputWidth int, active bool) string {
	labelPart := t.label.Width(registerLabelWidth).Render(label)
	gap := t.windowBody.Width(2).Render("")
	return labelPart + gap + renderTextField(t, field, inputWidth, active)
}

func renderRegisterButtons(t theme, width int, focus registerFocus, submitDisabled bool) string {
	registerButton := t.button.Render("< Register >")
	closeButton := t.button.Render("< Close >")

	if submitDisabled {
		registerButton = t.buttonDisabled.Render("< Register >")
		if focus == registerSubmit {
			registerButton = t.buttonDisabledActive.Render("< Register >")
		}
	} else if focus == registerSubmit {
		registerButton = t.buttonActive.Render("< Register >")
	}
	if focus == registerClose {
		closeButton = t.buttonActive.Render("< Close >")
	}

	buttonsWidth := lipgloss.Width(registerButton) + registerButtonGap + lipgloss.Width(closeButton)
	left := max(0, (width-buttonsWidth)/2)
	right := max(0, width-buttonsWidth-left)
	return t.windowBody.Width(left).Render("") +
		registerButton +
		t.windowBody.Width(registerButtonGap).Render("") +
		closeButton +
		t.windowBody.Width(right).Render("")
}
