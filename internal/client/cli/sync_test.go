package cli

import (
	"bytes"
	"context"
	"errors"
	"io"
	"strings"
	"testing"

	"github.com/xhrobj/gopherkeeper/internal/client/config"
	"github.com/xhrobj/gopherkeeper/internal/client/usecase"
	"github.com/xhrobj/gopherkeeper/internal/model"
)

const (
	testSyncRecordID       = "550e8400-e29b-41d4-a716-446655440001"
	testSecondSyncRecordID = "550e8400-e29b-41d4-a716-446655440002"
)

func TestSyncCommand_ConfigurationAndRefresh(t *testing.T) {
	isolateClientConfig(t)

	var gotConfig config.Config
	var gotRequest usecase.SyncRequest

	app := newApplicationStub(t)
	app.sync = func(_ context.Context, request usecase.SyncRequest) (usecase.SyncResult, error) {
		gotRequest = request
		return usecase.SyncResult{
			Updated: []usecase.RevisionChange{{
				Metadata: model.RecordMetadata{
					ID:       testSyncRecordID,
					Type:     model.RecordTypeText,
					Title:    "Work note",
					Revision: 2,
				},
				LocalRevision: 1,
			}},
			Unchanged: 3,
		}, nil
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
			"sync",
			"--refresh",
			"--address", "localhost:8082",
			"--ca-cert", "flag-ca.pem",
			"--session-file", "flag-session.json",
			"--cache-dir", "flag-cache",
		},
		strings.NewReader(testRegistrationPassword+"\n"),
		&output,
		io.Discard,
		factory,
	)
	if err != nil {
		t.Fatalf("run() error = %v", err)
	}

	wantConfig := config.Config{
		Address:     "localhost:8082",
		CACertFile:  "flag-ca.pem",
		SessionFile: "flag-session.json",
		CacheDir:    "flag-cache",
	}
	if gotConfig != wantConfig {
		t.Errorf("configuration = %+v, want %+v", gotConfig, wantConfig)
	}
	if gotRequest.Password != testRegistrationPassword {
		t.Error("Sync() received unexpected password")
	}
	if !gotRequest.RefreshStale {
		t.Error("RefreshStale = false, want true")
	}

	wantOutput := "Cache synchronization completed.\n" +
		"Added: 0\nUpdated: 1\nRemoved: 0\nUnchanged: 3\nStale: 0\n"
	if got := output.String(); got != wantOutput {
		t.Errorf("output = %q, want %q", got, wantOutput)
	}
	assertSyncOutputContainsNoSecrets(t, output.String())
}

func TestSyncCommand_HelpDoesNotOfferPasswordFlags(t *testing.T) {
	isolateClientConfig(t)

	var output bytes.Buffer
	err := runTestCommand(
		t,
		[]string{"gkeep", "sync", "--help"},
		strings.NewReader(""),
		&output,
		io.Discard,
		nil,
	)
	if err != nil {
		t.Fatalf("run() error = %v", err)
	}

	help := output.String()
	if !strings.Contains(help, "--refresh") {
		t.Errorf("sync help = %q, want refresh flag", help)
	}
	if strings.Contains(help, "--password") {
		t.Errorf("sync help exposes password flag: %q", help)
	}
	if strings.Contains(help, "stdin") {
		t.Errorf("sync help exposes technical stdin input: %q", help)
	}
}

func TestExecuteSync_ReportsStaleRecords(t *testing.T) {
	passwords := &passwordReaderStub{hiddenValues: []string{testRegistrationPassword}}
	var gotRequest usecase.SyncRequest
	var output bytes.Buffer

	app := newApplicationStub(t)
	app.sync = func(_ context.Context, request usecase.SyncRequest) (usecase.SyncResult, error) {
		gotRequest = request
		return usecase.SyncResult{
			Added:   []model.RecordMetadata{{ID: testSecondSyncRecordID}},
			Removed: []usecase.RecordState{{ID: testSyncRecordID, Revision: 1}},
			Stale: []usecase.RevisionChange{{
				Metadata: model.RecordMetadata{
					ID:       testSyncRecordID,
					Type:     model.RecordTypeText,
					Title:    "Work note",
					Revision: 2,
				},
				LocalRevision: 1,
			}},
			Unchanged: 3,
		}, nil
	}

	err := executeSync(
		context.Background(),
		app,
		passwords,
		passwordStreams{
			input:        strings.NewReader(""),
			output:       &output,
			promptOutput: io.Discard,
		},
		false,
	)
	if err != nil {
		t.Fatalf("executeSync() error = %v", err)
	}

	if passwords.hiddenCalls != 1 {
		t.Errorf("hidden password reads = %d, want 1", passwords.hiddenCalls)
	}
	if passwords.lineCalls != 0 {
		t.Errorf("line reads = %d, want 0", passwords.lineCalls)
	}
	if gotRequest.Password != testRegistrationPassword {
		t.Error("Sync() received unexpected password")
	}
	if gotRequest.RefreshStale {
		t.Error("RefreshStale = true, want false")
	}

	got := output.String()
	assertContainsAll(
		t,
		got,
		"Cache synchronization completed.",
		"Added: 1",
		"Updated: 0",
		"Removed: 1",
		"Unchanged: 3",
		"Stale: 1",
		"Stale records:",
		"ID",
		"TYPE",
		"TITLE",
		"LOCAL REVISION",
		"SERVER REVISION",
		testSyncRecordID,
		"text",
		"Work note",
		"Run `gkeep sync --refresh` to update stale records.",
	)
	assertSyncOutputContainsNoSecrets(t, got)
}

func TestExecuteSync_ReturnsPasswordReaderError(t *testing.T) {
	readError := errors.New("password input failed")
	app := newApplicationStub(t)

	err := executeSync(
		context.Background(),
		app,
		passwordReaderErrorStub{err: readError},
		passwordStreams{
			input:        strings.NewReader(""),
			output:       io.Discard,
			promptOutput: io.Discard,
		},
		false,
	)
	if !errors.Is(err, readError) {
		t.Fatalf("executeSync() error = %v, want %v", err, readError)
	}
}

func TestExecuteSync_ReturnsApplicationError(t *testing.T) {
	applicationError := errors.New("synchronization unavailable")
	var output bytes.Buffer

	app := newApplicationStub(t)
	app.sync = func(context.Context, usecase.SyncRequest) (usecase.SyncResult, error) {
		return usecase.SyncResult{}, applicationError
	}

	err := executeSync(
		context.Background(),
		app,
		&passwordReaderStub{hiddenValues: []string{testRegistrationPassword}},
		passwordStreams{
			input:        strings.NewReader(""),
			output:       &output,
			promptOutput: io.Discard,
		},
		false,
	)
	if !errors.Is(err, applicationError) {
		t.Fatalf("executeSync() error = %v, want %v", err, applicationError)
	}
	if output.Len() != 0 {
		t.Errorf("output = %q, want empty output", output.String())
	}
	assertSyncOutputContainsNoSecrets(t, err.Error())
}

func TestWriteSyncResult(t *testing.T) {
	tests := []struct {
		name   string
		result usecase.SyncResult
		wants  []string
		absent []string
	}{
		{
			name: "empty synchronization",
			wants: []string{
				"Cache synchronization completed.",
				"Added: 0",
				"Updated: 0",
				"Removed: 0",
				"Unchanged: 0",
				"Stale: 0",
			},
			absent: []string{"Stale records:", "sync --refresh"},
		},
		{
			name: "stale synchronization",
			result: usecase.SyncResult{
				Added:     []model.RecordMetadata{{ID: testSecondSyncRecordID}},
				Unchanged: 2,
				Stale: []usecase.RevisionChange{{
					Metadata: model.RecordMetadata{
						ID:       testSyncRecordID,
						Type:     model.RecordTypeCredentials,
						Title:    "GitHub",
						Revision: 4,
					},
					LocalRevision: 2,
				}},
			},
			wants: []string{
				"Added: 1",
				"Unchanged: 2",
				"Stale: 1",
				"Stale records:",
				testSyncRecordID,
				"credentials",
				"GitHub",
				"Run `gkeep sync --refresh` to update stale records.",
			},
		},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			var output bytes.Buffer
			if err := writeSyncResult(&output, tt.result); err != nil {
				t.Fatalf("writeSyncResult() error = %v", err)
			}

			got := output.String()
			assertContainsAll(t, got, tt.wants...)
			for _, value := range tt.absent {
				if strings.Contains(got, value) {
					t.Errorf("output = %q, must not contain %q", got, value)
				}
			}
			assertSyncOutputContainsNoSecrets(t, got)
		})
	}
}

func TestWriteSyncResult_ReturnsWriterError(t *testing.T) {
	writeError := errors.New("write failed")
	staleResult := usecase.SyncResult{
		Stale: []usecase.RevisionChange{{
			Metadata: model.RecordMetadata{
				ID:       testSyncRecordID,
				Type:     model.RecordTypeText,
				Title:    "Work note",
				Revision: 2,
			},
			LocalRevision: 1,
		}},
	}

	tests := []struct {
		name      string
		writer    io.Writer
		result    usecase.SyncResult
		wantError string
	}{
		{
			name:      "completion message",
			writer:    &failOnWriteWriter{failAt: 1, err: writeError},
			wantError: "write synchronization result",
		},
		{
			name:      "summary",
			writer:    &failOnWriteWriter{failAt: 2, err: writeError},
			wantError: "write synchronization summary",
		},
		{
			name:      "stale heading",
			writer:    &failOnWriteWriter{failAt: 3, err: writeError},
			result:    staleResult,
			wantError: "write stale record heading",
		},
		{
			name:      "stale table flush",
			writer:    &failOnWriteWriter{failAt: 4, err: writeError},
			result:    staleResult,
			wantError: "flush stale records",
		},
		{
			name: "stale hint",
			writer: &rejectTextWriter{
				rejected: "Run `gkeep sync --refresh`",
				err:      writeError,
			},
			result:    staleResult,
			wantError: "write stale record hint",
		},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			err := writeSyncResult(tt.writer, tt.result)
			if err == nil {
				t.Fatal("writeSyncResult() error = nil, want writer error")
			}
			if !errors.Is(err, writeError) {
				t.Errorf("writeSyncResult() error = %v, want %v in chain", err, writeError)
			}
			if !strings.Contains(err.Error(), tt.wantError) {
				t.Errorf("writeSyncResult() error = %q, want substring %q", err, tt.wantError)
			}
		})
	}
}

type passwordReaderErrorStub struct {
	err error
}

func (stub passwordReaderErrorStub) ReadHidden(io.Reader, io.Writer, string) (string, error) {
	return "", stub.err
}

func (stub passwordReaderErrorStub) ReadLine(io.Reader, io.Writer, string) (string, error) {
	return "", stub.err
}

type failOnWriteWriter struct {
	writes int
	failAt int
	err    error
}

func (writer *failOnWriteWriter) Write(data []byte) (int, error) {
	writer.writes++
	if writer.writes == writer.failAt {
		return 0, writer.err
	}

	return len(data), nil
}

type rejectTextWriter struct {
	buffer   bytes.Buffer
	rejected string
	err      error
}

func (writer *rejectTextWriter) Write(data []byte) (int, error) {
	if strings.Contains(string(data), writer.rejected) {
		return 0, writer.err
	}

	return writer.buffer.Write(data)
}

func assertSyncOutputContainsNoSecrets(t *testing.T, value string) {
	t.Helper()

	for _, secret := range []string{
		testRegistrationPassword,
		"test.jwt.token",
		"private payload",
	} {
		if strings.Contains(value, secret) {
			t.Errorf("synchronization output contains secret %q", secret)
		}
	}
}
