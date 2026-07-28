package tui

import (
	"context"
	"errors"
	"path/filepath"
	"testing"
	"time"

	"github.com/xhrobj/gopherkeeper/internal/buildinfo"
	"github.com/xhrobj/gopherkeeper/internal/client/config"
	recordmodel "github.com/xhrobj/gopherkeeper/internal/model"
)

type recordCreateBackendStub struct {
	backendStub
	createRecord func(context.Context, string, recordmodel.RecordPayload) (recordmodel.Record, error)
}

type recordCreateRecordsBackendStub struct {
	backendStub
	createRecord func(context.Context, string, recordmodel.RecordPayload) (recordmodel.Record, error)
	listCalls    *int
}

func (stub recordCreateBackendStub) CreateRecord(
	ctx context.Context,
	title string,
	payload recordmodel.RecordPayload,
) (recordmodel.Record, error) {
	if stub.createRecord == nil {
		return recordmodel.Record{}, errors.New("unexpected CreateRecord call")
	}
	return stub.createRecord(ctx, title, payload)
}

func TestModel_RecordCreateMenuAvailability(t *testing.T) {
	m := newRecordCreateTestModel(t, recordCreateBackendStub{})
	definitions := m.currentMenuDefinitions()
	if definitions[menuRecord].disabled || definitions[menuRecord].items[2].disabled {
		t.Fatalf("Records menu = %#v", definitions[menuRecord])
	}

	m.authentication.session = authSession{state: authGuest}
	definitions = m.currentMenuDefinitions()
	if !definitions[menuRecord].disabled || !definitions[menuRecord].items[2].disabled {
		t.Fatalf("guest Records menu = %#v", definitions[menuRecord])
	}
}

func TestModel_RecordCreateFlow(t *testing.T) {
	createdAt := time.Date(2026, time.July, 20, 12, 0, 0, 0, time.UTC)
	created := recordmodel.Record{
		Metadata: recordmodel.RecordMetadata{
			ID:        "7a79b627-0473-48a0-a001-887e79419719",
			Type:      recordmodel.RecordTypeText,
			Title:     "Recovery codes",
			Revision:  1,
			CreatedAt: createdAt,
			UpdatedAt: createdAt,
		},
		Payload: &recordmodel.TextPayload{Text: "code-1\ncode-2"},
	}
	backend := recordCreateBackendStub{
		createRecord: func(_ context.Context, title string, payload recordmodel.RecordPayload) (recordmodel.Record, error) {
			if title != "Recovery codes" {
				t.Fatalf("title = %q", title)
			}
			text := payload.(*recordmodel.TextPayload)
			if text.Text != "code-1\ncode-2" {
				t.Fatalf("payload = %#v", text)
			}
			return created, nil
		},
	}
	m := newRecordCreateTestModel(t, backend)
	m.recordFeature.workspace.open = true
	m.recordFeature.workspace.apply([]recordmodel.RecordMetadata{{ID: "old", Type: recordmodel.RecordTypeText, Title: "Old", Revision: 1}}, 10)

	updated, command := m.activate(actionNewRecord)
	m = updated.(model)
	if command != nil || m.dialog != dialogRecordType {
		t.Fatalf("picker state = dialog %d command %v", m.dialog, command)
	}

	m.openRecordCreate(2)
	m.recordFeature.createForm.title.setValue("Recovery codes")
	m.recordFeature.createForm.text.setValue("code-1\ncode-2")
	m.recordFeature.createForm.setFocus(recordFormSubmit)

	updated, command = m.activateRecordCreate()
	m = updated.(model)
	if command == nil || !m.operations.request(operationCreateRecord).pending {
		t.Fatalf("request state = %#v", m.operations.request(operationCreateRecord))
	}

	message := commandResult[recordCreateResultMsg](t, command)
	updated, _ = m.Update(message)
	got := updated.(model)
	if got.dialog != dialogNone || got.alert != alertNotice || got.alertTitle != "Record created" {
		t.Fatalf("result state = dialog %d alert %d title %q", got.dialog, got.alert, got.alertTitle)
	}
	if len(got.recordFeature.workspace.records) != 2 || got.recordFeature.workspace.records[0].ID != created.Metadata.ID || got.recordFeature.workspace.selected != 0 {
		t.Fatalf("records = %#v selected %d", got.recordFeature.workspace.records, got.recordFeature.workspace.selected)
	}
	if got.recordFeature.createForm.recordType != "" || got.operations.request(operationCreateRecord).pending {
		t.Fatalf("create state was not cleared: form %#v request %#v", got.recordFeature.createForm, got.operations.request(operationCreateRecord))
	}
}

func TestModel_RecordCreateErrorKeepsForm(t *testing.T) {
	backend := recordCreateBackendStub{
		createRecord: func(context.Context, string, recordmodel.RecordPayload) (recordmodel.Record, error) {
			return recordmodel.Record{}, unavailableTestError()
		},
	}
	m := newRecordCreateTestModel(t, backend)
	m.openRecordCreate(0)
	m.recordFeature.createForm.title.setValue("GitHub")
	m.recordFeature.createForm.login.setValue("alice")
	m.recordFeature.createForm.password.setValue("secret")
	m.recordFeature.createForm.setFocus(recordFormSubmit)

	updated, command := m.activateRecordCreate()
	m = updated.(model)
	message := commandResult[recordCreateResultMsg](t, command)
	updated, _ = m.Update(message)
	got := updated.(model)

	if got.alert != alertError || got.alertReturnDialog != dialogRecordCreate {
		t.Fatalf("alert state = %#v", got)
	}
	if got.recordFeature.createForm.password.value != "secret" || got.recordFeature.createForm.title.value != "GitHub" {
		t.Fatalf("form was cleared after recoverable error: %#v", got.recordFeature.createForm)
	}
}

func TestModel_RecordCreateBinaryUsesFilePicker(t *testing.T) {
	m := newRecordCreateTestModel(t, recordCreateBackendStub{})
	m.openRecordCreate(3)
	m.recordFeature.createForm.setFocus(recordFormFilePath)

	updated, command := m.activateRecordCreate()
	got := updated.(model)
	if command == nil || got.dialog != dialogPathPicker || got.pathPicker.target != pathPickerBinaryCreateFile {
		t.Fatalf("picker state = dialog %d target %d command %t", got.dialog, got.pathPicker.target, command != nil)
	}

	root := t.TempDir()
	selected := filepath.Join(root, "fixtures", "backup.bin")
	got.pathPicker.rootDirectory = root
	got.applyPathSelection(selected)

	want := filepath.Join("fixtures", "backup.bin")
	if got.dialog != dialogRecordCreate || got.recordFeature.createForm.filePath.value != want {
		t.Fatalf("selected path state = dialog %d path %q, want %q", got.dialog, got.recordFeature.createForm.filePath.value, want)
	}
	if got.recordFeature.createForm.activeControl() != recordFormFilePath ||
		got.pathPicker.rootDirectory != "" ||
		got.pathPicker.currentDirectory != "" ||
		got.pathPicker.entries != nil {
		t.Fatalf("picker was not cleared or focus was not restored: picker %#v form %#v", got.pathPicker, got.recordFeature.createForm)
	}
}

func TestModel_CloseRecordCreateClearsSecretsAndCancelsRequest(t *testing.T) {
	m := newRecordCreateTestModel(t, recordCreateBackendStub{})
	m.openRecordCreate(0)
	m.recordFeature.createForm.password.setValue("secret")
	_, requestID := m.operations.begin(m.operationDone, operationCreateRecord)

	m.closeRecordCreate()
	if m.dialog != dialogNone || m.recordFeature.createForm.password.value != "" {
		t.Fatalf("close state = dialog %d form %#v", m.dialog, m.recordFeature.createForm)
	}
	if m.operations.request(operationCreateRecord).pending || m.operations.request(operationCreateRecord).id == requestID {
		t.Fatalf("request state = %#v", m.operations.request(operationCreateRecord))
	}
}

func (stub recordCreateRecordsBackendStub) CreateRecord(
	ctx context.Context,
	title string,
	payload recordmodel.RecordPayload,
) (recordmodel.Record, error) {
	return stub.createRecord(ctx, title, payload)
}

func (stub recordCreateRecordsBackendStub) ListRecords(context.Context) ([]recordmodel.RecordMetadata, error) {
	if stub.listCalls != nil {
		(*stub.listCalls)++
	}
	return nil, errors.New("refresh failed")
}

func (stub recordCreateRecordsBackendStub) GetRecord(context.Context, string) (recordmodel.Record, error) {
	return recordmodel.Record{}, errors.New("unexpected GetRecord call")
}

func TestModel_RecordCreateSuccessDoesNotDependOnListRefresh(t *testing.T) {
	created := recordmodel.Record{
		Metadata: recordmodel.RecordMetadata{
			ID: "7a79b627-0473-48a0-a001-887e79419719", Type: recordmodel.RecordTypeText, Title: "Recovery codes", Revision: 1,
		},
		Payload: &recordmodel.TextPayload{Text: "code-1"},
	}
	listCalls := 0
	backend := recordCreateRecordsBackendStub{
		createRecord: func(context.Context, string, recordmodel.RecordPayload) (recordmodel.Record, error) {
			return created, nil
		},
		listCalls: &listCalls,
	}
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
	m.openRecordCreate(2)
	m.recordFeature.createForm.title.setValue("Recovery codes")
	m.recordFeature.createForm.text.setValue("code-1")
	m.recordFeature.createForm.setFocus(recordFormSubmit)

	updated, command := m.activateRecordCreate()
	m = updated.(model)
	updated, followUp := m.Update(commandResult[recordCreateResultMsg](t, command))
	got := updated.(model)

	if followUp != nil || listCalls != 0 {
		t.Fatalf("successful create triggered list refresh: command %t calls %d", followUp != nil, listCalls)
	}
	if got.alert != alertNotice || got.alertTitle != "Record created" || got.recordFeature.workspace.open || len(got.recordFeature.workspace.records) != 0 {
		t.Fatalf("successful create state = alert %d title %q workspace %#v", got.alert, got.alertTitle, got.recordFeature.workspace)
	}
}

func TestModel_RecordCreateInvalidatesLoadingListSnapshot(t *testing.T) {
	created := recordmodel.Record{
		Metadata: recordmodel.RecordMetadata{ID: "new", Type: recordmodel.RecordTypeText, Title: "New", Revision: 1},
		Payload:  &recordmodel.TextPayload{Text: "secret"},
	}
	m := newRecordCreateTestModel(t, recordCreateBackendStub{})
	m.recordFeature.workspace.beginServer()
	m.operations.request(operationListRecords).pending = true
	m.operations.request(operationListRecords).id = 42
	m.operations.request(operationCreateRecord).pending = true
	m.operations.request(operationCreateRecord).id = 7
	m.dialog = dialogRecordCreate

	updated, _ := m.Update(recordCreateResultMsg{requestID: 7, record: created})
	got := updated.(model)
	if got.recordFeature.workspace.open || got.recordFeature.workspace.state != recordListIdle || got.operations.request(operationListRecords).pending || got.operations.request(operationListRecords).id == 42 {
		t.Fatalf("workspace after create = %#v request %#v", got.recordFeature.workspace, got.operations.request(operationListRecords))
	}

	updated, _ = got.Update(recordListResultMsg{
		requestID: 42,
		records:   []recordmodel.RecordMetadata{{ID: "old", Title: "Stale snapshot"}},
	})
	got = updated.(model)
	if got.recordFeature.workspace.open || len(got.recordFeature.workspace.records) != 0 {
		t.Fatalf("stale list snapshot was applied: %#v", got.recordFeature.workspace)
	}
}

func TestModel_RecordTypeWizardDefaultsToCredentialsAndUsesWizardOnlyForSelection(t *testing.T) {
	m := newRecordCreateTestModel(t, recordCreateBackendStub{})

	updated, command := m.activate(actionNewRecord)
	m = updated.(model)
	if command != nil || m.dialog != dialogRecordType || m.recordFeature.typePicker.selected != 0 {
		t.Fatalf("wizard state = dialog %d selected %d command %t", m.dialog, m.recordFeature.typePicker.selected, command != nil)
	}
	assertViewContains(t, m.renderDialog(), "New Record Wizard", "CREDENTIALS")

	updated, command = m.Update(keyPress("enter"))
	got := updated.(model)
	if command != nil || got.dialog != dialogRecordCreate || got.recordFeature.createForm.recordType != recordmodel.RecordTypeCredentials {
		t.Fatalf("default wizard selection = dialog %d type %q command %t", got.dialog, got.recordFeature.createForm.recordType, command != nil)
	}
	assertViewContains(t, got.renderDialog(), "New Credentials Record")
	assertViewExcludes(t, got.renderDialog(), "Wizard")
}

func newRecordCreateTestModel(t *testing.T, backend recordCreateBackendStub) model {
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
