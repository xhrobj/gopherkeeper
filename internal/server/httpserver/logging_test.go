package httpserver

import (
	"net/http"
	"net/http/httptest"
	"testing"

	"go.uber.org/zap"
	"go.uber.org/zap/zapcore"
	"go.uber.org/zap/zaptest/observer"
)

func TestWithLogging(t *testing.T) {
	const responseBody = `{"status":"test"}`

	tests := []struct {
		name          string
		status        int
		observerLevel zapcore.Level
		wantLevel     zapcore.Level
	}{
		{
			name:          "successful request",
			status:        http.StatusOK,
			observerLevel: zap.DebugLevel,
			wantLevel:     zap.DebugLevel,
		},
		{
			name:          "server error",
			status:        http.StatusServiceUnavailable,
			observerLevel: zap.InfoLevel,
			wantLevel:     zap.ErrorLevel,
		},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			core, logs := observer.New(tt.observerLevel)
			lg := zap.New(core)

			handler := WithLogging(
				http.HandlerFunc(func(
					w http.ResponseWriter,
					_ *http.Request,
				) {
					w.WriteHeader(tt.status)

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

			assertHTTPLogEntry(
				t,
				logs.All(),
				tt.wantLevel,
				tt.status,
				len(responseBody),
			)
		})
	}
}

func assertHTTPLogEntry(
	t *testing.T,
	entries []observer.LoggedEntry,
	wantLevel zapcore.Level,
	wantStatus int,
	wantSize int,
) {
	t.Helper()

	if len(entries) != 1 {
		t.Fatalf("log entries = %d, want 1", len(entries))
	}

	entry := entries[0]
	if entry.Message != "http request" {
		t.Errorf("log message = %q, want %q", entry.Message, "http request")
	}

	if entry.Level != wantLevel {
		t.Errorf("log level = %s, want %s", entry.Level, wantLevel)
	}

	fields := entry.ContextMap()
	if fields["method"] != http.MethodGet {
		t.Errorf("method = %v, want %s", fields["method"], http.MethodGet)
	}

	if fields["path"] != "/health" {
		t.Errorf("path = %v, want /health", fields["path"])
	}

	if _, ok := fields["uri"]; ok {
		t.Error("uri field must not be logged")
	}

	if fields["status"] != int64(wantStatus) {
		t.Errorf("status = %v, want %d", fields["status"], wantStatus)
	}

	if fields["size"] != int64(wantSize) {
		t.Errorf("size = %v, want %d", fields["size"], wantSize)
	}

	if _, ok := fields["duration"]; !ok {
		t.Error("duration field is missing")
	}
}
