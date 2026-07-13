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

const testLoginPassword = "correct-horse-battery-staple"

func TestClient_Login(t *testing.T) {
	createdAt := time.Date(2026, time.July, 4, 12, 0, 0, 0, time.UTC)
	expiresAt := time.Date(2026, time.July, 4, 12, 15, 0, 0, time.UTC)
	server := newSuccessfulLoginServer(t, createdAt, expiresAt)
	defer server.Close()

	client, err := New(serverAddress(server), writeServerCertificate(t, server))
	if err != nil {
		t.Fatalf("New() error = %v", err)
	}

	result, err := client.Login(context.Background(), " Alice ", testLoginPassword)
	if err != nil {
		t.Fatalf("Login() error = %v", err)
	}

	if result.AccessToken != "test.jwt.token" {
		t.Errorf("access token = %q, want test.jwt.token", result.AccessToken)
	}
	if !result.ExpiresAt.Equal(expiresAt) {
		t.Errorf("expires at = %s, want %s", result.ExpiresAt, expiresAt)
	}
	if result.User.ID != 42 {
		t.Errorf("user ID = %d, want 42", result.User.ID)
	}
	if result.User.Login != "alice" {
		t.Errorf("user login = %q, want alice", result.User.Login)
	}
	if !result.User.CreatedAt.Equal(createdAt) {
		t.Errorf("user created at = %s, want %s", result.User.CreatedAt, createdAt)
	}
}

func newSuccessfulLoginServer(t *testing.T, createdAt time.Time, expiresAt time.Time) *httptest.Server {
	t.Helper()

	return httptest.NewTLSServer(http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		if !assertLoginRequest(t, r) {
			http.Error(w, "invalid login request", http.StatusBadRequest)
			return
		}

		w.Header().Set("Content-Type", "application/json")
		w.WriteHeader(http.StatusOK)
		if err := json.NewEncoder(w).Encode(loginResponse{
			AccessToken: "test.jwt.token",
			ExpiresAt:   expiresAt,
			User: userResponse{
				ID:        42,
				Login:     "alice",
				CreatedAt: createdAt,
			},
		}); err != nil {
			t.Errorf("encode response: %v", err)
		}
	}))
}

func assertLoginRequest(t *testing.T, r *http.Request) bool {
	t.Helper()

	valid := true
	if r.Method != http.MethodPost {
		t.Errorf("method = %s, want %s", r.Method, http.MethodPost)
		valid = false
	}
	if r.URL.Path != loginPath {
		t.Errorf("path = %s, want %s", r.URL.Path, loginPath)
		valid = false
	}
	if got := r.Header.Get("Content-Type"); got != "application/json" {
		t.Errorf("Content-Type = %q, want application/json", got)
		valid = false
	}

	var request loginRequest
	if err := json.NewDecoder(r.Body).Decode(&request); err != nil {
		t.Errorf("decode request: %v", err)
		return false
	}
	if request.Login != " Alice " {
		t.Errorf("login = %q, want %q", request.Login, " Alice ")
		valid = false
	}
	if request.Password != testLoginPassword {
		t.Error("password was not transferred unchanged")
		valid = false
	}

	return valid
}

func TestClient_LoginReturnsAPIError(t *testing.T) {
	server := httptest.NewTLSServer(http.HandlerFunc(func(w http.ResponseWriter, _ *http.Request) {
		w.Header().Set("Content-Type", "application/json")
		w.WriteHeader(http.StatusUnauthorized)
		_, _ = w.Write([]byte(`{
			"code":"invalid_credentials",
			"message":"invalid login or password"
		}`))
	}))
	defer server.Close()

	client, err := New(serverAddress(server), writeServerCertificate(t, server))
	if err != nil {
		t.Fatalf("New() error = %v", err)
	}

	_, err = client.Login(context.Background(), "eve", testLoginPassword)
	if err == nil {
		t.Fatal("Login() error = nil, want API error")
	}

	var apiError *APIError
	if !errors.As(err, &apiError) {
		t.Fatalf("Login() error = %T, want *APIError", err)
	}

	if apiError.StatusCode != http.StatusUnauthorized {
		t.Errorf("status code = %d, want %d", apiError.StatusCode, http.StatusUnauthorized)
	}
	if apiError.Code != "invalid_credentials" {
		t.Errorf("code = %q, want invalid_credentials", apiError.Code)
	}
	if apiError.Message != "invalid login or password" {
		t.Errorf("message = %q, want invalid login or password", apiError.Message)
	}
	if !errors.Is(err, model.ErrInvalidCredentials) {
		t.Errorf("Login() error = %v, want ErrInvalidCredentials", err)
	}
	if strings.Contains(err.Error(), testLoginPassword) {
		t.Error("login error contains password")
	}
}
