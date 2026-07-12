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
)

const testRegistrationPassword = "correct-horse-battery-staple"

func TestClient_Register(t *testing.T) {
	createdAt := time.Date(2026, time.July, 1, 12, 0, 0, 0, time.UTC)
	server := newSuccessfulRegistrationServer(t, createdAt)
	defer server.Close()

	client, err := New(serverAddress(server), writeServerCertificate(t, server))
	if err != nil {
		t.Fatalf("New() error = %v", err)
	}

	user, err := client.Register(context.Background(), " Alice ", testRegistrationPassword)
	if err != nil {
		t.Fatalf("Register() error = %v", err)
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

func newSuccessfulRegistrationServer(t *testing.T, createdAt time.Time) *httptest.Server {
	t.Helper()

	return httptest.NewTLSServer(http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		if !assertRegistrationRequest(t, r) {
			http.Error(w, "invalid registration request", http.StatusBadRequest)
			return
		}

		w.Header().Set("Content-Type", "application/json")
		w.WriteHeader(http.StatusCreated)
		if err := json.NewEncoder(w).Encode(registerResponse{
			ID:        42,
			Login:     "alice",
			CreatedAt: createdAt,
		}); err != nil {
			t.Errorf("encode response: %v", err)
		}
	}))
}

func assertRegistrationRequest(t *testing.T, r *http.Request) bool {
	t.Helper()

	valid := true
	if r.Method != http.MethodPost {
		t.Errorf("method = %s, want %s", r.Method, http.MethodPost)
		valid = false
	}
	if r.URL.Path != registerPath {
		t.Errorf("path = %s, want %s", r.URL.Path, registerPath)
		valid = false
	}
	if got := r.Header.Get("Content-Type"); got != "application/json" {
		t.Errorf("Content-Type = %q, want application/json", got)
		valid = false
	}

	var request registerRequest
	if err := json.NewDecoder(r.Body).Decode(&request); err != nil {
		t.Errorf("decode request: %v", err)
		return false
	}
	if request.Login != " Alice " {
		t.Errorf("login = %q, want %q", request.Login, " Alice ")
		valid = false
	}
	if request.Password != testRegistrationPassword {
		t.Error("password was not transferred unchanged")
		valid = false
	}

	return valid
}

func TestClient_RegisterReturnsAPIError(t *testing.T) {
	server := httptest.NewTLSServer(http.HandlerFunc(func(w http.ResponseWriter, _ *http.Request) {
		w.Header().Set("Content-Type", "application/json")
		w.WriteHeader(http.StatusConflict)
		_, _ = w.Write([]byte(`{
			"code":"login_already_exists",
			"message":"login is already registered"
		}`))
	}))
	defer server.Close()

	client, err := New(serverAddress(server), writeServerCertificate(t, server))
	if err != nil {
		t.Fatalf("New() error = %v", err)
	}

	_, err = client.Register(context.Background(), "eve", testRegistrationPassword)
	if err == nil {
		t.Fatal("Register() error = nil, want API error")
	}

	var apiError *APIError
	if !errors.As(err, &apiError) {
		t.Fatalf("Register() error = %T, want *APIError", err)
	}

	if apiError.StatusCode != http.StatusConflict {
		t.Errorf("status code = %d, want %d", apiError.StatusCode, http.StatusConflict)
	}
	if apiError.Code != "login_already_exists" {
		t.Errorf("code = %q, want login_already_exists", apiError.Code)
	}
	if apiError.Message != "login is already registered" {
		t.Errorf("message = %q, want login is already registered", apiError.Message)
	}
	if strings.Contains(err.Error(), testRegistrationPassword) {
		t.Error("registration error contains password")
	}
}
