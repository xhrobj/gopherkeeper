package httpserver

import (
	"context"
	"encoding/json"
	"errors"
	"net/http"
	"net/http/httptest"
	"strings"
	"testing"
)

type databasePingerFunc func(context.Context) error

func (f databasePingerFunc) Ping(ctx context.Context) error {
	return f(ctx)
}

func TestHealthHandler(t *testing.T) {
	tests := []struct {
		name       string
		pingErr    error
		wantCode   int
		wantStatus string
	}{
		{
			name:       "healthy",
			wantCode:   http.StatusOK,
			wantStatus: healthStatusOK,
		},
		{
			name:       "unavailable",
			pingErr:    errors.New("database connection failed"),
			wantCode:   http.StatusServiceUnavailable,
			wantStatus: healthStatusUnavailable,
		},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			handler := NewHandler(databasePingerFunc(func(context.Context) error {
				return tt.pingErr
			}))

			request := httptest.NewRequest(http.MethodGet, "/health", nil)
			response := httptest.NewRecorder()

			handler.ServeHTTP(response, request)

			assertHealthResponse(t, response, tt.wantCode, tt.wantStatus)
		})
	}
}

func assertHealthResponse(
	t *testing.T,
	response *httptest.ResponseRecorder,
	wantCode int,
	wantStatus string,
) {
	t.Helper()

	if response.Code != wantCode {
		t.Errorf("status code = %d, want %d", response.Code, wantCode)
	}

	if contentType := response.Header().Get("Content-Type"); contentType != "application/json" {
		t.Errorf("Content-Type = %q, want application/json", contentType)
	}

	bodyBytes := response.Body.Bytes()

	var body healthResponse
	if err := json.Unmarshal(bodyBytes, &body); err != nil {
		t.Fatalf("decode response body: %v", err)
	}

	if body.Status != wantStatus {
		t.Errorf("response status = %q, want %q", body.Status, wantStatus)
	}

	if strings.Contains(string(bodyBytes), "database connection failed") {
		t.Error("response body contains internal database error")
	}
}

func TestHealthHandlerRejectsUnsupportedMethod(t *testing.T) {
	handler := NewHandler(databasePingerFunc(func(context.Context) error {
		return nil
	}))

	request := httptest.NewRequest(http.MethodPost, "/health", nil)
	response := httptest.NewRecorder()

	handler.ServeHTTP(response, request)

	if response.Code != http.StatusMethodNotAllowed {
		t.Errorf("status code = %d, want %d", response.Code, http.StatusMethodNotAllowed)
	}
}
