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
	"github.com/xhrobj/gopherkeeper/internal/server/service"
)

const testLoginPassword = "correct-horse-battery-staple"

type userAuthenticatorFunc func(context.Context, string, string) (service.AuthenticationResult, error)

func (f userAuthenticatorFunc) Authenticate(
	ctx context.Context,
	login string,
	password string,
) (service.AuthenticationResult, error) {
	return f(ctx, login, password)
}

func TestLoginHandler_AuthenticatesUser(t *testing.T) {
	createdAt := time.Date(2026, time.July, 4, 12, 0, 0, 0, time.UTC)
	expiresAt := time.Date(2026, time.July, 4, 12, 15, 0, 0, time.UTC)

	var calls int
	var gotLogin string
	var gotPassword string

	authenticator := userAuthenticatorFunc(func(
		_ context.Context,
		login string,
		password string,
	) (service.AuthenticationResult, error) {
		calls++
		gotLogin = login
		gotPassword = password

		return service.AuthenticationResult{
			AccessToken: "access-token",
			ExpiresAt:   expiresAt,
			User: model.User{
				ID:        42,
				Login:     "alice",
				CreatedAt: createdAt,
			},
		}, nil
	})

	request := newLoginRequest(t, loginRequestBody(t, " Alice "))
	response := httptest.NewRecorder()

	loginHandler(authenticator).ServeHTTP(response, request)

	if calls != 1 {
		t.Errorf("Authenticate() calls = %d, want 1", calls)
	}
	if gotLogin != " Alice " {
		t.Errorf("Authenticate() login = %q, want %q", gotLogin, " Alice ")
	}
	if gotPassword != testLoginPassword {
		t.Error("Authenticate() password differs from request password")
	}

	if response.Code != http.StatusOK {
		t.Errorf("status code = %d, want %d", response.Code, http.StatusOK)
	}
	if contentType := response.Header().Get("Content-Type"); contentType != "application/json" {
		t.Errorf("Content-Type = %q, want application/json", contentType)
	}
	if cacheControl := response.Header().Get("Cache-Control"); cacheControl != "no-store" {
		t.Errorf("Cache-Control = %q, want no-store", cacheControl)
	}
	if pragma := response.Header().Get("Pragma"); pragma != "no-cache" {
		t.Errorf("Pragma = %q, want no-cache", pragma)
	}

	var body loginResponse
	if err := json.Unmarshal(response.Body.Bytes(), &body); err != nil {
		t.Fatalf("decode response body: %v", err)
	}

	if body.AccessToken != "access-token" {
		t.Errorf("response access_token = %q, want access-token", body.AccessToken)
	}
	if body.TokenType != tokenTypeBearer {
		t.Errorf("response token_type = %q, want %q", body.TokenType, tokenTypeBearer)
	}
	if !body.ExpiresAt.Equal(expiresAt) {
		t.Errorf("response expires_at = %s, want %s", body.ExpiresAt, expiresAt)
	}
	if body.User.ID != 42 {
		t.Errorf("response user.id = %d, want 42", body.User.ID)
	}
	if body.User.Login != "alice" {
		t.Errorf("response user.login = %q, want alice", body.User.Login)
	}
	if !body.User.CreatedAt.Equal(createdAt) {
		t.Errorf("response user.created_at = %s, want %s", body.User.CreatedAt, createdAt)
	}
	if strings.Contains(response.Body.String(), testLoginPassword) {
		t.Error("response body contains password")
	}
}

func TestLoginHandler_MapsServiceErrors(t *testing.T) {
	internalError := errors.New("database connection details")

	tests := []struct {
		name        string
		serviceErr  error
		wantStatus  int
		wantCode    string
		wantMessage string
	}{
		{
			name:        "invalid credentials",
			serviceErr:  service.ErrInvalidCredentials,
			wantStatus:  http.StatusUnauthorized,
			wantCode:    errorCodeInvalidCredentials,
			wantMessage: errorMessageInvalidCredentials,
		},
		{
			name:        "wrapped invalid credentials",
			serviceErr:  fmt.Errorf("authenticate: %w", service.ErrInvalidCredentials),
			wantStatus:  http.StatusUnauthorized,
			wantCode:    errorCodeInvalidCredentials,
			wantMessage: errorMessageInvalidCredentials,
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
			authenticator := userAuthenticatorFunc(func(
				context.Context,
				string,
				string,
			) (service.AuthenticationResult, error) {
				return service.AuthenticationResult{}, tt.serviceErr
			})

			request := newLoginRequest(t, loginRequestBody(t, "eve"))
			response := httptest.NewRecorder()

			loginHandler(authenticator).ServeHTTP(response, request)

			assertErrorResponse(
				t,
				response,
				tt.wantStatus,
				tt.wantCode,
				tt.wantMessage,
			)
			if strings.Contains(response.Body.String(), internalError.Error()) {
				t.Error("response body contains internal error details")
			}
		})
	}
}

func TestLoginHandler_RejectsInvalidRequest(t *testing.T) {
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
			body:        loginRequestBody(t, "eve"),
			wantStatus:  http.StatusUnsupportedMediaType,
			wantCode:    errorCodeUnsupportedMediaType,
			wantMessage: errorMessageUnsupportedMediaType,
		},
		{
			name:        "unsupported Content-Type",
			contentType: "text/plain",
			body:        loginRequestBody(t, "eve"),
			wantStatus:  http.StatusUnsupportedMediaType,
			wantCode:    errorCodeUnsupportedMediaType,
			wantMessage: errorMessageUnsupportedMediaType,
		},
		{
			name:        "empty body",
			contentType: "application/json",
			wantStatus:  http.StatusBadRequest,
			wantCode:    errorCodeInvalidRequest,
			wantMessage: errorMessageInvalidLoginRequest,
		},
		{
			name:        "malformed JSON",
			contentType: "application/json",
			body:        `{"login":"eve"`,
			wantStatus:  http.StatusBadRequest,
			wantCode:    errorCodeInvalidRequest,
			wantMessage: errorMessageInvalidLoginRequest,
		},
		{
			name:        "unknown field",
			contentType: "application/json",
			body:        `{"login":"eve","password":"` + testLoginPassword + `","role":"admin"}`,
			wantStatus:  http.StatusBadRequest,
			wantCode:    errorCodeInvalidRequest,
			wantMessage: errorMessageInvalidLoginRequest,
		},
		{
			name:        "multiple JSON values",
			contentType: "application/json",
			body: loginRequestBody(t, "eve") +
				loginRequestBody(t, "eve"),
			wantStatus:  http.StatusBadRequest,
			wantCode:    errorCodeInvalidRequest,
			wantMessage: errorMessageInvalidLoginRequest,
		},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			authenticator := userAuthenticatorFunc(func(
				context.Context,
				string,
				string,
			) (service.AuthenticationResult, error) {
				t.Fatal("authenticator must not be called")
				return service.AuthenticationResult{}, nil
			})

			request := httptest.NewRequest(
				http.MethodPost,
				"/api/v1/auth/login",
				strings.NewReader(tt.body),
			)
			if tt.contentType != "" {
				request.Header.Set("Content-Type", tt.contentType)
			}
			response := httptest.NewRecorder()

			loginHandler(authenticator).ServeHTTP(response, request)

			assertErrorResponse(
				t,
				response,
				tt.wantStatus,
				tt.wantCode,
				tt.wantMessage,
			)
		})
	}
}

func TestLoginHandler_RejectsOversizedBody(t *testing.T) {
	authenticator := userAuthenticatorFunc(func(
		context.Context,
		string,
		string,
	) (service.AuthenticationResult, error) {
		t.Fatal("authenticator must not be called")
		return service.AuthenticationResult{}, nil
	})

	body := `{"login":"eve","password":"` +
		strings.Repeat("a", int(maxRequestBodySize)) +
		`"}`
	request := newLoginRequest(t, body)
	response := httptest.NewRecorder()

	loginHandler(authenticator).ServeHTTP(response, request)

	assertErrorResponse(
		t,
		response,
		http.StatusRequestEntityTooLarge,
		errorCodePayloadTooLarge,
		errorMessagePayloadTooLarge,
	)
}

func newLoginRequest(t *testing.T, body string) *http.Request {
	t.Helper()

	request := httptest.NewRequest(
		http.MethodPost,
		"/api/v1/auth/login",
		strings.NewReader(body),
	)
	request.Header.Set("Content-Type", "application/json")

	return request
}

func loginRequestBody(t *testing.T, login string) string {
	t.Helper()

	body, err := json.Marshal(loginRequest{
		Login:    login,
		Password: testLoginPassword,
	})
	if err != nil {
		t.Fatalf("encode login request: %v", err)
	}

	return string(body)
}
