package httpserver

import (
	"context"
	"encoding/json"
	"errors"
	"fmt"
	"net/http"
	"net/http/httptest"
	"reflect"
	"strings"
	"testing"
	"time"

	"github.com/xhrobj/gopherkeeper/internal/model"
	"github.com/xhrobj/gopherkeeper/internal/server/middleware"
	"github.com/xhrobj/gopherkeeper/internal/server/service"
)

const testRecordID = "7b4c2d7d-0e2f-4c4b-8d4b-8f4f7c4d3a21"

var (
	testRecordCreatedAt = time.Date(2026, time.July, 12, 12, 0, 0, 0, time.UTC)
	testRecordUpdatedAt = time.Date(2026, time.July, 12, 12, 1, 0, 0, time.UTC)
)

type recordManagerStub struct {
	create func(context.Context, service.CreateRecordRequest) (model.Record, error)
	list   func(context.Context, int64) ([]model.RecordMetadata, error)
	get    func(context.Context, int64, string) (model.Record, error)
	update func(context.Context, service.UpdateRecordRequest) (model.Record, error)
	delete func(context.Context, service.DeleteRecordRequest) error
}

func (s recordManagerStub) Create(
	ctx context.Context,
	request service.CreateRecordRequest,
) (model.Record, error) {
	return s.create(ctx, request)
}

func (s recordManagerStub) List(ctx context.Context, userID int64) ([]model.RecordMetadata, error) {
	return s.list(ctx, userID)
}

func (s recordManagerStub) Get(
	ctx context.Context,
	userID int64,
	recordID string,
) (model.Record, error) {
	return s.get(ctx, userID, recordID)
}

func (s recordManagerStub) Update(
	ctx context.Context,
	request service.UpdateRecordRequest,
) (model.Record, error) {
	return s.update(ctx, request)
}

func (s recordManagerStub) Delete(ctx context.Context, request service.DeleteRecordRequest) error {
	return s.delete(ctx, request)
}

type recordResponseEnvelope struct {
	ID        string           `json:"id"`
	Type      model.RecordType `json:"type"`
	Title     string           `json:"title"`
	Revision  int64            `json:"revision"`
	CreatedAt time.Time        `json:"created_at"`
	UpdatedAt time.Time        `json:"updated_at"`
	Payload   json.RawMessage  `json:"payload"`
}

type recordPayloadCase struct {
	name       string
	title      string
	payload    model.RecordPayload
	wantBase64 string
}

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
			wantCode:    errorCodeInvalidRequest,
			wantMessage: errorMessageInvalidRecordRequest,
		},
		{
			name:        "null data",
			body:        `{"type":"binary","title":"Empty","payload":{"filename":"empty.bin","data":null}}`,
			wantStatus:  http.StatusBadRequest,
			wantCode:    errorCodeInvalidRequest,
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
			name:        "unknown field",
			contentType: "application/json",
			body:        `{"type":"text","title":"my note","payload":{"text":"secret"},"extra":42}`,
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
			body:        `{"type":"card","title":"Joel's card","payload":{"number":"2013 0614 2020 0619","text":"secret"}}`,
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
		{
			name:        "multiple JSON values",
			contentType: "application/json",
			body:        validBody + validBody,
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
			name:        "unknown field",
			ifMatch:     `"1"`,
			contentType: "application/json",
			body:        `{"type":"text","title":"my note","payload":{"text":"secret"},"extra":42}`,
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
		{
			name:        "multiple JSON values",
			ifMatch:     `"1"`,
			contentType: "application/json",
			body:        validBody + validBody,
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

func TestRecordHandlers_RejectOversizedBody(t *testing.T) {
	body := `{"type":"text","title":"my note","payload":{"text":"` +
		strings.Repeat("a", int(maxRequestBodySize)) +
		`"}}`
	tests := []struct {
		name    string
		handler http.Handler
		request *http.Request
	}{
		{
			name: "create",
			handler: createRecordHandler(recordManagerStub{
				create: func(context.Context, service.CreateRecordRequest) (model.Record, error) {
					t.Fatal("record service must not be called")
					return model.Record{}, nil
				},
			}),
			request: newCreateRecordRequest(body),
		},
		{
			name: "update",
			handler: updateRecordHandler(recordManagerStub{
				update: func(context.Context, service.UpdateRecordRequest) (model.Record, error) {
					t.Fatal("record service must not be called")
					return model.Record{}, nil
				},
			}),
			request: func() *http.Request {
				request := newUpdateRecordRequest(testRecordID, body)
				request.Header.Set("If-Match", `"1"`)
				return request
			}(),
		},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			response := httptest.NewRecorder()

			serveAuthenticatedRecordHandler(t, tt.handler, response, tt.request)

			assertErrorResponse(
				t,
				response,
				http.StatusRequestEntityTooLarge,
				errorCodePayloadTooLarge,
				errorMessagePayloadTooLarge,
			)
		})
	}
}

func TestDeleteRecordHandler_DeletesRecord(t *testing.T) {
	var gotRequest service.DeleteRecordRequest
	records := recordManagerStub{
		delete: func(_ context.Context, request service.DeleteRecordRequest) error {
			gotRequest = request
			return nil
		},
	}
	request := newDeleteRecordRequest(testRecordID)
	request.Header.Set("If-Match", `"2"`)
	response := httptest.NewRecorder()

	serveAuthenticatedRecordHandler(t, deleteRecordHandler(records), response, request)

	if gotRequest.UserID != 42 {
		t.Errorf("Delete() userID = %d, want 42", gotRequest.UserID)
	}
	if gotRequest.RecordID != testRecordID {
		t.Errorf("Delete() recordID = %q, want %q", gotRequest.RecordID, testRecordID)
	}
	if gotRequest.ExpectedRevision != 2 {
		t.Errorf("Delete() expected revision = %d, want 2", gotRequest.ExpectedRevision)
	}
	if response.Code != http.StatusNoContent {
		t.Errorf("status code = %d, want %d", response.Code, http.StatusNoContent)
	}
	if response.Body.Len() != 0 {
		t.Errorf("response body = %q, want empty", response.Body.String())
	}
}

func TestDeleteRecordHandler_RejectsInvalidIfMatch(t *testing.T) {
	preconditionRequired := errorResponseExpectation{
		status:  http.StatusPreconditionRequired,
		code:    errorCodePreconditionRequired,
		message: errorMessagePreconditionRequired,
	}
	invalidRequest := errorResponseExpectation{
		status:  http.StatusBadRequest,
		code:    errorCodeInvalidRequest,
		message: errorMessageInvalidRecordRequest,
	}

	tests := []struct {
		name    string
		ifMatch string
		want    errorResponseExpectation
	}{
		{name: "missing If-Match", want: preconditionRequired},
		{name: "unquoted If-Match", ifMatch: "1", want: invalidRequest},
		{name: "zero If-Match revision", ifMatch: `"0"`, want: invalidRequest},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			records := recordManagerStub{
				delete: func(context.Context, service.DeleteRecordRequest) error {
					t.Fatal("record service must not be called")
					return nil
				},
			}
			request := newDeleteRecordRequest(testRecordID)
			if tt.ifMatch != "" {
				request.Header.Set("If-Match", tt.ifMatch)
			}
			response := httptest.NewRecorder()

			serveAuthenticatedRecordHandler(t, deleteRecordHandler(records), response, request)

			assertErrorResponse(t, response, tt.want.status, tt.want.code, tt.want.message)
		})
	}
}

func TestRecordHandlers_MapServiceError(t *testing.T) {
	internalError := errors.New("database connection details")
	validBody := recordRequestBody(t, model.RecordTypeText, "my note", &model.TextPayload{Text: "secret"})
	tests := []struct {
		name    string
		handler http.Handler
		request *http.Request
	}{
		{
			name: "create",
			handler: createRecordHandler(recordManagerStub{
				create: func(context.Context, service.CreateRecordRequest) (model.Record, error) {
					return model.Record{}, internalError
				},
			}),
			request: newCreateRecordRequest(validBody),
		},
		{
			name: "list",
			handler: listRecordsHandler(recordManagerStub{
				list: func(context.Context, int64) ([]model.RecordMetadata, error) {
					return nil, internalError
				},
			}),
			request: httptest.NewRequest(http.MethodGet, "/api/v1/records", nil),
		},
		{
			name: "get",
			handler: getRecordHandler(recordManagerStub{
				get: func(context.Context, int64, string) (model.Record, error) {
					return model.Record{}, internalError
				},
			}),
			request: func() *http.Request {
				request := httptest.NewRequest(http.MethodGet, "/api/v1/records/"+testRecordID, nil)
				request.SetPathValue("id", testRecordID)
				return request
			}(),
		},
		{
			name: "update",
			handler: updateRecordHandler(recordManagerStub{
				update: func(context.Context, service.UpdateRecordRequest) (model.Record, error) {
					return model.Record{}, internalError
				},
			}),
			request: func() *http.Request {
				request := newUpdateRecordRequest(testRecordID, validBody)
				request.Header.Set("If-Match", `"1"`)
				return request
			}(),
		},
		{
			name: "delete",
			handler: deleteRecordHandler(recordManagerStub{
				delete: func(context.Context, service.DeleteRecordRequest) error {
					return internalError
				},
			}),
			request: func() *http.Request {
				request := newDeleteRecordRequest(testRecordID)
				request.Header.Set("If-Match", `"1"`)
				return request
			}(),
		},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			response := httptest.NewRecorder()

			serveAuthenticatedRecordHandler(t, tt.handler, response, tt.request)

			assertErrorResponse(t, response, http.StatusInternalServerError, errorCodeInternal, errorMessageInternal)
			if strings.Contains(response.Body.String(), internalError.Error()) {
				t.Error("response body contains internal error details")
			}
		})
	}
}

func TestWriteRecordError(t *testing.T) {
	internalError := errors.New("database connection details")
	payloadTooLargeError := fmt.Errorf("create record: %w", model.ErrPayloadTooLarge)
	recordNotFoundError := fmt.Errorf("get record: %w", model.ErrRecordNotFound)
	revisionConflictError := fmt.Errorf("update record: %w", model.ErrRecordRevisionConflict)
	preconditionRequiredError := fmt.Errorf("update record: %w", model.ErrRecordPreconditionRequired)

	payloadTooLarge := errorResponseExpectation{
		status:  http.StatusRequestEntityTooLarge,
		code:    errorCodePayloadTooLarge,
		message: errorMessagePayloadTooLarge,
	}
	recordNotFound := errorResponseExpectation{
		status:  http.StatusNotFound,
		code:    errorCodeRecordNotFound,
		message: errorMessageRecordNotFound,
	}
	revisionConflict := errorResponseExpectation{
		status:  http.StatusConflict,
		code:    errorCodeRevisionConflict,
		message: errorMessageRevisionConflict,
	}
	preconditionRequired := errorResponseExpectation{
		status:  http.StatusPreconditionRequired,
		code:    errorCodePreconditionRequired,
		message: errorMessagePreconditionRequired,
	}
	invalidRequest := errorResponseExpectation{
		status:  http.StatusBadRequest,
		code:    errorCodeInvalidRequest,
		message: errorMessageInvalidRecordRequest,
	}
	internal := errorResponseExpectation{
		status:  http.StatusInternalServerError,
		code:    errorCodeInternal,
		message: errorMessageInternal,
	}

	tests := []struct {
		name string
		err  error
		want errorResponseExpectation
	}{
		{name: "payload too large", err: payloadTooLargeError, want: payloadTooLarge},
		{name: "record not found", err: recordNotFoundError, want: recordNotFound},
		{name: "revision conflict", err: revisionConflictError, want: revisionConflict},
		{name: "precondition required", err: preconditionRequiredError, want: preconditionRequired},
		{name: "invalid record ID", err: model.ErrInvalidRecordID, want: invalidRequest},
		{name: "invalid record revision", err: model.ErrInvalidRecordRevision, want: invalidRequest},
		{name: "invalid record title", err: model.ErrInvalidRecordTitle, want: invalidRequest},
		{name: "invalid text payload", err: model.ErrInvalidTextPayload, want: invalidRequest},
		{name: "invalid credentials payload", err: model.ErrInvalidCredentialsPayload, want: invalidRequest},
		{name: "invalid card payload", err: model.ErrInvalidCardPayload, want: invalidRequest},
		{name: "invalid binary payload", err: model.ErrInvalidBinaryPayload, want: invalidRequest},
		{name: "unsupported record type", err: model.ErrRecordTypeUnsupported, want: invalidRequest},
		{name: "internal error", err: internalError, want: internal},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			response := httptest.NewRecorder()

			writeRecordError(response, tt.err)

			assertErrorResponse(t, response, tt.want.status, tt.want.code, tt.want.message)
			if strings.Contains(response.Body.String(), internalError.Error()) {
				t.Error("response body contains internal error details")
			}
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

func TestRecordHandlers_RequireAuthentication(t *testing.T) {
	records := recordManagerStub{
		create: func(context.Context, service.CreateRecordRequest) (model.Record, error) {
			t.Fatal("record service must not be called")
			return model.Record{}, nil
		},
		list: func(context.Context, int64) ([]model.RecordMetadata, error) {
			t.Fatal("record service must not be called")
			return nil, nil
		},
		get: func(context.Context, int64, string) (model.Record, error) {
			t.Fatal("record service must not be called")
			return model.Record{}, nil
		},
		update: func(context.Context, service.UpdateRecordRequest) (model.Record, error) {
			t.Fatal("record service must not be called")
			return model.Record{}, nil
		},
		delete: func(context.Context, service.DeleteRecordRequest) error {
			t.Fatal("record service must not be called")
			return nil
		},
	}
	validBody := recordRequestBody(t, model.RecordTypeText, "my note", &model.TextPayload{Text: "secret"})
	tests := []struct {
		name    string
		handler http.Handler
		request *http.Request
	}{
		{name: "create", handler: createRecordHandler(records), request: newCreateRecordRequest(validBody)},
		{name: "list", handler: listRecordsHandler(records), request: httptest.NewRequest(http.MethodGet, "/api/v1/records", nil)},
		{name: "get", handler: getRecordHandler(records), request: httptest.NewRequest(http.MethodGet, "/api/v1/records/"+testRecordID, nil)},
		{name: "update", handler: updateRecordHandler(records), request: newUpdateRecordRequest(testRecordID, validBody)},
		{name: "delete", handler: deleteRecordHandler(records), request: newDeleteRecordRequest(testRecordID)},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			response := httptest.NewRecorder()

			tt.handler.ServeHTTP(response, tt.request)

			assertUnauthorizedResponse(t, response)
		})
	}
}

func recordPayloadCases() []recordPayloadCase {
	month := 3
	year := 2038

	return []recordPayloadCase{
		{
			name:  "text",
			title: "my note",
			payload: &model.TextPayload{
				Text:     "secret note",
				Metadata: "private metadata",
			},
		},
		{
			name:  "credentials",
			title: "GitHub",
			payload: &model.CredentialsPayload{
				Login:    "alice",
				Password: "correct-horse-battery-staple",
				URL:      "https://github.com",
				Metadata: "personal account",
			},
		},
		{
			name:  "card",
			title: "Joel's card",
			payload: &model.CardPayload{
				Number:      "2013 0614 2020 0619",
				Cardholder:  "Joel Miller",
				ExpiryMonth: &month,
				ExpiryYear:  &year,
				CVV:         "014",
				Metadata:    "test card",
			},
		},
		{
			name:  "binary",
			title: "Backup",
			payload: &model.BinaryPayload{
				Filename:    "backup.bin",
				Data:        []byte{0x00, 0x01, 0x02, 0xff},
				ContentType: "application/octet-stream",
				Metadata:    "encrypted backup",
			},
			wantBase64: `"data":"AAEC/w=="`,
		},
	}
}

func testRecord(title string, revision int64, payload model.RecordPayload) model.Record {
	return model.Record{
		Metadata: testRecordMetadata(title, payload.RecordType(), revision),
		Payload:  payload,
	}
}

func testRecordMetadata(title string, recordType model.RecordType, revision int64) model.RecordMetadata {
	return model.RecordMetadata{
		ID:        testRecordID,
		Type:      recordType,
		Title:     title,
		Revision:  revision,
		CreatedAt: testRecordCreatedAt,
		UpdatedAt: testRecordUpdatedAt,
	}
}

func serveAuthenticatedRecordHandler(
	t *testing.T,
	handler http.Handler,
	response *httptest.ResponseRecorder,
	request *http.Request,
) {
	t.Helper()

	validator := tokenValidatorFunc(func(_ context.Context, token string) (int64, error) {
		if token != "valid-token" {
			t.Fatalf("Validate() token = %q, want valid-token", token)
		}

		return 42, nil
	})
	request.Header.Set("Authorization", "Bearer valid-token")

	middleware.WithAuthentication(handler, validator).ServeHTTP(response, request)
}

func newCreateRecordRequest(body string) *http.Request {
	request := httptest.NewRequest(http.MethodPost, "/api/v1/records", strings.NewReader(body))
	request.Header.Set("Content-Type", "application/json")

	return request
}

func newUpdateRecordRequest(recordID, body string) *http.Request {
	request := httptest.NewRequest(http.MethodPut, "/api/v1/records/"+recordID, strings.NewReader(body))
	request.SetPathValue("id", recordID)
	request.Header.Set("Content-Type", "application/json")

	return request
}

func newDeleteRecordRequest(recordID string) *http.Request {
	request := httptest.NewRequest(http.MethodDelete, "/api/v1/records/"+recordID, nil)
	request.SetPathValue("id", recordID)

	return request
}

func recordRequestBody(t *testing.T, recordType model.RecordType, title string, payload any) string {
	t.Helper()

	body, err := json.Marshal(struct {
		Type    model.RecordType `json:"type"`
		Title   string           `json:"title"`
		Payload any              `json:"payload"`
	}{
		Type:    recordType,
		Title:   title,
		Payload: payload,
	})
	if err != nil {
		t.Fatalf("encode record request: %v", err)
	}

	return string(body)
}

func decodeJSONResponse(t *testing.T, response *httptest.ResponseRecorder, target any) {
	t.Helper()

	if contentType := response.Header().Get("Content-Type"); contentType != "application/json" {
		t.Errorf("Content-Type = %q, want application/json", contentType)
	}
	if err := json.Unmarshal(response.Body.Bytes(), target); err != nil {
		t.Fatalf("decode response body: %v", err)
	}
}

func assertCreateRecordRequest(
	t *testing.T,
	got service.CreateRecordRequest,
	wantTitle string,
	wantPayload model.RecordPayload,
) {
	t.Helper()

	if got.UserID != 42 {
		t.Errorf("Create() userID = %d, want 42", got.UserID)
	}
	if got.Title != wantTitle {
		t.Errorf("Create() title = %q, want %q", got.Title, wantTitle)
	}
	if !reflect.DeepEqual(got.Payload, wantPayload) {
		t.Errorf("Create() payload = %#v, want %#v", got.Payload, wantPayload)
	}
}

func assertUpdateRecordRequest(
	t *testing.T,
	got service.UpdateRecordRequest,
	wantTitle string,
	wantPayload model.RecordPayload,
) {
	t.Helper()

	if got.UserID != 42 {
		t.Errorf("Update() userID = %d, want 42", got.UserID)
	}
	if got.RecordID != testRecordID {
		t.Errorf("Update() recordID = %q, want %q", got.RecordID, testRecordID)
	}
	if got.ExpectedRevision != 1 {
		t.Errorf("Update() expected revision = %d, want 1", got.ExpectedRevision)
	}
	if got.Title != wantTitle {
		t.Errorf("Update() title = %q, want %q", got.Title, wantTitle)
	}
	if !reflect.DeepEqual(got.Payload, wantPayload) {
		t.Errorf("Update() payload = %#v, want %#v", got.Payload, wantPayload)
	}
}

func assertRecordResponse(
	t *testing.T,
	response *httptest.ResponseRecorder,
	wantStatus int,
	wantRevision int64,
	wantTitle string,
	wantPayload model.RecordPayload,
) {
	t.Helper()

	if response.Code != wantStatus {
		t.Errorf("status code = %d, want %d", response.Code, wantStatus)
	}
	wantETag := fmt.Sprintf(`"%d"`, wantRevision)
	if got := response.Header().Get("ETag"); got != wantETag {
		t.Errorf("ETag = %q, want %q", got, wantETag)
	}

	var body recordResponseEnvelope
	decodeJSONResponse(t, response, &body)
	assertRecordMetadataResponse(t, recordMetadataResponse{
		ID:        body.ID,
		Type:      body.Type,
		Title:     body.Title,
		Revision:  body.Revision,
		CreatedAt: body.CreatedAt,
		UpdatedAt: body.UpdatedAt,
	}, newRecordMetadataResponse(testRecordMetadata(wantTitle, wantPayload.RecordType(), wantRevision)))
	assertJSONValue(t, body.Payload, wantPayload)
}

func assertRecordMetadataResponse(t *testing.T, got, want recordMetadataResponse) {
	t.Helper()

	if got.ID != want.ID {
		t.Errorf("response id = %q, want %q", got.ID, want.ID)
	}
	if got.Type != want.Type {
		t.Errorf("response type = %q, want %q", got.Type, want.Type)
	}
	if got.Title != want.Title {
		t.Errorf("response title = %q, want %q", got.Title, want.Title)
	}
	if got.Revision != want.Revision {
		t.Errorf("response revision = %d, want %d", got.Revision, want.Revision)
	}
	if !got.CreatedAt.Equal(want.CreatedAt) {
		t.Errorf("response created_at = %s, want %s", got.CreatedAt, want.CreatedAt)
	}
	if !got.UpdatedAt.Equal(want.UpdatedAt) {
		t.Errorf("response updated_at = %s, want %s", got.UpdatedAt, want.UpdatedAt)
	}
}

func assertJSONValue(t *testing.T, gotJSON []byte, want any) {
	t.Helper()

	var gotValue any
	if err := json.Unmarshal(gotJSON, &gotValue); err != nil {
		t.Fatalf("decode response payload: %v", err)
	}
	wantJSON, err := json.Marshal(want)
	if err != nil {
		t.Fatalf("encode expected payload: %v", err)
	}
	var wantValue any
	if err := json.Unmarshal(wantJSON, &wantValue); err != nil {
		t.Fatalf("decode expected payload: %v", err)
	}
	if !reflect.DeepEqual(gotValue, wantValue) {
		t.Errorf("response payload = %#v, want %#v", gotValue, wantValue)
	}
}

func requireBinaryPayload(t *testing.T, payload model.RecordPayload) model.BinaryPayload {
	t.Helper()

	value, ok := payload.(*model.BinaryPayload)
	if !ok || value == nil {
		t.Fatalf("payload type = %T, want non-nil *BinaryPayload", payload)
	}

	return *value
}
