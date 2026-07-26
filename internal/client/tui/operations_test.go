package tui

import (
	"context"
	"strings"
	"testing"

	tea "charm.land/bubbletea/v2"
	"github.com/charmbracelet/x/ansi"
	"github.com/xhrobj/gopherkeeper/internal/buildinfo"
	"github.com/xhrobj/gopherkeeper/internal/client/config"
	recordmodel "github.com/xhrobj/gopherkeeper/internal/model"
)

func TestOperationCoordinator_Lifecycle(t *testing.T) {
	operations := operationCoordinator{}

	firstContext, firstID := operations.begin(context.Background().Done(), operationLogin)
	if !operations.pending(operationLogin) || !operations.accepts(operationLogin, firstID) {
		t.Fatalf("first operation = pending %t accepts %t", operations.pending(operationLogin), operations.accepts(operationLogin, firstID))
	}

	secondContext, secondID := operations.begin(context.Background().Done(), operationLogin)
	if secondID == firstID || operations.accepts(operationLogin, firstID) || !operations.accepts(operationLogin, secondID) {
		t.Fatalf("restarted operation = first %d second %d accepts first %t accepts second %t", firstID, secondID, operations.accepts(operationLogin, firstID), operations.accepts(operationLogin, secondID))
	}
	assertContextCanceled(t, firstContext)

	operations.finish(operationLogin)
	if operations.pending(operationLogin) || operations.accepts(operationLogin, secondID) {
		t.Fatalf("finished operation = pending %t accepts %t", operations.pending(operationLogin), operations.accepts(operationLogin, secondID))
	}
	assertContextCanceled(t, secondContext)
}

func TestOperationCoordinator_TracksKindsIndependently(t *testing.T) {
	operations := operationCoordinator{}
	_, loginID := operations.begin(context.Background().Done(), operationLogin)
	_, binaryID := operations.begin(context.Background().Done(), operationBinarySave)

	operations.finish(operationLogin)

	if operations.pending(operationLogin) || operations.accepts(operationLogin, loginID) {
		t.Fatal("finished Login remained active")
	}
	if !operations.pending(operationBinarySave) || !operations.accepts(operationBinarySave, binaryID) {
		t.Fatal("finishing Login changed Binary Save")
	}
}

func TestOperationCoordinator_CancelInvalidatesResult(t *testing.T) {
	operations := operationCoordinator{}
	operationContext, requestID := operations.begin(context.Background().Done(), operationViewRecord)

	operations.cancel(operationViewRecord)

	if operations.pending(operationViewRecord) || operations.accepts(operationViewRecord, requestID) {
		t.Fatalf("canceled operation = pending %t accepts %t", operations.pending(operationViewRecord), operations.accepts(operationViewRecord, requestID))
	}
	assertContextCanceled(t, operationContext)
}

func TestOperationCoordinator_CancelAllIncludesLocalOperations(t *testing.T) {
	operations := operationCoordinator{}
	loginContext, _ := operations.begin(context.Background().Done(), operationLogin)
	logoutContext, _ := operations.begin(context.Background().Done(), operationLogout)
	binaryContext, _ := operations.begin(context.Background().Done(), operationBinarySave)
	cacheContext, _ := operations.begin(context.Background().Done(), operationOpenCache)
	cachedRecordContext, _ := operations.begin(context.Background().Done(), operationViewCachedRecord)
	syncContext, _ := operations.begin(context.Background().Done(), operationSync)

	operations.cancelAll()

	for _, operation := range []operationKind{
		operationLogin,
		operationLogout,
		operationBinarySave,
		operationOpenCache,
		operationViewCachedRecord,
		operationSync,
	} {
		if operations.pending(operation) {
			t.Fatalf("operation %d remained pending", operation)
		}
	}
	assertContextCanceled(t, loginContext)
	assertContextCanceled(t, logoutContext)
	assertContextCanceled(t, binaryContext)
	assertContextCanceled(t, cacheContext)
	assertContextCanceled(t, cachedRecordContext)
	assertContextCanceled(t, syncContext)
}

func assertContextCanceled(t *testing.T, ctx context.Context) {
	t.Helper()
	select {
	case <-ctx.Done():
	default:
		t.Fatal("operation context was not canceled")
	}
}

func TestModel_CurrentNetworkBusyStateCoversBlockingOperations(t *testing.T) {
	tests := []struct {
		name      string
		prepare   func(*model)
		operation operationKind
		message   string
		inline    bool
	}{
		{
			name: "restore session",
			prepare: func(m *model) {
				m.authentication.currentUserCheck = currentUserCheckRestore
				m.operations.request(operationCurrentUser).pending = true
			},
			operation: operationCurrentUser,
			message:   "Restoring session",
		},
		{
			name: "reconfigure session",
			prepare: func(m *model) {
				m.authentication.currentUserCheck = currentUserCheckReconfigure
				m.operations.request(operationCurrentUser).pending = true
			},
			operation: operationCurrentUser,
			message:   "Checking session",
		},
		{
			name: "manual current user",
			prepare: func(m *model) {
				m.dialog = dialogCurrentUser
				m.authentication.currentUserCheck = currentUserCheckManual
				m.operations.request(operationCurrentUser).pending = true
			},
			operation: operationCurrentUser,
			message:   "Checking current user",
			inline:    true,
		},
		{name: "login", prepare: func(m *model) { m.operations.request(operationLogin).pending = true }, operation: operationLogin, message: "Logging in", inline: true},
		{name: "register", prepare: func(m *model) { m.operations.request(operationRegister).pending = true }, operation: operationRegister, message: "Registering", inline: true},
		{
			name: "status",
			prepare: func(m *model) {
				m.dialog = dialogServerStatus
				m.operations.request(operationServerStatus).pending = true
			},
			operation: operationServerStatus,
			message:   "Checking server",
			inline:    true,
		},
		{name: "list", prepare: func(m *model) { m.operations.request(operationListRecords).pending = true }, operation: operationListRecords, message: "Loading records", inline: true},
		{name: "view", prepare: func(m *model) { m.operations.request(operationViewRecord).pending = true }, operation: operationViewRecord, message: "Loading record", inline: true},
		{name: "edit load", prepare: func(m *model) { m.operations.request(operationLoadRecordForEdit).pending = true }, operation: operationLoadRecordForEdit, message: "Loading record", inline: true},
		{name: "create", prepare: func(m *model) { m.operations.request(operationCreateRecord).pending = true }, operation: operationCreateRecord, message: "Creating record", inline: true},
		{name: "edit", prepare: func(m *model) { m.operations.request(operationEditRecord).pending = true }, operation: operationEditRecord, message: "Saving record", inline: true},
		{name: "delete", prepare: func(m *model) { m.operations.request(operationDeleteRecord).pending = true }, operation: operationDeleteRecord, message: "Deleting record", inline: true},
		{name: "open cache", prepare: func(m *model) { m.operations.request(operationOpenCache).pending = true }, operation: operationOpenCache, message: "Opening local cache", inline: true},
		{name: "view cached record", prepare: func(m *model) { m.operations.request(operationViewCachedRecord).pending = true }, operation: operationViewCachedRecord, message: "Loading cached record", inline: true},
		{name: "sync", prepare: func(m *model) { m.operations.request(operationSync).pending = true }, operation: operationSync, message: "Synchronizing local cache", inline: true},
	}

	for _, test := range tests {
		t.Run(test.name, func(t *testing.T) {
			assertBlockingOperationState(t, test.prepare, test.operation, test.message, test.inline)
		})
	}
}

func assertBlockingOperationState(
	t *testing.T,
	prepare func(*model),
	operation operationKind,
	message string,
	inline bool,
) {
	t.Helper()
	m := newTestModel(t, config.Config{}, buildinfo.Info{})
	prepare(&m)

	state := m.currentNetworkBusyState()
	if state.operation != operation || state.message != message || state.inline != inline {
		t.Fatalf("busy state = %#v, want operation %d message %q inline %t", state, operation, message, inline)
	}
	if !m.networkBusy() || !m.interactionBlocked() || !m.spinnerPending() {
		t.Fatal("network operation did not activate the global busy state")
	}

	_, visible := m.networkBusyPlacement()
	if visible == inline {
		t.Fatalf("busy overlay visible = %t, want %t", visible, !inline)
	}
	if inline {
		assertViewExcludes(t, m.View().Content, "Please wait")
		return
	}
	assertViewContains(t, m.View().Content, "Please wait", message+"...", m.spinnerFrameValue())
}

func TestRenderNetworkBusyWindow_KeepsSpinnerInTitleAndUsesASCIIDotsInBody(t *testing.T) {
	plain := ansi.Strip(renderNetworkBusyWindow(newTheme(), 40, "Restoring session", "⠋"))
	lines := strings.Split(plain, "\n")
	if len(lines) < 2 {
		t.Fatalf("busy window has %d lines, want at least 2: %q", len(lines), plain)
	}
	if !strings.Contains(lines[0], "Please wait ⠋") {
		t.Fatalf("busy title = %q, want spinner after Please wait", lines[0])
	}
	if !strings.Contains(plain, "Restoring session...") {
		t.Fatalf("busy body = %q, want three ASCII dots", plain)
	}
	if strings.Contains(plain, "Restoring session ⠋") {
		t.Fatalf("spinner remained in busy body: %q", plain)
	}
}

func TestModel_NetworkBusyBlocksUserInputButAcceptsResizeAndQuit(t *testing.T) {
	m := newTestModel(t, config.Config{}, buildinfo.Info{})
	m.width = 80
	m.height = 25
	m.dialog = dialogLogin
	m.authentication.loginForm.login.setValue("alice")
	m.authentication.loginForm.password.setValue("secret")
	m.authentication.loginForm.focus = loginSubmit
	m.operations.request(operationLogin).pending = true
	m.operations.request(operationLogin).id = 7
	canceled := false
	m.operations.request(operationLogin).cancel = func() { canceled = true }

	updated, command := m.Update(keyPress("tab"))
	got := updated.(model)
	if command != nil || got.authentication.loginForm.focus != loginSubmit {
		t.Fatalf("keyboard changed busy form: focus %d command %t", got.authentication.loginForm.focus, command != nil)
	}

	updated, command = got.Update(tea.PasteMsg{Content: "changed"})
	got = updated.(model)
	if command != nil || got.authentication.loginForm.login.value != "alice" {
		t.Fatalf("paste changed busy form: login %q command %t", got.authentication.loginForm.login.value, command != nil)
	}

	updated, command = got.Update(mouseClick(1, menuBarY))
	got = updated.(model)
	if command != nil || got.menuFocused || got.dropdownOpen {
		t.Fatalf("mouse opened menu while busy: menu %t dropdown %t command %t", got.menuFocused, got.dropdownOpen, command != nil)
	}

	updated, command = got.Update(tea.WindowSizeMsg{Width: 100, Height: 30})
	got = updated.(model)
	if command != nil || got.width != 100 || got.height != 30 || !got.operations.request(operationLogin).pending {
		t.Fatalf("resize while busy = %dx%d pending %t command %t", got.width, got.height, got.operations.request(operationLogin).pending, command != nil)
	}

	updated, command = got.Update(keyPress("ctrl+q"))
	got = updated.(model)
	if command == nil || got.operations.request(operationLogin).pending || !canceled {
		t.Fatalf("quit while busy = command %t pending %t canceled %t", command != nil, got.operations.request(operationLogin).pending, canceled)
	}
}

func TestModel_LocalOperationsDoNotActivateNetworkBusy(t *testing.T) {
	tests := []struct {
		name      string
		operation operationKind
	}{
		{name: "logout", operation: operationLogout},
		{name: "binary save", operation: operationBinarySave},
	}

	for _, test := range tests {
		t.Run(test.name, func(t *testing.T) {
			m := newTestModel(t, config.Config{}, buildinfo.Info{})
			m.operations.request(test.operation).pending = true

			if m.networkBusy() || m.interactionBlocked() || m.spinnerPending() {
				t.Fatalf("local operation %d was treated as a server operation", test.operation)
			}
		})
	}
}

func TestModel_CurrentNetworkBusyStateUsesStablePriority(t *testing.T) {
	m := newTestModel(t, config.Config{}, buildinfo.Info{})
	m.operations.request(operationListRecords).pending = true
	m.operations.request(operationCreateRecord).pending = true

	state := m.currentNetworkBusyState()
	if state.operation != operationCreateRecord {
		t.Fatalf("busy operation = %d, want create %d", state.operation, operationCreateRecord)
	}
}

func TestRenderWindowTitle_KeepsBaseTitleCenteredWhileSpinnerIsVisible(t *testing.T) {
	theme := newTheme()
	idle := ansi.Strip(renderWindowTitle(theme.windowTitle, 40, "Records", "", false))
	pending := ansi.Strip(renderWindowTitle(theme.windowTitle, 40, "Records", "⠋", true))

	idleIndex := strings.Index(idle, "Records")
	pendingIndex := strings.Index(pending, "Records")
	if idleIndex < 0 || pendingIndex < 0 || idleIndex != pendingIndex {
		t.Fatalf("title positions = idle %d pending %d; idle %q pending %q", idleIndex, pendingIndex, idle, pending)
	}
	if !strings.Contains(pending, "Records ⠋") {
		t.Fatalf("pending title = %q, want spinner after title", pending)
	}
}

func TestRenderLoginButtons_DisablesBothButtonsAndPreservesFocusWhileBlocked(t *testing.T) {
	theme := newTheme()
	rendered := renderLoginButtons(theme, 40, loginSubmit, false, true)
	plain := ansi.Strip(rendered)

	if !strings.Contains(rendered, theme.buttonDisabledActive.Render("< Login >")) {
		t.Fatalf("focused Login button is not rendered with disabled-active style: %q", plain)
	}
	if !strings.Contains(rendered, theme.buttonDisabled.Render("< Close >")) {
		t.Fatalf("Close button is not rendered with disabled style: %q", plain)
	}
	if strings.Contains(plain, "⠋") {
		t.Fatalf("spinner remained inside a button: %q", plain)
	}
}

func TestRenderRecordFormButtons_DisablesAllButtonsAndPreservesFocusWhileBlocked(t *testing.T) {
	form := newRecordCreateForm(recordmodel.RecordTypeText)
	form.title.setValue("Note")
	form.text.setValue("secret")
	form.setFocus(recordFormSubmit)
	theme := newTheme()

	rendered := renderRecordFormButtons(theme, 60, form, true, "< Create >")
	plain := ansi.Strip(rendered)
	if !strings.Contains(rendered, theme.recordFormButtonDisabledActive.Render("< Create >")) {
		t.Fatalf("focused Create button is not rendered with disabled-active style: %q", plain)
	}
	if !strings.Contains(rendered, theme.recordFormButtonDisabled.Render("< Cancel >")) {
		t.Fatalf("Cancel button is not rendered with disabled style: %q", plain)
	}
	if strings.Contains(plain, "⠋") {
		t.Fatalf("spinner remained inside a button: %q", plain)
	}
}

func TestModel_SpinnerTickAdvancesOnlyWhileOperationPending(t *testing.T) {
	m := newTestModel(t, config.Config{}, buildinfo.Info{})
	m.operations.request(operationCreateRecord).pending = true
	initialFrame := m.activitySpinner.View()

	updated, command := m.Update(m.activitySpinner.Tick())
	got := updated.(model)
	if got.activitySpinner.View() == initialFrame || command == nil {
		t.Fatalf("spinner state = frame %q command %t", got.activitySpinner.View(), command != nil)
	}

	got.operations.request(operationCreateRecord).pending = false
	stoppedFrame := got.activitySpinner.View()
	updated, command = got.Update(got.activitySpinner.Tick())
	got = updated.(model)
	if got.activitySpinner.View() != stoppedFrame || command != nil {
		t.Fatalf("stopped spinner state = frame %q command %t", got.activitySpinner.View(), command != nil)
	}
}
