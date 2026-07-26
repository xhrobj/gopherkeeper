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

func renderLoginWindow(
	t theme,
	width int,
	form loginForm,
	pending bool,
	blocked bool,
	spinnerFrame string,
) string {
	layout := newLabeledFieldColumnLayout(
		width,
		loginLabelWidth,
		16,
		2,
		loginFirstFieldRow,
		loginFieldRowStep,
	)
	contentWidth := layout.contentWidth
	inputWidth := layout.inputWidth

	rows := []string{
		renderLoginField(t, "Login", form.login, inputWidth, form.focus == loginName && !blocked),
		t.windowBody.Width(contentWidth).Render(""),
		renderLoginField(t, "Password", form.password, inputWidth, form.focus == loginPassword && !blocked),
		t.windowBody.Width(contentWidth).Render(""),
		renderLoginButtons(t, contentWidth, form.focus, !form.canSubmit(), blocked),
	}

	title := renderWindowTitle(t.windowTitle, width, "Login", spinnerFrame, pending)
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

func loginButtonsLayout(
	t theme,
	width int,
	focus loginFocus,
	submitDisabled bool,
	blocked bool,
) buttonRowLayout {
	return formButtonsLayout(t, width, loginButtonGap, []formButton{
		{label: "< Login >", active: focus == loginSubmit, disabled: blocked || submitDisabled},
		{label: "< Close >", active: focus == loginClose, disabled: blocked},
	})
}

func renderLoginButtons(
	t theme,
	width int,
	focus loginFocus,
	submitDisabled bool,
	blocked bool,
) string {
	return loginButtonsLayout(t, width, focus, submitDisabled, blocked).content
}

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

func renderRegisterWindow(
	t theme,
	width int,
	form registerForm,
	pending bool,
	blocked bool,
	spinnerFrame string,
) string {
	layout := newLabeledFieldColumnLayout(
		width,
		registerLabelWidth,
		16,
		3,
		registerFirstFieldRow,
		registerFieldRowStep,
	)
	contentWidth := layout.contentWidth
	inputWidth := layout.inputWidth

	rows := []string{
		renderRegisterField(t, "Login", form.login, inputWidth, form.focus == registerName && !blocked),
		t.windowBody.Width(contentWidth).Render(""),
		renderRegisterField(t, "Password", form.password, inputWidth, form.focus == registerPassword && !blocked),
		t.windowBody.Width(contentWidth).Render(""),
		renderRegisterField(t, "Repeat password", form.repeatPassword, inputWidth, form.focus == registerRepeatPassword && !blocked),
		t.windowBody.Width(contentWidth).Render(""),
		renderRegisterButtons(t, contentWidth, form.focus, !form.canSubmit(), blocked),
	}

	title := renderWindowTitle(t.windowTitle, width, "Register", spinnerFrame, pending)
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

func registerButtonsLayout(
	t theme,
	width int,
	focus registerFocus,
	submitDisabled bool,
	blocked bool,
) buttonRowLayout {
	return formButtonsLayout(t, width, registerButtonGap, []formButton{
		{label: "< Register >", active: focus == registerSubmit, disabled: blocked || submitDisabled},
		{label: "< Close >", active: focus == registerClose, disabled: blocked},
	})
}

func renderRegisterButtons(
	t theme,
	width int,
	focus registerFocus,
	submitDisabled bool,
	blocked bool,
) string {
	return registerButtonsLayout(t, width, focus, submitDisabled, blocked).content
}
