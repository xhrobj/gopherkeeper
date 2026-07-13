package middleware

import (
	"net/http"
	"net/http/httptest"
	"testing"
)

func TestWithNoStore(t *testing.T) {
	nextCalled := false
	next := http.HandlerFunc(func(w http.ResponseWriter, _ *http.Request) {
		nextCalled = true
		w.WriteHeader(http.StatusNoContent)
	})
	request := httptest.NewRequest(http.MethodGet, "/private", nil)
	response := httptest.NewRecorder()

	WithNoStore(next).ServeHTTP(response, request)

	if !nextCalled {
		t.Fatal("next handler was not called")
	}
	if response.Code != http.StatusNoContent {
		t.Errorf("status code = %d, want %d", response.Code, http.StatusNoContent)
	}
	if got := response.Header().Get("Cache-Control"); got != "no-store" {
		t.Errorf("Cache-Control = %q, want no-store", got)
	}
	if got := response.Header().Get("Pragma"); got != "no-cache" {
		t.Errorf("Pragma = %q, want no-cache", got)
	}
}
