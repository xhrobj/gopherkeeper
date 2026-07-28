package httpserver

import (
	"context"
	"errors"
	"net/http"
	"net/http/httptest"
	"strings"
	"testing"

	"github.com/xhrobj/gopherkeeper/internal/model"
)

func TestListRecordsHandler_ReturnsMetadataOnly(t *testing.T) {
	var gotUserID int64
	records := recordManagerStub{
		list: func(_ context.Context, userID int64) ([]model.RecordMetadata, error) {
			gotUserID = userID
			return []model.RecordMetadata{testRecordMetadata("my note", model.RecordTypeText, 1)}, nil
		},
	}
	response := httptest.NewRecorder()

	serveAuthenticatedRecordHandler(
		t,
		listRecordsHandler(records),
		response,
		httptest.NewRequest(http.MethodGet, "/api/v1/records", nil),
	)

	if gotUserID != 42 {
		t.Errorf("List() userID = %d, want 42", gotUserID)
	}
	if response.Code != http.StatusOK {
		t.Errorf("status code = %d, want %d", response.Code, http.StatusOK)
	}

	var body listRecordsResponse
	decodeJSONResponse(t, response, &body)
	if len(body.Records) != 1 {
		t.Fatalf("records count = %d, want 1", len(body.Records))
	}
	assertRecordMetadataResponse(t, body.Records[0], newRecordMetadataResponse(
		testRecordMetadata("my note", model.RecordTypeText, 1),
	))
	if strings.Contains(response.Body.String(), "secret note") {
		t.Error("list response contains payload")
	}
}

func TestListRecordsHandler_RejectsInvalidServiceResult(t *testing.T) {
	records := recordManagerStub{
		list: func(context.Context, int64) ([]model.RecordMetadata, error) {
			return []model.RecordMetadata{
				testRecordMetadata("line\nbreak", model.RecordTypeText, 1),
			}, nil
		},
	}
	response := httptest.NewRecorder()

	serveAuthenticatedRecordHandler(
		t,
		listRecordsHandler(records),
		response,
		httptest.NewRequest(http.MethodGet, "/api/v1/records", nil),
	)

	assertErrorResponse(t, response, http.StatusInternalServerError, errorCodeInternal, errorMessageInternal)
}

func TestGetRecordHandler_ReturnsRecords(t *testing.T) {
	for _, tt := range recordPayloadCases() {
		t.Run(tt.name, func(t *testing.T) {
			records := recordManagerStub{
				get: func(_ context.Context, userID int64, recordID string) (model.Record, error) {
					if userID != 42 {
						t.Fatalf("Get() userID = %d, want 42", userID)
					}
					if recordID != testRecordID {
						t.Fatalf("Get() recordID = %q, want %q", recordID, testRecordID)
					}

					return testRecord(tt.title, model.RecordInitialRevision, tt.payload), nil
				},
			}
			request := httptest.NewRequest(http.MethodGet, "/api/v1/records/"+testRecordID, nil)
			request.SetPathValue("id", testRecordID)
			response := httptest.NewRecorder()

			serveAuthenticatedRecordHandler(t, getRecordHandler(records), response, request)

			assertRecordResponse(
				t,
				response,
				http.StatusOK,
				model.RecordInitialRevision,
				tt.title,
				tt.payload,
			)
		})
	}
}

func TestGetRecordHandler_RejectsInvalidServiceResult(t *testing.T) {
	records := recordManagerStub{
		get: func(context.Context, int64, string) (model.Record, error) {
			return model.Record{
				Metadata: testRecordMetadata("GitHub", model.RecordTypeCredentials, 1),
			}, nil
		},
	}
	request := httptest.NewRequest(http.MethodGet, "/api/v1/records/"+testRecordID, nil)
	request.SetPathValue("id", testRecordID)
	response := httptest.NewRecorder()

	serveAuthenticatedRecordHandler(t, getRecordHandler(records), response, request)

	assertErrorResponse(t, response, http.StatusInternalServerError, errorCodeInternal, errorMessageInternal)
	if response.Header().Get("ETag") != "" {
		t.Errorf("ETag = %q, want empty", response.Header().Get("ETag"))
	}
}

func TestNewRecordResponse_RejectsInvalidRecord(t *testing.T) {
	validMetadata := testRecordMetadata("Private note", model.RecordTypeText, 1)
	tests := []struct {
		name   string
		record model.Record
	}{
		{
			name: "typed nil payload",
			record: model.Record{
				Metadata: validMetadata,
				Payload:  (*model.TextPayload)(nil),
			},
		},
		{
			name: "invalid metadata",
			record: model.Record{
				Metadata: testRecordMetadata("line\nbreak", model.RecordTypeText, 1),
				Payload:  &model.TextPayload{Text: "secret"},
			},
		},
		{
			name: "payload type mismatch",
			record: model.Record{
				Metadata: validMetadata,
				Payload: &model.CredentialsPayload{
					Login:    "alice",
					Password: "correct-horse-battery-staple",
				},
			},
		},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			_, err := newRecordResponse(tt.record)
			if !errors.Is(err, errInvalidRecordResponse) {
				t.Fatalf("newRecordResponse() error = %v, want %v", err, errInvalidRecordResponse)
			}
		})
	}
}
