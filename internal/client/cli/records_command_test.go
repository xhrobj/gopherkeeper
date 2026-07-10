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
	"time"

	"github.com/xhrobj/gopherkeeper/internal/client/config"
	"github.com/xhrobj/gopherkeeper/internal/client/usecase"
	"github.com/xhrobj/gopherkeeper/internal/model"
)

const testRecordID = "550e8400-e29b-41d4-a716-446655440000"

func TestRecordsUpdateTextCommand(t *testing.T) {
	isolateClientConfig(t)

	textFile := writeTestFile(t, "note.txt", "updated secret")
	metadataFile := writeTestFile(t, "metadata.txt", "updated private metadata")
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
		&bytes.Buffer{},
		commandRunners{
			updateTextRecord: func(
				_ context.Context,
				cfg config.Config,
				output io.Writer,
				request textRecordUpdateCommandRequest,
			) error {
				assertUpdateTextCommandRequest(t, cfg, output, &stdout, request)
				return nil
			},
		},
	)
	if err != nil {
		t.Fatalf("run update-text command error = %v", err)
	}
}

func TestRecordsUpdateTextCommandRequiresRecordID(t *testing.T) {
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
		&bytes.Buffer{},
		&bytes.Buffer{},
		commandRunners{
			updateTextRecord: unexpectedUpdateTextRecordRunner(t),
		},
	)
	if err == nil || err.Error() != "record id is required" {
		t.Fatalf("run update-text command error = %v, want record id is required", err)
	}
}

func TestRecordsDeleteCommand(t *testing.T) {
	isolateClientConfig(t)

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
		&bytes.Buffer{},
		commandRunners{
			deleteRecord: func(
				_ context.Context,
				cfg config.Config,
				output io.Writer,
				recordID string,
				expectedRevision int64,
			) error {
				if cfg.Address != "localhost:9090" {
					t.Errorf("address = %q, want localhost:9090", cfg.Address)
				}
				if output != &stdout {
					t.Error("output writer was not passed to runner")
				}
				if recordID != testRecordID {
					t.Errorf("record ID = %q, want %q", recordID, testRecordID)
				}
				if expectedRevision != 2 {
					t.Errorf("expected revision = %d, want 2", expectedRevision)
				}

				return nil
			},
		},
	)
	if err != nil {
		t.Fatalf("run delete command error = %v", err)
	}
}

func TestRecordsDeleteCommandRequiresRecordID(t *testing.T) {
	isolateClientConfig(t)

	err := runTestCommand(
		t,
		[]string{"gkeep", "records", "delete", "--revision", "2"},
		nil,
		&bytes.Buffer{},
		&bytes.Buffer{},
		commandRunners{
			deleteRecord: unexpectedDeleteRecordRunner(t),
		},
	)
	if err == nil || err.Error() != "record id is required" {
		t.Fatalf("run delete command error = %v, want record id is required", err)
	}
}

func TestExecuteUpdateTextRecord(t *testing.T) {
	textFile := writeTestFile(t, "note.txt", "updated secret")
	metadataFile := writeTestFile(t, "metadata.txt", "updated private metadata")
	updatedAt := time.Date(2026, time.July, 9, 12, 5, 0, 0, time.UTC)
	updater := textRecordUpdaterFunc(func(_ context.Context, request usecase.UpdateTextRecordRequest) (usecase.TextRecord, error) {
		if request.RecordID != testRecordID {
			t.Errorf("record ID = %q, want %q", request.RecordID, testRecordID)
		}
		if request.ExpectedRevision != 1 {
			t.Errorf("expected revision = %d, want 1", request.ExpectedRevision)
		}
		if request.Title != "Updated note" {
			t.Errorf("title = %q, want Updated note", request.Title)
		}
		if request.Text != "updated secret" {
			t.Errorf("text = %q, want updated secret", request.Text)
		}
		if request.Metadata != "updated private metadata" {
			t.Errorf("metadata = %q, want updated private metadata", request.Metadata)
		}

		return usecase.TextRecord{
			Metadata: model.RecordMetadata{
				ID:        testRecordID,
				Type:      model.RecordTypeText,
				Title:     request.Title,
				Revision:  2,
				UpdatedAt: updatedAt,
			},
			Payload: model.TextPayload{Text: request.Text, Metadata: request.Metadata},
		}, nil
	})

	var output bytes.Buffer
	if err := executeUpdateTextRecord(
		context.Background(),
		updater,
		&output,
		textRecordUpdateCommandRequest{
			recordID:         testRecordID,
			expectedRevision: 1,
			title:            "Updated note",
			textFile:         textFile,
			metadataFile:     metadataFile,
		},
	); err != nil {
		t.Fatalf("executeUpdateTextRecord() error = %v", err)
	}

	want := "Updated text record " + testRecordID + " to revision 2.\n"
	if output.String() != want {
		t.Errorf("output = %q, want %q", output.String(), want)
	}
}

func TestExecuteUpdateTextRecordReturnsReadError(t *testing.T) {
	updater := textRecordUpdaterFunc(func(context.Context, usecase.UpdateTextRecordRequest) (usecase.TextRecord, error) {
		t.Fatal("updater must not be called")
		return usecase.TextRecord{}, nil
	})

	err := executeUpdateTextRecord(
		context.Background(),
		updater,
		&bytes.Buffer{},
		textRecordUpdateCommandRequest{
			recordID:         testRecordID,
			expectedRevision: 1,
			title:            "Updated note",
			textFile:         "missing.txt",
		},
	)
	if err == nil || !strings.Contains(err.Error(), "stat text file") {
		t.Fatalf("executeUpdateTextRecord() error = %v, want text file stat error", err)
	}
}

func TestExecuteDeleteRecord(t *testing.T) {
	deleter := recordDeleterFunc(func(_ context.Context, request usecase.DeleteRecordRequest) error {
		if request.RecordID != testRecordID {
			t.Errorf("record ID = %q, want %q", request.RecordID, testRecordID)
		}
		if request.ExpectedRevision != 2 {
			t.Errorf("expected revision = %d, want 2", request.ExpectedRevision)
		}
		return nil
	})

	var output bytes.Buffer
	if err := executeDeleteRecord(context.Background(), deleter, &output, testRecordID, 2); err != nil {
		t.Fatalf("executeDeleteRecord() error = %v", err)
	}

	want := "Deleted record " + testRecordID + ".\n"
	if output.String() != want {
		t.Errorf("output = %q, want %q", output.String(), want)
	}
}

func TestExecuteDeleteRecordReturnsUsecaseError(t *testing.T) {
	wantErr := errors.New("record revision conflict")
	deleter := recordDeleterFunc(func(context.Context, usecase.DeleteRecordRequest) error {
		return wantErr
	})

	err := executeDeleteRecord(context.Background(), deleter, &bytes.Buffer{}, testRecordID, 1)
	if !errors.Is(err, wantErr) {
		t.Fatalf("executeDeleteRecord() error = %v, want %v", err, wantErr)
	}
}

func assertUpdateTextCommandRequest(
	t *testing.T,
	cfg config.Config,
	output, wantOutput io.Writer,
	request textRecordUpdateCommandRequest,
) {
	t.Helper()

	if cfg.Address != "localhost:9090" {
		t.Errorf("address = %q, want localhost:9090", cfg.Address)
	}
	if output != wantOutput {
		t.Error("output writer was not passed to runner")
	}
	if request.recordID != testRecordID {
		t.Errorf("record ID = %q, want %q", request.recordID, testRecordID)
	}
	if request.expectedRevision != 1 {
		t.Errorf("expected revision = %d, want 1", request.expectedRevision)
	}
	if request.title != "Updated note" {
		t.Errorf("title = %q, want Updated note", request.title)
	}
	assertFileContent(t, request.textFile, "updated secret")
	assertFileContent(t, request.metadataFile, "updated private metadata")
}

func assertFileContent(t *testing.T, path string, want string) {
	t.Helper()

	got, err := os.ReadFile(path)
	if err != nil {
		t.Fatalf("read file %q: %v", path, err)
	}
	if string(got) != want {
		t.Errorf("file %q = %q, want %q", path, got, want)
	}
}

type textRecordUpdaterFunc func(context.Context, usecase.UpdateTextRecordRequest) (usecase.TextRecord, error)

func (f textRecordUpdaterFunc) UpdateTextRecord(
	ctx context.Context,
	request usecase.UpdateTextRecordRequest,
) (usecase.TextRecord, error) {
	return f(ctx, request)
}

type recordDeleterFunc func(context.Context, usecase.DeleteRecordRequest) error

func (f recordDeleterFunc) DeleteRecord(ctx context.Context, request usecase.DeleteRecordRequest) error {
	return f(ctx, request)
}

func writeTestFile(t *testing.T, filename, content string) string {
	t.Helper()

	path := filepath.Join(t.TempDir(), filename)
	if err := os.WriteFile(path, []byte(content), 0o600); err != nil {
		t.Fatalf("write test file: %v", err)
	}

	return path
}
