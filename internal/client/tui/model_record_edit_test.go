package tui

import (
	"bytes"
	"context"
	"errors"
	"path/filepath"
	"testing"
	"time"

	"github.com/xhrobj/gopherkeeper/internal/buildinfo"
	"github.com/xhrobj/gopherkeeper/internal/client/config"
	recordmodel "github.com/xhrobj/gopherkeeper/internal/model"
)

type recordEditBackendStub struct {
	backendStub
	getRecord    func(context.Context, string) (recordmodel.Record, error)
	updateRecord func(context.Context, string, int64, string, recordmodel.RecordPayload) (recordmodel.Record, error)
}

func (stub recordEditBackendStub) ListRecords(context.Context) ([]recordmodel.RecordMetadata, error) {
	return nil, nil
}

func (stub recordEditBackendStub) GetRecord(ctx context.Context, recordID string) (recordmodel.Record, error) {
	if stub.getRecord == nil {
		return recordmodel.Record{}, errors.New("unexpected GetRecord call")
	}
	return stub.getRecord(ctx, recordID)
}

func (stub recordEditBackendStub) UpdateRecord(
	ctx context.Context,
	recordID string,
	revision int64,
	title string,
	payload recordmodel.RecordPayload,
) (recordmodel.Record, error) {
	if stub.updateRecord == nil {
		return recordmodel.Record{}, errors.New("unexpected UpdateRecord call")
	}
	return stub.updateRecord(ctx, recordID, revision, title, payload)
}

func newRecordEditTestModel(t *testing.T, backend recordEditBackendStub) model {
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

func textRecordForEdit() recordmodel.Record {
	return recordmodel.Record{
		Metadata: recordmodel.RecordMetadata{
			ID:        "7a79b627-0473-48a0-a001-887e79419719",
			Type:      recordmodel.RecordTypeText,
			Title:     "Recovery codes",
			Revision:  2,
			CreatedAt: time.Date(2026, time.July, 20, 12, 0, 0, 0, time.UTC),
			UpdatedAt: time.Date(2026, time.July, 20, 13, 0, 0, 0, time.UTC),
		},
		Payload: &recordmodel.TextPayload{Text: "code-1\ncode-2", Metadata: "personal"},
	}
}

func TestModel_RecordEditMenuAvailability(t *testing.T) {
	m := newRecordEditTestModel(t, recordEditBackendStub{})
	m.recordFeature.workspace.apply([]recordmodel.RecordMetadata{textRecordForEdit().Metadata}, 10)

	definitions := m.currentMenuDefinitions()
	if definitions[menuRecord].items[4].disabled {
		t.Fatalf("Edit menu item is disabled: %#v", definitions[menuRecord])
	}

	m.authentication.session = authSession{state: authGuest}
	definitions = m.currentMenuDefinitions()
	if !definitions[menuRecord].items[4].disabled {
		t.Fatal("guest Edit menu item is enabled")
	}
}

func TestModel_RecordEditLoadsSelectedRecord(t *testing.T) {
	record := textRecordForEdit()
	backend := recordEditBackendStub{
		getRecord: func(_ context.Context, recordID string) (recordmodel.Record, error) {
			if recordID != record.Metadata.ID {
				t.Fatalf("record id = %q, want %q", recordID, record.Metadata.ID)
			}
			return record, nil
		},
	}
	m := newRecordEditTestModel(t, backend)
	m.recordFeature.workspace.apply([]recordmodel.RecordMetadata{record.Metadata}, 10)

	command := m.openRecordEdit()
	if command == nil || m.dialog != dialogRecordEdit || m.recordFeature.edit.status != recordEditLoading {
		t.Fatalf("loading state = dialog %d edit %#v", m.dialog, m.recordFeature.edit)
	}
	assertViewContains(t, m.renderDialog(), "Edit Text Record "+m.spinnerFrameValue(), record.Metadata.Title, "< Save >", "< Cancel >")
	assertViewExcludes(t, m.renderDialog(), "Loading record")

	updated, _ := m.Update(commandResult[recordEditLoadResultMsg](t, command))
	got := updated.(model)
	if got.recordFeature.edit.status != recordEditReady || got.recordFeature.edit.form.title.value != record.Metadata.Title {
		t.Fatalf("edit state = %#v", got.recordFeature.edit)
	}
	if got.recordFeature.edit.form.text.value != "code-1\ncode-2" || got.recordFeature.edit.form.metadata.value != "personal" {
		t.Fatalf("edit form = %#v", got.recordFeature.edit.form)
	}
}

func TestModel_RecordEditFlow(t *testing.T) {
	record := textRecordForEdit()
	updatedRecord := record
	updatedRecord.Metadata.Title = "Updated recovery codes"
	updatedRecord.Metadata.Revision = 3
	updatedRecord.Payload = &recordmodel.TextPayload{Text: "code-3", Metadata: "updated"}

	backend := recordEditBackendStub{
		updateRecord: func(
			_ context.Context,
			recordID string,
			revision int64,
			title string,
			payload recordmodel.RecordPayload,
		) (recordmodel.Record, error) {
			if recordID != record.Metadata.ID || revision != 2 || title != updatedRecord.Metadata.Title {
				t.Fatalf("update request = id %q revision %d title %q", recordID, revision, title)
			}
			text := payload.(*recordmodel.TextPayload)
			if text.Text != "code-3" || text.Metadata != "updated" {
				t.Fatalf("payload = %#v", text)
			}
			return updatedRecord, nil
		},
	}
	m := newRecordEditTestModel(t, backend)
	m.recordFeature.workspace.apply([]recordmodel.RecordMetadata{record.Metadata}, 10)
	m.recordFeature.edit.apply(record, dialogNone)
	m.dialog = dialogRecordEdit
	m.recordFeature.edit.form.title.setValue(updatedRecord.Metadata.Title)
	m.recordFeature.edit.form.mutateText(func() { m.recordFeature.edit.form.text.setValue("code-3") })
	m.recordFeature.edit.form.setFocus(recordFormMetadata)
	for m.recordFeature.edit.form.metadata.value != "" {
		m.recordFeature.edit.form.backspace()
	}
	m.recordFeature.edit.form.insert("updated")
	m.recordFeature.edit.form.setFocus(recordFormSubmit)

	updated, command := m.activateRecordEdit()
	m = updated.(model)
	if command == nil || !m.operations.request(operationEditRecord).pending {
		t.Fatalf("request state = %#v", m.operations.request(operationEditRecord))
	}

	updated, _ = m.Update(commandResult[recordEditResultMsg](t, command))
	got := updated.(model)
	if got.alert != alertNotice || got.alertTitle != "Record updated" || got.alertReturnDialog != dialogRecordView {
		t.Fatalf("result state = alert %d title %q return %d", got.alert, got.alertTitle, got.alertReturnDialog)
	}
	if got.recordFeature.view.record.Metadata.Revision != 3 || got.recordFeature.workspace.records[0].Revision != 3 {
		t.Fatalf("updated states = view %#v list %#v", got.recordFeature.view.record.Metadata, got.recordFeature.workspace.records)
	}
}

func TestModel_RecordEditConflictKeepsForm(t *testing.T) {
	record := textRecordForEdit()
	backend := recordEditBackendStub{
		updateRecord: func(context.Context, string, int64, string, recordmodel.RecordPayload) (recordmodel.Record, error) {
			return recordmodel.Record{}, recordmodel.ErrRecordRevisionConflict
		},
	}
	m := newRecordEditTestModel(t, backend)
	m.recordFeature.edit.apply(record, dialogNone)
	m.dialog = dialogRecordEdit
	m.recordFeature.edit.form.title.setValue("Locally changed")
	m.recordFeature.edit.form.setFocus(recordFormSubmit)

	updated, command := m.activateRecordEdit()
	m = updated.(model)
	updated, _ = m.Update(commandResult[recordEditResultMsg](t, command))
	got := updated.(model)
	if got.alert != alertError || got.alertReturnDialog != dialogRecordEdit {
		t.Fatalf("alert state = %#v", got)
	}
	if got.recordFeature.edit.form.title.value != "Locally changed" || got.recordFeature.edit.record.Metadata.Revision != 2 {
		t.Fatalf("edit state was lost: %#v", got.recordFeature.edit)
	}
}

func TestModel_RecordEditBinaryUsesFilePicker(t *testing.T) {
	record := recordmodel.Record{
		Metadata: recordmodel.RecordMetadata{Type: recordmodel.RecordTypeBinary, Title: "Backup", Revision: 1},
		Payload:  &recordmodel.BinaryPayload{Filename: "backup.bin", Data: []byte{0x01}},
	}
	m := newRecordEditTestModel(t, recordEditBackendStub{})
	m.recordFeature.edit.apply(record, dialogNone)
	m.dialog = dialogRecordEdit
	m.recordFeature.edit.form.setFocus(recordFormFilePath)

	updated, command := m.activateRecordEdit()
	got := updated.(model)
	if command == nil || got.dialog != dialogPathPicker || got.pathPicker.target != pathPickerBinaryEditFile {
		t.Fatalf("picker state = dialog %d target %d command %t", got.dialog, got.pathPicker.target, command != nil)
	}

	root := t.TempDir()
	selected := filepath.Join(root, "fixtures", "replacement.bin")
	got.pathPicker.rootDirectory = root
	got.applyPathSelection(selected)

	want := filepath.Join("fixtures", "replacement.bin")
	if got.dialog != dialogRecordEdit || got.recordFeature.edit.form.filePath.value != want {
		t.Fatalf("selected path state = dialog %d path %q, want %q", got.dialog, got.recordFeature.edit.form.filePath.value, want)
	}
	if got.recordFeature.edit.form.activeControl() != recordFormFilePath ||
		got.pathPicker.rootDirectory != "" ||
		got.pathPicker.currentDirectory != "" ||
		got.pathPicker.entries != nil {
		t.Fatalf("picker was not cleared or focus was not restored: picker %#v form %#v", got.pathPicker, got.recordFeature.edit.form)
	}
}

func TestRecordEditForm_BinaryKeepsExistingData(t *testing.T) {
	data := []byte{0x01, 0x02, 0xff}
	record := recordmodel.Record{
		Metadata: recordmodel.RecordMetadata{Type: recordmodel.RecordTypeBinary, Title: "Backup", Revision: 1},
		Payload: &recordmodel.BinaryPayload{
			Filename: "backup.bin", Data: data, Metadata: "nightly",
		},
	}
	form := newRecordEditForm(record)
	if !form.canSubmit() {
		t.Fatal("binary edit form cannot submit without a replacement file")
	}
	input := recordFormInputFrom(form)
	payload, err := input.buildPayload(func(string) (string, []byte, error) {
		return "", nil, errors.New("must not be called")
	}, record.Payload)
	if err != nil {
		t.Fatalf("payload() error = %v", err)
	}
	binary := payload.(*recordmodel.BinaryPayload)
	if binary.Filename != "backup.bin" || !bytes.Equal(binary.Data, data) || binary.Metadata != "nightly" {
		t.Fatalf("binary payload = %#v", binary)
	}
}

func TestModel_CloseRecordEditClearsSecretsAndBinary(t *testing.T) {
	record := recordmodel.Record{
		Metadata: recordmodel.RecordMetadata{Type: recordmodel.RecordTypeCredentials, Title: "GitHub", Revision: 1},
		Payload:  &recordmodel.CredentialsPayload{Login: "alice", Password: "secret"},
	}
	m := newRecordEditTestModel(t, recordEditBackendStub{})
	m.recordFeature.edit.apply(record, dialogNone)
	m.dialog = dialogRecordEdit
	_, requestID := m.operations.begin(m.operationDone, operationEditRecord)

	m.closeRecordEdit()
	if m.dialog != dialogNone || m.recordFeature.edit.form.password.value != "" || m.recordFeature.edit.record.Payload != nil {
		t.Fatalf("close state = dialog %d edit %#v", m.dialog, m.recordFeature.edit)
	}
	if m.operations.request(operationEditRecord).pending || m.operations.request(operationEditRecord).id == requestID {
		t.Fatalf("request state = %#v", m.operations.request(operationEditRecord))
	}
}

func TestRecordEditForm_TextKeepsOriginalLineEndingsAndTabsUntilTextChanges(t *testing.T) {
	const original = "first\tcolumn\r\nsecond"
	record := recordmodel.Record{
		Metadata: recordmodel.RecordMetadata{Type: recordmodel.RecordTypeText, Title: "Formatting", Revision: 1},
		Payload:  &recordmodel.TextPayload{Text: original, Metadata: "personal"},
	}

	form := newRecordEditForm(record)
	form.title.setValue("Renamed formatting")
	input := recordFormInputFrom(form)
	payload, err := input.buildPayload(func(string) (string, []byte, error) {
		return "", nil, errors.New("must not be called")
	}, record.Payload)
	if err != nil {
		t.Fatalf("payload() error = %v", err)
	}
	text := payload.(*recordmodel.TextPayload)
	if text.Text != original {
		t.Fatalf("text = %q, want original %q", text.Text, original)
	}

	form.mutateText(func() { form.text.setValue("changed") })
	input = recordFormInputFrom(form)
	payload, err = input.buildPayload(func(string) (string, []byte, error) {
		return "", nil, errors.New("must not be called")
	}, record.Payload)
	if err != nil {
		t.Fatalf("changed payload() error = %v", err)
	}
	if got := payload.(*recordmodel.TextPayload).Text; got != "changed" {
		t.Fatalf("changed text = %q, want %q", got, "changed")
	}
}

func TestRecordEditForm_BinaryKeepsEmptyNonNilData(t *testing.T) {
	record := recordmodel.Record{
		Metadata: recordmodel.RecordMetadata{Type: recordmodel.RecordTypeBinary, Title: "Empty backup", Revision: 1},
		Payload: &recordmodel.BinaryPayload{
			Filename: "empty.bin", Data: []byte{}, Metadata: "empty",
		},
	}

	form := newRecordEditForm(record)
	if !form.canSubmit() {
		t.Fatal("empty binary edit form cannot submit without a replacement file")
	}
	input := recordFormInputFrom(form)
	if !input.binaryExisting {
		t.Fatal("empty binary data was not recognized in edit input")
	}
	payload, err := input.buildPayload(func(string) (string, []byte, error) {
		return "", nil, errors.New("must not be called")
	}, record.Payload)
	if err != nil {
		t.Fatalf("payload() error = %v", err)
	}
	binary := payload.(*recordmodel.BinaryPayload)
	if binary.Data == nil || len(binary.Data) != 0 {
		t.Fatalf("binary data = %#v, want non-nil empty slice", binary.Data)
	}
}
