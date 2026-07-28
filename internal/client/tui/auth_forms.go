package tui

import (
	"strings"

	"charm.land/lipgloss/v2"
	domainmodel "github.com/xhrobj/gopherkeeper/internal/model"
)

const (
	loginName loginFocus = iota
	loginPassword
	loginSubmit
	loginClose
	loginFocusCount
)

const (
	loginLabelWidth   = 8
	loginFieldRowStep = 2
	loginButtonGap    = 3
)

const (
	registerName registerFocus = iota
	registerPassword
	registerRepeatPassword
	registerSubmit
	registerClose
	registerFocusCount
)

const (
	registerLabelWidth        = 15
	registerFieldRowStep      = 2
	registerButtonGap         = 3
	registerMinPasswordLength = 8
	registerMaxPasswordLength = 64

	registerLoginHint          = `^[A-Za-z0-9][A-Za-z0-9._-]{2,31}$`
	registerPasswordHint       = `^[!-~]{8,64}$`
	registerRepeatPasswordHint = `== password`
	registerWelcomeHint        = `Welcome (^-^)/`
	registerCloseWelcomeHint   = `\(O_O)/`
)

type loginFocus int

type loginForm struct {
	login    textField
	password textField
	focus    loginFocus
}

type authWindowLayout struct {
	content      string
	fieldBounds  []layoutBounds
	buttonBounds []layoutBounds
}

type registerFocus int

type registerForm struct {
	login          textField
	password       textField
	repeatPassword textField
	focus          registerFocus
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

func newLoginWindowLayout(
	t theme,
	width int,
	form loginForm,
	pending bool,
	blocked bool,
	spinnerFrame string,
) authWindowLayout {
	const (
		horizontalPadding = 2
		verticalPadding   = 1
	)

	title := renderWindowTitle(t.windowTitle, width, "Login", spinnerFrame, pending)
	fieldLayout := newLabeledFieldColumnLayout(
		width,
		loginLabelWidth,
		16,
		2,
		lipgloss.Height(title)+verticalPadding,
		loginFieldRowStep,
	)

	rows := []string{
		renderLoginField(t, "Login", form.login, fieldLayout.inputWidth, form.focus == loginName && !blocked),
		t.windowBody.Width(fieldLayout.contentWidth).Render(""),
		renderLoginField(t, "Password", form.password, fieldLayout.inputWidth, form.focus == loginPassword && !blocked),
		t.windowBody.Width(fieldLayout.contentWidth).Render(""),
	}
	buttonLayout := loginButtonsLayout(t, fieldLayout.contentWidth, form.focus, !form.canSubmit(), blocked).
		positioned(horizontalPadding, lipgloss.Height(title)+verticalPadding+len(rows))
	rows = append(rows, buttonLayout.content)

	body := t.windowBody.
		Width(width).
		Padding(verticalPadding, horizontalPadding).
		Render(strings.Join(rows, "\n"))

	return authWindowLayout{
		content:      lipgloss.JoinVertical(lipgloss.Left, title, body),
		fieldBounds:  fieldLayout.bounds,
		buttonBounds: buttonLayout.bounds,
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
	return newLoginWindowLayout(t, width, form, pending, blocked, spinnerFrame).content
}

func (m model) loginWindowLayout() authWindowLayout {
	return newLoginWindowLayout(
		m.theme,
		clamp(m.width-18, 44, 58),
		m.authentication.loginForm,
		m.operations.pending(operationLogin),
		m.interactionBlocked(),
		m.spinnerFrameValue(),
	)
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

func newRegisterForm() registerForm {
	return registerForm{
		login:          newASCIITextField("", false),
		password:       newASCIITextField("", true),
		repeatPassword: newASCIITextField("", true),
		focus:          registerName,
	}
}

func (form registerForm) canSubmit() bool {
	if _, err := domainmodel.CanonicalizeLogin(form.login.value); err != nil {
		return false
	}

	return validRegistrationPassword(form.password.value) &&
		form.password.value == form.repeatPassword.value
}

func validRegistrationPassword(password string) bool {
	if len(password) < registerMinPasswordLength || len(password) > registerMaxPasswordLength {
		return false
	}

	for index := 0; index < len(password); index++ {
		if password[index] < '!' || password[index] > '~' {
			return false
		}
	}

	return true
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

func newRegisterWindowLayout(
	t theme,
	width int,
	form registerForm,
	pending bool,
	blocked bool,
	spinnerFrame string,
) authWindowLayout {
	const (
		horizontalPadding = 2
		verticalPadding   = 1
	)

	title := renderWindowTitle(t.windowTitle, width, "Register", spinnerFrame, pending)
	fieldLayout := newLabeledFieldColumnLayout(
		width,
		registerLabelWidth,
		16,
		3,
		lipgloss.Height(title)+verticalPadding,
		registerFieldRowStep,
	)

	rows := []string{
		renderRegisterField(t, "Login", form.login, fieldLayout.inputWidth, form.focus == registerName && !blocked),
		t.windowBody.Width(fieldLayout.contentWidth).Render(""),
		renderRegisterField(t, "Password", form.password, fieldLayout.inputWidth, form.focus == registerPassword && !blocked),
		t.windowBody.Width(fieldLayout.contentWidth).Render(""),
		renderRegisterField(t, "Repeat password", form.repeatPassword, fieldLayout.inputWidth, form.focus == registerRepeatPassword && !blocked),
		t.windowBody.Width(fieldLayout.contentWidth).Render(""),
		renderRegisterHint(t, fieldLayout.contentWidth, form.focus),
		t.windowBody.Width(fieldLayout.contentWidth).Render(""),
	}
	buttonLayout := registerButtonsLayout(t, fieldLayout.contentWidth, form.focus, !form.canSubmit(), blocked).
		positioned(horizontalPadding, lipgloss.Height(title)+verticalPadding+len(rows))
	rows = append(rows, buttonLayout.content)

	body := t.windowBody.
		Width(width).
		Padding(verticalPadding, horizontalPadding).
		Render(strings.Join(rows, "\n"))

	return authWindowLayout{
		content:      lipgloss.JoinVertical(lipgloss.Left, title, body),
		fieldBounds:  fieldLayout.bounds,
		buttonBounds: buttonLayout.bounds,
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
	return newRegisterWindowLayout(t, width, form, pending, blocked, spinnerFrame).content
}

func (m model) registerWindowLayout() authWindowLayout {
	return newRegisterWindowLayout(
		m.theme,
		clamp(m.width-18, 48, 62),
		m.authentication.registerForm,
		m.operations.pending(operationRegister),
		m.interactionBlocked(),
		m.spinnerFrameValue(),
	)
}

func renderRegisterField(t theme, label string, field textField, inputWidth int, active bool) string {
	labelPart := t.label.Width(registerLabelWidth).Render(label)
	gap := t.windowBody.Width(2).Render("")

	return labelPart + gap + renderTextField(t, field, inputWidth, active)
}

func renderRegisterHint(t theme, width int, focus registerFocus) string {
	return t.recordInfo.
		Width(width).
		AlignHorizontal(lipgloss.Center).
		Render(registerHint(focus))
}

func registerHint(focus registerFocus) string {
	switch focus {
	case registerName:
		return registerLoginHint
	case registerPassword:
		return registerPasswordHint
	case registerRepeatPassword:
		return registerRepeatPasswordHint
	case registerClose:
		return registerCloseWelcomeHint
	default:
		return registerWelcomeHint
	}
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
