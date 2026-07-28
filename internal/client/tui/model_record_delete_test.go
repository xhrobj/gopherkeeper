package tui

import (
	"context"
	"errors"
	"strings"
	"testing"
	"time"

	"github.com/charmbracelet/x/ansi"

	"github.com/xhrobj/gopherkeeper/internal/buildinfo"
	"github.com/xhrobj/gopherkeeper/internal/client/config"
	recordmodel "github.com/xhrobj/gopherkeeper/internal/model"
)

type recordDeleteBackendStub struct {
	recordsBackendStub
	deleteRecord func(context.Context, string, int64) error
}

func (stub recordDeleteBackendStub) DeleteRecord(ctx context.Context, id string, revision int64) error {
	if stub.deleteRecord == nil {
		return errors.New("unexpected DeleteRecord call")
	}
	return stub.deleteRecord(ctx, id, revision)
}

func TestModel_RecordDeleteMenuAvailability(t *testing.T) {
	m := newRecordDeleteTestModel(t, recordDeleteBackendStub{})
	m.recordFeature.workspace.open = true
	m.recordFeature.workspace.apply([]recordmodel.RecordMetadata{{
		ID: "7a79b627-0473-48a0-a001-887e79419719", Type: recordmodel.RecordTypeText, Title: "Recovery codes", Revision: 3,
	}}, 10)

	definitions := m.currentMenuDefinitions()
	if definitions[menuRecord].items[5].disabled {
		t.Fatalf("Delete menu item is disabled for selected record: %#v", definitions[menuRecord])
	}

	m.authentication.session = authSession{state: authGuest}
	definitions = m.currentMenuDefinitions()
	if !definitions[menuRecord].items[5].disabled {
		t.Fatalf("Delete menu item is enabled for guest: %#v", definitions[menuRecord])
	}
}

func TestModel_DeleteKeyOpensSafeConfirmation(t *testing.T) {
	m := newRecordDeleteTestModel(t, recordDeleteBackendStub{})
	m.recordFeature.workspace.open = true
	m.recordFeature.workspace.apply([]recordmodel.RecordMetadata{{
		ID: "7a79b627-0473-48a0-a001-887e79419719", Type: recordmodel.RecordTypeText, Title: "Recovery codes", Revision: 3,
	}}, 10)

	updated, command := m.updateRecordWorkspace("delete")
	got := updated.(model)
	if command != nil || got.dialog != dialogRecordDelete {
		t.Fatalf("delete confirmation = dialog %d command %t", got.dialog, command != nil)
	}
	if got.activeButton != 1 {
		t.Fatalf("active button = %d, want safe Cancel button", got.activeButton)
	}
	if got.recordFeature.deletion.metadata.Title != "Recovery codes" || got.recordFeature.deletion.metadata.Revision != 3 {
		t.Fatalf("delete target = %#v", got.recordFeature.deletion.metadata)
	}
}

func TestModel_RecordDeleteFlowRemovesRecord(t *testing.T) {
	const recordID = "7a79b627-0473-48a0-a001-887e79419719"
	backend := recordDeleteBackendStub{
		deleteRecord: func(_ context.Context, id string, revision int64) error {
			if id != recordID || revision != 3 {
				t.Fatalf("DeleteRecord(%q, %d)", id, revision)
			}
			return nil
		},
	}
	m := newRecordDeleteTestModel(t, backend)
	m.recordFeature.workspace.open = true
	m.recordFeature.workspace.apply([]recordmodel.RecordMetadata{
		{ID: recordID, Type: recordmodel.RecordTypeText, Title: "Recovery codes", Revision: 3},
		{ID: "second", Type: recordmodel.RecordTypeText, Title: "Second", Revision: 1},
	}, 10)
	m.openRecordDelete(m.recordFeature.workspace.records[0])
	m.activeButton = 0

	updated, command := m.activateRecordDelete()
	m = updated.(model)
	if command == nil || !m.operations.request(operationDeleteRecord).pending {
		t.Fatalf("delete request = command %t state %#v", command != nil, m.operations.request(operationDeleteRecord))
	}

	message := commandResult[recordDeleteResultMsg](t, command)
	updated, _ = m.Update(message)
	got := updated.(model)
	if got.operations.request(operationDeleteRecord).pending || got.dialog != dialogNone {
		t.Fatalf("delete result = dialog %d request %#v", got.dialog, got.operations.request(operationDeleteRecord))
	}
	if len(got.recordFeature.workspace.records) != 1 || got.recordFeature.workspace.records[0].ID != "second" || got.recordFeature.workspace.selected != 0 {
		t.Fatalf("records after deletion = %#v selected %d", got.recordFeature.workspace.records, got.recordFeature.workspace.selected)
	}
	if got.alert != alertNotice || got.alertTitle != "Record deleted" {
		t.Fatalf("delete notice = state %d title %q", got.alert, got.alertTitle)
	}
}

func TestModel_RecordDeleteErrorKeepsConfirmation(t *testing.T) {
	backend := recordDeleteBackendStub{
		deleteRecord: func(context.Context, string, int64) error {
			return errors.New("delete record: connection refused")
		},
	}
	m := newRecordDeleteTestModel(t, backend)
	metadata := recordmodel.RecordMetadata{ID: "7a79b627-0473-48a0-a001-887e79419719", Title: "Recovery codes", Revision: 3}
	m.openRecordDelete(metadata)
	m.activeButton = 0

	updated, command := m.activateRecordDelete()
	m = updated.(model)
	updated, _ = m.Update(commandResult[recordDeleteResultMsg](t, command))
	got := updated.(model)
	if got.alert != alertError || got.alertReturnDialog != dialogRecordDelete {
		t.Fatalf("delete error alert = %#v return dialog %d", got.alert, got.alertReturnDialog)
	}
	if got.recordFeature.deletion.metadata.ID != metadata.ID || got.activeButton != 1 {
		t.Fatalf("delete confirmation was not preserved: %#v button %d", got.recordFeature.deletion, got.activeButton)
	}
}

func TestModel_RecordMutationPendingBlocksMenuKeyboardAndMouse(t *testing.T) {
	m := newRecordDeleteTestModel(t, recordDeleteBackendStub{})
	m.width = 80
	m.height = 25
	m.dialog = dialogRecordDelete
	m.operations.request(operationDeleteRecord).pending = true
	m.operations.request(operationDeleteRecord).id = 42

	updated, _ := m.Update(keyPress("f10"))
	got := updated.(model)
	if got.menuFocused || got.dropdownOpen {
		t.Fatal("F10 opened menu while record deletion was pending")
	}

	updated, _ = got.Update(mouseClick(1, menuBarY))
	got = updated.(model)
	if got.menuFocused || got.dropdownOpen {
		t.Fatal("mouse opened menu while record deletion was pending")
	}
}

func TestModel_DeleteFromRecordViewKeepsTargetBeforeClosingView(t *testing.T) {
	m := newRecordDeleteTestModel(t, recordDeleteBackendStub{})
	m.dialog = dialogRecordView
	m.recordFeature.view = recordViewState{
		status: recordViewReady,
		record: recordmodel.Record{Metadata: recordmodel.RecordMetadata{
			ID: "7a79b627-0473-48a0-a001-887e79419719", Title: "Recovery codes", Revision: 3,
		}},
	}

	updated, _ := m.activate(actionDeleteRecord)
	got := updated.(model)
	if got.dialog != dialogRecordDelete || got.recordFeature.deletion.metadata.Title != "Recovery codes" {
		t.Fatalf("delete target after closing record view = dialog %d target %#v", got.dialog, got.recordFeature.deletion.metadata)
	}
	if got.recordFeature.view.status != recordViewIdle {
		t.Fatalf("record view was not cleared: %#v", got.recordFeature.view)
	}
}

func TestModel_RecordDeleteIsUnavailableFromEdit(t *testing.T) {
	record := recordmodel.Record{
		Metadata: recordmodel.RecordMetadata{
			ID: "7a79b627-0473-48a0-a001-887e79419719", Type: recordmodel.RecordTypeText,
			Title: "Recovery codes", Revision: 3,
		},
		Payload: &recordmodel.TextPayload{Text: "code-1"},
	}
	m := newRecordDeleteTestModel(t, recordDeleteBackendStub{})
	m.recordFeature.workspace.open = true
	m.recordFeature.workspace.apply([]recordmodel.RecordMetadata{record.Metadata}, 10)
	m.recordFeature.edit.apply(record, dialogNone)
	m.recordFeature.edit.form.title.setValue("Unsaved title")
	m.dialog = dialogRecordEdit

	definitions := m.currentMenuDefinitions()
	if !definitions[menuRecord].items[5].disabled {
		t.Fatalf("Delete menu item is enabled while editing: %#v", definitions[menuRecord])
	}

	updated, command := m.activate(actionDeleteRecord)
	got := updated.(model)
	if command != nil || got.dialog != dialogRecordEdit {
		t.Fatalf("delete from edit = dialog %d command %t", got.dialog, command != nil)
	}
	if got.recordFeature.edit.form.title.value != "Unsaved title" {
		t.Fatalf("unsaved edit was lost: %#v", got.recordFeature.edit.form)
	}
	if got.recordFeature.deletion.metadata.ID != "" {
		t.Fatalf("delete confirmation was opened: %#v", got.recordFeature.deletion)
	}
}

func TestRenderRecordDeleteWindow_UsesRedMetadataPreviewWithoutWarning(t *testing.T) {
	theme := newTheme()
	state := recordDeleteState{metadata: recordmodel.RecordMetadata{
		ID:        "7a79b627-0473-48a0-a001-887e79419719",
		Type:      recordmodel.RecordTypeText,
		Title:     "Recovery codes",
		Revision:  3,
		CreatedAt: time.Date(2026, 7, 19, 11, 0, 0, 0, time.Local),
		UpdatedAt: time.Date(2026, 7, 19, 12, 0, 0, 0, time.Local),
	}}

	rendered := renderRecordDeleteWindow(theme, 62, state, false, false, "", 1)
	plain := ansi.Strip(rendered)
	for _, want := range []string{
		"Delete Record Text",
		"ID: 7a79b627-0473-48a0-a001-887e79419719",
		"Revision: 3",
		"Created at: 2026-07-19 11:00",
		"Updated at: 2026-07-19 12:00",
		"Title: Recovery codes",
		"< Delete >",
		"< Cancel >",
	} {
		if !strings.Contains(plain, want) {
			t.Fatalf("delete dialog does not contain %q:\n%s", want, plain)
		}
	}
	for _, unwanted := range []string{"Delete this record?", "This action cannot be undone", "Type:"} {
		if strings.Contains(plain, unwanted) {
			t.Fatalf("delete dialog still contains %q:\n%s", unwanted, plain)
		}
	}
	lines := strings.Split(plain, "\n")
	updatedRow := lineIndexContaining(lines, "Updated at:")
	titleRow := lineIndexContaining(lines, "Title:")
	if updatedRow < 0 || titleRow != updatedRow+2 || strings.TrimSpace(lines[updatedRow+1]) != "" {
		t.Fatalf("delete dialog does not separate Title with one blank row:\n%s", plain)
	}
	if !strings.Contains(rendered, theme.errorLabel.Render("ID: ")) ||
		!strings.Contains(rendered, theme.errorBody.Render(state.metadata.ID)) {
		t.Fatal("delete dialog metadata preview does not use the red error styles")
	}
}

func TestRenderRecordDeleteWindow_PendingKeepsPreviewAndDisablesButtons(t *testing.T) {
	theme := newTheme()
	state := recordDeleteState{metadata: recordmodel.RecordMetadata{
		ID:       "7a79b627-0473-48a0-a001-887e79419719",
		Type:     recordmodel.RecordTypeBinary,
		Revision: 1,
	}}

	rendered := renderRecordDeleteWindow(theme, 62, state, true, true, "⠋", 0)
	plain := ansi.Strip(rendered)
	for _, want := range []string{"Delete Record Binary ⠋", "ID: ", state.metadata.ID, "Revision: 1", "< Delete >", "< Cancel >"} {
		if !strings.Contains(plain, want) {
			t.Fatalf("pending delete dialog does not contain %q:\n%s", want, plain)
		}
	}
	if strings.Contains(plain, "Deleting record") {
		t.Fatalf("pending delete dialog contains a body loading message:\n%s", plain)
	}
	if !strings.Contains(rendered, theme.errorButtonDisabledActive.Render("< Delete >")) ||
		!strings.Contains(rendered, theme.errorButtonDisabled.Render("< Cancel >")) {
		t.Fatal("pending delete dialog does not render both buttons disabled")
	}
}

func TestRecordDeleteWindowLayout_TracksWrappedTitle(t *testing.T) {
	theme := newTheme()
	state := recordDeleteState{metadata: recordmodel.RecordMetadata{
		ID:       "7a79b627-0473-48a0-a001-887e79419719",
		Type:     recordmodel.RecordTypeText,
		Title:    strings.Repeat("very long title ", 8),
		Revision: 1,
	}}
	layout := newRecordDeleteWindowLayout(theme, 44, state, false, false, "", 0)
	lines := strings.Split(ansi.Strip(layout.content), "\n")
	buttonRow := lineIndexContaining(lines, "< Delete >")

	if buttonRow < 0 || len(layout.buttonBounds) != 2 {
		t.Fatalf("button row = %d, bounds = %d", buttonRow, len(layout.buttonBounds))
	}
	if buttonRow <= 10 {
		t.Fatalf("wrapped title did not move the button row: %d", buttonRow)
	}
	for index, bounds := range layout.buttonBounds {
		if bounds.y != buttonRow {
			t.Fatalf("button %d y = %d, want rendered row %d", index, bounds.y, buttonRow)
		}
	}
}

func newRecordDeleteTestModel(t *testing.T, backend recordDeleteBackendStub) model {
	m := mustNewModel(t,
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
