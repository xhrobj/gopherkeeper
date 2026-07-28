package httpserver

import (
	"context"
	"errors"
	"fmt"
	"net/http"
	"net/http/httptest"
	"strings"
	"testing"

	"github.com/xhrobj/gopherkeeper/internal/model"
	"github.com/xhrobj/gopherkeeper/internal/server/service"
)

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
				errorCodeRequestTooLarge,
				errorMessageRequestTooLarge,
			)
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
	decryptionError := fmt.Errorf("get record: %w", model.ErrRecordDecryptionFailed)
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
	recordDecryption := errorResponseExpectation{
		status:  http.StatusInternalServerError,
		code:    errorCodeRecordDecryption,
		message: errorMessageRecordDecryption,
	}
	preconditionRequired := errorResponseExpectation{
		status:  http.StatusPreconditionRequired,
		code:    errorCodePreconditionRequired,
		message: errorMessagePreconditionRequired,
	}
	invalidRecordData := errorResponseExpectation{
		status:  http.StatusBadRequest,
		code:    errorCodeInvalidRecordData,
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
		{name: "record decryption", err: decryptionError, want: recordDecryption},
		{name: "precondition required", err: preconditionRequiredError, want: preconditionRequired},
		{name: "invalid record title", err: model.ErrInvalidRecordTitle, want: invalidRecordData},
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
