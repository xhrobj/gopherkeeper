package cli

import (
	"bytes"
	"context"
	"errors"
	"io"
	"os"
	"path/filepath"
	"strings"
	"testing"

	"github.com/xhrobj/gopherkeeper/internal/client/config"
	"github.com/xhrobj/gopherkeeper/internal/client/usecase"
	"github.com/xhrobj/gopherkeeper/internal/model"
)

const testRecordID = "550e8400-e29b-41d4-a716-446655440000"

func TestRecordsUpdateTextCommand(t *testing.T) {
	isolateClientConfig(t)

	textFile := writeTestFile(t, "note.txt", "updated secret")
	metadataFile := writeTestFile(t, "metadata.txt", "updated private metadata")
	var gotConfig config.Config
	app := newApplicationStub(t)
	app.updateRecord = func(_ context.Context, request usecase.UpdateRecordRequest) (model.Record, error) {
		assertTextUpdateRequest(t, request)
		return model.Record{Metadata: model.RecordMetadata{ID: testRecordID, Revision: 2}}, nil
	}
	factory := newClientFactoryStub(t)
	factory.newApplication = func(cfg config.Config) (application, error) {
		gotConfig = cfg
		return app, nil
	}

	var stdout bytes.Buffer
	err := runTestCommand(
		t,
		[]string{
			"gkeep",
			"--address", "localhost:9090",
			"records", "update-text", testRecordID,
			"-r", "1",
			"--title", "Updated note",
			"--text-file", textFile,
			"--metadata-file", metadataFile,
		},
		nil,
		&stdout,
		io.Discard,
		factory,
	)
	if err != nil {
		t.Fatalf("run update-text command error = %v", err)
	}
	if gotConfig.Address != "localhost:9090" {
		t.Errorf("address = %q, want localhost:9090", gotConfig.Address)
	}
	if got := stdout.String(); got != "Updated text record "+testRecordID+" to revision 2.\n" {
		t.Errorf("output = %q", got)
	}
}

func TestRecordsUpdateTextCommand_RequiresRecordID(t *testing.T) {
	isolateClientConfig(t)

	textFile := writeTestFile(t, "note.txt", "updated secret")
	err := runTestCommand(
		t,
		[]string{
			"gkeep",
			"records", "update-text",
			"--revision", "1",
			"--title", "Updated note",
			"--text-file", textFile,
		},
		nil,
		io.Discard,
		io.Discard,
		nil,
	)
	if err == nil || err.Error() != "record id is required" {
		t.Fatalf("run update-text command error = %v, want record id is required", err)
	}
}

func TestRecordsDeleteCommand(t *testing.T) {
	isolateClientConfig(t)

	var gotConfig config.Config
	app := newApplicationStub(t)
	app.deleteRecord = func(_ context.Context, request usecase.DeleteRecordRequest) error {
		if request.RecordID != testRecordID || request.ExpectedRevision != 2 {
			t.Errorf("request = %+v, want ID and revision 2", request)
		}
		return nil
	}
	factory := newClientFactoryStub(t)
	factory.newApplication = func(cfg config.Config) (application, error) {
		gotConfig = cfg
		return app, nil
	}

	var stdout bytes.Buffer
	err := runTestCommand(
		t,
		[]string{
			"gkeep",
			"--address", "localhost:9090",
			"records", "delete", testRecordID,
			"-r", "2",
		},
		nil,
		&stdout,
		io.Discard,
		factory,
	)
	if err != nil {
		t.Fatalf("run delete command error = %v", err)
	}
	if gotConfig.Address != "localhost:9090" {
		t.Errorf("address = %q, want localhost:9090", gotConfig.Address)
	}
	if got := stdout.String(); got != "Deleted record "+testRecordID+".\n" {
		t.Errorf("output = %q", got)
	}
}

func TestRecordsDeleteCommand_RequiresRecordID(t *testing.T) {
	isolateClientConfig(t)

	err := runTestCommand(
		t,
		[]string{"gkeep", "records", "delete", "--revision", "2"},
		nil,
		io.Discard,
		io.Discard,
		nil,
	)
	if err == nil || err.Error() != "record id is required" {
		t.Fatalf("run delete command error = %v, want record id is required", err)
	}
}

func TestExecuteUpdateTextRecord_ReturnsReadError(t *testing.T) {
	app := newApplicationStub(t)
	err := executeUpdateTextRecord(
		context.Background(),
		app,
		&bytes.Buffer{},
		textRecordUpdateCommandRequest{
			recordID:         testRecordID,
			expectedRevision: 1,
			title:            "Updated note",
			textFile:         "missing.txt",
		},
	)
	if err == nil || !strings.Contains(err.Error(), "open text file") {
		t.Fatalf("executeUpdateTextRecord() error = %v, want text file open error", err)
	}
}

func TestExecuteDeleteRecord(t *testing.T) {
	app := newApplicationStub(t)
	app.deleteRecord = func(_ context.Context, request usecase.DeleteRecordRequest) error {
		if request.RecordID != testRecordID || request.ExpectedRevision != 2 {
			t.Errorf("request = %+v", request)
		}
		return nil
	}

	var output bytes.Buffer
	if err := executeDeleteRecord(context.Background(), app, &output, testRecordID, 2); err != nil {
		t.Fatalf("executeDeleteRecord() error = %v", err)
	}

	want := "Deleted record " + testRecordID + ".\n"
	if output.String() != want {
		t.Errorf("output = %q, want %q", output.String(), want)
	}
}

func TestExecuteDeleteRecord_ReturnsUsecaseError(t *testing.T) {
	wantErr := errors.New("record revision conflict")
	app := newApplicationStub(t)
	app.deleteRecord = func(context.Context, usecase.DeleteRecordRequest) error {
		return wantErr
	}

	err := executeDeleteRecord(context.Background(), app, &bytes.Buffer{}, testRecordID, 1)
	if !errors.Is(err, wantErr) {
		t.Fatalf("executeDeleteRecord() error = %v, want %v", err, wantErr)
	}
}

func assertTextUpdateRequest(t *testing.T, request usecase.UpdateRecordRequest) {
	t.Helper()

	if request.RecordID != testRecordID || request.ExpectedRevision != 1 || request.Title != "Updated note" {
		t.Errorf("request = %+v, want text update values", request)
	}
	payload, ok := request.Payload.(*model.TextPayload)
	if !ok {
		t.Fatalf("payload type = %T, want *model.TextPayload", request.Payload)
	}
	if payload.Text != "updated secret" || payload.Metadata != "updated private metadata" {
		t.Errorf("payload = %+v, want updated values", payload)
	}
}

func writeTestFile(t *testing.T, filename, content string) string {
	t.Helper()

	path := filepath.Join(t.TempDir(), filename)
	if err := os.WriteFile(path, []byte(content), 0o600); err != nil {
		t.Fatalf("write test file: %v", err)
	}

	return path
}
