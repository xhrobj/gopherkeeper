package tui

import (
	"strings"
	"testing"

	"github.com/xhrobj/gopherkeeper/internal/client/config"
	recordmodel "github.com/xhrobj/gopherkeeper/internal/model"
)

func TestModel_MouseClickRecordTypeSelectsAndCancels(t *testing.T) {
	m := newRecordsTestModel(t, config.Config{}, recordsBackendStub{})
	m.width = 100
	m.height = 36
	m.openRecordTypePicker()

	window, ok := m.dialogPlacement()
	if !ok {
		t.Fatal("record type dialog placement is unavailable")
	}
	layout := newRecordTypePickerLayout()
	typeBounds := window.screenBounds(layout.typeBounds)

	updated, command := m.updateMouse(clickInside(typeBounds[1]))
	got := updated.(model)
	if command != nil || got.dialog != dialogRecordCreate || got.recordFeature.createForm.recordType != recordCreateTypes[1] {
		t.Fatalf("selected type state = dialog %d type %q command %t", got.dialog, got.recordFeature.createForm.recordType, command != nil)
	}

	m = newRecordsTestModel(t, config.Config{}, recordsBackendStub{})
	m.width = 100
	m.height = 36
	m.openRecordTypePicker()
	window, _ = m.dialogPlacement()
	cancel := layout.cancelBounds.translated(window.x, window.y)
	updated, command = m.updateMouse(clickInside(cancel))
	got = updated.(model)
	if command != nil || got.dialog != dialogNone {
		t.Fatalf("cancel state = dialog %d command %t", got.dialog, command != nil)
	}
}

func TestModel_MouseClickRecordCreateFieldsAndButtons(t *testing.T) {
	m := newRecordsTestModel(t, config.Config{}, recordsBackendStub{})
	m.width = 100
	m.height = 40
	m.dialog = dialogRecordCreate
	m.recordFeature.createForm = newRecordCreateForm(recordmodel.RecordTypeCredentials)

	password := recordControlBounds(t, m.recordCreateFieldBounds(), recordFormPassword)
	updated, _ := m.updateMouse(clickInside(password))
	m = updated.(model)
	if m.recordFeature.createForm.activeControl() != recordFormPassword {
		t.Fatalf("active control = %d, want password", m.recordFeature.createForm.activeControl())
	}

	submit := recordControlBounds(t, m.recordCreateButtonBounds(), recordFormSubmit)
	updated, command := m.updateMouse(clickInside(submit))
	m = updated.(model)
	if command != nil || m.operations.pending(operationCreateRecord) {
		t.Fatal("disabled create submit started a request")
	}

	m.recordFeature.createForm.title.setValue("Account")
	m.recordFeature.createForm.login.setValue("alice")
	m.recordFeature.createForm.password.setValue("secret")
	submit = recordControlBounds(t, m.recordCreateButtonBounds(), recordFormSubmit)
	updated, command = m.updateMouse(clickInside(submit))
	m = updated.(model)
	if command == nil || !m.operations.pending(operationCreateRecord) {
		t.Fatal("enabled create submit did not start a request")
	}

	m = newRecordsTestModel(t, config.Config{}, recordsBackendStub{})
	m.width = 100
	m.height = 40
	m.dialog = dialogRecordCreate
	m.recordFeature.createForm = newRecordCreateForm(recordmodel.RecordTypeCredentials)
	cancel := recordControlBounds(t, m.recordCreateButtonBounds(), recordFormCancel)
	updated, command = m.updateMouse(clickInside(cancel))
	got := updated.(model)
	if command != nil || got.dialog != dialogNone {
		t.Fatalf("cancel state = dialog %d command %t", got.dialog, command != nil)
	}
}

func TestModel_MouseClickBinaryCreateAndEditFieldsOpenPickers(t *testing.T) {
	m := newRecordsTestModel(t, config.Config{}, recordsBackendStub{})
	m.width = 100
	m.height = 40
	m.dialog = dialogRecordCreate
	m.recordFeature.createForm = newRecordCreateForm(recordmodel.RecordTypeBinary)
	filePath := recordControlBounds(t, m.recordCreateFieldBounds(), recordFormFilePath)

	updated, command := m.updateMouse(clickInside(filePath))
	got := updated.(model)
	if command == nil || got.dialog != dialogPathPicker || got.pathPicker.target != pathPickerBinaryCreateFile {
		t.Fatalf("create picker = dialog %d target %d command %t", got.dialog, got.pathPicker.target, command != nil)
	}

	record := recordmodel.Record{
		Metadata: recordmodel.RecordMetadata{ID: "binary", Type: recordmodel.RecordTypeBinary, Title: "Backup", Revision: 1},
		Payload:  &recordmodel.BinaryPayload{Filename: "old.bin", Data: []byte{1}},
	}
	m = newRecordsTestModel(t, config.Config{}, recordsBackendStub{})
	m.width = 100
	m.height = 40
	m.recordFeature.edit.apply(record, dialogNone)
	m.dialog = dialogRecordEdit
	filePath = recordControlBounds(t, m.recordEditFieldBounds(), recordFormFilePath)

	updated, command = m.updateMouse(clickInside(filePath))
	got = updated.(model)
	if command == nil || got.dialog != dialogPathPicker || got.pathPicker.target != pathPickerBinaryEditFile {
		t.Fatalf("edit picker = dialog %d target %d command %t", got.dialog, got.pathPicker.target, command != nil)
	}
}

func TestModel_MouseClickRecordEditFieldsAndButtons(t *testing.T) {
	record := recordmodel.Record{
		Metadata: recordmodel.RecordMetadata{ID: "text", Type: recordmodel.RecordTypeText, Title: "Note", Revision: 2},
		Payload:  &recordmodel.TextPayload{Text: "body"},
	}
	m := newRecordsTestModel(t, config.Config{}, recordsBackendStub{})
	m.width = 100
	m.height = 40
	m.recordFeature.edit.apply(record, dialogNone)
	m.dialog = dialogRecordEdit

	title := recordControlBounds(t, m.recordEditFieldBounds(), recordFormTitle)
	updated, _ := m.updateMouse(clickInside(title))
	m = updated.(model)
	if m.recordFeature.edit.form.activeControl() != recordFormTitle {
		t.Fatalf("active control = %d, want title", m.recordFeature.edit.form.activeControl())
	}

	submit := recordControlBounds(t, m.recordEditButtonBounds(), recordFormSubmit)
	updated, command := m.updateMouse(clickInside(submit))
	m = updated.(model)
	if command == nil || !m.operations.pending(operationEditRecord) {
		t.Fatal("edit submit did not start a request")
	}

	m = newRecordsTestModel(t, config.Config{}, recordsBackendStub{})
	m.width = 100
	m.height = 40
	m.recordFeature.edit.apply(record, dialogNone)
	m.dialog = dialogRecordEdit
	m.operations.request(operationEditRecord).pending = true
	updated, command = m.updateMouse(clickInside(title))
	got := updated.(model)
	if command != nil || got.recordFeature.edit.form.activeControl() != recordFormTitle {
		t.Fatal("pending edit click changed the form")
	}
}

func TestModel_MouseClickRecordViewScrollsAndCloses(t *testing.T) {
	m := newRecordsTestModel(t, config.Config{}, recordsBackendStub{})
	m.width = 100
	m.height = 42
	m.dialog = dialogRecordView
	m.recordFeature.view.apply(recordmodel.Record{
		Metadata: recordmodel.RecordMetadata{ID: "text", Type: recordmodel.RecordTypeText, Title: "Long", Revision: 1},
		Payload:  &recordmodel.TextPayload{Text: strings.Repeat("line of text\n", 40)},
	}, m.width, m.height)

	window, ok := m.dialogPlacement()
	if !ok {
		t.Fatal("record view placement is unavailable")
	}
	layout := newRecordViewContentLayout(m.theme, window.width, window.height, m.recordFeature.view)
	down := layout.textAreaDown.translated(window.x, window.y)
	updated, command := m.updateMouse(clickInside(down))
	m = updated.(model)
	if command != nil || m.recordFeature.view.textArea.offset != 1 {
		t.Fatalf("text offset = %d command %t", m.recordFeature.view.textArea.offset, command != nil)
	}

	buttons := m.dialogButtonBounds()
	updated, command = m.updateMouse(clickInside(buttons[len(buttons)-1]))
	got := updated.(model)
	if command != nil || got.dialog != dialogNone {
		t.Fatalf("close state = dialog %d command %t", got.dialog, command != nil)
	}
}

func TestModel_MouseClickRecordDeleteAndBinarySaveButtons(t *testing.T) {
	metadata := recordmodel.RecordMetadata{ID: "record", Type: recordmodel.RecordTypeText, Title: "Note", Revision: 3}
	m := newRecordsTestModel(t, config.Config{}, recordsBackendStub{})
	m.width = 100
	m.height = 36
	m.authentication.session = authSession{state: authAuthenticated, login: "alice"}
	m.openRecordDelete(metadata)

	buttons := m.dialogButtonBounds()
	updated, command := m.updateMouse(clickInside(buttons[0]))
	m = updated.(model)
	if command == nil || !m.operations.pending(operationDeleteRecord) {
		t.Fatal("delete click did not start a request")
	}

	m = newRecordsTestModel(t, config.Config{}, recordsBackendStub{})
	m.width = 100
	m.height = 36
	m.openRecordDelete(metadata)
	buttons = m.dialogButtonBounds()
	updated, command = m.updateMouse(clickInside(buttons[1]))
	got := updated.(model)
	if command != nil || got.dialog != dialogNone {
		t.Fatalf("delete cancel = dialog %d command %t", got.dialog, command != nil)
	}

	m = binarySaveMouseTestModel(t)
	field := m.binarySaveFieldBounds()[0]
	updated, command = m.updateMouse(clickInside(field))
	got = updated.(model)
	if command == nil || got.dialog != dialogPathPicker || got.pathPicker.target != pathPickerBinarySaveDirectory {
		t.Fatalf("binary picker = dialog %d target %d command %t", got.dialog, got.pathPicker.target, command != nil)
	}

	m = binarySaveMouseTestModel(t)
	buttons = m.dialogButtonBounds()
	updated, command = m.updateMouse(clickInside(buttons[1]))
	got = updated.(model)
	if command != nil || got.dialog != dialogRecordView {
		t.Fatalf("binary cancel = dialog %d command %t", got.dialog, command != nil)
	}
}

func TestModel_RecordRowAtRejectsUnavailableRows(t *testing.T) {
	m := newRecordsTestModel(t, config.Config{}, recordsBackendStub{})
	if _, ok := m.recordRowAt(0, 0); ok {
		t.Fatal("record row exists without workspace placement")
	}

	m.width = 100
	m.height = 30
	m.recordFeature.workspace = recordWorkspace{open: true, state: recordListLoading}
	if _, ok := m.recordRowAt(10, 10); ok {
		t.Fatal("record row exists while list is loading")
	}

	m.recordFeature.workspace = recordWorkspace{
		open:    true,
		state:   recordListReady,
		records: []recordmodel.RecordMetadata{{ID: "one"}},
	}
	window, ok := m.workspacePlacement()
	if !ok {
		t.Fatal("workspace placement is unavailable")
	}
	if _, ok := m.recordRowAt(window.x-1, window.y-1); ok {
		t.Fatal("record row exists outside workspace")
	}

	layout := newRecordWorkspaceLayout(window.width, window.height)
	rows := layout.rowBounds.translated(window.x, window.y)
	if _, ok := m.recordRowAt(rows.x, rows.y+1); ok {
		t.Fatal("record row exists beyond available records")
	}
}

func recordControlBounds(t *testing.T, bounds []recordFormControlBound, control recordFormControl) layoutBounds {
	t.Helper()
	for _, bound := range bounds {
		if bound.control == control {
			return bound.bounds
		}
	}
	t.Fatalf("control %d bounds were not found", control)
	return layoutBounds{}
}

func binarySaveMouseTestModel(t *testing.T) model {
	t.Helper()
	m := newRecordsTestModel(t, config.Config{}, recordsBackendStub{})
	m.width = 100
	m.height = 36
	m.dialog = dialogRecordView
	m.recordFeature.view = recordViewState{
		status: recordViewReady,
		record: recordmodel.Record{
			Metadata: recordmodel.RecordMetadata{ID: "binary", Type: recordmodel.RecordTypeBinary, Title: "Backup", Revision: 1},
			Payload:  &recordmodel.BinaryPayload{Filename: "backup.bin", Data: []byte{1}},
		},
	}
	m.openBinarySave()
	return m
}
