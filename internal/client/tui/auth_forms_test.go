package tui

import (
	"strings"
	"testing"

	"github.com/charmbracelet/x/ansi"
)

const (
	authFormTestLogin    = "alice"
	authFormTestPassword = "secret"
	authFormTestSpinner  = "spinner"
)

func TestLoginForm_EditingAndNavigation(t *testing.T) {
	form := newLoginForm()
	if form.canSubmit() {
		t.Fatal("empty login form can be submitted")
	}

	form.insert(authFormTestLogin)
	form.setFocus(loginPassword, false)
	form.insert(authFormTestPassword)
	if !form.canSubmit() {
		t.Fatal("complete login form cannot be submitted")
	}

	form.moveCursorToStart()
	form.delete()
	form.insertKey("s")
	form.moveCursorToEnd()
	form.backspace()
	form.insert("t")
	if form.password.value != authFormTestPassword {
		t.Fatalf("password = %q, want secret", form.password.value)
	}

	form.setFocus(loginSubmit, true)
	if form.focus != loginPassword {
		t.Fatalf("disabled submit changed focus to %d", form.focus)
	}

	form.move(1, true)
	if form.focus != loginClose {
		t.Fatalf("focus after skipping submit = %d, want close", form.focus)
	}

	form.setFocus(loginFocusCount, false)
	if form.focus != loginClose {
		t.Fatalf("invalid focus changed form focus to %d", form.focus)
	}

	form.moveCursor(-1)
	form.moveCursorToStart()
	form.moveCursorToEnd()
	form.insert("ignored")
	if form.insertKey("x") {
		t.Fatal("button focus accepted a text key")
	}
	form.backspace()
	form.delete()

	form.move(1, false)
	if form.focus != loginName || form.activeField() == nil {
		t.Fatalf("wrapped focus = %d, want login field", form.focus)
	}
}

func TestRegisterForm_EditingAndNavigation(t *testing.T) {
	form := newRegisterForm()
	form.insert(authFormTestLogin)
	form.setFocus(registerPassword, false)
	form.insert(authFormTestPassword)
	form.setFocus(registerRepeatPassword, false)
	form.insert("different")
	if form.canSubmit() {
		t.Fatal("mismatched registration passwords can be submitted")
	}

	form.moveCursorToStart()
	form.delete()
	form.repeatPassword.setValue("")
	form.insertKey("s")
	form.insert("ecret")
	if !form.canSubmit() {
		t.Fatal("complete registration form cannot be submitted")
	}

	form.moveCursor(-1)
	form.backspace()
	form.insert("e")
	form.moveCursorToEnd()
	if form.repeatPassword.value != authFormTestPassword {
		t.Fatalf("repeat password = %q, want secret", form.repeatPassword.value)
	}

	form.setFocus(registerSubmit, true)
	if form.focus != registerRepeatPassword {
		t.Fatalf("disabled submit changed focus to %d", form.focus)
	}
	form.move(1, true)
	if form.focus != registerClose {
		t.Fatalf("focus after skipping submit = %d, want close", form.focus)
	}

	form.setFocus(registerFocus(-1), false)
	if form.focus != registerClose {
		t.Fatalf("invalid focus changed form focus to %d", form.focus)
	}

	form.moveCursorToStart()
	form.moveCursorToEnd()
	form.insert("ignored")
	if form.insertKey("x") {
		t.Fatal("button focus accepted a text key")
	}
	form.backspace()
	form.delete()

	form.move(1, false)
	if form.focus != registerName || form.activeField() == nil {
		t.Fatalf("wrapped focus = %d, want login field", form.focus)
	}
}

func TestRenderAuthForms_ShowsFieldsButtonsAndPendingState(t *testing.T) {
	theme := newTheme()

	login := newLoginForm()
	login.login.setValue(authFormTestLogin)
	login.password.setValue(authFormTestPassword)
	login.focus = loginSubmit
	loginView := ansi.Strip(renderLoginWindow(theme, 48, login, true, false, authFormTestSpinner))
	assertAuthFormView(t, loginView, "Login", "Password", "< Login >", closeButtonLabel, authFormTestSpinner)

	blockedLogin := ansi.Strip(renderLoginWindow(theme, 48, login, false, true, ""))
	assertAuthFormView(t, blockedLogin, "Login", "Password", "< Login >", closeButtonLabel)

	register := newRegisterForm()
	register.login.setValue(authFormTestLogin)
	register.password.setValue(authFormTestPassword)
	register.repeatPassword.setValue(authFormTestPassword)
	register.focus = registerSubmit
	registerView := ansi.Strip(renderRegisterWindow(theme, 56, register, true, false, authFormTestSpinner))
	assertAuthFormView(t, registerView, "Register", "Repeat password", "< Register >", closeButtonLabel, authFormTestSpinner)

	blockedRegister := ansi.Strip(renderRegisterWindow(theme, 56, register, false, true, ""))
	assertAuthFormView(t, blockedRegister, "Register", "Repeat password", "< Register >", closeButtonLabel)
}

func assertAuthFormView(t *testing.T, view string, parts ...string) {
	t.Helper()
	for _, part := range parts {
		if !strings.Contains(view, part) {
			t.Errorf("view does not contain %q: %q", part, view)
		}
	}
}

func TestRenderRegisterWindow_ShowsCenteredHintForFocusedControl(t *testing.T) {
	theme := newTheme()

	tests := []struct {
		name  string
		focus registerFocus
		want  string
	}{
		{name: "login", focus: registerName, want: registerLoginHint},
		{name: "password", focus: registerPassword, want: registerPasswordHint},
		{name: "repeat password", focus: registerRepeatPassword, want: registerRepeatPasswordHint},
		{name: "register button", focus: registerSubmit, want: registerWelcomeHint},
		{name: "close button", focus: registerClose, want: registerCloseWelcomeHint},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			form := newRegisterForm()
			form.focus = tt.focus

			view := ansi.Strip(renderRegisterWindow(theme, 56, form, false, false, ""))
			lines := strings.Split(view, "\n")
			lineIndex := -1
			for index, line := range lines {
				if strings.TrimSpace(line) == tt.want {
					lineIndex = index
					break
				}
			}
			if lineIndex < 1 || lineIndex+1 >= len(lines) {
				t.Fatalf("hint %q is missing or not separated by rows:\n%s", tt.want, view)
			}
			if strings.TrimSpace(lines[lineIndex-1]) != "" || strings.TrimSpace(lines[lineIndex+1]) != "" {
				t.Fatalf("hint %q is not surrounded by empty rows:\n%s", tt.want, view)
			}

			line := lines[lineIndex]
			left := len(line) - len(strings.TrimLeft(line, " "))
			right := len(line) - len(strings.TrimRight(line, " "))
			if left-right < -1 || left-right > 1 {
				t.Fatalf("hint %q is not centered: left padding %d, right padding %d", tt.want, left, right)
			}
		})
	}
}
