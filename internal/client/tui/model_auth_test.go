package tui

import (
	"context"
	"errors"
	"strings"
	"testing"

	tea "charm.land/bubbletea/v2"
	"github.com/xhrobj/gopherkeeper/internal/buildinfo"
	"github.com/xhrobj/gopherkeeper/internal/client/config"
	"github.com/xhrobj/gopherkeeper/internal/client/usecase"
	recordmodel "github.com/xhrobj/gopherkeeper/internal/model"
)

type registrationConflictError struct {
	login string
}

func (err registrationConflictError) Error() string {
	return `login "` + err.login + `" is already registered`
}

func (err registrationConflictError) Unwrap() error {
	return recordmodel.ErrLoginAlreadyExists
}

func TestModel_F10OpensCurrentAccountMenuAtLogin(t *testing.T) {
	m := newTestModel(t, config.Config{}, buildinfo.Info{})

	updated, _ := m.Update(keyPress("f10"))
	got := updated.(model)
	if !got.menuFocused || !got.dropdownOpen {
		t.Fatal("F10 did not open the current menu")
	}

	if got.activeMenu != int(menuAccount) {
		t.Fatalf("active menu = %d, want Account (%d)", got.activeMenu, menuAccount)
	}

	if got.selectedItem != 0 {
		t.Fatalf("selected item = %d, want Login (0)", got.selectedItem)
	}
}

func TestModel_LoginAcceptsOnlyFirstPastedLine(t *testing.T) {
	m := newTestModel(t, config.Config{}, buildinfo.Info{})
	m.authentication.loginForm.focus = loginName

	updated, _ := m.Update(tea.PasteMsg{Content: "alice\r\nignored"})
	got := updated.(model)
	if got.authentication.loginForm.login.value != "alice" {
		t.Fatalf("pasted login = %q, want alice", got.authentication.loginForm.login.value)
	}
}

func TestModel_LoginSuccessShowsPurpleConfirmation(t *testing.T) {
	m := newTestModel(t, config.Config{Address: "localhost:8888"}, buildinfo.Info{})
	m.authentication.loginForm.login.setValue("alice")
	m.authentication.loginForm.password.setValue("secret")
	m.authentication.loginForm.focus = loginSubmit
	m.backend = backendStub{login: func(
		context.Context,
		string,
		string,
	) (string, error) {
		return "alice", nil
	}}

	updated, cmd := m.Update(keyPress("enter"))
	m = updated.(model)
	if cmd == nil || !m.operations.request(operationLogin).pending {
		t.Fatal("Login did not start an asynchronous request")
	}
	if m.authentication.loginForm.focus != loginSubmit {
		t.Fatalf("focus = %d, want Login button", m.authentication.loginForm.focus)
	}
	frame := m.spinnerFrameValue()
	view := m.View().Content
	assertViewContains(t, view, "Login "+frame, "< Login >", "< Close >")
	assertViewExcludes(t, view, "Please wait", "Logging in", "< Login"+frame+">")
	dialog := m.renderDialog()
	if !strings.Contains(dialog, m.theme.buttonDisabledActive.Render("< Login >")) {
		t.Fatal("focused Login button is not visually disabled in the rendered dialog while the request is pending")
	}
	if !strings.Contains(dialog, m.theme.buttonDisabled.Render("< Close >")) {
		t.Fatal("Close button is not visually disabled in the rendered dialog while the request is pending")
	}

	updated, _ = m.Update(keyPress("tab"))
	m = updated.(model)
	if m.authentication.loginForm.focus != loginSubmit {
		t.Fatalf("focus = %d, want unchanged Login while request is pending", m.authentication.loginForm.focus)
	}
	updated, _ = m.Update(keyPress("shift+tab"))
	m = updated.(model)
	if m.authentication.loginForm.focus != loginSubmit {
		t.Fatalf("pending Login changed focus: %d", m.authentication.loginForm.focus)
	}

	updated, listCmd := m.Update(commandResult[loginResultMsg](t, cmd))
	m = updated.(model)
	if m.operations.request(operationLogin).pending || !m.authentication.session.authenticated() || m.dialog != dialogCurrentUser {
		t.Fatalf("pending = %t, logged in = %t, dialog = %d; want Current User", m.operations.request(operationLogin).pending, m.authentication.session.authenticated(), m.dialog)
	}
	if m.authentication.session.login != "alice" {
		t.Fatalf("current user = %q, want alice", m.authentication.session.login)
	}
	assertViewContains(t, m.View().Content, "Current User", "Logged in as alice", "< OK >")

	updated, _ = m.Update(commandResult[recordListResultMsg](t, listCmd))
	m = updated.(model)
	updated, _ = m.Update(keyPress("enter"))
	got := updated.(model)
	if got.dialog != dialogNone || got.alert != alertNone {
		t.Fatalf("dialog = %d, alert = %d; want empty desktop", got.dialog, got.alert)
	}
}

func TestModel_LoginErrorShowsRedDialog(t *testing.T) {
	m := newTestModel(t, config.Config{}, buildinfo.Info{})
	m.authentication.loginForm.login.setValue("alice")
	m.authentication.loginForm.password.setValue("wrong")
	m.authentication.loginForm.focus = loginSubmit
	m.backend = backendStub{login: func(
		context.Context,
		string,
		string,
	) (string, error) {
		return "", recordmodel.ErrInvalidCredentials
	}}

	updated, cmd := m.Update(keyPress("enter"))
	m = updated.(model)
	updated, _ = m.Update(commandResult[loginResultMsg](t, cmd))
	got := updated.(model)
	if got.alert != alertError {
		t.Fatalf("alert = %d, want login error", got.alert)
	}
	assertViewContains(t, got.View().Content, "Login failed", "Invalid login or password", "< OK >")
}

func TestModel_SwitchingWindowCancelsPendingLogin(t *testing.T) {
	m := newTestModel(t, config.Config{}, buildinfo.Info{})
	m.authentication.loginForm.login.setValue("alice")
	m.authentication.loginForm.password.setValue("secret")
	m.operations.request(operationLogin).pending = true
	m.operations.request(operationLogin).id = 7
	canceled := false
	m.operations.request(operationLogin).cancel = func() { canceled = true }

	updated, _ := m.activate(actionConfig)
	got := updated.(model)
	if !canceled {
		t.Fatal("pending Login context was not canceled")
	}
	if got.operations.request(operationLogin).pending {
		t.Fatal("Login remains pending after opening Config")
	}
	if got.operations.request(operationLogin).id != 8 {
		t.Fatalf("login request ID = %d, want invalidated 8", got.operations.request(operationLogin).id)
	}
	if got.authentication.loginForm.login.value != "" || got.authentication.loginForm.password.value != "" {
		t.Fatalf("Login form was not cleared: %#v", got.authentication.loginForm)
	}
}

func TestModel_ClosingRegisterClearsCredentials(t *testing.T) {
	m := newTestModel(t, config.Config{}, buildinfo.Info{})
	m.dialog = dialogRegister
	m.authentication.registerForm.login.setValue("alice")
	m.authentication.registerForm.password.setValue("secret")
	m.authentication.registerForm.repeatPassword.setValue("secret")

	updated, _ := m.Update(keyPress("esc"))
	got := updated.(model)
	if got.dialog != dialogNone {
		t.Fatalf("dialog = %d, want none", got.dialog)
	}
	if got.authentication.registerForm.login.value != "" || got.authentication.registerForm.password.value != "" || got.authentication.registerForm.repeatPassword.value != "" {
		t.Fatalf("Register form was not cleared: %#v", got.authentication.registerForm)
	}
}

func TestModel_ManualAuthenticationCancelsStartupSessionCheck(t *testing.T) {
	tests := []struct {
		name       string
		action     actionID
		wantDialog dialogID
	}{
		{name: "login", action: actionLogin, wantDialog: dialogLogin},
		{name: "register", action: actionRegister, wantDialog: dialogRegister},
	}

	for _, test := range tests {
		t.Run(test.name, func(t *testing.T) {
			m := newTestModel(t, config.Config{}, buildinfo.Info{})
			m.dialog = dialogNone
			m.authentication.session = authSession{state: authUnknown}
			m.operations.request(operationCurrentUser).pending = true
			m.operations.request(operationCurrentUser).id = 7
			canceled := false
			m.operations.request(operationCurrentUser).cancel = func() { canceled = true }

			updated, _ := m.activate(test.action)
			got := updated.(model)

			if !canceled {
				t.Fatal("startup session check was not canceled")
			}
			if got.operations.request(operationCurrentUser).pending || got.operations.request(operationCurrentUser).id != 8 {
				t.Fatalf("session check state = pending %t request %d, want false/8", got.operations.request(operationCurrentUser).pending, got.operations.request(operationCurrentUser).id)
			}
			if got.authentication.session.state != authGuest || got.dialog != test.wantDialog {
				t.Fatalf("manual auth state = auth %d dialog %d", got.authentication.session.state, got.dialog)
			}
		})
	}
}

func TestModel_StaleStartupSessionResultDoesNotOverwriteLogin(t *testing.T) {
	m := newTestModel(t, config.Config{}, buildinfo.Info{})
	m.dialog = dialogNone
	m.authentication.session = authSession{state: authUnknown}
	m.operations.request(operationCurrentUser).pending = true
	m.operations.request(operationCurrentUser).id = 7
	m.operations.request(operationCurrentUser).cancel = func() {}
	m.backend = backendStub{login: func(context.Context, string, string) (string, error) {
		return "alice", nil
	}}

	updated, _ := m.activate(actionLogin)
	m = updated.(model)
	m.authentication.loginForm.login.setValue("alice")
	m.authentication.loginForm.password.setValue("secret")
	m.authentication.loginForm.focus = loginSubmit

	updated, cmd := m.Update(keyPress("enter"))
	m = updated.(model)
	if cmd == nil {
		t.Fatal("Login did not start an asynchronous request")
	}
	updated, _ = m.Update(commandResult[loginResultMsg](t, cmd))
	m = updated.(model)
	if !m.authentication.session.authenticated() || m.authentication.session.login != "alice" {
		t.Fatalf("state after Login = %#v", m.authentication.session)
	}

	updated, _ = m.Update(currentUserResultMsg{requestID: 7, err: usecase.ErrNotLoggedIn})
	got := updated.(model)
	if !got.authentication.session.authenticated() || got.authentication.session.login != "alice" || got.dialog != dialogCurrentUser {
		t.Fatalf("state after stale startup result = auth %#v dialog %d", got.authentication.session, got.dialog)
	}
}

func TestModel_CurrentUserChecksServerBeforeShowingDialog(t *testing.T) {
	m := newTestModel(t, config.Config{}, buildinfo.Info{})
	m.authentication.session = authSession{state: authAuthenticated, login: "alice"}
	m.dialog = dialogNone
	m.recordFeature.workspace = recordWorkspace{open: true, state: recordListReady}
	m.backend = backendStub{currentUser: func(context.Context) (string, error) {
		return "alice", nil
	}}

	updated, command := m.activate(actionCurrentUser)
	m = updated.(model)
	if command == nil || !m.operations.request(operationCurrentUser).pending || m.authentication.currentUserCheck != currentUserCheckManual {
		t.Fatalf("Current User check state = command %t pending %t mode %d", command != nil, m.operations.request(operationCurrentUser).pending, m.authentication.currentUserCheck)
	}
	if m.dialog != dialogCurrentUser {
		t.Fatalf("dialog = %d, want Current User while Whoami is pending", m.dialog)
	}
	assertViewContains(t, m.View().Content, "Current User", "< OK >")
	assertViewExcludes(t, m.View().Content, "Checking current user", "Logged in as")

	updated, followup := m.Update(commandResult[currentUserResultMsg](t, command))
	got := updated.(model)
	if followup != nil {
		t.Fatal("manual Current User check unexpectedly started record loading")
	}
	if !got.authentication.session.authenticated() || got.authentication.session.login != "alice" || got.dialog != dialogCurrentUser {
		t.Fatalf("Current User result = auth %#v dialog %d", got.authentication.session, got.dialog)
	}
	if !got.recordFeature.workspace.open {
		t.Fatal("successful Current User check closed the Records workspace")
	}
}

func TestModel_CurrentUserIdentityChangeClosesStaleRecords(t *testing.T) {
	m := newTestModel(t, config.Config{}, buildinfo.Info{})
	m.authentication.session = authSession{state: authAuthenticated, login: "alice"}
	m.recordFeature.workspace = recordWorkspace{
		open:    true,
		state:   recordListReady,
		records: []recordmodel.RecordMetadata{{ID: "alice-record"}},
	}
	m.backend = backendStub{currentUser: func(context.Context) (string, error) {
		return "bob", nil
	}}

	updated, command := m.activate(actionCurrentUser)
	m = updated.(model)
	updated, _ = m.Update(commandResult[currentUserResultMsg](t, command))
	got := updated.(model)

	if !got.authentication.session.authenticated() || got.authentication.session.login != "bob" || got.dialog != dialogCurrentUser {
		t.Fatalf("Current User result = auth %#v dialog %d", got.authentication.session, got.dialog)
	}
	if got.recordFeature.workspace.open || len(got.recordFeature.workspace.records) != 0 {
		t.Fatalf("records from previous user remain visible: %#v", got.recordFeature.workspace)
	}
}

func TestModel_CurrentUserSessionExpiryUsesGlobalReset(t *testing.T) {
	m := newTestModel(t, config.Config{}, buildinfo.Info{})
	m.authentication.session = authSession{state: authAuthenticated, login: "alice"}
	m.dialog = dialogAbout
	m.recordFeature.workspace = recordWorkspace{
		open:  true,
		state: recordListReady,
	}
	m.backend = backendStub{currentUser: func(context.Context) (string, error) {
		return "", usecase.ErrNotLoggedIn
	}}

	updated, command := m.activate(actionCurrentUser)
	m = updated.(model)
	updated, _ = m.Update(commandResult[currentUserResultMsg](t, command))
	got := updated.(model)

	if got.authentication.session.authenticated() || got.authentication.session.login != "" || got.dialog != dialogNone || got.recordFeature.workspace.open {
		t.Fatalf("expired Current User state = auth %#v dialog %d records %#v", got.authentication.session, got.dialog, got.recordFeature.workspace)
	}
	if got.alert != alertError || got.alertTitle != sessionExpiredTitle || got.alertMessage != sessionExpiredMessage {
		t.Fatalf("expiry alert = state %d title %q message %q", got.alert, got.alertTitle, got.alertMessage)
	}
	assertViewExcludes(t, got.View().Content, "Logged in as alice")
}

func TestModel_CurrentUserNetworkFailureKeepsAuthenticatedState(t *testing.T) {
	m := newTestModel(t, config.Config{}, buildinfo.Info{})
	m.authentication.session = authSession{state: authAuthenticated, login: "alice"}
	m.dialog = dialogNone
	m.recordFeature.workspace = recordWorkspace{open: true, state: recordListReady}
	m.backend = backendStub{currentUser: func(context.Context) (string, error) {
		return "", errors.New("get current user: connection refused")
	}}

	updated, command := m.activate(actionCurrentUser)
	m = updated.(model)
	updated, _ = m.Update(commandResult[currentUserResultMsg](t, command))
	got := updated.(model)

	if !got.authentication.session.authenticated() || got.authentication.session.login != "alice" || !got.recordFeature.workspace.open {
		t.Fatalf("network failure changed authenticated state: auth %#v records %#v", got.authentication.session, got.recordFeature.workspace)
	}
	if got.alert != alertError || got.alertTitle != "Current user check failed" || got.alertMessage != "Connection refused" {
		t.Fatalf("network alert = state %d title %q message %q", got.alert, got.alertTitle, got.alertMessage)
	}
}

func TestLoginButton_DisabledUntilBothFieldsHaveValues(t *testing.T) {
	m := newTestModel(t, config.Config{}, buildinfo.Info{})
	m.authentication.loginForm.focus = loginPassword
	m.authentication.loginForm.login.setValue("alice")

	updated, _ := m.Update(keyPress("tab"))
	got := updated.(model)
	if got.authentication.loginForm.focus == loginSubmit {
		t.Fatal("Login button received focus while password is empty")
	}

	m.authentication.loginForm.password.setValue("secret")
	m.authentication.loginForm.focus = loginPassword
	updated, _ = m.Update(keyPress("tab"))
	got = updated.(model)
	if got.authentication.loginForm.focus != loginSubmit {
		t.Fatalf("focus = %d, want enabled Login button", got.authentication.loginForm.focus)
	}
}

func TestModel_LoginEnablesCurrentUserAndLogoutMenuItems(t *testing.T) {
	m := newTestModel(t, config.Config{}, buildinfo.Info{})
	m.authentication.loginForm.login.setValue("alice")
	m.authentication.loginForm.password.setValue("secret")
	m.authentication.loginForm.focus = loginSubmit
	m.backend = backendStub{login: func(context.Context, string, string) (string, error) {
		return "alice", nil
	}}

	updated, cmd := m.Update(keyPress("enter"))
	m = updated.(model)
	updated, listCmd := m.Update(commandResult[loginResultMsg](t, cmd))
	m = updated.(model)
	updated, _ = m.Update(commandResult[recordListResultMsg](t, listCmd))
	m = updated.(model)

	definitions := menuDefinitions(m.dialog, m.authentication.session.authenticated())
	account := definitions[menuAccount].items
	if !account[0].disabled || !account[1].disabled || account[2].disabled || account[4].disabled {
		t.Fatal("Account menu state was not switched after login")
	}

	updated, _ = m.Update(keyPress("f10"))
	got := updated.(model)
	if got.activeMenu != int(menuAccount) || got.selectedItem != 2 {
		t.Fatalf("F10 selection = menu %d item %d, want Account/Current User", got.activeMenu, got.selectedItem)
	}
}

func TestModel_LogoutClearsUserAndShowsAlert(t *testing.T) {
	m := newTestModel(t, config.Config{}, buildinfo.Info{})
	m.authentication.session = authSession{state: authAuthenticated, login: "alice"}
	m.dialog = dialogCurrentUser
	m.backend = backendStub{logout: func(context.Context) error { return nil }}

	updated, cmd := m.activate(actionLogout)
	m = updated.(model)
	if cmd == nil {
		t.Fatal("Logout did not return a command")
	}
	updated, _ = m.Update(commandResult[logoutResultMsg](t, cmd))
	got := updated.(model)
	if got.authentication.session.authenticated() || got.authentication.session.login != "" {
		t.Fatal("Logout did not clear current user")
	}
	if got.alert != alertNotice || got.alertTitle != "Logged out" {
		t.Fatalf("alert = %d/%q, want Logged out notice", got.alert, got.alertTitle)
	}
}

func TestModel_LogoutInvalidatesPendingCurrentUserCheck(t *testing.T) {
	m := newTestModel(t, config.Config{}, buildinfo.Info{})
	m.authentication.session = authSession{state: authAuthenticated, login: "alice"}
	m.dialog = dialogNone
	m.backend = backendStub{
		currentUser: func(context.Context) (string, error) { return "alice", nil },
		logout:      func(context.Context) error { return nil },
	}

	updated, currentUserCmd := m.activate(actionCurrentUser)
	m = updated.(model)
	if currentUserCmd == nil || !m.operations.request(operationCurrentUser).pending {
		t.Fatal("Current User check did not start")
	}

	updated, logoutCmd := m.activate(actionLogout)
	m = updated.(model)
	if logoutCmd == nil || m.operations.request(operationCurrentUser).pending {
		t.Fatalf("Logout state = command %t auth pending %t", logoutCmd != nil, m.operations.request(operationCurrentUser).pending)
	}

	updated, _ = m.Update(commandResult[logoutResultMsg](t, logoutCmd))
	m = updated.(model)
	updated, _ = m.Update(commandResult[currentUserResultMsg](t, currentUserCmd))
	got := updated.(model)

	if got.authentication.session.authenticated() || got.authentication.session.login != "" {
		t.Fatalf("stale Current User result restored session: %#v", got.authentication.session)
	}
	if got.dialog == dialogCurrentUser {
		t.Fatal("stale Current User result reopened the dialog after Logout")
	}
}

func TestModel_SessionRecheckIgnoresStaleLogoutResult(t *testing.T) {
	oldConfig := config.Config{Address: "old.example:8443", SessionDir: "/tmp/old-session"}
	newConfig := config.Config{Address: "new.example:9443", SessionDir: "/tmp/new-session"}
	m := newTestModel(t, oldConfig, buildinfo.Info{})
	m.authentication.session = authSession{state: authAuthenticated, login: "alice"}
	m.dialog = dialogCurrentUser
	m.backend = backendStub{
		logout: func(context.Context) error { return nil },
	}

	updated, logoutCmd := m.activate(actionLogout)
	m = updated.(model)
	if logoutCmd == nil || !m.operations.request(operationLogout).pending || m.operations.request(operationLogout).id != 1 {
		t.Fatalf("Logout state = command %t pending %t request %d", logoutCmd != nil, m.operations.request(operationLogout).pending, m.operations.request(operationLogout).id)
	}

	m.config = newConfig
	m.backend = backendStub{
		currentUser: func(context.Context) (string, error) {
			return "bob", nil
		},
	}
	currentUserCmd := m.beginCurrentUserCheck(currentUserCheckRestore)
	if currentUserCmd == nil {
		t.Fatal("session recheck command = nil")
	}
	if m.operations.request(operationLogout).pending || m.operations.request(operationLogout).id != 2 {
		t.Fatalf("invalidated Logout state = pending %t request %d, want false/2", m.operations.request(operationLogout).pending, m.operations.request(operationLogout).id)
	}

	updated, _ = m.Update(commandResult[currentUserResultMsg](t, currentUserCmd))
	m = updated.(model)
	if !m.authentication.session.authenticated() || m.authentication.session.login != "bob" {
		t.Fatalf("state after session recheck = %#v", m.authentication.session)
	}

	updated, _ = m.Update(commandResult[logoutResultMsg](t, logoutCmd))
	got := updated.(model)
	if !got.authentication.session.authenticated() || got.authentication.session.login != "bob" {
		t.Fatalf("state after stale Logout result = %#v", got.authentication.session)
	}
}

func TestModel_RegisterFailureHighlightsLogin(t *testing.T) {
	m := newTestModel(t, config.Config{}, buildinfo.Info{})
	m.dialog = dialogRegister
	m.authentication.registerForm.login.setValue("alice")
	m.authentication.registerForm.password.setValue("secret42")
	m.authentication.registerForm.repeatPassword.setValue("secret42")
	m.authentication.registerForm.focus = registerSubmit
	m.backend = backendStub{register: func(context.Context, string, string) (string, error) {
		return "", registrationConflictError{login: "alice"}
	}}

	updated, cmd := m.Update(keyPress("enter"))
	m = updated.(model)
	if cmd == nil {
		t.Fatal("Register did not start an asynchronous request")
	}
	updated, _ = m.Update(commandResult[registerResultMsg](t, cmd))
	m = updated.(model)

	if m.alert != alertError || m.alertHighlight != "alice" {
		t.Fatalf("alert = %d highlight %q, want registration login highlighted", m.alert, m.alertHighlight)
	}
	if !strings.Contains(m.renderAlert(), m.theme.errorTitle.Render("alice")) {
		t.Fatal("failed registration login is not rendered with yellow highlight")
	}
}

func TestModel_RegisterSuccessReturnsToPrefilledLogin(t *testing.T) {
	m := newTestModel(t, config.Config{}, buildinfo.Info{})
	m.dialog = dialogRegister
	m.authentication.registerForm.login.setValue("alice")
	m.authentication.registerForm.password.setValue("secret42")
	m.authentication.registerForm.repeatPassword.setValue("secret42")
	m.authentication.registerForm.focus = registerSubmit
	m.backend = backendStub{register: func(context.Context, string, string) (string, error) {
		return "alice", nil
	}}

	updated, cmd := m.Update(keyPress("enter"))
	m = updated.(model)
	if cmd == nil || !m.operations.request(operationRegister).pending {
		t.Fatal("Register did not start an asynchronous request")
	}
	updated, _ = m.Update(commandResult[registerResultMsg](t, cmd))
	m = updated.(model)
	if m.alert != alertNotice || m.alertTitle != "Registration successful" {
		t.Fatalf("alert = %d/%q, want registration notice", m.alert, m.alertTitle)
	}

	if m.dialog != dialogNone {
		t.Fatalf("dialog during registration notice = %d, want none", m.dialog)
	}
	assertViewContains(t, m.View().Content, "Registered as alice. Please log in.")

	updated, _ = m.Update(keyPress("enter"))
	got := updated.(model)
	if got.dialog != dialogLogin || got.authentication.loginForm.login.value != "alice" || got.authentication.loginForm.focus != loginPassword {
		t.Fatalf("state after registration notice = dialog %d login %q focus %d", got.dialog, got.authentication.loginForm.login.value, got.authentication.loginForm.focus)
	}
}

func TestModel_StartupRestoresExistingSession(t *testing.T) {
	cfg := config.Config{Address: "localhost:8888"}
	m := mustNewModel(t,
		context.Background(),
		cfg,
		"",
		buildinfo.Info{},
		func(config.Config) (Backend, error) {
			return backendStub{currentUser: func(context.Context) (string, error) {
				return "alice", nil
			}}, nil
		},
	)

	if m.authentication.session.state != authUnknown || m.dialog != dialogNone {
		t.Fatalf("startup state = auth %d dialog %d, want unknown desktop", m.authentication.session.state, m.dialog)
	}
	cmd := m.Init()
	if cmd == nil {
		t.Fatal("startup session check command = nil")
	}

	updated, _ := m.Update(commandResult[currentUserResultMsg](t, cmd))
	got := updated.(model)
	if !got.authentication.session.authenticated() || got.authentication.session.login != "alice" || got.dialog != dialogNone {
		t.Fatalf("restored state = auth %#v dialog %d, want authenticated desktop", got.authentication.session, got.dialog)
	}
}
func TestModel_StartupOpensLoginWithoutSession(t *testing.T) {
	m := mustNewModel(t,
		context.Background(),
		config.Config{},
		"",
		buildinfo.Info{},
		func(config.Config) (Backend, error) {
			return backendStub{currentUser: func(context.Context) (string, error) {
				return "", usecase.ErrNotLoggedIn
			}}, nil
		},
	)

	updated, _ := m.Update(commandResult[currentUserResultMsg](t, m.Init()))
	got := updated.(model)
	if got.authentication.session.state != authGuest || got.dialog != dialogLogin || got.alert != alertNone {
		t.Fatalf("guest state = auth %d dialog %d alert %d", got.authentication.session.state, got.dialog, got.alert)
	}
}

func TestModel_LogoutIsLocalAndClearsRecordState(t *testing.T) {
	m := newTestModel(t, config.Config{}, buildinfo.Info{})
	m.authentication.session = authSession{state: authAuthenticated, login: "alice"}
	m.dialog = dialogNone
	m.backend = backendStub{logout: func(context.Context) error { return nil }}
	m.recordFeature.workspace.apply([]recordmodel.RecordMetadata{{ID: "42", Title: "Secret"}}, 10)
	m.recordFeature.workspace.open = true
	m.recordFeature.view.apply(recordmodel.Record{
		Metadata: recordmodel.RecordMetadata{ID: "42", Type: recordmodel.RecordTypeCredentials},
		Payload:  &recordmodel.CredentialsPayload{Password: "secret"},
	}, m.width, m.height)
	m.recordFeature.createForm = newRecordCreateForm(recordmodel.RecordTypeCredentials)
	m.recordFeature.createForm.password.setValue("new-secret")
	m.recordFeature.edit.apply(recordmodel.Record{
		Metadata: recordmodel.RecordMetadata{ID: "43", Type: recordmodel.RecordTypeCredentials},
		Payload:  &recordmodel.CredentialsPayload{Password: "edited-secret"},
	}, dialogNone)
	m.recordFeature.deletion = recordDeleteState{metadata: recordmodel.RecordMetadata{ID: "44", Title: "Delete me"}}

	updated, command := m.activate(actionLogout)
	m = updated.(model)
	if command == nil || m.dialog != dialogNone || !m.operations.request(operationLogout).pending {
		t.Fatalf("logout state = dialog %d pending %t command %t", m.dialog, m.operations.request(operationLogout).pending, command != nil)
	}
	if m.networkBusy() || m.interactionBlocked() || m.spinnerPending() {
		t.Fatal("local Logout was presented as a network request")
	}
	assertViewExcludes(t, m.View().Content, "Logout", "Logging out", "Please wait")

	updated, _ = m.Update(commandResult[logoutResultMsg](t, command))
	got := updated.(model)
	if got.authentication.session.authenticated() || got.dialog != dialogNone || got.alert != alertNotice {
		t.Fatalf("logout result state = auth %#v dialog %d alert %d", got.authentication.session, got.dialog, got.alert)
	}
	if got.recordFeature.workspace.open || got.recordFeature.view.record.Payload != nil || got.recordFeature.createForm.password.value != "" ||
		got.recordFeature.edit.record.Payload != nil || got.recordFeature.deletion.metadata.ID != "" {
		t.Fatalf("record state survived logout: records %#v view %#v form %#v edit %#v delete %#v",
			got.recordFeature.workspace, got.recordFeature.view, got.recordFeature.createForm, got.recordFeature.edit, got.recordFeature.deletion)
	}
}
