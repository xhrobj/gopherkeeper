package tui

import (
	"context"
	"errors"
	"testing"

	recordmodel "github.com/xhrobj/gopherkeeper/internal/model"
)

const syncFormTestPassword = "secret"

func TestSyncForm_EditingAndNavigation(t *testing.T) {
	form := newSyncForm()
	if form.canSubmit() {
		t.Fatal("empty sync form can be submitted")
	}

	form.insert(syncFormTestPassword)
	if !form.canSubmit() {
		t.Fatal("sync form with password cannot be submitted")
	}

	form.moveCursorToStart()
	form.delete()
	form.insertKey("s")
	form.moveCursor(1)
	form.backspace()
	form.insert("e")
	form.moveCursorToEnd()
	if form.password.value != syncFormTestPassword {
		t.Fatalf("password = %q, want secret", form.password.value)
	}

	form.setFocus(syncSubmit, true)
	if form.focus != syncPassword {
		t.Fatalf("disabled submit changed focus to %d", form.focus)
	}
	form.move(1, true)
	if form.focus != syncCancel {
		t.Fatalf("focus after skipping submit = %d, want cancel", form.focus)
	}

	form.setFocus(syncFocusCount, false)
	if form.focus != syncCancel {
		t.Fatalf("invalid focus changed form focus to %d", form.focus)
	}

	before := form.password.value
	form.insert("ignored")
	if form.insertKey("x") {
		t.Fatal("button focus accepted a text key")
	}
	form.backspace()
	form.delete()
	form.moveCursor(1)
	form.moveCursorToStart()
	form.moveCursorToEnd()
	if form.password.value != before {
		t.Fatalf("button focus changed password to %q", form.password.value)
	}

	form.move(1, false)
	if form.focus != syncPassword {
		t.Fatalf("wrapped focus = %d, want password", form.focus)
	}
}

func TestSyncCommand_ReturnsSummaryAndError(t *testing.T) {
	wantResult := SyncSummary{Added: 2, Updated: 1}
	wantErr := errors.New("sync failed")
	backend := backendStub{sync: func(_ context.Context, password string) (SyncSummary, error) {
		if password != syncFormTestPassword {
			t.Fatalf("password = %q, want secret", password)
		}
		return wantResult, wantErr
	}}

	message := syncCommand(context.Background(), backend, 41, syncFormTestPassword)().(syncResultMsg)
	if message.requestID != 41 || message.result != wantResult || !errors.Is(message.err, wantErr) {
		t.Fatalf("message = %#v", message)
	}
}

func TestCleanSyncError_KnownFailures(t *testing.T) {
	tests := []struct {
		name string
		err  error
		want string
	}{
		{name: "nil", want: "Unknown synchronization error"},
		{name: "canceled", err: context.Canceled, want: "Synchronization canceled"},
		{name: "invalid credentials", err: recordmodel.ErrInvalidCredentials, want: "Invalid login or password"},
		{name: "fallback", err: errors.New("internal details"), want: "Unable to synchronize local cache"},
	}

	for _, test := range tests {
		t.Run(test.name, func(t *testing.T) {
			if got := cleanSyncError(test.err); got != test.want {
				t.Fatalf("cleanSyncError() = %q, want %q", got, test.want)
			}
		})
	}
}

func TestModel_UpdateSyncNavigatesAndEditsForm(t *testing.T) {
	m := newSyncTestModel(t, backendStub{})
	m.dialog = dialogSync
	m.syncFeature.form.password.setValue(syncFormTestPassword)

	updated, _ := m.updateSync("tab")
	m = updated.(model)
	if m.syncFeature.form.focus != syncSubmit {
		t.Fatalf("focus after tab = %d, want submit", m.syncFeature.form.focus)
	}

	updated, _ = m.updateSync("left")
	m = updated.(model)
	if m.syncFeature.form.focus != syncPassword {
		t.Fatalf("focus after left = %d, want password", m.syncFeature.form.focus)
	}

	updated, _ = m.updateSync("home")
	m = updated.(model)
	updated, _ = m.updateSync("delete")
	m = updated.(model)
	updated, _ = m.updateSync("s")
	m = updated.(model)
	updated, _ = m.updateSync("end")
	m = updated.(model)
	updated, _ = m.updateSync("backspace")
	m = updated.(model)
	updated, _ = m.updateSync("t")
	m = updated.(model)
	if m.syncFeature.form.password.value != syncFormTestPassword {
		t.Fatalf("edited password = %q, want secret", m.syncFeature.form.password.value)
	}

	updated, _ = m.updateSync("right")
	m = updated.(model)
	updated, _ = m.updateSync("shift+tab")
	m = updated.(model)
	if m.syncFeature.form.focus != syncCancel {
		t.Fatalf("focus after reverse move = %d, want cancel", m.syncFeature.form.focus)
	}

	updated, _ = m.updateSync("up")
	m = updated.(model)
	if m.syncFeature.form.focus != syncSubmit {
		t.Fatalf("focus after up = %d, want submit", m.syncFeature.form.focus)
	}
}

func TestModel_UpdateSyncGuardsAndCancel(t *testing.T) {
	m := newSyncTestModel(t, backendStub{})
	m.dialog = dialogSync
	m.operations.request(operationSync).pending = true

	updated, command := m.updateSync("esc")
	got := updated.(model)
	if command != nil || got.dialog != dialogSync {
		t.Fatalf("pending update changed state: dialog=%d command=%t", got.dialog, command != nil)
	}

	m.operations.request(operationSync).pending = false
	m.syncFeature.form.focus = syncCancel
	updated, command = m.updateSync("enter")
	got = updated.(model)
	if command != nil || got.dialog != dialogNone {
		t.Fatalf("cancel state: dialog=%d command=%t", got.dialog, command != nil)
	}
}
