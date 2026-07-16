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
	testCardNumber = "2013 0614 2020 0619"
	testCardCVV    = "014"
)

type cardReaderStub struct {
	hiddenValues []string
	lineValues   []string
	hiddenCalls  int
	lineCalls    int
}

func (r *cardReaderStub) ReadHidden(io.Reader, io.Writer, string) (string, error) {
	if r.hiddenCalls >= len(r.hiddenValues) {
		return "", errors.New("unexpected hidden read")
	}

	value := r.hiddenValues[r.hiddenCalls]
	r.hiddenCalls++
	return value, nil
}

func (r *cardReaderStub) ReadLine(io.Reader, io.Writer, string) (string, error) {
	if r.lineCalls >= len(r.lineValues) {
		return "", errors.New("unexpected line read")
	}

	value := r.lineValues[r.lineCalls]
	r.lineCalls++
	return value, nil
}

func TestRecordsCreateCardCommand(t *testing.T) {
	isolateClientConfig(t)

	input := strings.NewReader(testCardNumber + "\nJoel Miller\n03/2038\n" + testCardCVV + "\n")
	var gotConfig config.Config
	app := newApplicationStub(t)
	app.createRecord = func(_ context.Context, request usecase.CreateRecordRequest) (model.Record, error) {
		payload := cardPayloadFromRequest(t, request.Payload)
		if request.Title != "Joel's card" || payload.Number != testCardNumber ||
			payload.Cardholder != "Joel Miller" || payload.CVV != testCardCVV {
			t.Errorf("request = %+v, payload = %+v, want card values", request, payload)
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
			"records", "create-card",
			"--title", "Joel's card",
		},
		input,
		&stdout,
		io.Discard,
		factory,
	)
	if err != nil {
		t.Fatalf("run create-card command error = %v", err)
	}
	if gotConfig.Address != "localhost:9090" {
		t.Errorf("address = %q, want localhost:9090", gotConfig.Address)
	}
}

func TestRecordsUpdateCardCommand(t *testing.T) {
	isolateClientConfig(t)

	input := strings.NewReader(testCardNumber + "\nJoel Miller\n03/2038\n" + testCardCVV + "\n")
	app := newApplicationStub(t)
	app.updateRecord = func(_ context.Context, request usecase.UpdateRecordRequest) (model.Record, error) {
		payload := cardPayloadFromRequest(t, request.Payload)
		if request.RecordID != testRecordID || request.ExpectedRevision != 2 ||
			request.Title != "Joel's card updated" || payload.Number != testCardNumber ||
			payload.CVV != testCardCVV {
			t.Errorf("request = %+v, payload = %+v, want update-card values", request, payload)
		}
		return model.Record{Metadata: model.RecordMetadata{ID: testRecordID, Revision: 3}}, nil
	}
	factory := newClientFactoryStub(t)
	factory.newApplication = func(config.Config) (application, error) { return app, nil }

	err := runTestCommand(
		t,
		[]string{
			"gkeep",
			"records", "update-card", testRecordID,
			"--revision", "2",
			"--title", "Joel's card updated",
		},
		input,
		io.Discard,
		io.Discard,
		factory,
	)
	if err != nil {
		t.Fatalf("run update-card command error = %v", err)
	}
}

func TestRecordsCreateCardCommand_HelpDoesNotOfferSensitiveFlags(t *testing.T) {
	isolateClientConfig(t)

	var output bytes.Buffer
	if err := runTestCommand(
		t,
		[]string{"gkeep", "records", "create-card", "--help"},
		nil,
		&output,
		io.Discard,
		nil,
	); err != nil {
		t.Fatalf("run create-card help error = %v", err)
	}

	help := output.String()
	if strings.Contains(help, "stdin") {
		t.Errorf("help exposes technical stdin input: %q", help)
	}
	for _, sensitiveFlag := range []string{"--number", "--cvv"} {
		if strings.Contains(help, sensitiveFlag) {
			t.Errorf("help exposes sensitive flag %q", sensitiveFlag)
		}
	}
}

func TestReadCardPayload(t *testing.T) {
	metadataFile := writeTestFile(t, "metadata.txt", "test card")
	reader := &cardReaderStub{
		hiddenValues: []string{testCardNumber, testCardCVV},
		lineValues:   []string{"Joel Miller", "03/2038"},
	}

	payload, err := readCardPayload(
		reader,
		strings.NewReader(""),
		io.Discard,
		metadataFile,
	)
	if err != nil {
		t.Fatalf("readCardPayload() error = %v", err)
	}
	if reader.hiddenCalls != 2 || reader.lineCalls != 2 {
		t.Errorf("reads = hidden %d, line %d, want 2 and 2", reader.hiddenCalls, reader.lineCalls)
	}
	if payload.Number != testCardNumber || payload.Cardholder != "Joel Miller" ||
		payload.ExpiryMonth == nil || *payload.ExpiryMonth != 3 ||
		payload.ExpiryYear == nil || *payload.ExpiryYear != 2038 ||
		payload.CVV != testCardCVV || payload.Metadata != "test card" {
		t.Errorf("payload = %#v, want card values", payload)
	}
}

func TestReadOptionalCardExpiry_RetriesInvalidValue(t *testing.T) {
	reader := &cardReaderStub{lineValues: []string{"13/2038", "01/0000", "03/2038"}}
	var promptOutput bytes.Buffer

	month, year, err := readOptionalCardExpiry(reader, strings.NewReader(""), &promptOutput)
	if err != nil {
		t.Fatalf("readOptionalCardExpiry() error = %v", err)
	}
	if month == nil || *month != 3 || year == nil || *year != 2038 {
		t.Fatalf("expiry = %v/%v, want 3/2038", month, year)
	}
	if reader.lineCalls != 3 {
		t.Errorf("line calls = %d, want 3", reader.lineCalls)
	}
	if got := strings.Count(promptOutput.String(), "Invalid expiry. Use MM/YYYY or leave it empty."); got != 2 {
		t.Errorf("validation messages = %d, want 2", got)
	}
}

func TestReadOptionalCardExpiry_EmptyValue(t *testing.T) {
	reader := &cardReaderStub{lineValues: []string{""}}

	month, year, err := readOptionalCardExpiry(reader, strings.NewReader(""), io.Discard)
	if err != nil {
		t.Fatalf("readOptionalCardExpiry() error = %v", err)
	}
	if month != nil || year != nil {
		t.Fatalf("expiry = %v/%v, want nil values", month, year)
	}
}

func TestParseCardExpiry(t *testing.T) {
	tests := []struct {
		name      string
		value     string
		wantMonth int
		wantYear  int
		wantOK    bool
	}{
		{name: "valid", value: "03/2038", wantMonth: 3, wantYear: 2038, wantOK: true},
		{name: "minimum values", value: "01/0001", wantMonth: 1, wantYear: 1, wantOK: true},
		{name: "zero year", value: "01/0000", wantOK: false},
		{name: "empty", value: "", wantOK: false},
		{name: "single digit month", value: "3/2038", wantOK: false},
		{name: "short year", value: "03/38", wantOK: false},
		{name: "month out of range", value: "13/2038", wantOK: false},
		{name: "non digit", value: "03/20x8", wantOK: false},
	}

	for _, test := range tests {
		t.Run(test.name, func(t *testing.T) {
			month, year, ok := parseCardExpiry(test.value)
			if month != test.wantMonth || year != test.wantYear || ok != test.wantOK {
				t.Errorf("parseCardExpiry(%q) = (%d, %d, %t), want (%d, %d, %t)",
					test.value, month, year, ok, test.wantMonth, test.wantYear, test.wantOK)
			}
		})
	}
}

func cardPayloadFromRequest(t *testing.T, payload model.RecordPayload) *model.CardPayload {
	t.Helper()

	card, ok := payload.(*model.CardPayload)
	if !ok {
		t.Fatalf("payload type = %T, want *model.CardPayload", payload)
	}

	return card
}
