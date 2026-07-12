package cli

import (
	"bytes"
	"context"
	"io"
	"os"
	"path/filepath"
	"strings"
	"testing"
	"time"

	"github.com/xhrobj/gopherkeeper/internal/client/config"
	"github.com/xhrobj/gopherkeeper/internal/model"
)

func TestRecordsGetBinaryCommand(t *testing.T) {
	isolateClientConfig(t)

	data := []byte("binary-secret-42")
	outputPath := filepath.Join(t.TempDir(), "restored-backup.bin")
	recordedAt := time.Date(2026, time.July, 12, 12, 0, 0, 0, time.UTC)
	var gotConfig config.Config
	app := newApplicationStub(t)
	app.getRecord = func(_ context.Context, recordID string) (model.Record, error) {
		if recordID != testRecordID {
			t.Errorf("record ID = %q, want %q", recordID, testRecordID)
		}
		return binaryTestRecord(recordedAt, data), nil
	}
	factory := newClientFactoryStub(t)
	factory.newApplication = func(cfg config.Config) (application, error) {
		gotConfig = cfg
		return app, nil
	}

	var output bytes.Buffer
	err := runTestCommand(
		t,
		[]string{
			"gkeep",
			"--address", "localhost:9090",
			"records", "get", testRecordID,
			"--output", outputPath,
		},
		nil,
		&output,
		io.Discard,
		factory,
	)
	if err != nil {
		t.Fatalf("run get binary command error = %v", err)
	}
	if gotConfig.Address != "localhost:9090" {
		t.Errorf("address = %q, want localhost:9090", gotConfig.Address)
	}

	got, err := os.ReadFile(outputPath)
	if err != nil {
		t.Fatalf("read restored file: %v", err)
	}
	if !bytes.Equal(got, data) {
		t.Errorf("restored data = %q, want %q", got, data)
	}
	for _, want := range []string{
		"Type: binary",
		"Filename: backup.bin",
		"Size: 16 bytes",
		"Saved to: " + outputPath,
		"Content type: application/octet-stream",
		"Metadata:\nprivate backup",
	} {
		if !strings.Contains(output.String(), want) {
			t.Errorf("output = %q, want %q", output.String(), want)
		}
	}
	if strings.Contains(output.String(), string(data)) {
		t.Error("output contains binary data")
	}
}

func TestExecuteGetRecord_BinaryRequiresOutput(t *testing.T) {
	recordedAt := time.Date(2026, time.July, 12, 12, 0, 0, 0, time.UTC)
	getter := recordGetterFunc(func(context.Context, string) (model.Record, error) {
		return binaryTestRecord(recordedAt, []byte("backup")), nil
	})

	var output bytes.Buffer
	err := executeGetRecord(context.Background(), getter, &output, testRecordID, "")
	if err == nil || err.Error() != "output path is required for binary record" {
		t.Fatalf("executeGetRecord() error = %v, want output path error", err)
	}
	if output.Len() != 0 {
		t.Errorf("output = %q, want empty", output.String())
	}
}

func TestExecuteGetRecord_RejectsOutputForText(t *testing.T) {
	outputPath := filepath.Join(t.TempDir(), "note.txt")
	getter := recordGetterFunc(func(context.Context, string) (model.Record, error) {
		return model.Record{
			Metadata: model.RecordMetadata{
				ID:       testRecordID,
				Type:     model.RecordTypeText,
				Title:    "Note",
				Revision: 1,
			},
			Payload: &model.TextPayload{Text: "private note"},
		}, nil
	})

	var output bytes.Buffer
	err := executeGetRecord(context.Background(), getter, &output, testRecordID, outputPath)
	if err == nil || err.Error() != "--output can only be used with binary records" {
		t.Fatalf("executeGetRecord() error = %v, want non-binary output error", err)
	}
	if output.Len() != 0 {
		t.Errorf("output = %q, want empty", output.String())
	}
	if _, statErr := os.Stat(outputPath); !os.IsNotExist(statErr) {
		t.Fatalf("output file stat error = %v, want not exist", statErr)
	}
}

func TestExecuteGetRecord_DoesNotOverwriteBinaryFile(t *testing.T) {
	outputPath := filepath.Join(t.TempDir(), "restored-backup.bin")
	if err := os.WriteFile(outputPath, []byte("original"), 0o600); err != nil {
		t.Fatalf("write existing output: %v", err)
	}
	recordedAt := time.Date(2026, time.July, 12, 12, 0, 0, 0, time.UTC)
	getter := recordGetterFunc(func(context.Context, string) (model.Record, error) {
		return binaryTestRecord(recordedAt, []byte("replacement")), nil
	})

	err := executeGetRecord(context.Background(), getter, io.Discard, testRecordID, outputPath)
	if err == nil || !strings.Contains(err.Error(), "create output file") {
		t.Fatalf("executeGetRecord() error = %v, want create output error", err)
	}
	got, readErr := os.ReadFile(outputPath)
	if readErr != nil {
		t.Fatalf("read existing output: %v", readErr)
	}
	if string(got) != "original" {
		t.Errorf("existing output = %q, want original", got)
	}
}

func TestExecuteGetRecord_RejectsNilBinaryPayload(t *testing.T) {
	getter := recordGetterFunc(func(context.Context, string) (model.Record, error) {
		return model.Record{
			Metadata: model.RecordMetadata{ID: testRecordID, Type: model.RecordTypeBinary},
			Payload:  (*model.BinaryPayload)(nil),
		}, nil
	})

	err := executeGetRecord(
		context.Background(),
		getter,
		io.Discard,
		testRecordID,
		filepath.Join(t.TempDir(), "backup.bin"),
	)
	if err == nil || err.Error() != "unexpected binary payload" {
		t.Fatalf("executeGetRecord() error = %v, want unexpected binary payload", err)
	}
}

func binaryTestRecord(recordedAt time.Time, data []byte) model.Record {
	return model.Record{
		Metadata: model.RecordMetadata{
			ID:        testRecordID,
			Type:      model.RecordTypeBinary,
			Title:     "Encrypted backup",
			Revision:  1,
			CreatedAt: recordedAt,
			UpdatedAt: recordedAt,
		},
		Payload: &model.BinaryPayload{
			Filename:    "backup.bin",
			Data:        data,
			ContentType: "application/octet-stream",
			Metadata:    "private backup",
		},
	}
}
