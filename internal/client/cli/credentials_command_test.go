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

const testCredentialsPassword = "vault-secret-42"

type credentialsReaderStub struct {
	hiddenValue string
	lineValues  []string
	hiddenErr   error
	lineErr     error
	hiddenCalls int
	lineCalls   int
}

func (r *credentialsReaderStub) ReadHidden(
	io.Reader,
	io.Writer,
	string,
) (string, error) {
	r.hiddenCalls++
	if r.hiddenErr != nil {
		return "", r.hiddenErr
	}

	return r.hiddenValue, nil
}

func (r *credentialsReaderStub) ReadLine(
	io.Reader,
	io.Writer,
	string,
) (string, error) {
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

	input := strings.NewReader("alice\n" + testCredentialsPassword + "\nhttps://github.com\n")
	var gotConfig config.Config
	app := newApplicationStub(t)
	app.createRecord = func(_ context.Context, request usecase.CreateRecordRequest) (model.Record, error) {
		payload, ok := request.Payload.(*model.CredentialsPayload)
		if !ok {
			t.Fatalf("payload type = %T, want *model.CredentialsPayload", request.Payload)
		}
		if request.Title != "GitHub" || payload.Login != "alice" ||
			payload.Password != testCredentialsPassword || payload.URL != "https://github.com" {
			t.Errorf("request = %+v, payload = %+v, want credentials values", request, payload)
		}
		return model.Record{Metadata: model.RecordMetadata{ID: testRecordID, Revision: 1}}, nil
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
			"records", "create-credentials",
			"--title", "GitHub",
		},
		input,
		&stdout,
		io.Discard,
		factory,
	)
	if err != nil {
		t.Fatalf("run create-credentials command error = %v", err)
	}
	if gotConfig.Address != "localhost:9090" {
		t.Errorf("address = %q, want localhost:9090", gotConfig.Address)
	}
}

func TestRecordsUpdateCredentialsCommand(t *testing.T) {
	isolateClientConfig(t)

	input := strings.NewReader("alice\n" + testCredentialsPassword + "\nhttps://github.com\n")
	app := newApplicationStub(t)
	app.updateRecord = func(_ context.Context, request usecase.UpdateRecordRequest) (model.Record, error) {
		payload, ok := request.Payload.(*model.CredentialsPayload)
		if !ok {
			t.Fatalf("payload type = %T, want *model.CredentialsPayload", request.Payload)
		}
		if request.RecordID != testRecordID || request.ExpectedRevision != 2 ||
			request.Title != "GitHub updated" || payload.Login != "alice" ||
			payload.Password != testCredentialsPassword {
			t.Errorf("request = %+v, payload = %+v, want update values", request, payload)
		}
		return model.Record{Metadata: model.RecordMetadata{ID: testRecordID, Revision: 3}}, nil
	}
	factory := newClientFactoryStub(t)
	factory.newApplication = func(config.Config) (application, error) { return app, nil }

	err := runTestCommand(
		t,
		[]string{
			"gkeep",
			"records", "update-credentials", testRecordID,
			"--revision", "2",
			"--title", "GitHub updated",
		},
		input,
		io.Discard,
		io.Discard,
		factory,
	)
	if err != nil {
		t.Fatalf("run update-credentials command error = %v", err)
	}
}

func TestRecordsUpdateCredentialsCommand_RequiresRecordID(t *testing.T) {
	isolateClientConfig(t)

	err := runTestCommand(
		t,
		[]string{
			"gkeep",
			"records", "update-credentials",
			"--revision", "2",
			"--title", "GitHub",
		},
		strings.NewReader("alice\n"+testCredentialsPassword+"\n\n"),
		io.Discard,
		io.Discard,
		nil,
	)
	if err == nil || err.Error() != "record id is required" {
		t.Fatalf("run update-credentials command error = %v, want record id is required", err)
	}
}

func TestRecordsCreateCredentialsCommand_HelpDoesNotOfferSensitiveFlags(t *testing.T) {
	isolateClientConfig(t)

	var output bytes.Buffer
	if err := runTestCommand(
		t,
		[]string{"gkeep", "records", "create-credentials", "--help"},
		nil,
		&output,
		io.Discard,
		nil,
	); err != nil {
		t.Fatalf("run create-credentials help error = %v", err)
	}

	help := output.String()
	if strings.Contains(help, "stdin") {
		t.Errorf("help exposes technical stdin input: %q", help)
	}
	if strings.Contains(help, "--password") {
		t.Errorf("help exposes password flag: %q", help)
	}
}

func TestReadCredentialsPayload(t *testing.T) {
	metadataFile := writeTestFile(t, "metadata.txt", "private metadata")
	reader := &credentialsReaderStub{
		hiddenValue: testCredentialsPassword,
		lineValues:  []string{"alice", "https://github.com"},
	}

	payload, err := readCredentialsPayload(
		reader,
		strings.NewReader(""),
		io.Discard,
		metadataFile,
	)
	if err != nil {
		t.Fatalf("readCredentialsPayload() error = %v", err)
	}
	if reader.hiddenCalls != 1 || reader.lineCalls != 2 {
		t.Errorf("reads = hidden %d, line %d, want 1 and 2", reader.hiddenCalls, reader.lineCalls)
	}
	want := model.CredentialsPayload{
		Login:    "alice",
		Password: testCredentialsPassword,
		URL:      "https://github.com",
		Metadata: "private metadata",
	}
	if payload != want {
		t.Errorf("payload = %#v, want %#v", payload, want)
	}
}

func TestReadCredentialsPayload_ReturnsSecretInputError(t *testing.T) {
	inputError := errors.New("secret input failed")
	reader := &credentialsReaderStub{
		lineValues: []string{"alice"},
		hiddenErr:  inputError,
	}

	_, err := readCredentialsPayload(
		reader,
		strings.NewReader(""),
		io.Discard,
		"",
	)
	if !errors.Is(err, inputError) {
		t.Fatalf("readCredentialsPayload() error = %v, want %v", err, inputError)
	}
}
