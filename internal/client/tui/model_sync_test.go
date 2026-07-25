package tui

import (
	"context"
	"errors"
	"strings"
	"testing"

	"github.com/charmbracelet/x/ansi"
	"github.com/xhrobj/gopherkeeper/internal/buildinfo"
	"github.com/xhrobj/gopherkeeper/internal/client/config"
	"github.com/xhrobj/gopherkeeper/internal/client/usecase"
)

func newSyncTestModel(t *testing.T, backend backendStub) model {
	t.Helper()
	m := mustNewModel(
		t,
		context.Background(),
		config.Config{},
		"",
		buildinfo.Info{},
		staticBackendFactory(backend),
	)
	m.operations.cancel(operationCurrentUser)
	m.startupCmd = nil
	m.authentication.session = authSession{state: authAuthenticated, login: "alice"}
	m.dialog = dialogNone
	return m
}

func TestModel_SyncMenuAvailability(t *testing.T) {
	m := newSyncTestModel(t, backendStub{})
	definitions := m.currentMenuDefinitions()
	if definitions[menuCache].disabled {
		t.Fatalf("authenticated Cache menu = %#v", definitions[menuCache])
	}
	assertMenuActionDisabledState(t, definitions[menuCache], actionSynchronize, false)

	m.authentication.session = authSession{state: authGuest}
	definitions = m.currentMenuDefinitions()
	if definitions[menuCache].disabled {
		t.Fatalf("guest Cache menu = %#v", definitions[menuCache])
	}
	assertMenuActionDisabledState(t, definitions[menuCache], actionBrowseCache, false)
	assertMenuActionDisabledState(t, definitions[menuCache], actionSynchronize, true)
}

func TestModel_SyncFlow(t *testing.T) {
	backend := backendStub{sync: func(_ context.Context, password string) (SyncSummary, error) {
		if password != "old" {
			t.Fatalf("password = %q", password)
		}
		return SyncSummary{Added: 2, Updated: 3, Removed: 1, Unchanged: 5}, nil
	}}
	m := newSyncTestModel(t, backend)

	updated, command := m.activate(actionSynchronize)
	m = updated.(model)
	if command != nil || m.dialog != dialogSync || m.syncFeature.form.focus != syncPassword {
		t.Fatalf("open sync = dialog %d focus %d command %t", m.dialog, m.syncFeature.form.focus, command != nil)
	}

	m.syncFeature.form.password.setValue("old")
	m.syncFeature.form.focus = syncSubmit
	updated, command = m.activateSync()
	m = updated.(model)
	if command == nil || !m.operations.pending(operationSync) {
		t.Fatalf("sync pending = %t command %t", m.operations.pending(operationSync), command != nil)
	}
	if m.syncFeature.form.password.value != "" {
		t.Fatal("password remains in model after starting Sync")
	}

	message := commandResult[syncResultMsg](t, command)
	updated, _ = m.handleSyncResult(message)
	m = updated.(model)
	if m.dialog != dialogSyncResult || m.operations.pending(operationSync) {
		t.Fatalf("sync result = dialog %d pending %t", m.dialog, m.operations.pending(operationSync))
	}
	if m.syncFeature.result != (SyncSummary{Added: 2, Updated: 3, Removed: 1, Unchanged: 5}) {
		t.Fatalf("summary = %#v", m.syncFeature.result)
	}

	plain := ansi.Strip(m.renderDialog())
	for _, value := range []string{"Synchronization Complete", "Added:", "2", "Updated:", "3", "Removed:", "1", "Unchanged:", "5", "< OK >"} {
		if !strings.Contains(plain, value) {
			t.Fatalf("result window does not contain %q: %q", value, plain)
		}
	}
}

func TestModel_SyncFailureKeepsOnlineState(t *testing.T) {
	m := newSyncTestModel(t, backendStub{sync: func(context.Context, string) (SyncSummary, error) {
		return SyncSummary{}, errors.New("sync: connection refused")
	}})
	m.recordFeature.workspace.open = true
	m.dialog = dialogSync
	m.syncFeature.form.password.setValue("old")
	m.syncFeature.form.focus = syncSubmit

	updated, command := m.activateSync()
	m = updated.(model)
	message := commandResult[syncResultMsg](t, command)
	updated, _ = m.handleSyncResult(message)
	m = updated.(model)

	if m.alert != alertError || m.alertTitle != "Synchronization failed" || m.alertMessage != "Connection refused" {
		t.Fatalf("alert = %d %q %q", m.alert, m.alertTitle, m.alertMessage)
	}
	if !m.authentication.session.authenticated() || !m.recordFeature.workspace.open {
		t.Fatal("failed Sync changed online session or workspace")
	}
	if m.alertReturnDialog != dialogSync || m.syncFeature.form.password.value != "" {
		t.Fatalf("failure return = dialog %d password %q", m.alertReturnDialog, m.syncFeature.form.password.value)
	}
}

func TestModel_SyncSessionExpiry(t *testing.T) {
	m := newSyncTestModel(t, backendStub{sync: func(context.Context, string) (SyncSummary, error) {
		return SyncSummary{}, usecase.ErrNotLoggedIn
	}})
	m.dialog = dialogSync
	m.syncFeature.form.password.setValue("old")
	m.syncFeature.form.focus = syncSubmit

	updated, command := m.activateSync()
	m = updated.(model)
	message := commandResult[syncResultMsg](t, command)
	updated, _ = m.handleSyncResult(message)
	m = updated.(model)

	if m.authentication.session.state != authGuest || m.alertTitle != sessionExpiredTitle {
		t.Fatalf("expired session = state %d alert %q", m.authentication.session.state, m.alertTitle)
	}
}

func TestModel_CloseSyncInvalidatesLateResult(t *testing.T) {
	m := newSyncTestModel(t, backendStub{sync: func(context.Context, string) (SyncSummary, error) {
		return SyncSummary{Added: 1}, nil
	}})
	m.dialog = dialogSync
	m.syncFeature.form.password.setValue("old")
	m.syncFeature.form.focus = syncSubmit

	updated, command := m.activateSync()
	m = updated.(model)
	message := commandResult[syncResultMsg](t, command)
	m.closeSync()

	updated, _ = m.handleSyncResult(message)
	m = updated.(model)
	if m.dialog != dialogNone || m.syncFeature.result != (SyncSummary{}) {
		t.Fatalf("late result changed closed Sync state: dialog %d result %#v", m.dialog, m.syncFeature.result)
	}
}

func assertMenuActionDisabledState(t *testing.T, definition menuDefinition, action actionID, want bool) {
	t.Helper()
	for _, item := range definition.items {
		if item.action == action {
			if item.disabled != want {
				t.Fatalf("action %d disabled = %t, want %t", action, item.disabled, want)
			}
			return
		}
	}
	t.Fatalf("action %d not found", action)
}

func TestRenderSyncWindows_UsePurpleBodyAndCompactResult(t *testing.T) {
	theme := newTheme()
	form := newSyncForm()
	form.password.setValue("old")

	syncWidth := syncWindowWidth(80)
	resultWidth := syncResultWindowWidth(80)
	if resultWidth != 44 || resultWidth >= syncWidth {
		t.Fatalf("sync widths = form %d result %d, want compact result width 44", syncWidth, resultWidth)
	}

	formWindow := renderSyncWindow(theme, syncWidth, "alice", form, false, false, "")
	if !strings.Contains(formWindow, theme.recordFormLabel.Width(syncLabelWidth).Render("User")) {
		t.Fatal("Sync form does not use the purple record-form label style")
	}
	if strings.Contains(formWindow, theme.label.Width(syncLabelWidth).Render("User")) {
		t.Fatal("Sync form still uses the cyan dialog label style")
	}

	resultWindow := renderSyncResultWindow(theme, resultWidth, SyncSummary{Added: 1}, false)
	if !strings.Contains(resultWindow, theme.recordFormInfo.Width(12).Render("Added:")) {
		t.Fatal("Sync result does not use the purple record-form body styles")
	}
	if !strings.Contains(resultWindow, theme.recordFormButtonActive.Render("< OK >")) {
		t.Fatal("Sync result does not render the purple OK button")
	}
}
