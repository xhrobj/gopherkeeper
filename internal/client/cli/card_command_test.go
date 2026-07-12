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

func (r *cardReaderStub) ReadLine(io.Reader) (string, error) {
	if r.lineCalls >= len(r.lineValues) {
		return "", errors.New("unexpected line read")
	}

	value := r.lineValues[r.lineCalls]
	r.lineCalls++
	return value, nil
}

func TestRecordsCreateCardCommand(t *testing.T) {
	isolateClientConfig(t)

	input := strings.NewReader(`{"number":"2013 0614 2020 0619","cvv":"014"}`)
	var gotConfig config.Config
	app := newApplicationStub(t)
	app.createRecord = func(_ context.Context, request usecase.CreateRecordRequest) (model.Record, error) {
		payload := cardPayloadFromRequest(t, request.Payload)
		if request.Title != "Joel's card" || payload.Number != testCardNumber || payload.CVV != testCardCVV {
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
	var stderr bytes.Buffer
	err := runTestCommand(
		t,
		[]string{
			"gkeep",
			"--address", "localhost:9090",
			"records", "create-card",
			"--title", "Joel's card",
			"--card-stdin",
		},
		input,
		&stdout,
		&stderr,
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

	input := strings.NewReader(`{"number":"2013 0614 2020 0619","cvv":"014"}`)
	app := newApplicationStub(t)
	app.updateRecord = func(_ context.Context, request usecase.UpdateRecordRequest) (model.Record, error) {
		payload := cardPayloadFromRequest(t, request.Payload)
		if request.RecordID != testRecordID || request.ExpectedRevision != 2 ||
			request.Title != "Joel's card updated" || payload.Number != testCardNumber || payload.CVV != testCardCVV {
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
			"--card-stdin",
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
	if !strings.Contains(help, "--card-stdin") {
		t.Errorf("help = %q, want card-stdin flag", help)
	}
	for _, sensitiveFlag := range []string{"--number", "--cvv"} {
		if strings.Contains(help, sensitiveFlag) {
			t.Errorf("help exposes sensitive flag %q", sensitiveFlag)
		}
	}
}

func TestExecuteCreateCardRecord_Interactive(t *testing.T) {
	metadataFile := writeTestFile(t, "metadata.txt", "test card")
	reader := &cardReaderStub{
		hiddenValues: []string{testCardNumber, testCardCVV},
		lineValues:   []string{"Joel Miller", "3", "2038"},
	}
	var output bytes.Buffer

	err := executeCreateCardRecord(
		context.Background(),
		recordCreatorFunc(func(
			_ context.Context,
			request usecase.CreateRecordRequest,
		) (model.Record, error) {
			payload := cardPayloadFromRequest(t, request.Payload)
			if request.Title != "Joel's card" || payload.Number != testCardNumber ||
				payload.Cardholder != "Joel Miller" || payload.ExpiryMonth == nil ||
				*payload.ExpiryMonth != 3 || payload.ExpiryYear == nil || *payload.ExpiryYear != 2038 ||
				payload.CVV != testCardCVV || payload.Metadata != "test card" {
				t.Errorf("request = %+v, payload = %+v, want interactive card values", request, payload)
			}

			return model.Record{
				Metadata: model.RecordMetadata{ID: testRecordID, Revision: 1},
			}, nil
		}),
		reader,
		passwordStreams{input: strings.NewReader(""), output: &output, promptOutput: io.Discard},
		cardRecordCreateCommandRequest{title: "Joel's card", metadataFile: metadataFile},
	)
	if err != nil {
		t.Fatalf("executeCreateCardRecord() error = %v", err)
	}
	if reader.hiddenCalls != 2 || reader.lineCalls != 3 {
		t.Errorf("reads = hidden %d, line %d, want 2 and 3", reader.hiddenCalls, reader.lineCalls)
	}

	want := "Created card record " + testRecordID + " with revision 1.\n"
	if output.String() != want {
		t.Errorf("output = %q, want %q", output.String(), want)
	}
	if strings.Contains(output.String(), testCardNumber) || strings.Contains(output.String(), testCardCVV) {
		t.Error("create output contains card secrets")
	}
}

func TestExecuteUpdateCardRecord_Stdin(t *testing.T) {
	input := strings.NewReader(`{
		"number":"2013 0614 2020 0619",
		"cardholder":"Joel Miller",
		"expiry_month":3,
		"expiry_year":2038,
		"cvv":"014",
		"metadata":"test card updated"
	}`)
	reader := &cardReaderStub{}
	var output bytes.Buffer

	err := executeUpdateCardRecord(
		context.Background(),
		recordUpdaterFunc(func(
			_ context.Context,
			request usecase.UpdateRecordRequest,
		) (model.Record, error) {
			payload := cardPayloadFromRequest(t, request.Payload)
			if request.RecordID != testRecordID || request.ExpectedRevision != 1 ||
				payload.Number != testCardNumber || payload.CVV != testCardCVV {
				t.Errorf("request = %+v, payload = %+v, want stdin card update", request, payload)
			}

			return model.Record{
				Metadata: model.RecordMetadata{ID: testRecordID, Revision: 2},
			}, nil
		}),
		reader,
		passwordStreams{input: input, output: &output, promptOutput: io.Discard},
		cardRecordUpdateCommandRequest{
			recordID:         testRecordID,
			expectedRevision: 1,
			title:            "Joel's card updated",
			cardStdin:        true,
		},
	)
	if err != nil {
		t.Fatalf("executeUpdateCardRecord() error = %v", err)
	}
	if reader.hiddenCalls != 0 || reader.lineCalls != 0 {
		t.Fatalf("interactive reads = hidden %d, line %d, want 0", reader.hiddenCalls, reader.lineCalls)
	}

	want := fmt.Sprintf("Updated card record %s to revision 2.\n", testRecordID)
	if output.String() != want {
		t.Errorf("output = %q, want %q", output.String(), want)
	}
	if strings.Contains(output.String(), testCardNumber) || strings.Contains(output.String(), testCardCVV) {
		t.Error("update output contains card secrets")
	}
}

func TestExecuteGetRecord_Card(t *testing.T) {
	expiryMonth := 3
	expiryYear := 2038
	recordedAt := time.Date(2026, time.July, 11, 12, 0, 0, 0, time.UTC)
	getter := cardRecordGetterFunc(func(context.Context, string) (model.Record, error) {
		return model.Record{
			Metadata: model.RecordMetadata{
				ID:        testRecordID,
				Type:      model.RecordTypeCard,
				Title:     "Joel's card",
				Revision:  1,
				CreatedAt: recordedAt,
				UpdatedAt: recordedAt,
			},
			Payload: &model.CardPayload{
				Number:      testCardNumber,
				Cardholder:  "Joel Miller",
				ExpiryMonth: &expiryMonth,
				ExpiryYear:  &expiryYear,
				CVV:         testCardCVV,
				Metadata:    "test card",
			},
		}, nil
	})

	var output bytes.Buffer
	if err := executeGetRecord(context.Background(), getter, &output, testRecordID, ""); err != nil {
		t.Fatalf("executeGetRecord() error = %v", err)
	}

	for _, want := range []string{
		"Type: card",
		"Number: " + testCardNumber,
		"Cardholder: Joel Miller",
		"Expiry: 03/2038",
		"CVV: " + testCardCVV,
		"test card",
	} {
		if !strings.Contains(output.String(), want) {
			t.Errorf("output = %q, want %q", output.String(), want)
		}
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
