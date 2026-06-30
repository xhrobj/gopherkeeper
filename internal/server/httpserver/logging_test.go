package httpserver

import (
	"net/http"
	"net/http/httptest"
	"testing"

	"go.uber.org/zap"
	"go.uber.org/zap/zaptest/observer"
)

func TestWithLogging(t *testing.T) {
	const responseBody = `{"status":"unavailable"}`

	core, logs := observer.New(zap.InfoLevel)
	lg := zap.New(core)

	handler := WithLogging(
		http.HandlerFunc(func(w http.ResponseWriter, _ *http.Request) {
			w.WriteHeader(http.StatusServiceUnavailable)

			if _, err := w.Write([]byte(responseBody)); err != nil {
				t.Fatalf("write response: %v", err)
			}
		}),
		lg,
	)

	request := httptest.NewRequest(
		http.MethodGet,
		"/health?details=true",
		nil,
	)
	response := httptest.NewRecorder()

	handler.ServeHTTP(response, request)

	entries := logs.All()
	if len(entries) != 1 {
		t.Fatalf("log entries = %d, want 1", len(entries))
	}

	if entries[0].Message != "http request" {
		t.Errorf(
			"log message = %q, want %q",
			entries[0].Message,
			"http request",
		)
	}

	fields := entries[0].ContextMap()

	if fields["method"] != http.MethodGet {
		t.Errorf("method = %v, want %s", fields["method"], http.MethodGet)
	}

	if fields["uri"] != "/health?details=true" {
		t.Errorf(
			"uri = %v, want /health?details=true",
			fields["uri"],
		)
	}

	if fields["status"] != int64(http.StatusServiceUnavailable) {
		t.Errorf(
			"status = %v, want %d",
			fields["status"],
			http.StatusServiceUnavailable,
		)
	}

	if fields["size"] != int64(len(responseBody)) {
		t.Errorf(
			"size = %v, want %d",
			fields["size"],
			len(responseBody),
		)
	}

	if _, ok := fields["duration"]; !ok {
		t.Error("duration field is missing")
	}
}
