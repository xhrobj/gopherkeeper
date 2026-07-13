package httpclient

import (
	"context"
	"encoding/json"
	"errors"
	"net/http"
	"net/http/httptest"
	"strings"
	"testing"
	"time"

	"github.com/xhrobj/gopherkeeper/internal/model"
)

func TestClient_CurrentUser(t *testing.T) {
	createdAt := time.Date(2026, time.July, 4, 12, 0, 0, 0, time.UTC)
	server := newSuccessfulCurrentUserServer(t, createdAt)
	defer server.Close()

	client, err := New(serverAddress(server), writeServerCertificate(t, server))
	if err != nil {
		t.Fatalf("New() error = %v", err)
	}

	user, err := client.CurrentUser(context.Background(), "test.jwt.token")
	if err != nil {
		t.Fatalf("CurrentUser() error = %v", err)
	}

	if user.ID != 42 {
		t.Errorf("user ID = %d, want 42", user.ID)
	}
	if user.Login != "alice" {
		t.Errorf("user login = %q, want alice", user.Login)
	}
	if !user.CreatedAt.Equal(createdAt) {
		t.Errorf("user created at = %s, want %s", user.CreatedAt, createdAt)
	}
}

func newSuccessfulCurrentUserServer(t *testing.T, createdAt time.Time) *httptest.Server {
	t.Helper()

	return httptest.NewTLSServer(http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		if !assertCurrentUserRequest(t, r) {
			http.Error(w, "invalid current user request", http.StatusBadRequest)
			return
		}

		w.Header().Set("Content-Type", "application/json")
		w.WriteHeader(http.StatusOK)
		if err := json.NewEncoder(w).Encode(userResponse{
			ID:        42,
			Login:     "alice",
			CreatedAt: createdAt,
		}); err != nil {
			t.Errorf("encode response: %v", err)
		}
	}))
}

func assertCurrentUserRequest(t *testing.T, r *http.Request) bool {
	t.Helper()

	valid := true
	if r.Method != http.MethodGet {
		t.Errorf("method = %s, want %s", r.Method, http.MethodGet)
		valid = false
	}
	if r.URL.Path != currentUserPath {
		t.Errorf("path = %s, want %s", r.URL.Path, currentUserPath)
		valid = false
	}
	if got := r.Header.Get("Authorization"); got != "Bearer test.jwt.token" {
		t.Errorf("Authorization = %q, want bearer token", got)
		valid = false
	}

	return valid
}

func TestClient_CurrentUserReturnsAPIError(t *testing.T) {
	server := httptest.NewTLSServer(http.HandlerFunc(func(w http.ResponseWriter, _ *http.Request) {
		w.Header().Set("Content-Type", "application/json")
		w.WriteHeader(http.StatusUnauthorized)
		_, _ = w.Write([]byte(`{
			"code":"unauthorized",
			"message":"missing or invalid bearer token"
		}`))
	}))
	defer server.Close()

	client, err := New(serverAddress(server), writeServerCertificate(t, server))
	if err != nil {
		t.Fatalf("New() error = %v", err)
	}

	_, err = client.CurrentUser(context.Background(), "test.jwt.token")
	if err == nil {
		t.Fatal("CurrentUser() error = nil, want API error")
	}

	var apiError *APIError
	if !errors.As(err, &apiError) {
		t.Fatalf("CurrentUser() error = %T, want *APIError", err)
	}

	if apiError.StatusCode != http.StatusUnauthorized {
		t.Errorf("status code = %d, want %d", apiError.StatusCode, http.StatusUnauthorized)
	}
	if apiError.Code != "unauthorized" {
		t.Errorf("code = %q, want unauthorized", apiError.Code)
	}
	if !errors.Is(err, model.ErrUnauthorized) {
		t.Errorf("CurrentUser() error = %v, want ErrUnauthorized", err)
	}
	if strings.Contains(err.Error(), "test.jwt.token") {
		t.Error("current user error contains access token")
	}
}
