package cli

import (
	"bytes"
	"context"
	"errors"
	"fmt"
	"io"
	"strings"
	"testing"
	"time"

	"github.com/xhrobj/gopherkeeper/internal/client/config"
	"github.com/xhrobj/gopherkeeper/internal/client/usecase"
	"github.com/xhrobj/gopherkeeper/internal/model"
)

const testCredentialsPassword = "vault-secret-42"

type credentialsRecordCreatorFunc func(
	context.Context,
	usecase.CreateCredentialsRecordRequest,
) (usecase.CredentialsRecord, error)

func (f credentialsRecordCreatorFunc) CreateCredentialsRecord(
	ctx context.Context,
	request usecase.CreateCredentialsRecordRequest,
) (usecase.CredentialsRecord, error) {
	return f(ctx, request)
}

type credentialsRecordUpdaterFunc func(
	context.Context,
	usecase.UpdateCredentialsRecordRequest,
) (usecase.CredentialsRecord, error)

func (f credentialsRecordUpdaterFunc) UpdateCredentialsRecord(
	ctx context.Context,
	request usecase.UpdateCredentialsRecordRequest,
) (usecase.CredentialsRecord, error) {
	return f(ctx, request)
}

type recordGetterFunc func(context.Context, string) (usecase.Record, error)

func (f recordGetterFunc) GetRecord(ctx context.Context, recordID string) (usecase.Record, error) {
	return f(ctx, recordID)
}

type credentialsReaderStub struct {
	hiddenValue string
	lineValues  []string
	hiddenErr   error
	lineErr     error
	hiddenCalls int
	lineCalls   int
}

func (r *credentialsReaderStub) ReadHidden(io.Reader, io.Writer, string) (string, error) {
	r.hiddenCalls++
	if r.hiddenErr != nil {
		return "", r.hiddenErr
	}

	return r.hiddenValue, nil
}

func (r *credentialsReaderStub) ReadLine(io.Reader) (string, error) {
	if r.lineErr != nil {
		return "", r.lineErr
	}
	if r.lineCalls >= len(r.lineValues) {
		return "", errors.New("unexpected line read")
	}

	value := r.lineValues[r.lineCalls]
	r.lineCalls++
	return value, nil
}

func TestRecordsCreateCredentialsCommand(t *testing.T) {
	isolateClientConfig(t)

	input := strings.NewReader(`{"login":"alice","password":"vault-secret-42"}`)
	var stdout bytes.Buffer
	var stderr bytes.Buffer

	err := runTestCommand(
		t,
		[]string{
			"gkeep",
			"--address", "localhost:9090",
			"records", "create-credentials",
			"--title", "GitHub",
			"--credentials-stdin",
		},
		input,
		&stdout,
		&stderr,
		commandRunners{
			createCredentialsRecord: func(
				_ context.Context,
				cfg config.Config,
				commandInput io.Reader,
				output io.Writer,
				promptOutput io.Writer,
				request credentialsRecordCreateCommandRequest,
			) error {
				if cfg.Address != "localhost:9090" {
					t.Errorf("address = %q, want localhost:9090", cfg.Address)
				}
				if commandInput != input {
					t.Error("standard input was not passed to runner")
				}
				if output != &stdout {
					t.Error("output writer was not passed to runner")
				}
				if promptOutput != &stderr {
					t.Error("prompt writer was not passed to runner")
				}
				if request.title != "GitHub" {
					t.Errorf("title = %q, want GitHub", request.title)
				}
				if !request.credentialsStdin {
					t.Error("credentials-stdin = false, want true")
				}
				return nil
			},
		},
	)
	if err != nil {
		t.Fatalf("run create-credentials command error = %v", err)
	}
}

func TestRecordsUpdateCredentialsCommand(t *testing.T) {
	isolateClientConfig(t)

	input := strings.NewReader(`{"login":"alice","password":"vault-secret-42"}`)
	var stdout bytes.Buffer

	err := runTestCommand(
		t,
		[]string{
			"gkeep",
			"records", "update-credentials", testRecordID,
			"--revision", "2",
			"--title", "GitHub updated",
			"--credentials-stdin",
		},
		input,
		&stdout,
		io.Discard,
		commandRunners{
			updateCredentialsRecord: func(
				_ context.Context,
				_ config.Config,
				commandInput io.Reader,
				output io.Writer,
				_ io.Writer,
				request credentialsRecordUpdateCommandRequest,
			) error {
				if commandInput != input {
					t.Error("standard input was not passed to runner")
				}
				if output != &stdout {
					t.Error("output writer was not passed to runner")
				}
				if request.recordID != testRecordID {
					t.Errorf("record ID = %q, want %q", request.recordID, testRecordID)
				}
				if request.expectedRevision != 2 {
					t.Errorf("expected revision = %d, want 2", request.expectedRevision)
				}
				if request.title != "GitHub updated" {
					t.Errorf("title = %q, want GitHub updated", request.title)
				}
				if !request.credentialsStdin {
					t.Error("credentials-stdin = false, want true")
				}
				return nil
			},
		},
	)
	if err != nil {
		t.Fatalf("run update-credentials command error = %v", err)
	}
}

func TestRecordsUpdateCredentialsCommandRequiresRecordID(t *testing.T) {
	isolateClientConfig(t)

	err := runTestCommand(
		t,
		[]string{
			"gkeep",
			"records", "update-credentials",
			"--revision", "2",
			"--title", "GitHub",
			"--credentials-stdin",
		},
		strings.NewReader(`{"login":"alice","password":"vault-secret-42"}`),
		io.Discard,
		io.Discard,
		commandRunners{
			updateCredentialsRecord: unexpectedUpdateCredentialsRecordRunner(t),
		},
	)
	if err == nil || err.Error() != "record id is required" {
		t.Fatalf("run update-credentials command error = %v, want record id is required", err)
	}
}

func TestRecordsCreateCredentialsHelpDoesNotOfferPasswordFlag(t *testing.T) {
	isolateClientConfig(t)

	var output bytes.Buffer
	if err := runTestCommand(
		t,
		[]string{"gkeep", "records", "create-credentials", "--help"},
		nil,
		&output,
		io.Discard,
		commandRunners{},
	); err != nil {
		t.Fatalf("run create-credentials help error = %v", err)
	}

	help := output.String()
	if !strings.Contains(help, "--credentials-stdin") {
		t.Errorf("help = %q, want credentials-stdin flag", help)
	}
	if strings.Contains(help, "--password") {
		t.Errorf("help exposes password flag: %q", help)
	}
}

func TestExecuteCreateCredentialsRecord_Stdin(t *testing.T) {
	input := strings.NewReader(`{
		"login":"alice",
		"password":"vault-secret-42",
		"url":"https://github.com",
		"metadata":"recovery codes"
	}`)
	reader := &credentialsReaderStub{}
	var output bytes.Buffer

	err := executeCreateCredentialsRecord(
		context.Background(),
		credentialsRecordCreatorFunc(func(
			_ context.Context,
			request usecase.CreateCredentialsRecordRequest,
		) (usecase.CredentialsRecord, error) {
			assertCreateCredentialsRequest(t, request)
			return usecase.CredentialsRecord{
				Metadata: model.RecordMetadata{ID: testRecordID, Revision: 1},
			}, nil
		}),
		reader,
		passwordStreams{input: input, output: &output, promptOutput: io.Discard},
		credentialsRecordCreateCommandRequest{
			title:            "GitHub",
			credentialsStdin: true,
		},
	)
	if err != nil {
		t.Fatalf("executeCreateCredentialsRecord() error = %v", err)
	}
	if reader.hiddenCalls != 0 || reader.lineCalls != 0 {
		t.Fatalf("interactive reads = hidden %d, line %d, want 0", reader.hiddenCalls, reader.lineCalls)
	}

	want := "Created credentials record " + testRecordID + " with revision 1.\n"
	if output.String() != want {
		t.Errorf("output = %q, want %q", output.String(), want)
	}
	if strings.Contains(output.String(), testCredentialsPassword) {
		t.Error("create output contains password")
	}
}

func TestExecuteCreateCredentialsRecord_Interactive(t *testing.T) {
	metadataFile := writeTestFile(t, "metadata.txt", "private metadata")
	reader := &credentialsReaderStub{
		hiddenValue: testCredentialsPassword,
		lineValues:  []string{"alice", "https://github.com"},
	}
	var output bytes.Buffer
	var prompts bytes.Buffer

	err := executeCreateCredentialsRecord(
		context.Background(),
		credentialsRecordCreatorFunc(func(
			_ context.Context,
			request usecase.CreateCredentialsRecordRequest,
		) (usecase.CredentialsRecord, error) {
			if request.Title != "GitHub" || request.Login != "alice" ||
				request.Password != testCredentialsPassword ||
				request.URL != "https://github.com" ||
				request.Metadata != "private metadata" {
				t.Errorf("request = %+v, want interactive credentials", request)
			}

			return usecase.CredentialsRecord{
				Metadata: model.RecordMetadata{ID: testRecordID, Revision: 1},
			}, nil
		}),
		reader,
		passwordStreams{input: strings.NewReader(""), output: &output, promptOutput: &prompts},
		credentialsRecordCreateCommandRequest{
			title:        "GitHub",
			metadataFile: metadataFile,
		},
	)
	if err != nil {
		t.Fatalf("executeCreateCredentialsRecord() error = %v", err)
	}
	if reader.hiddenCalls != 1 || reader.lineCalls != 2 {
		t.Errorf("reads = hidden %d, line %d, want 1 and 2", reader.hiddenCalls, reader.lineCalls)
	}
	if got := prompts.String(); got != "Login: URL (optional): " {
		t.Errorf("prompts = %q, want login and URL prompts", got)
	}
}

func TestReadCredentialsPayloadInteractiveSuggestsCredentialsStdin(t *testing.T) {
	reader := &credentialsReaderStub{
		lineValues: []string{"alice"},
		hiddenErr:  fmt.Errorf("%w; use --password-stdin", errPasswordInputNotTerminal),
	}

	_, err := readCredentialsPayload(
		reader,
		strings.NewReader(""),
		io.Discard,
		"",
		false,
	)
	if err == nil || err.Error() != "password input is not a terminal; use --credentials-stdin" {
		t.Fatalf("readCredentialsPayload() error = %v, want credentials-stdin hint", err)
	}
}

func TestExecuteCreateCredentialsRecordRejectsConflictingInput(t *testing.T) {
	creator := credentialsRecordCreatorFunc(func(
		context.Context,
		usecase.CreateCredentialsRecordRequest,
	) (usecase.CredentialsRecord, error) {
		t.Fatal("creator must not be called")
		return usecase.CredentialsRecord{}, nil
	})

	err := executeCreateCredentialsRecord(
		context.Background(),
		creator,
		&credentialsReaderStub{},
		passwordStreams{input: strings.NewReader("{}"), output: io.Discard, promptOutput: io.Discard},
		credentialsRecordCreateCommandRequest{
			title:            "GitHub",
			metadataFile:     "metadata.txt",
			credentialsStdin: true,
		},
	)
	if err == nil || err.Error() != "--metadata-file cannot be used with --credentials-stdin" {
		t.Fatalf("executeCreateCredentialsRecord() error = %v, want conflicting input error", err)
	}
}

func TestExecuteUpdateCredentialsRecord(t *testing.T) {
	input := strings.NewReader(`{"login":"alice","password":"vault-secret-42"}`)
	var output bytes.Buffer

	err := executeUpdateCredentialsRecord(
		context.Background(),
		credentialsRecordUpdaterFunc(func(
			_ context.Context,
			request usecase.UpdateCredentialsRecordRequest,
		) (usecase.CredentialsRecord, error) {
			if request.RecordID != testRecordID || request.ExpectedRevision != 1 ||
				request.Title != "GitHub updated" || request.Login != "alice" ||
				request.Password != testCredentialsPassword {
				t.Errorf("request = %+v, want update credentials request", request)
			}

			return usecase.CredentialsRecord{
				Metadata: model.RecordMetadata{ID: testRecordID, Revision: 2},
			}, nil
		}),
		&credentialsReaderStub{},
		passwordStreams{input: input, output: &output, promptOutput: io.Discard},
		credentialsRecordUpdateCommandRequest{
			recordID:         testRecordID,
			expectedRevision: 1,
			title:            "GitHub updated",
			credentialsStdin: true,
		},
	)
	if err != nil {
		t.Fatalf("executeUpdateCredentialsRecord() error = %v", err)
	}

	want := "Updated credentials record " + testRecordID + " to revision 2.\n"
	if output.String() != want {
		t.Errorf("output = %q, want %q", output.String(), want)
	}
}

func TestReadCredentialsPayloadJSONRejectsUnknownFieldWithoutSecretLeak(t *testing.T) {
	errPayload := `{"login":"alice","password":"vault-secret-42","unexpected":true}`
	_, err := readCredentialsPayloadJSON(strings.NewReader(errPayload))
	if err == nil {
		t.Fatal("readCredentialsPayloadJSON() error = nil, want unknown field error")
	}
	if strings.Contains(err.Error(), testCredentialsPassword) {
		t.Fatalf("error contains password: %v", err)
	}
}

func TestExecuteGetRecord_Credentials(t *testing.T) {
	createdAt := time.Date(2026, time.July, 10, 12, 0, 0, 0, time.UTC)
	updatedAt := time.Date(2026, time.July, 10, 12, 5, 0, 0, time.UTC)
	var output bytes.Buffer

	err := executeGetRecord(
		context.Background(),
		recordGetterFunc(func(_ context.Context, recordID string) (usecase.Record, error) {
			if recordID != testRecordID {
				t.Errorf("record ID = %q, want %q", recordID, testRecordID)
			}
			return usecase.Record{
				Metadata: model.RecordMetadata{
					ID:        testRecordID,
					Type:      model.RecordTypeCredentials,
					Title:     "GitHub",
					Revision:  2,
					CreatedAt: createdAt,
					UpdatedAt: updatedAt,
				},
				Payload: &model.CredentialsPayload{
					Login:    "alice",
					Password: testCredentialsPassword,
					URL:      "https://github.com",
					Metadata: "recovery codes",
				},
			}, nil
		}),
		&output,
		testRecordID,
	)
	if err != nil {
		t.Fatalf("executeGetRecord() error = %v", err)
	}

	for _, want := range []string{
		"Type: credentials",
		"Login: alice",
		"Password: " + testCredentialsPassword,
		"URL: https://github.com",
		"Metadata:\nrecovery codes",
	} {
		if !strings.Contains(output.String(), want) {
			t.Errorf("output = %q, want %q", output.String(), want)
		}
	}
}

func TestExecuteGetRecord_Text(t *testing.T) {
	var output bytes.Buffer

	err := executeGetRecord(
		context.Background(),
		recordGetterFunc(func(context.Context, string) (usecase.Record, error) {
			return usecase.Record{
				Metadata: model.RecordMetadata{
					ID:       testRecordID,
					Type:     model.RecordTypeText,
					Title:    "Note",
					Revision: 1,
				},
				Payload: &model.TextPayload{Text: "private note"},
			}, nil
		}),
		&output,
		testRecordID,
	)
	if err != nil {
		t.Fatalf("executeGetRecord() error = %v", err)
	}
	if !strings.Contains(output.String(), "Text:\nprivate note") {
		t.Errorf("output = %q, want text payload", output.String())
	}
}

func assertCreateCredentialsRequest(t *testing.T, request usecase.CreateCredentialsRecordRequest) {
	t.Helper()

	if request.Title != "GitHub" {
		t.Errorf("title = %q, want GitHub", request.Title)
	}
	if request.Login != "alice" {
		t.Errorf("login = %q, want alice", request.Login)
	}
	if request.Password != testCredentialsPassword {
		t.Error("creator received unexpected password")
	}
	if request.URL != "https://github.com" {
		t.Errorf("URL = %q, want https://github.com", request.URL)
	}
	if request.Metadata != "recovery codes" {
		t.Errorf("metadata = %q, want recovery codes", request.Metadata)
	}
}
