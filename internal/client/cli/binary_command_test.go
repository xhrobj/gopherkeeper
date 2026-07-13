package cli

import (
	"bytes"
	"context"
	"errors"
	"io"
	"os"
	"path/filepath"
	"testing"

	"github.com/xhrobj/gopherkeeper/internal/client/config"
	"github.com/xhrobj/gopherkeeper/internal/client/usecase"
	"github.com/xhrobj/gopherkeeper/internal/model"
)

func TestRecordsCreateBinaryCommand(t *testing.T) {
	isolateClientConfig(t)

	binaryFile := writeBinaryTestFile(t, "backup.bin", []byte{0x00, 0x01, 0x02, 0xff})
	metadataFile := writeTestFile(t, "metadata.txt", "private backup")
	var gotConfig config.Config
	app := newApplicationStub(t)
	app.createRecord = func(_ context.Context, request usecase.CreateRecordRequest) (model.Record, error) {
		payload := binaryPayloadFromRequest(t, request.Payload)
		if request.Title != "Encrypted backup" || payload.Filename != "backup.bin" ||
			!bytes.Equal(payload.Data, []byte{0x00, 0x01, 0x02, 0xff}) ||
			payload.ContentType != "application/octet-stream" || payload.Metadata != "private backup" {
			t.Errorf("request = %+v, payload = %+v, want binary values", request, payload)
		}
		return model.Record{Metadata: model.RecordMetadata{ID: testRecordID, Revision: 1}}, nil
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
			"records", "create-binary",
			"--title", "Encrypted backup",
			"--binary-file", binaryFile,
			"--content-type", "application/octet-stream",
			"--metadata-file", metadataFile,
		},
		nil,
		&output,
		io.Discard,
		factory,
	)
	if err != nil {
		t.Fatalf("run create-binary command error = %v", err)
	}
	if gotConfig.Address != "localhost:9090" {
		t.Errorf("address = %q, want localhost:9090", gotConfig.Address)
	}
	want := "Created binary record " + testRecordID + " with revision 1.\n"
	if output.String() != want {
		t.Errorf("output = %q, want %q", output.String(), want)
	}
}

func TestRecordsUpdateBinaryCommand(t *testing.T) {
	isolateClientConfig(t)

	binaryFile := writeBinaryTestFile(t, "backup-v2.bin", []byte("updated backup"))
	app := newApplicationStub(t)
	app.updateRecord = func(_ context.Context, request usecase.UpdateRecordRequest) (model.Record, error) {
		payload := binaryPayloadFromRequest(t, request.Payload)
		if request.RecordID != testRecordID || request.ExpectedRevision != 1 ||
			request.Title != "Updated backup" || payload.Filename != "backup-v2.bin" ||
			string(payload.Data) != "updated backup" {
			t.Errorf("request = %+v, payload = %+v, want binary update", request, payload)
		}
		return model.Record{Metadata: model.RecordMetadata{ID: testRecordID, Revision: 2}}, nil
	}
	factory := newClientFactoryStub(t)
	factory.newApplication = func(config.Config) (application, error) { return app, nil }

	var output bytes.Buffer
	err := runTestCommand(
		t,
		[]string{
			"gkeep",
			"records", "update-binary", testRecordID,
			"--revision", "1",
			"--title", "Updated backup",
			"--binary-file", binaryFile,
		},
		nil,
		&output,
		io.Discard,
		factory,
	)
	if err != nil {
		t.Fatalf("run update-binary command error = %v", err)
	}
	want := "Updated binary record " + testRecordID + " to revision 2.\n"
	if output.String() != want {
		t.Errorf("output = %q, want %q", output.String(), want)
	}
}

func TestRecordsUpdateBinaryCommand_RequiresRecordID(t *testing.T) {
	isolateClientConfig(t)

	binaryFile := writeBinaryTestFile(t, "backup.bin", []byte("backup"))
	err := runTestCommand(
		t,
		[]string{
			"gkeep",
			"records", "update-binary",
			"--revision", "1",
			"--title", "Updated backup",
			"--binary-file", binaryFile,
		},
		nil,
		io.Discard,
		io.Discard,
		nil,
	)
	if err == nil || err.Error() != "record id is required" {
		t.Fatalf("run update-binary command error = %v, want record id is required", err)
	}
}

func TestExecuteCreateBinaryRecord_ReturnsReadError(t *testing.T) {
	app := newApplicationStub(t)
	err := executeCreateBinaryRecord(
		context.Background(),
		app,
		io.Discard,
		binaryRecordCreateCommandRequest{title: "Backup", binaryFile: "missing.bin"},
	)
	if err == nil || !errors.Is(err, os.ErrNotExist) {
		t.Fatalf("executeCreateBinaryRecord() error = %v, want not exist", err)
	}
}

func binaryPayloadFromRequest(t *testing.T, payload model.RecordPayload) *model.BinaryPayload {
	t.Helper()

	binaryPayload, ok := payload.(*model.BinaryPayload)
	if !ok {
		t.Fatalf("payload type = %T, want *model.BinaryPayload", payload)
	}

	return binaryPayload
}

func writeBinaryTestFile(t *testing.T, filename string, data []byte) string {
	t.Helper()

	path := filepath.Join(t.TempDir(), filename)
	if err := os.WriteFile(path, data, 0o600); err != nil {
		t.Fatalf("write binary test file: %v", err)
	}

	return path
}
