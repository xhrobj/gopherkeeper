package httpserver

import (
	"context"
	"errors"
	"net/http"
	"net/http/httptest"
	"strings"
	"testing"

	"github.com/xhrobj/gopherkeeper/internal/model"
	"github.com/xhrobj/gopherkeeper/internal/server/service"
)

func TestUpdateRecordHandler_UpdatesRecords(t *testing.T) {
	tests := recordPayloadCases()[:2]
	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			var gotRequest service.UpdateRecordRequest
			records := recordManagerStub{
				update: func(_ context.Context, request service.UpdateRecordRequest) (model.Record, error) {
					gotRequest = request
					return testRecord(request.Title, 2, request.Payload), nil
				},
			}
			request := newUpdateRecordRequest(
				testRecordID,
				recordRequestBody(t, tt.payload.RecordType(), tt.title, tt.payload),
			)
			request.Header.Set("If-Match", `"1"`)
			response := httptest.NewRecorder()

			serveAuthenticatedRecordHandler(t, updateRecordHandler(records), response, request)

			assertUpdateRecordRequest(t, gotRequest, tt.title, tt.payload)
			assertRecordResponse(t, response, http.StatusOK, 2, tt.title, tt.payload)
		})
	}
}

func TestUpdateRecordHandler_RejectsInvalidRequest(t *testing.T) {
	validBody := recordRequestBody(t, model.RecordTypeText, "my note", &model.TextPayload{Text: "secret note"})
	tests := []struct {
		name        string
		ifMatch     string
		contentType string
		body        string
		wantStatus  int
		wantCode    string
		wantMessage string
	}{
		{
			name:        "missing If-Match",
			contentType: "application/json",
			body:        validBody,
			wantStatus:  http.StatusPreconditionRequired,
			wantCode:    errorCodePreconditionRequired,
			wantMessage: errorMessagePreconditionRequired,
		},
		{
			name:        "unquoted If-Match",
			ifMatch:     "1",
			contentType: "application/json",
			body:        validBody,
			wantStatus:  http.StatusBadRequest,
			wantCode:    errorCodeInvalidRequest,
			wantMessage: errorMessageInvalidRecordRequest,
		},
		{
			name:        "weak If-Match",
			ifMatch:     `W/"1"`,
			contentType: "application/json",
			body:        validBody,
			wantStatus:  http.StatusBadRequest,
			wantCode:    errorCodeInvalidRequest,
			wantMessage: errorMessageInvalidRecordRequest,
		},
		{
			name:        "zero If-Match revision",
			ifMatch:     `"0"`,
			contentType: "application/json",
			body:        validBody,
			wantStatus:  http.StatusBadRequest,
			wantCode:    errorCodeInvalidRequest,
			wantMessage: errorMessageInvalidRecordRequest,
		},
		{
			name:        "missing Content-Type",
			ifMatch:     `"1"`,
			body:        validBody,
			wantStatus:  http.StatusUnsupportedMediaType,
			wantCode:    errorCodeUnsupportedMediaType,
			wantMessage: errorMessageUnsupportedMediaType,
		},
		{
			name:        "malformed JSON",
			ifMatch:     `"1"`,
			contentType: "application/json",
			body:        `{"type":"text"`,
			wantStatus:  http.StatusBadRequest,
			wantCode:    errorCodeInvalidRequest,
			wantMessage: errorMessageInvalidRecordRequest,
		},
		{
			name:        "malformed binary Base64",
			ifMatch:     `"1"`,
			contentType: "application/json",
			body:        `{"type":"binary","title":"Backup","payload":{"filename":"backup.bin","data":"not-base64***"}}`,
			wantStatus:  http.StatusBadRequest,
			wantCode:    errorCodeInvalidRequest,
			wantMessage: errorMessageInvalidRecordRequest,
		},
		{
			name:        "unsupported record type",
			ifMatch:     `"1"`,
			contentType: "application/json",
			body:        `{"type":"card","title":"my note","payload":{"text":"secret"}}`,
			wantStatus:  http.StatusBadRequest,
			wantCode:    errorCodeInvalidRequest,
			wantMessage: errorMessageInvalidRecordRequest,
		},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			records := recordManagerStub{
				update: func(context.Context, service.UpdateRecordRequest) (model.Record, error) {
					t.Fatal("record service must not be called")
					return model.Record{}, nil
				},
			}
			request := httptest.NewRequest(
				http.MethodPut,
				"/api/v1/records/"+testRecordID,
				strings.NewReader(tt.body),
			)
			request.SetPathValue("id", testRecordID)
			if tt.ifMatch != "" {
				request.Header.Set("If-Match", tt.ifMatch)
			}
			if tt.contentType != "" {
				request.Header.Set("Content-Type", tt.contentType)
			}
			response := httptest.NewRecorder()

			serveAuthenticatedRecordHandler(t, updateRecordHandler(records), response, request)

			assertErrorResponse(t, response, tt.wantStatus, tt.wantCode, tt.wantMessage)
		})
	}
}

func TestParseIfMatchRevision(t *testing.T) {
	tests := []struct {
		name    string
		value   string
		want    int64
		wantErr error
	}{
		{name: "strong ETag", value: `"42"`, want: 42},
		{name: "missing", wantErr: model.ErrRecordPreconditionRequired},
		{name: "unquoted", value: "42", wantErr: model.ErrInvalidRecordRevision},
		{name: "weak ETag", value: `W/"42"`, wantErr: model.ErrInvalidRecordRevision},
		{name: "zero", value: `"0"`, wantErr: model.ErrInvalidRecordRevision},
		{name: "negative", value: `"-1"`, wantErr: model.ErrInvalidRecordRevision},
		{name: "overflow", value: `"9223372036854775808"`, wantErr: model.ErrInvalidRecordRevision},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			got, err := parseIfMatchRevision(tt.value)
			if !errors.Is(err, tt.wantErr) {
				t.Fatalf("parseIfMatchRevision() error = %v, want %v", err, tt.wantErr)
			}
			if got != tt.want {
				t.Errorf("parseIfMatchRevision() = %d, want %d", got, tt.want)
			}
		})
	}
}
