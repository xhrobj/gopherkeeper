package httpserver

import (
	"context"
	"net/http"
	"net/http/httptest"
	"strings"
	"testing"

	"github.com/xhrobj/gopherkeeper/internal/model"
	"github.com/xhrobj/gopherkeeper/internal/server/service"
)

func TestCreateRecordHandler_CreatesRecords(t *testing.T) {
	for _, tt := range recordPayloadCases() {
		t.Run(tt.name, func(t *testing.T) {
			var gotRequest service.CreateRecordRequest
			records := recordManagerStub{
				create: func(_ context.Context, request service.CreateRecordRequest) (model.Record, error) {
					gotRequest = request

					return testRecord(request.Title, model.RecordInitialRevision, request.Payload), nil
				},
			}
			requestBody := recordRequestBody(t, tt.payload.RecordType(), tt.title, tt.payload)
			if tt.wantBase64 != "" && !strings.Contains(requestBody, tt.wantBase64) {
				t.Fatalf("request body = %s, want Base64 fragment %s", requestBody, tt.wantBase64)
			}
			response := httptest.NewRecorder()

			serveAuthenticatedRecordHandler(
				t,
				createRecordHandler(records),
				response,
				newCreateRecordRequest(requestBody),
			)

			assertCreateRecordRequest(t, gotRequest, tt.title, tt.payload)
			assertRecordResponse(
				t,
				response,
				http.StatusCreated,
				model.RecordInitialRevision,
				tt.title,
				tt.payload,
			)
			if tt.wantBase64 != "" && !strings.Contains(response.Body.String(), tt.wantBase64) {
				t.Errorf("response body = %s, want Base64 fragment %s", response.Body.String(), tt.wantBase64)
			}
		})
	}
}

func TestCreateRecordHandler_AcceptsEmptyBinaryData(t *testing.T) {
	records := recordManagerStub{
		create: func(_ context.Context, request service.CreateRecordRequest) (model.Record, error) {
			payload := requireBinaryPayload(t, request.Payload)
			if payload.Data == nil {
				t.Fatal("Create() binary data = nil, want present empty slice")
			}
			if len(payload.Data) != 0 {
				t.Fatalf("Create() binary data length = %d, want 0", len(payload.Data))
			}

			return testRecord(request.Title, model.RecordInitialRevision, request.Payload), nil
		},
	}
	request := newCreateRecordRequest(
		`{"type":"binary","title":"Empty","payload":{"filename":"empty.bin","data":""}}`,
	)
	response := httptest.NewRecorder()

	serveAuthenticatedRecordHandler(t, createRecordHandler(records), response, request)

	if response.Code != http.StatusCreated {
		t.Errorf("status code = %d, want %d", response.Code, http.StatusCreated)
	}
}

func TestCreateRecordHandler_MapsBinaryPayloadValidation(t *testing.T) {
	tests := []struct {
		name        string
		body        string
		wantStatus  int
		wantCode    string
		wantMessage string
	}{
		{
			name:        "missing data",
			body:        `{"type":"binary","title":"Empty","payload":{"filename":"empty.bin"}}`,
			wantStatus:  http.StatusBadRequest,
			wantCode:    errorCodeInvalidRecordData,
			wantMessage: errorMessageInvalidRecordRequest,
		},
		{
			name:        "null data",
			body:        `{"type":"binary","title":"Empty","payload":{"filename":"empty.bin","data":null}}`,
			wantStatus:  http.StatusBadRequest,
			wantCode:    errorCodeInvalidRecordData,
			wantMessage: errorMessageInvalidRecordRequest,
		},
		{
			name: "decoded data too large",
			body: recordRequestBody(t, model.RecordTypeBinary, "Large", &model.BinaryPayload{
				Filename: "large.bin",
				Data:     make([]byte, model.BinaryPayloadMaxSize+1),
			}),
			wantStatus:  http.StatusRequestEntityTooLarge,
			wantCode:    errorCodePayloadTooLarge,
			wantMessage: errorMessagePayloadTooLarge,
		},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			records := recordManagerStub{
				create: func(_ context.Context, request service.CreateRecordRequest) (model.Record, error) {
					return model.Record{}, request.Payload.Validate()
				},
			}
			response := httptest.NewRecorder()

			serveAuthenticatedRecordHandler(
				t,
				createRecordHandler(records),
				response,
				newCreateRecordRequest(tt.body),
			)

			assertErrorResponse(t, response, tt.wantStatus, tt.wantCode, tt.wantMessage)
		})
	}
}

func TestCreateRecordHandler_RejectsInvalidRequest(t *testing.T) {
	validBody := recordRequestBody(t, model.RecordTypeText, "my note", &model.TextPayload{Text: "secret note"})
	tests := []struct {
		name        string
		contentType string
		body        string
		wantStatus  int
		wantCode    string
		wantMessage string
	}{
		{
			name:        "missing Content-Type",
			body:        validBody,
			wantStatus:  http.StatusUnsupportedMediaType,
			wantCode:    errorCodeUnsupportedMediaType,
			wantMessage: errorMessageUnsupportedMediaType,
		},
		{
			name:        "unsupported Content-Type",
			contentType: "text/plain",
			body:        validBody,
			wantStatus:  http.StatusUnsupportedMediaType,
			wantCode:    errorCodeUnsupportedMediaType,
			wantMessage: errorMessageUnsupportedMediaType,
		},
		{
			name:        "malformed JSON",
			contentType: "application/json",
			body:        `{"type":"text"`,
			wantStatus:  http.StatusBadRequest,
			wantCode:    errorCodeInvalidRequest,
			wantMessage: errorMessageInvalidRecordRequest,
		},
		{
			name:        "malformed binary Base64",
			contentType: "application/json",
			body:        `{"type":"binary","title":"Backup","payload":{"filename":"backup.bin","data":"not-base64***"}}`,
			wantStatus:  http.StatusBadRequest,
			wantCode:    errorCodeInvalidRequest,
			wantMessage: errorMessageInvalidRecordRequest,
		},
		{
			name:        "text payload contains credentials field",
			contentType: "application/json",
			body:        `{"type":"text","title":"my note","payload":{"text":"secret","login":"alice"}}`,
			wantStatus:  http.StatusBadRequest,
			wantCode:    errorCodeInvalidRequest,
			wantMessage: errorMessageInvalidRecordRequest,
		},
		{
			name:        "credentials payload contains text field",
			contentType: "application/json",
			body:        `{"type":"credentials","title":"GitHub","payload":{"login":"alice","password":"secret","text":"note"}}`,
			wantStatus:  http.StatusBadRequest,
			wantCode:    errorCodeInvalidRequest,
			wantMessage: errorMessageInvalidRecordRequest,
		},
		{
			name:        "card payload contains text field",
			contentType: "application/json",
			body:        `{"type":"card","title":"Joel's card","payload":{"number":"2013061420200619","text":"secret"}}`,
			wantStatus:  http.StatusBadRequest,
			wantCode:    errorCodeInvalidRequest,
			wantMessage: errorMessageInvalidRecordRequest,
		},
		{
			name:        "unsupported record type",
			contentType: "application/json",
			body:        `{"type":"otp","title":"token","payload":{"secret":"value"}}`,
			wantStatus:  http.StatusBadRequest,
			wantCode:    errorCodeInvalidRequest,
			wantMessage: errorMessageInvalidRecordRequest,
		},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			records := recordManagerStub{
				create: func(context.Context, service.CreateRecordRequest) (model.Record, error) {
					t.Fatal("record service must not be called")
					return model.Record{}, nil
				},
			}
			request := httptest.NewRequest(http.MethodPost, "/api/v1/records", strings.NewReader(tt.body))
			if tt.contentType != "" {
				request.Header.Set("Content-Type", tt.contentType)
			}
			response := httptest.NewRecorder()

			serveAuthenticatedRecordHandler(t, createRecordHandler(records), response, request)

			assertErrorResponse(t, response, tt.wantStatus, tt.wantCode, tt.wantMessage)
		})
	}
}
