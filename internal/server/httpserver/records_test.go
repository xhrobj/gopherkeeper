package httpserver

import (
	"context"
	"encoding/json"
	"errors"
	"fmt"
	"net/http"
	"net/http/httptest"
	"strings"
	"testing"
	"time"

	"github.com/xhrobj/gopherkeeper/internal/model"
	"github.com/xhrobj/gopherkeeper/internal/server/middleware"
	"github.com/xhrobj/gopherkeeper/internal/server/service"
)

const testRecordID = "7b4c2d7d-0e2f-4c4b-8d4b-8f4f7c4d3a21"

type recordManagerStub struct {
	createText func(context.Context, service.CreateTextRecordRequest) (service.TextRecord, error)
	list       func(context.Context, int64) ([]model.RecordMetadata, error)
	getText    func(context.Context, int64, string) (service.TextRecord, error)
}

func (s recordManagerStub) CreateText(
	ctx context.Context,
	request service.CreateTextRecordRequest,
) (service.TextRecord, error) {
	return s.createText(ctx, request)
}

func (s recordManagerStub) List(ctx context.Context, userID int64) ([]model.RecordMetadata, error) {
	return s.list(ctx, userID)
}

func (s recordManagerStub) GetText(ctx context.Context, userID int64, recordID string) (service.TextRecord, error) {
	return s.getText(ctx, userID, recordID)
}

func TestCreateRecordHandler_CreatesTextRecord(t *testing.T) {
	createdAt := time.Date(2026, time.July, 8, 12, 0, 0, 0, time.UTC)
	updatedAt := time.Date(2026, time.July, 8, 12, 1, 0, 0, time.UTC)
	var gotRequest service.CreateTextRecordRequest
	records := recordManagerStub{
		createText: func(_ context.Context, request service.CreateTextRecordRequest) (service.TextRecord, error) {
			gotRequest = request

			return service.TextRecord{
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
		t.Errorf("CreateText() userID = %d, want 42", gotRequest.UserID)
	}
	if gotRequest.Title != "my note" {
		t.Errorf("CreateText() title = %q, want my note", gotRequest.Title)
	}
	if gotRequest.Payload.Text != "secret note" {
		t.Errorf("CreateText() payload text = %q, want secret note", gotRequest.Payload.Text)
	}
	if gotRequest.Payload.Metadata != "private metadata" {
		t.Errorf("CreateText() metadata = %q, want private metadata", gotRequest.Payload.Metadata)
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
			name:        "unknown field",
			contentType: "application/json",
			body:        `{"type":"text","title":"my note","payload":{"text":"secret"},"extra":42}`,
			wantStatus:  http.StatusBadRequest,
			wantCode:    errorCodeInvalidRequest,
			wantMessage: errorMessageInvalidRecordRequest,
		},
		{
			name:        "unsupported record type",
			contentType: "application/json",
			body:        `{"type":"card","title":"my note","payload":{"text":"secret"}}`,
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
				createText: func(context.Context, service.CreateTextRecordRequest) (service.TextRecord, error) {
					t.Fatal("record service must not be called")
					return service.TextRecord{}, nil
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
		createText: func(context.Context, service.CreateTextRecordRequest) (service.TextRecord, error) {
			t.Fatal("record service must not be called")
			return service.TextRecord{}, nil
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

func TestCreateRecordHandler_MapsServiceErrors(t *testing.T) {
	internalError := errors.New("database connection details")
	tests := []struct {
		name        string
		serviceErr  error
		wantStatus  int
		wantCode    string
		wantMessage string
	}{
		{
			name:        "invalid title",
			serviceErr:  model.ErrInvalidRecordTitle,
			wantStatus:  http.StatusBadRequest,
			wantCode:    errorCodeInvalidRequest,
			wantMessage: errorMessageInvalidRecordRequest,
		},
		{
			name:        "payload too large",
			serviceErr:  model.ErrPayloadTooLarge,
			wantStatus:  http.StatusRequestEntityTooLarge,
			wantCode:    errorCodePayloadTooLarge,
			wantMessage: errorMessagePayloadTooLarge,
		},
		{
			name:        "internal error",
			serviceErr:  internalError,
			wantStatus:  http.StatusInternalServerError,
			wantCode:    errorCodeInternal,
			wantMessage: errorMessageInternal,
		},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			records := recordManagerStub{
				createText: func(context.Context, service.CreateTextRecordRequest) (service.TextRecord, error) {
					return service.TextRecord{}, tt.serviceErr
				},
			}
			request := newCreateRecordRequest(t, createTextRecordRequestBody(t, "my note", "secret", ""))
			response := httptest.NewRecorder()

			serveAuthenticatedRecordHandler(t, createRecordHandler(records), response, request)

			assertErrorResponse(t, response, tt.wantStatus, tt.wantCode, tt.wantMessage)
			if strings.Contains(response.Body.String(), internalError.Error()) {
				t.Error("response body contains internal error details")
			}
		})
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
	var gotUserID int64
	var gotRecordID string
	records := recordManagerStub{
		getText: func(_ context.Context, userID int64, recordID string) (service.TextRecord, error) {
			gotUserID = userID
			gotRecordID = recordID

			return service.TextRecord{
				Metadata: model.RecordMetadata{
					ID:        testRecordID,
					Type:      model.RecordTypeText,
					Title:     "my note",
					Revision:  model.RecordInitialRevision,
					CreatedAt: createdAt,
					UpdatedAt: updatedAt,
				},
				Payload: model.TextPayload{
					Text:     "secret note",
					Metadata: "private metadata",
				},
			}, nil
		},
	}
	request := httptest.NewRequest(http.MethodGet, "/api/v1/records/"+testRecordID, nil)
	request.SetPathValue("id", testRecordID)
	response := httptest.NewRecorder()

	serveAuthenticatedRecordHandler(t, getRecordHandler(records), response, request)

	if gotUserID != 42 {
		t.Errorf("GetText() userID = %d, want 42", gotUserID)
	}
	if gotRecordID != testRecordID {
		t.Errorf("GetText() recordID = %q, want %q", gotRecordID, testRecordID)
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
		Payload: model.TextPayload{
			Text:     "secret note",
			Metadata: "private metadata",
		},
	})
}

func TestRecordHandlers_RequireAuthentication(t *testing.T) {
	records := recordManagerStub{
		createText: func(context.Context, service.CreateTextRecordRequest) (service.TextRecord, error) {
			t.Fatal("record service must not be called")
			return service.TextRecord{}, nil
		},
		list: func(context.Context, int64) ([]model.RecordMetadata, error) {
			t.Fatal("record service must not be called")
			return nil, nil
		},
		getText: func(context.Context, int64, string) (service.TextRecord, error) {
			t.Fatal("record service must not be called")
			return service.TextRecord{}, nil
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
		getText: func(context.Context, int64, string) (service.TextRecord, error) {
			return service.TextRecord{}, fmt.Errorf("get record: %w", model.ErrRecordNotFound)
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

func createTextRecordRequestBody(t *testing.T, title string, text string, metadata string) string {
	t.Helper()

	body, err := json.Marshal(createRecordRequest{
		Type:  model.RecordTypeText,
		Title: title,
		Payload: model.TextPayload{
			Text:     text,
			Metadata: metadata,
		},
	})
	if err != nil {
		t.Fatalf("encode create record request: %v", err)
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
