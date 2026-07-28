package httpserver

import (
	"context"
	"net/http"
	"net/http/httptest"
	"testing"

	"github.com/xhrobj/gopherkeeper/internal/server/service"
)

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
