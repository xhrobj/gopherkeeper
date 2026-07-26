package tui

import (
	"context"
	"errors"
	"path/filepath"
	"strings"
	"testing"

	tea "charm.land/bubbletea/v2"
	"github.com/charmbracelet/x/ansi"
	"github.com/xhrobj/gopherkeeper/internal/client/config"
	recordmodel "github.com/xhrobj/gopherkeeper/internal/model"
)

func TestBinarySaveCommand(t *testing.T) {
	var gotPath string
	var gotData []byte
	command := binarySaveCommand(
		context.Background(),
		func(path string, data []byte) error {
			gotPath = path
			gotData = append([]byte(nil), data...)
			return nil
		},
		42,
		"backup.bin",
		[]byte{0x01, 0x02},
	)

	message := command().(binarySaveResultMsg)
	if message.requestID != 42 || message.path != "backup.bin" || message.err != nil {
		t.Fatalf("message = %#v", message)
	}
	if gotPath != "backup.bin" || len(gotData) != 2 || gotData[1] != 0x02 {
		t.Fatalf("writer input = %q %v", gotPath, gotData)
	}
}

func TestCleanBinarySaveError(t *testing.T) {
	if got := cleanBinarySaveError(nil); got != "Unknown binary save error" {
		t.Fatalf("nil error = %q", got)
	}
	if got := cleanBinarySaveError(errors.New("create output file: file exists")); got != "Create output file: file exists" {
		t.Fatalf("create error = %q", got)
	}

	long := strings.Repeat("x", binarySaveErrorMaxRunes+1)
	got := cleanBinarySaveError(errors.New(long))
	if got != "X"+strings.Repeat("x", binarySaveErrorMaxRunes-1)+"..." {
		t.Fatalf("long error = %q", got)
	}
}

func TestModel_OpenBinarySavePrefillsFilename(t *testing.T) {
	m := newRecordsTestModel(t, config.Config{}, recordsBackendStub{})
	m.dialog = dialogRecordView
	m.recordFeature.view = recordViewState{
		status: recordViewReady,
		record: recordmodel.Record{
			Metadata: recordmodel.RecordMetadata{Type: recordmodel.RecordTypeBinary},
			Payload:  &recordmodel.BinaryPayload{Filename: "backup.bin", Data: []byte("data")},
		},
	}

	m.openBinarySave()
	if m.dialog != dialogBinarySave {
		t.Fatalf("dialog = %v, want binary save", m.dialog)
	}
	if m.recordFeature.binarySaveForm.path != "backup.bin" || m.recordFeature.binarySaveForm.fileName != "backup.bin" || m.recordFeature.binarySaveForm.focus != binarySavePath {
		t.Fatalf("form = %#v", m.recordFeature.binarySaveForm)
	}
}

func TestModel_BinarySavePathOpensDirectoryPicker(t *testing.T) {
	m := newRecordsTestModel(t, config.Config{}, recordsBackendStub{})
	m.dialog = dialogBinarySave
	m.recordFeature.binarySaveForm = newBinarySaveForm("backup.bin")

	updated, command := m.activateBinarySave()
	got := updated.(model)
	if command == nil || got.dialog != dialogPathPicker {
		t.Fatalf("picker state = dialog %v command %t", got.dialog, command != nil)
	}
	if got.pathPicker.target != pathPickerBinarySaveDirectory || got.pathPicker.fileName != "backup.bin" {
		t.Fatalf("picker = %#v", got.pathPicker)
	}
}

func TestBinarySaveForm_SelectedDirectoryBuildsFinalPath(t *testing.T) {
	form := newBinarySaveForm("backup.bin")
	form.setDirectory("exports")
	if form.path != "exports/backup.bin" {
		t.Fatalf("path = %q, want exports/backup.bin", form.path)
	}
	form.setDirectory(".")
	if form.path != "backup.bin" {
		t.Fatalf("root path = %q, want backup.bin", form.path)
	}
}

func TestModel_BinarySaveResultReturnsToRecordView(t *testing.T) {
	m := newRecordsTestModel(t, config.Config{}, recordsBackendStub{})
	m.dialog = dialogBinarySave
	m.recordFeature.view = recordViewState{
		status: recordViewReady,
		record: recordmodel.Record{
			Metadata: recordmodel.RecordMetadata{Type: recordmodel.RecordTypeBinary},
			Payload:  &recordmodel.BinaryPayload{Filename: "backup.bin", Data: []byte("data")},
		},
	}
	m.operations.request(operationBinarySave).pending = true
	m.operations.request(operationBinarySave).id = 42

	updated, _ := m.Update(binarySaveResultMsg{requestID: 42, path: "backup.bin"})
	got := updated.(model)
	if got.alert != alertNotice || got.alertTitle != "Binary saved" {
		t.Fatalf("alert = %v %q", got.alert, got.alertTitle)
	}
	if got.alertReturnDialog != dialogRecordView {
		t.Fatalf("return dialog = %v, want record view", got.alertReturnDialog)
	}
	if got.activeButton != 1 {
		t.Fatalf("active button = %d, want Close (1)", got.activeButton)
	}
	got.dismissAlert()
	if got.dialog != dialogRecordView || got.activeButton != 1 {
		t.Fatalf("after alert: dialog = %v, active button = %d; want record view and Close (1)", got.dialog, got.activeButton)
	}
}

func TestModel_CloseBinarySaveReturnsToClose(t *testing.T) {
	tests := []struct {
		name string
		act  func(model) (model, tea.Cmd)
	}{
		{
			name: "Esc",
			act: func(m model) (model, tea.Cmd) {
				updated, command := m.updateBinarySave("esc")
				return updated.(model), command
			},
		},
		{
			name: "cancel",
			act: func(m model) (model, tea.Cmd) {
				m.recordFeature.binarySaveForm.focus = binarySaveCancel
				updated, command := m.activateBinarySave()
				return updated.(model), command
			},
		},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			m := newRecordsTestModel(t, config.Config{}, recordsBackendStub{})
			m.dialog = dialogBinarySave
			m.recordFeature.view = recordViewState{
				status: recordViewReady,
				record: recordmodel.Record{
					Metadata: recordmodel.RecordMetadata{Type: recordmodel.RecordTypeBinary},
					Payload:  &recordmodel.BinaryPayload{Filename: "backup.bin", Data: []byte("data")},
				},
			}
			m.recordFeature.binarySaveForm = newBinarySaveForm("backup.bin")

			got, command := tt.act(m)
			if command != nil {
				t.Fatal("close action returned an unexpected command")
			}
			if got.dialog != dialogRecordView || got.activeButton != 1 {
				t.Fatalf("dialog = %v, active button = %d; want record view and Close (1)", got.dialog, got.activeButton)
			}
		})
	}
}

func TestModel_BinarySaveErrorPreservesForm(t *testing.T) {
	m := newRecordsTestModel(t, config.Config{}, recordsBackendStub{})
	m.dialog = dialogBinarySave
	m.recordFeature.binarySaveForm = newBinarySaveForm("backup.bin")
	m.recordFeature.binarySaveForm.setDirectory("exports")
	m.operations.request(operationBinarySave).pending = true
	m.operations.request(operationBinarySave).id = 42

	updated, _ := m.Update(binarySaveResultMsg{requestID: 42, err: errors.New("file exists")})
	got := updated.(model)
	if got.alert != alertError || got.alertReturnDialog != dialogBinarySave {
		t.Fatalf("alert = %v, return dialog = %v", got.alert, got.alertReturnDialog)
	}
	if got.recordFeature.binarySaveForm.path != "exports/backup.bin" {
		t.Fatalf("path = %q, want preserved value", got.recordFeature.binarySaveForm.path)
	}
}

func TestModel_ActivateBinarySaveWritesPayload(t *testing.T) {
	var gotPath string
	var gotData []byte
	m := newRecordsTestModel(t, config.Config{}, recordsBackendStub{})
	m.dialog = dialogBinarySave
	m.recordFeature.view = recordViewState{
		status: recordViewReady,
		record: recordmodel.Record{
			Metadata: recordmodel.RecordMetadata{Type: recordmodel.RecordTypeBinary},
			Payload:  &recordmodel.BinaryPayload{Filename: "backup.bin", Data: []byte{0x01, 0x02}},
		},
	}
	m.recordFeature.binarySaveForm = newBinarySaveForm("backup.bin")
	m.recordFeature.binarySaveForm.setDirectory("exports")
	m.recordFeature.binarySaveForm.focus = binarySaveSubmit
	m.writeBinaryFile = func(path string, data []byte) error {
		gotPath = path
		gotData = append([]byte(nil), data...)
		return nil
	}

	updated, command := m.activateBinarySave()
	got := updated.(model)
	if command == nil || !got.operations.request(operationBinarySave).pending {
		t.Fatal("binary save request did not start")
	}
	_ = command()
	if gotPath != "exports/backup.bin" || len(gotData) != 2 || gotData[1] != 0x02 {
		t.Fatalf("writer input = %q %v", gotPath, gotData)
	}
}

func TestModel_BinarySaveDirectoryPickerAppliesDirectoryAndFilename(t *testing.T) {
	root := t.TempDir()
	m := newRecordsTestModel(t, config.Config{}, recordsBackendStub{})
	m.dialog = dialogPathPicker
	m.recordFeature.binarySaveForm = newBinarySaveForm("backup.bin")
	m.pathPicker = pathPicker{
		target:        pathPickerBinarySaveDirectory,
		rootDirectory: root,
		fileName:      "backup.bin",
	}

	m.applyPathSelection(filepath.Join(root, "exports"))
	if m.dialog != dialogBinarySave || m.recordFeature.binarySaveForm.path != "exports/backup.bin" {
		t.Fatalf("selection result = dialog %v path %q, want binary save and exports/backup.bin", m.dialog, m.recordFeature.binarySaveForm.path)
	}
}

func TestRenderBinarySaveWindow(t *testing.T) {
	view := ansi.Strip(renderBinarySaveWindow(newTheme(), 64, newBinarySaveForm("backup.bin"), false))
	for _, want := range []string{"Save Binary As", "Path", "backup.bin", "<...>", "< Save >", "< Cancel >"} {
		if !strings.Contains(view, want) {
			t.Fatalf("binary save window does not contain %q:\n%s", want, view)
		}
	}
}
