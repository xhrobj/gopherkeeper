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

type textRecordResponse struct {
	ID        string            `json:"id"`
	Type      model.RecordType  `json:"type"`
	Title     string            `json:"title"`
	Revision  int64             `json:"revision"`
	CreatedAt time.Time         `json:"created_at"`
	UpdatedAt time.Time         `json:"updated_at"`
	Payload   model.TextPayload `json:"payload"`
}

type credentialsRecordResponse struct {
	ID        string                   `json:"id"`
	Type      model.RecordType         `json:"type"`
	Title     string                   `json:"title"`
	Revision  int64                    `json:"revision"`
	CreatedAt time.Time                `json:"created_at"`
	UpdatedAt time.Time                `json:"updated_at"`
	Payload   model.CredentialsPayload `json:"payload"`
}

type cardRecordResponse struct {
	ID        string            `json:"id"`
	Type      model.RecordType  `json:"type"`
	Title     string            `json:"title"`
	Revision  int64             `json:"revision"`
	CreatedAt time.Time         `json:"created_at"`
	UpdatedAt time.Time         `json:"updated_at"`
	Payload   model.CardPayload `json:"payload"`
}

type binaryRecordResponse struct {
	ID        string              `json:"id"`
	Type      model.RecordType    `json:"type"`
	Title     string              `json:"title"`
	Revision  int64               `json:"revision"`
	CreatedAt time.Time           `json:"created_at"`
	UpdatedAt time.Time           `json:"updated_at"`
	Payload   model.BinaryPayload `json:"payload"`
}

func TestCreateRecordHandler_CreatesTextRecord(t *testing.T) {
	createdAt := time.Date(2026, time.July, 8, 12, 0, 0, 0, time.UTC)
	updatedAt := time.Date(2026, time.July, 8, 12, 1, 0, 0, time.UTC)
	var gotRequest service.CreateRecordRequest
	records := recordManagerStub{
		create: func(_ context.Context, request service.CreateRecordRequest) (model.Record, error) {
			gotRequest = request

			return model.Record{
				Metadata: model.RecordMetadata{
					ID:        testRecordID,
					Type:      model.RecordTypeText,
					Title:     request.Title,
					Revision:  model.RecordInitialRevision,
					CreatedAt: createdAt,
					UpdatedAt: updatedAt,
				},
				Payload: request.Payload,
			}, nil
		},
	}
	request := newCreateRecordRequest(t, createTextRecordRequestBody(t, "my note", "secret note", "private metadata"))
	response := httptest.NewRecorder()

	serveAuthenticatedRecordHandler(t, createRecordHandler(records), response, request)

	if gotRequest.UserID != 42 {
		t.Errorf("Create() userID = %d, want 42", gotRequest.UserID)
	}
	if gotRequest.Title != "my note" {
		t.Errorf("Create() title = %q, want my note", gotRequest.Title)
	}
	gotPayload := requireTextPayload(t, gotRequest.Payload)
	if gotPayload.Text != "secret note" {
		t.Errorf("Create() payload text = %q, want secret note", gotPayload.Text)
	}
	if gotPayload.Metadata != "private metadata" {
		t.Errorf("Create() metadata = %q, want private metadata", gotPayload.Metadata)
	}
	if response.Code != http.StatusCreated {
		t.Errorf("status code = %d, want %d", response.Code, http.StatusCreated)
	}
	if etag := response.Header().Get("ETag"); etag != `"1"` {
		t.Errorf("ETag = %q, want %q", etag, `"1"`)
	}

	var body textRecordResponse
	decodeJSONResponse(t, response, &body)
	assertTextRecordResponse(t, body, textRecordResponse{
		ID:        testRecordID,
		Type:      model.RecordTypeText,
		Title:     "my note",
		Revision:  model.RecordInitialRevision,
		CreatedAt: createdAt,
		UpdatedAt: updatedAt,
		Payload: model.TextPayload{
			Text:     "secret note",
			Metadata: "private metadata",
		},
	})
}

func TestCreateRecordHandler_CreatesCredentialsRecord(t *testing.T) {
	createdAt := time.Date(2026, time.July, 10, 12, 0, 0, 0, time.UTC)
	updatedAt := time.Date(2026, time.July, 10, 12, 1, 0, 0, time.UTC)
	payload := model.CredentialsPayload{
		Login:    "alice",
		Password: "correct-horse-battery-staple",
		URL:      "https://github.com",
		Metadata: "personal account",
	}
	var gotRequest service.CreateRecordRequest
	records := recordManagerStub{
		create: func(
			_ context.Context,
			request service.CreateRecordRequest,
		) (model.Record, error) {
			gotRequest = request

			return model.Record{
				Metadata: model.RecordMetadata{
					ID:        testRecordID,
					Type:      model.RecordTypeCredentials,
					Title:     request.Title,
					Revision:  model.RecordInitialRevision,
					CreatedAt: createdAt,
					UpdatedAt: updatedAt,
				},
				Payload: request.Payload,
			}, nil
		},
	}
	request := newCreateRecordRequest(t, createCredentialsRecordRequestBody(
		t,
		"GitHub",
		payload.Login,
		payload.Password,
		payload.URL,
		payload.Metadata,
	))
	response := httptest.NewRecorder()

	serveAuthenticatedRecordHandler(t, createRecordHandler(records), response, request)

	if gotRequest.UserID != 42 {
		t.Errorf("Create() userID = %d, want 42", gotRequest.UserID)
	}
	if gotRequest.Title != "GitHub" {
		t.Errorf("Create() title = %q, want GitHub", gotRequest.Title)
	}
	if gotPayload := requireCredentialsPayload(t, gotRequest.Payload); gotPayload != payload {
		t.Errorf("Create() payload = %+v, want %+v", gotPayload, payload)
	}
	if response.Code != http.StatusCreated {
		t.Errorf("status code = %d, want %d", response.Code, http.StatusCreated)
	}
	if etag := response.Header().Get("ETag"); etag != `"1"` {
		t.Errorf("ETag = %q, want %q", etag, `"1"`)
	}

	var body credentialsRecordResponse
	decodeJSONResponse(t, response, &body)
	assertCredentialsRecordResponse(t, body, credentialsRecordResponse{
		ID:        testRecordID,
		Type:      model.RecordTypeCredentials,
		Title:     "GitHub",
		Revision:  model.RecordInitialRevision,
		CreatedAt: createdAt,
		UpdatedAt: updatedAt,
		Payload:   payload,
	})
}

func TestCreateRecordHandler_CreatesCardRecord(t *testing.T) {
	createdAt := time.Date(2026, time.July, 11, 12, 0, 0, 0, time.UTC)
	updatedAt := time.Date(2026, time.July, 11, 12, 1, 0, 0, time.UTC)
	month := 3
	year := 2038
	payload := model.CardPayload{
		Number:      "2013 0614 2020 0619",
		Cardholder:  "Joel Miller",
		ExpiryMonth: &month,
		ExpiryYear:  &year,
		CVV:         "014",
		Metadata:    "test card",
	}
	var gotRequest service.CreateRecordRequest
	records := recordManagerStub{
		create: func(
			_ context.Context,
			request service.CreateRecordRequest,
		) (model.Record, error) {
			gotRequest = request

			return model.Record{
				Metadata: model.RecordMetadata{
					ID:        testRecordID,
					Type:      model.RecordTypeCard,
					Title:     request.Title,
					Revision:  model.RecordInitialRevision,
					CreatedAt: createdAt,
					UpdatedAt: updatedAt,
				},
				Payload: request.Payload,
			}, nil
		},
	}
	request := newCreateRecordRequest(t, createCardRecordRequestBody(t, "Joel's card", payload))
	response := httptest.NewRecorder()

	serveAuthenticatedRecordHandler(t, createRecordHandler(records), response, request)

	if gotRequest.UserID != 42 {
		t.Errorf("Create() userID = %d, want 42", gotRequest.UserID)
	}
	if gotRequest.Title != "Joel's card" {
		t.Errorf("Create() title = %q, want Joel's card", gotRequest.Title)
	}
	if gotPayload := requireCardPayload(t, gotRequest.Payload); !reflect.DeepEqual(gotPayload, payload) {
		t.Errorf("Create() payload = %+v, want %+v", gotPayload, payload)
	}
	if response.Code != http.StatusCreated {
		t.Errorf("status code = %d, want %d", response.Code, http.StatusCreated)
	}
	if etag := response.Header().Get("ETag"); etag != `"1"` {
		t.Errorf("ETag = %q, want %q", etag, `"1"`)
	}

	var body cardRecordResponse
	decodeJSONResponse(t, response, &body)
	assertRecordMetadataResponse(t, recordMetadataResponse{
		ID:        body.ID,
		Type:      body.Type,
		Title:     body.Title,
		Revision:  body.Revision,
		CreatedAt: body.CreatedAt,
		UpdatedAt: body.UpdatedAt,
	}, recordMetadataResponse{
		ID:        testRecordID,
		Type:      model.RecordTypeCard,
		Title:     "Joel's card",
		Revision:  model.RecordInitialRevision,
		CreatedAt: createdAt,
		UpdatedAt: updatedAt,
	})
	if !reflect.DeepEqual(body.Payload, payload) {
		t.Errorf("response payload = %+v, want %+v", body.Payload, payload)
	}
}

func TestCreateRecordHandler_CreatesBinaryRecord(t *testing.T) {
	createdAt := time.Date(2026, time.July, 12, 12, 0, 0, 0, time.UTC)
	updatedAt := time.Date(2026, time.July, 12, 12, 1, 0, 0, time.UTC)
	payload := model.BinaryPayload{
		Filename:    "backup.bin",
		Data:        []byte{0x00, 0x01, 0x02, 0xff},
		ContentType: "application/octet-stream",
		Metadata:    "encrypted backup",
	}
	var gotRequest service.CreateRecordRequest
	records := recordManagerStub{
		create: func(
			_ context.Context,
			request service.CreateRecordRequest,
		) (model.Record, error) {
			gotRequest = request

			return model.Record{
				Metadata: model.RecordMetadata{
					ID:        testRecordID,
					Type:      model.RecordTypeBinary,
					Title:     request.Title,
					Revision:  model.RecordInitialRevision,
					CreatedAt: createdAt,
					UpdatedAt: updatedAt,
				},
				Payload: request.Payload,
			}, nil
		},
	}
	requestBody := createBinaryRecordRequestBody(t, "Backup", payload)
	if !strings.Contains(requestBody, `"data":"AAEC/w=="`) {
		t.Fatalf("request body does not contain expected Base64 data: %s", requestBody)
	}
	request := newCreateRecordRequest(t, requestBody)
	response := httptest.NewRecorder()

	serveAuthenticatedRecordHandler(t, createRecordHandler(records), response, request)

	if gotRequest.UserID != 42 {
		t.Errorf("Create() userID = %d, want 42", gotRequest.UserID)
	}
	if gotRequest.Title != "Backup" {
		t.Errorf("Create() title = %q, want Backup", gotRequest.Title)
	}
	if gotPayload := requireBinaryPayload(t, gotRequest.Payload); !reflect.DeepEqual(gotPayload, payload) {
		t.Errorf("Create() payload = %+v, want %+v", gotPayload, payload)
	}
	if response.Code != http.StatusCreated {
		t.Errorf("status code = %d, want %d", response.Code, http.StatusCreated)
	}
	if etag := response.Header().Get("ETag"); etag != `"1"` {
		t.Errorf("ETag = %q, want %q", etag, `"1"`)
	}

	if !strings.Contains(response.Body.String(), `"data":"AAEC/w=="`) {
		t.Errorf("response body does not contain expected Base64 data: %s", response.Body.String())
	}

	var body binaryRecordResponse
	decodeJSONResponse(t, response, &body)
	assertRecordMetadataResponse(t, recordMetadataResponse{
		ID:        body.ID,
		Type:      body.Type,
		Title:     body.Title,
		Revision:  body.Revision,
		CreatedAt: body.CreatedAt,
		UpdatedAt: body.UpdatedAt,
	}, recordMetadataResponse{
		ID:        testRecordID,
		Type:      model.RecordTypeBinary,
		Title:     "Backup",
		Revision:  model.RecordInitialRevision,
		CreatedAt: createdAt,
		UpdatedAt: updatedAt,
	})
	if !reflect.DeepEqual(body.Payload, payload) {
		t.Errorf("response payload = %+v, want %+v", body.Payload, payload)
	}
}

func TestCreateRecordHandler_AcceptsEmptyBinaryFile(t *testing.T) {
	records := recordManagerStub{
		create: func(
			_ context.Context,
			request service.CreateRecordRequest,
		) (model.Record, error) {
			payload := requireBinaryPayload(t, request.Payload)
			if payload.Data == nil {
				t.Fatal("Create() binary data = nil, want present empty slice")
			}
			if len(payload.Data) != 0 {
				t.Fatalf("Create() binary data length = %d, want 0", len(payload.Data))
			}

			return model.Record{
				Metadata: model.RecordMetadata{
					ID:       testRecordID,
					Type:     model.RecordTypeBinary,
					Title:    request.Title,
					Revision: model.RecordInitialRevision,
				},
				Payload: request.Payload,
			}, nil
		},
	}
	request := newCreateRecordRequest(
		t,
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
		name       string
		body       string
		wantStatus int
		wantCode   string
	}{
		{
			name:       "missing data",
			body:       `{"type":"binary","title":"Empty","payload":{"filename":"empty.bin"}}`,
			wantStatus: http.StatusBadRequest,
			wantCode:   errorCodeInvalidRequest,
		},
		{
			name:       "null data",
			body:       `{"type":"binary","title":"Empty","payload":{"filename":"empty.bin","data":null}}`,
			wantStatus: http.StatusBadRequest,
			wantCode:   errorCodeInvalidRequest,
		},
		{
			name: "decoded data too large",
			body: createBinaryRecordRequestBody(t, "Large", model.BinaryPayload{
				Filename: "large.bin",
				Data:     make([]byte, model.BinaryPayloadMaxSize+1),
			}),
			wantStatus: http.StatusRequestEntityTooLarge,
			wantCode:   errorCodePayloadTooLarge,
		},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			records := recordManagerStub{
				create: func(
					_ context.Context,
					request service.CreateRecordRequest,
				) (model.Record, error) {
					return model.Record{}, request.Payload.Validate()
				},
			}
			request := newCreateRecordRequest(t, tt.body)
			response := httptest.NewRecorder()

			serveAuthenticatedRecordHandler(t, createRecordHandler(records), response, request)

			assertErrorResponse(t, response, tt.wantStatus, tt.wantCode, map[int]string{
				http.StatusBadRequest:            errorMessageInvalidRecordRequest,
				http.StatusRequestEntityTooLarge: errorMessagePayloadTooLarge,
			}[tt.wantStatus])
		})
	}
}

func TestCreateRecordHandler_RejectsInvalidRequest(t *testing.T) {
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
			body:        createTextRecordRequestBody(t, "my note", "secret note", ""),
			wantStatus:  http.StatusUnsupportedMediaType,
			wantCode:    errorCodeUnsupportedMediaType,
			wantMessage: errorMessageUnsupportedMediaType,
		},
		{
			name:        "unsupported Content-Type",
			contentType: "text/plain",
			body:        createTextRecordRequestBody(t, "my note", "secret note", ""),
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
			body: `{"type":"credentials","title":"GitHub","payload":` +
				`{"login":"alice","password":"secret","text":"note"}}`,
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
			body: createTextRecordRequestBody(t, "my note", "secret note", "") +
				createTextRecordRequestBody(t, "my note", "secret note", ""),
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

func TestCreateRecordHandler_RejectsOversizedBody(t *testing.T) {
	records := recordManagerStub{
		create: func(context.Context, service.CreateRecordRequest) (model.Record, error) {
			t.Fatal("record service must not be called")
			return model.Record{}, nil
		},
	}
	body := `{"type":"text","title":"my note","payload":{"text":"` +
		strings.Repeat("a", int(maxRequestBodySize)) +
		`"}}`
	request := newCreateRecordRequest(t, body)
	response := httptest.NewRecorder()

	serveAuthenticatedRecordHandler(t, createRecordHandler(records), response, request)

	assertErrorResponse(
		t,
		response,
		http.StatusRequestEntityTooLarge,
		errorCodePayloadTooLarge,
		errorMessagePayloadTooLarge,
	)
}

func TestCreateRecordHandler_MapsServiceError(t *testing.T) {
	internalError := errors.New("database connection details")
	records := recordManagerStub{
		create: func(context.Context, service.CreateRecordRequest) (model.Record, error) {
			return model.Record{}, internalError
		},
	}
	request := newCreateRecordRequest(t, createTextRecordRequestBody(t, "my note", "secret", ""))
	response := httptest.NewRecorder()

	serveAuthenticatedRecordHandler(t, createRecordHandler(records), response, request)

	assertErrorResponse(
		t,
		response,
		http.StatusInternalServerError,
		errorCodeInternal,
		errorMessageInternal,
	)
	if strings.Contains(response.Body.String(), internalError.Error()) {
		t.Error("response body contains internal error details")
	}
}

func TestListRecordsHandler_ReturnsMetadataOnly(t *testing.T) {
	createdAt := time.Date(2026, time.July, 8, 12, 0, 0, 0, time.UTC)
	updatedAt := time.Date(2026, time.July, 8, 12, 1, 0, 0, time.UTC)
	var gotUserID int64
	records := recordManagerStub{
		list: func(_ context.Context, userID int64) ([]model.RecordMetadata, error) {
			gotUserID = userID

			return []model.RecordMetadata{{
				ID:        testRecordID,
				Type:      model.RecordTypeText,
				Title:     "my note",
				Revision:  model.RecordInitialRevision,
				CreatedAt: createdAt,
				UpdatedAt: updatedAt,
			}}, nil
		},
	}
	request := httptest.NewRequest(http.MethodGet, "/api/v1/records", nil)
	response := httptest.NewRecorder()

	serveAuthenticatedRecordHandler(t, listRecordsHandler(records), response, request)

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
	assertRecordMetadataResponse(t, body.Records[0], recordMetadataResponse{
		ID:        testRecordID,
		Type:      model.RecordTypeText,
		Title:     "my note",
		Revision:  model.RecordInitialRevision,
		CreatedAt: createdAt,
		UpdatedAt: updatedAt,
	})
	if strings.Contains(response.Body.String(), "secret note") {
		t.Error("list response contains payload")
	}
}

func TestGetRecordHandler_ReturnsTextRecord(t *testing.T) {
	createdAt := time.Date(2026, time.July, 8, 12, 0, 0, 0, time.UTC)
	updatedAt := time.Date(2026, time.July, 8, 12, 1, 0, 0, time.UTC)
	payload := model.TextPayload{
		Text:     "secret note",
		Metadata: "private metadata",
	}
	var gotUserID int64
	var gotRecordID string
	records := recordManagerStub{
		get: func(_ context.Context, userID int64, recordID string) (model.Record, error) {
			gotUserID = userID
			gotRecordID = recordID

			return model.Record{
				Metadata: model.RecordMetadata{
					ID:        testRecordID,
					Type:      model.RecordTypeText,
					Title:     "my note",
					Revision:  model.RecordInitialRevision,
					CreatedAt: createdAt,
					UpdatedAt: updatedAt,
				},
				Payload: &payload,
			}, nil
		},
	}
	request := httptest.NewRequest(http.MethodGet, "/api/v1/records/"+testRecordID, nil)
	request.SetPathValue("id", testRecordID)
	response := httptest.NewRecorder()

	serveAuthenticatedRecordHandler(t, getRecordHandler(records), response, request)

	if gotUserID != 42 {
		t.Errorf("Get() userID = %d, want 42", gotUserID)
	}
	if gotRecordID != testRecordID {
		t.Errorf("Get() recordID = %q, want %q", gotRecordID, testRecordID)
	}
	if response.Code != http.StatusOK {
		t.Errorf("status code = %d, want %d", response.Code, http.StatusOK)
	}
	if etag := response.Header().Get("ETag"); etag != `"1"` {
		t.Errorf("ETag = %q, want %q", etag, `"1"`)
	}

	var body textRecordResponse
	decodeJSONResponse(t, response, &body)
	assertTextRecordResponse(t, body, textRecordResponse{
		ID:        testRecordID,
		Type:      model.RecordTypeText,
		Title:     "my note",
		Revision:  model.RecordInitialRevision,
		CreatedAt: createdAt,
		UpdatedAt: updatedAt,
		Payload:   payload,
	})
}

func TestGetRecordHandler_ReturnsCredentialsRecord(t *testing.T) {
	createdAt := time.Date(2026, time.July, 10, 12, 0, 0, 0, time.UTC)
	updatedAt := time.Date(2026, time.July, 10, 12, 1, 0, 0, time.UTC)
	payload := model.CredentialsPayload{
		Login:    "alice",
		Password: "correct-horse-battery-staple",
		URL:      "https://github.com",
		Metadata: "personal account",
	}
	records := recordManagerStub{
		get: func(_ context.Context, userID int64, recordID string) (model.Record, error) {
			if userID != 42 {
				t.Fatalf("Get() userID = %d, want 42", userID)
			}
			if recordID != testRecordID {
				t.Fatalf("Get() recordID = %q, want %q", recordID, testRecordID)
			}

			return model.Record{
				Metadata: model.RecordMetadata{
					ID:        testRecordID,
					Type:      model.RecordTypeCredentials,
					Title:     "GitHub",
					Revision:  model.RecordInitialRevision,
					CreatedAt: createdAt,
					UpdatedAt: updatedAt,
				},
				Payload: &payload,
			}, nil
		},
	}
	request := httptest.NewRequest(http.MethodGet, "/api/v1/records/"+testRecordID, nil)
	request.SetPathValue("id", testRecordID)
	response := httptest.NewRecorder()

	serveAuthenticatedRecordHandler(t, getRecordHandler(records), response, request)

	if response.Code != http.StatusOK {
		t.Errorf("status code = %d, want %d", response.Code, http.StatusOK)
	}
	if etag := response.Header().Get("ETag"); etag != `"1"` {
		t.Errorf("ETag = %q, want %q", etag, `"1"`)
	}

	var body credentialsRecordResponse
	decodeJSONResponse(t, response, &body)
	assertCredentialsRecordResponse(t, body, credentialsRecordResponse{
		ID:        testRecordID,
		Type:      model.RecordTypeCredentials,
		Title:     "GitHub",
		Revision:  model.RecordInitialRevision,
		CreatedAt: createdAt,
		UpdatedAt: updatedAt,
		Payload:   payload,
	})
}

func TestGetRecordHandler_ReturnsBinaryRecord(t *testing.T) {
	createdAt := time.Date(2026, time.July, 12, 12, 0, 0, 0, time.UTC)
	updatedAt := time.Date(2026, time.July, 12, 12, 1, 0, 0, time.UTC)
	payload := model.BinaryPayload{
		Filename:    "backup.bin",
		Data:        []byte{0x00, 0x01, 0x02, 0xff},
		ContentType: "application/octet-stream",
		Metadata:    "encrypted backup",
	}
	records := recordManagerStub{
		get: func(_ context.Context, userID int64, recordID string) (model.Record, error) {
			if userID != 42 {
				t.Fatalf("Get() userID = %d, want 42", userID)
			}
			if recordID != testRecordID {
				t.Fatalf("Get() recordID = %q, want %q", recordID, testRecordID)
			}

			return model.Record{
				Metadata: model.RecordMetadata{
					ID:        testRecordID,
					Type:      model.RecordTypeBinary,
					Title:     "Backup",
					Revision:  model.RecordInitialRevision,
					CreatedAt: createdAt,
					UpdatedAt: updatedAt,
				},
				Payload: &payload,
			}, nil
		},
	}
	request := httptest.NewRequest(http.MethodGet, "/api/v1/records/"+testRecordID, nil)
	request.SetPathValue("id", testRecordID)
	response := httptest.NewRecorder()

	serveAuthenticatedRecordHandler(t, getRecordHandler(records), response, request)

	if response.Code != http.StatusOK {
		t.Errorf("status code = %d, want %d", response.Code, http.StatusOK)
	}
	if etag := response.Header().Get("ETag"); etag != `"1"` {
		t.Errorf("ETag = %q, want %q", etag, `"1"`)
	}

	var body binaryRecordResponse
	decodeJSONResponse(t, response, &body)
	assertRecordMetadataResponse(t, recordMetadataResponse{
		ID:        body.ID,
		Type:      body.Type,
		Title:     body.Title,
		Revision:  body.Revision,
		CreatedAt: body.CreatedAt,
		UpdatedAt: body.UpdatedAt,
	}, recordMetadataResponse{
		ID:        testRecordID,
		Type:      model.RecordTypeBinary,
		Title:     "Backup",
		Revision:  model.RecordInitialRevision,
		CreatedAt: createdAt,
		UpdatedAt: updatedAt,
	})
	if !reflect.DeepEqual(body.Payload, payload) {
		t.Errorf("response payload = %+v, want %+v", body.Payload, payload)
	}
}

func TestGetRecordHandler_RejectsInvalidServiceResult(t *testing.T) {
	records := recordManagerStub{
		get: func(context.Context, int64, string) (model.Record, error) {
			return model.Record{
				Metadata: model.RecordMetadata{
					ID:       testRecordID,
					Type:     model.RecordTypeCredentials,
					Title:    "GitHub",
					Revision: model.RecordInitialRevision,
				},
			}, nil
		},
	}
	request := httptest.NewRequest(http.MethodGet, "/api/v1/records/"+testRecordID, nil)
	request.SetPathValue("id", testRecordID)
	response := httptest.NewRecorder()

	serveAuthenticatedRecordHandler(t, getRecordHandler(records), response, request)

	assertErrorResponse(
		t,
		response,
		http.StatusInternalServerError,
		errorCodeInternal,
		errorMessageInternal,
	)
	if response.Header().Get("ETag") != "" {
		t.Errorf("ETag = %q, want empty", response.Header().Get("ETag"))
	}
}

func TestNewRecordResponse_RejectsNilPayload(t *testing.T) {
	var payload *model.TextPayload

	_, err := newRecordResponse(model.Record{
		Metadata: model.RecordMetadata{Type: model.RecordTypeText},
		Payload:  payload,
	})
	if !errors.Is(err, errInvalidRecordResponse) {
		t.Fatalf("newRecordResponse() error = %v, want %v", err, errInvalidRecordResponse)
	}
}

func TestUpdateRecordHandler_UpdatesTextRecord(t *testing.T) {
	createdAt := time.Date(2026, time.July, 9, 12, 0, 0, 0, time.UTC)
	updatedAt := time.Date(2026, time.July, 9, 12, 1, 0, 0, time.UTC)
	var gotRequest service.UpdateRecordRequest
	records := recordManagerStub{
		update: func(_ context.Context, request service.UpdateRecordRequest) (model.Record, error) {
			gotRequest = request

			return model.Record{
				Metadata: model.RecordMetadata{
					ID:        request.RecordID,
					Type:      model.RecordTypeText,
					Title:     request.Title,
					Revision:  2,
					CreatedAt: createdAt,
					UpdatedAt: updatedAt,
				},
				Payload: request.Payload,
			}, nil
		},
	}
	request := newUpdateRecordRequest(t, testRecordID, updateTextRecordRequestBody(t, "new note", "new secret", "new metadata"))
	request.Header.Set("If-Match", `"1"`)
	response := httptest.NewRecorder()

	serveAuthenticatedRecordHandler(t, updateRecordHandler(records), response, request)

	if gotRequest.UserID != 42 {
		t.Errorf("Update() userID = %d, want 42", gotRequest.UserID)
	}
	if gotRequest.RecordID != testRecordID {
		t.Errorf("Update() recordID = %q, want %q", gotRequest.RecordID, testRecordID)
	}
	if gotRequest.ExpectedRevision != 1 {
		t.Errorf("Update() expected revision = %d, want 1", gotRequest.ExpectedRevision)
	}
	if gotRequest.Title != "new note" {
		t.Errorf("Update() title = %q, want new note", gotRequest.Title)
	}
	gotPayload := requireTextPayload(t, gotRequest.Payload)
	if gotPayload.Text != "new secret" {
		t.Errorf("Update() payload text = %q, want new secret", gotPayload.Text)
	}
	if gotPayload.Metadata != "new metadata" {
		t.Errorf("Update() metadata = %q, want new metadata", gotPayload.Metadata)
	}
	if response.Code != http.StatusOK {
		t.Errorf("status code = %d, want %d", response.Code, http.StatusOK)
	}
	if etag := response.Header().Get("ETag"); etag != `"2"` {
		t.Errorf("ETag = %q, want %q", etag, `"2"`)
	}

	var body textRecordResponse
	decodeJSONResponse(t, response, &body)
	assertTextRecordResponse(t, body, textRecordResponse{
		ID:        testRecordID,
		Type:      model.RecordTypeText,
		Title:     "new note",
		Revision:  2,
		CreatedAt: createdAt,
		UpdatedAt: updatedAt,
		Payload: model.TextPayload{
			Text:     "new secret",
			Metadata: "new metadata",
		},
	})
}

func TestUpdateRecordHandler_UpdatesCredentialsRecord(t *testing.T) {
	createdAt := time.Date(2026, time.July, 10, 12, 0, 0, 0, time.UTC)
	updatedAt := time.Date(2026, time.July, 10, 12, 1, 0, 0, time.UTC)
	payload := model.CredentialsPayload{
		Login:    "alice@example.com",
		Password: "new-secret",
		URL:      "https://github.com/login",
		Metadata: "updated account",
	}
	var gotRequest service.UpdateRecordRequest
	records := recordManagerStub{
		update: func(
			_ context.Context,
			request service.UpdateRecordRequest,
		) (model.Record, error) {
			gotRequest = request

			return model.Record{
				Metadata: model.RecordMetadata{
					ID:        request.RecordID,
					Type:      model.RecordTypeCredentials,
					Title:     request.Title,
					Revision:  2,
					CreatedAt: createdAt,
					UpdatedAt: updatedAt,
				},
				Payload: request.Payload,
			}, nil
		},
	}
	request := newUpdateRecordRequest(t, testRecordID, updateCredentialsRecordRequestBody(
		t,
		"GitHub updated",
		payload.Login,
		payload.Password,
		payload.URL,
		payload.Metadata,
	))
	request.Header.Set("If-Match", `"1"`)
	response := httptest.NewRecorder()

	serveAuthenticatedRecordHandler(t, updateRecordHandler(records), response, request)

	if gotRequest.UserID != 42 {
		t.Errorf("Update() userID = %d, want 42", gotRequest.UserID)
	}
	if gotRequest.RecordID != testRecordID {
		t.Errorf("Update() recordID = %q, want %q", gotRequest.RecordID, testRecordID)
	}
	if gotRequest.ExpectedRevision != 1 {
		t.Errorf("Update() expected revision = %d, want 1", gotRequest.ExpectedRevision)
	}
	if gotRequest.Title != "GitHub updated" {
		t.Errorf("Update() title = %q, want GitHub updated", gotRequest.Title)
	}
	if gotPayload := requireCredentialsPayload(t, gotRequest.Payload); gotPayload != payload {
		t.Errorf("Update() payload = %+v, want %+v", gotPayload, payload)
	}
	if response.Code != http.StatusOK {
		t.Errorf("status code = %d, want %d", response.Code, http.StatusOK)
	}
	if etag := response.Header().Get("ETag"); etag != `"2"` {
		t.Errorf("ETag = %q, want %q", etag, `"2"`)
	}

	var body credentialsRecordResponse
	decodeJSONResponse(t, response, &body)
	assertCredentialsRecordResponse(t, body, credentialsRecordResponse{
		ID:        testRecordID,
		Type:      model.RecordTypeCredentials,
		Title:     "GitHub updated",
		Revision:  2,
		CreatedAt: createdAt,
		UpdatedAt: updatedAt,
		Payload:   payload,
	})
}

func TestUpdateRecordHandler_RejectsInvalidRequest(t *testing.T) {
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
			body:        updateTextRecordRequestBody(t, "my note", "secret note", ""),
			wantStatus:  http.StatusPreconditionRequired,
			wantCode:    errorCodePreconditionRequired,
			wantMessage: errorMessagePreconditionRequired,
		},
		{
			name:        "unquoted If-Match",
			ifMatch:     "1",
			contentType: "application/json",
			body:        updateTextRecordRequestBody(t, "my note", "secret note", ""),
			wantStatus:  http.StatusBadRequest,
			wantCode:    errorCodeInvalidRequest,
			wantMessage: errorMessageInvalidRecordRequest,
		},
		{
			name:        "weak If-Match",
			ifMatch:     `W/"1"`,
			contentType: "application/json",
			body:        updateTextRecordRequestBody(t, "my note", "secret note", ""),
			wantStatus:  http.StatusBadRequest,
			wantCode:    errorCodeInvalidRequest,
			wantMessage: errorMessageInvalidRecordRequest,
		},
		{
			name:        "zero If-Match revision",
			ifMatch:     `"0"`,
			contentType: "application/json",
			body:        updateTextRecordRequestBody(t, "my note", "secret note", ""),
			wantStatus:  http.StatusBadRequest,
			wantCode:    errorCodeInvalidRequest,
			wantMessage: errorMessageInvalidRecordRequest,
		},
		{
			name:        "missing Content-Type",
			ifMatch:     `"1"`,
			body:        updateTextRecordRequestBody(t, "my note", "secret note", ""),
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
			body: updateTextRecordRequestBody(t, "my note", "secret note", "") +
				updateTextRecordRequestBody(t, "my note", "secret note", ""),
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
			request := httptest.NewRequest(http.MethodPut, "/api/v1/records/"+testRecordID, strings.NewReader(tt.body))
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

func TestUpdateRecordHandler_RejectsOversizedBody(t *testing.T) {
	records := recordManagerStub{
		update: func(context.Context, service.UpdateRecordRequest) (model.Record, error) {
			t.Fatal("record service must not be called")
			return model.Record{}, nil
		},
	}
	body := `{"type":"text","title":"my note","payload":{"text":"` +
		strings.Repeat("a", int(maxRequestBodySize)) +
		`"}}`
	request := newUpdateRecordRequest(t, testRecordID, body)
	request.Header.Set("If-Match", `"1"`)
	response := httptest.NewRecorder()

	serveAuthenticatedRecordHandler(t, updateRecordHandler(records), response, request)

	assertErrorResponse(
		t,
		response,
		http.StatusRequestEntityTooLarge,
		errorCodePayloadTooLarge,
		errorMessagePayloadTooLarge,
	)
}

func TestUpdateRecordHandler_MapsServiceError(t *testing.T) {
	records := recordManagerStub{
		update: func(context.Context, service.UpdateRecordRequest) (model.Record, error) {
			return model.Record{}, model.ErrRecordRevisionConflict
		},
	}
	request := newUpdateRecordRequest(t, testRecordID, updateTextRecordRequestBody(t, "my note", "secret", ""))
	request.Header.Set("If-Match", `"1"`)
	response := httptest.NewRecorder()

	serveAuthenticatedRecordHandler(t, updateRecordHandler(records), response, request)

	assertErrorResponse(
		t,
		response,
		http.StatusConflict,
		errorCodeRevisionConflict,
		errorMessageRevisionConflict,
	)
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
	tests := []struct {
		name        string
		ifMatch     string
		wantStatus  int
		wantCode    string
		wantMessage string
	}{
		{
			name:        "missing If-Match",
			wantStatus:  http.StatusPreconditionRequired,
			wantCode:    errorCodePreconditionRequired,
			wantMessage: errorMessagePreconditionRequired,
		},
		{
			name:        "unquoted If-Match",
			ifMatch:     "1",
			wantStatus:  http.StatusBadRequest,
			wantCode:    errorCodeInvalidRequest,
			wantMessage: errorMessageInvalidRecordRequest,
		},
		{
			name:        "zero If-Match revision",
			ifMatch:     `"0"`,
			wantStatus:  http.StatusBadRequest,
			wantCode:    errorCodeInvalidRequest,
			wantMessage: errorMessageInvalidRecordRequest,
		},
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

			assertErrorResponse(t, response, tt.wantStatus, tt.wantCode, tt.wantMessage)
		})
	}
}

func TestDeleteRecordHandler_MapsServiceError(t *testing.T) {
	records := recordManagerStub{
		delete: func(context.Context, service.DeleteRecordRequest) error {
			return model.ErrRecordRevisionConflict
		},
	}
	request := newDeleteRecordRequest(testRecordID)
	request.Header.Set("If-Match", `"1"`)
	response := httptest.NewRecorder()

	serveAuthenticatedRecordHandler(t, deleteRecordHandler(records), response, request)

	assertErrorResponse(
		t,
		response,
		http.StatusConflict,
		errorCodeRevisionConflict,
		errorMessageRevisionConflict,
	)
}

func TestWriteRecordError(t *testing.T) {
	internalError := errors.New("database connection details")
	tests := []struct {
		name        string
		err         error
		wantStatus  int
		wantCode    string
		wantMessage string
	}{
		{
			name:        "payload too large",
			err:         fmt.Errorf("create record: %w", model.ErrPayloadTooLarge),
			wantStatus:  http.StatusRequestEntityTooLarge,
			wantCode:    errorCodePayloadTooLarge,
			wantMessage: errorMessagePayloadTooLarge,
		},
		{
			name:        "record not found",
			err:         fmt.Errorf("get record: %w", model.ErrRecordNotFound),
			wantStatus:  http.StatusNotFound,
			wantCode:    errorCodeRecordNotFound,
			wantMessage: errorMessageRecordNotFound,
		},
		{
			name:        "revision conflict",
			err:         fmt.Errorf("update record: %w", model.ErrRecordRevisionConflict),
			wantStatus:  http.StatusConflict,
			wantCode:    errorCodeRevisionConflict,
			wantMessage: errorMessageRevisionConflict,
		},
		{
			name:        "precondition required",
			err:         fmt.Errorf("update record: %w", model.ErrRecordPreconditionRequired),
			wantStatus:  http.StatusPreconditionRequired,
			wantCode:    errorCodePreconditionRequired,
			wantMessage: errorMessagePreconditionRequired,
		},
		{
			name:        "invalid record ID",
			err:         model.ErrInvalidRecordID,
			wantStatus:  http.StatusBadRequest,
			wantCode:    errorCodeInvalidRequest,
			wantMessage: errorMessageInvalidRecordRequest,
		},
		{
			name:        "invalid record revision",
			err:         model.ErrInvalidRecordRevision,
			wantStatus:  http.StatusBadRequest,
			wantCode:    errorCodeInvalidRequest,
			wantMessage: errorMessageInvalidRecordRequest,
		},
		{
			name:        "invalid record title",
			err:         model.ErrInvalidRecordTitle,
			wantStatus:  http.StatusBadRequest,
			wantCode:    errorCodeInvalidRequest,
			wantMessage: errorMessageInvalidRecordRequest,
		},
		{
			name:        "invalid text payload",
			err:         model.ErrInvalidTextPayload,
			wantStatus:  http.StatusBadRequest,
			wantCode:    errorCodeInvalidRequest,
			wantMessage: errorMessageInvalidRecordRequest,
		},
		{
			name:        "invalid credentials payload",
			err:         model.ErrInvalidCredentialsPayload,
			wantStatus:  http.StatusBadRequest,
			wantCode:    errorCodeInvalidRequest,
			wantMessage: errorMessageInvalidRecordRequest,
		},
		{
			name:        "invalid card payload",
			err:         model.ErrInvalidCardPayload,
			wantStatus:  http.StatusBadRequest,
			wantCode:    errorCodeInvalidRequest,
			wantMessage: errorMessageInvalidRecordRequest,
		},
		{
			name:        "invalid binary payload",
			err:         model.ErrInvalidBinaryPayload,
			wantStatus:  http.StatusBadRequest,
			wantCode:    errorCodeInvalidRequest,
			wantMessage: errorMessageInvalidRecordRequest,
		},
		{
			name:        "unsupported record type",
			err:         model.ErrRecordTypeUnsupported,
			wantStatus:  http.StatusBadRequest,
			wantCode:    errorCodeInvalidRequest,
			wantMessage: errorMessageInvalidRecordRequest,
		},
		{
			name:        "internal error",
			err:         internalError,
			wantStatus:  http.StatusInternalServerError,
			wantCode:    errorCodeInternal,
			wantMessage: errorMessageInternal,
		},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			response := httptest.NewRecorder()

			writeRecordError(response, tt.err)

			assertErrorResponse(t, response, tt.wantStatus, tt.wantCode, tt.wantMessage)
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
	tests := []struct {
		name    string
		handler http.Handler
		request *http.Request
	}{
		{
			name:    "create",
			handler: createRecordHandler(records),
			request: newCreateRecordRequest(t, createTextRecordRequestBody(t, "my note", "secret", "")),
		},
		{
			name:    "list",
			handler: listRecordsHandler(records),
			request: httptest.NewRequest(http.MethodGet, "/api/v1/records", nil),
		},
		{
			name:    "get",
			handler: getRecordHandler(records),
			request: httptest.NewRequest(http.MethodGet, "/api/v1/records/"+testRecordID, nil),
		},
		{
			name:    "update",
			handler: updateRecordHandler(records),
			request: newUpdateRecordRequest(t, testRecordID, updateTextRecordRequestBody(t, "my note", "secret", "")),
		},
		{
			name:    "delete",
			handler: deleteRecordHandler(records),
			request: newDeleteRecordRequest(testRecordID),
		},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			response := httptest.NewRecorder()

			tt.handler.ServeHTTP(response, tt.request)

			assertUnauthorizedResponse(t, response)
		})
	}
}

func TestRecordHandlers_MapRecordNotFound(t *testing.T) {
	records := recordManagerStub{
		get: func(context.Context, int64, string) (model.Record, error) {
			return model.Record{}, fmt.Errorf("get record: %w", model.ErrRecordNotFound)
		},
	}
	request := httptest.NewRequest(http.MethodGet, "/api/v1/records/"+testRecordID, nil)
	request.SetPathValue("id", testRecordID)
	response := httptest.NewRecorder()

	serveAuthenticatedRecordHandler(t, getRecordHandler(records), response, request)

	assertErrorResponse(t, response, http.StatusNotFound, errorCodeRecordNotFound, errorMessageRecordNotFound)
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

func newCreateRecordRequest(t *testing.T, body string) *http.Request {
	t.Helper()

	request := httptest.NewRequest(http.MethodPost, "/api/v1/records", strings.NewReader(body))
	request.Header.Set("Content-Type", "application/json")

	return request
}

func newUpdateRecordRequest(t *testing.T, recordID string, body string) *http.Request {
	t.Helper()

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

func createTextRecordRequestBody(t *testing.T, title string, text string, metadata string) string {
	t.Helper()

	return recordRequestBody(t, model.RecordTypeText, title, model.TextPayload{
		Text:     text,
		Metadata: metadata,
	})
}

func createCredentialsRecordRequestBody(
	t *testing.T,
	title string,
	login string,
	password string,
	url string,
	metadata string,
) string {
	t.Helper()

	return recordRequestBody(t, model.RecordTypeCredentials, title, model.CredentialsPayload{
		Login:    login,
		Password: password,
		URL:      url,
		Metadata: metadata,
	})
}

func createCardRecordRequestBody(t *testing.T, title string, payload model.CardPayload) string {
	t.Helper()

	return recordRequestBody(t, model.RecordTypeCard, title, payload)
}

func createBinaryRecordRequestBody(t *testing.T, title string, payload model.BinaryPayload) string {
	t.Helper()

	return recordRequestBody(t, model.RecordTypeBinary, title, payload)
}

func updateTextRecordRequestBody(t *testing.T, title string, text string, metadata string) string {
	t.Helper()

	return createTextRecordRequestBody(t, title, text, metadata)
}

func updateCredentialsRecordRequestBody(
	t *testing.T,
	title string,
	login string,
	password string,
	url string,
	metadata string,
) string {
	t.Helper()

	return createCredentialsRecordRequestBody(t, title, login, password, url, metadata)
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

func assertRecordMetadataResponse(t *testing.T, got recordMetadataResponse, want recordMetadataResponse) {
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

func requireTextPayload(t *testing.T, payload model.RecordPayload) model.TextPayload {
	t.Helper()

	value, ok := payload.(*model.TextPayload)
	if !ok || value == nil {
		t.Fatalf("payload type = %T, want non-nil *TextPayload", payload)
	}

	return *value
}

func requireCredentialsPayload(t *testing.T, payload model.RecordPayload) model.CredentialsPayload {
	t.Helper()

	value, ok := payload.(*model.CredentialsPayload)
	if !ok || value == nil {
		t.Fatalf("payload type = %T, want non-nil *CredentialsPayload", payload)
	}

	return *value
}

func requireCardPayload(t *testing.T, payload model.RecordPayload) model.CardPayload {
	t.Helper()

	value, ok := payload.(*model.CardPayload)
	if !ok || value == nil {
		t.Fatalf("payload type = %T, want non-nil *CardPayload", payload)
	}

	return *value
}

func requireBinaryPayload(t *testing.T, payload model.RecordPayload) model.BinaryPayload {
	t.Helper()

	value, ok := payload.(*model.BinaryPayload)
	if !ok || value == nil {
		t.Fatalf("payload type = %T, want non-nil *BinaryPayload", payload)
	}

	return *value
}

func assertTextRecordResponse(t *testing.T, got textRecordResponse, want textRecordResponse) {
	t.Helper()

	assertRecordMetadataResponse(t, recordMetadataResponse{
		ID:        got.ID,
		Type:      got.Type,
		Title:     got.Title,
		Revision:  got.Revision,
		CreatedAt: got.CreatedAt,
		UpdatedAt: got.UpdatedAt,
	}, recordMetadataResponse{
		ID:        want.ID,
		Type:      want.Type,
		Title:     want.Title,
		Revision:  want.Revision,
		CreatedAt: want.CreatedAt,
		UpdatedAt: want.UpdatedAt,
	})
	if got.Payload.Text != want.Payload.Text {
		t.Errorf("response payload.text = %q, want %q", got.Payload.Text, want.Payload.Text)
	}
	if got.Payload.Metadata != want.Payload.Metadata {
		t.Errorf("response payload.metadata = %q, want %q", got.Payload.Metadata, want.Payload.Metadata)
	}
}

func assertCredentialsRecordResponse(
	t *testing.T,
	got credentialsRecordResponse,
	want credentialsRecordResponse,
) {
	t.Helper()

	assertRecordMetadataResponse(t, recordMetadataResponse{
		ID:        got.ID,
		Type:      got.Type,
		Title:     got.Title,
		Revision:  got.Revision,
		CreatedAt: got.CreatedAt,
		UpdatedAt: got.UpdatedAt,
	}, recordMetadataResponse{
		ID:        want.ID,
		Type:      want.Type,
		Title:     want.Title,
		Revision:  want.Revision,
		CreatedAt: want.CreatedAt,
		UpdatedAt: want.UpdatedAt,
	})
	if got.Payload != want.Payload {
		t.Errorf("response payload = %+v, want %+v", got.Payload, want.Payload)
	}
}
