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
	"github.com/xhrobj/gopherkeeper/internal/server/httperror"
	"github.com/xhrobj/gopherkeeper/internal/server/service"
)

const testRegistrationPassword = "correct-horse-battery-staple"

type userRegistererFunc func(context.Context, string, string) (model.User, error)

func (f userRegistererFunc) Register(
	ctx context.Context,
	login string,
	password string,
) (model.User, error) {
	return f(ctx, login, password)
}

func TestRegisterHandler_CreatesUser(t *testing.T) {
	createdAt := time.Date(2026, time.July, 1, 12, 0, 0, 0, time.UTC)

	var calls int
	var gotLogin string
	var gotPassword string

	registrar := userRegistererFunc(func(
		_ context.Context,
		login string,
		password string,
	) (model.User, error) {
		calls++
		gotLogin = login
		gotPassword = password

		return model.User{
			ID:        42,
			Login:     "alice",
			CreatedAt: createdAt,
		}, nil
	})

	request := httptest.NewRequest(
		http.MethodPost,
		"/api/v1/auth/register",
		strings.NewReader(`{"login":" Alice ","password":"`+testRegistrationPassword+`"}`),
	)
	request.Header.Set("Content-Type", "application/json; charset=utf-8")
	response := httptest.NewRecorder()

	registerHandler(registrar).ServeHTTP(response, request)

	if calls != 1 {
		t.Errorf("Register() calls = %d, want 1", calls)
	}
	if gotLogin != " Alice " {
		t.Errorf("Register() login = %q, want %q", gotLogin, " Alice ")
	}
	if gotPassword != testRegistrationPassword {
		t.Error("Register() password differs from request password")
	}

	if response.Code != http.StatusCreated {
		t.Errorf("status code = %d, want %d", response.Code, http.StatusCreated)
	}
	if contentType := response.Header().Get("Content-Type"); contentType != "application/json" {
		t.Errorf("Content-Type = %q, want application/json", contentType)
	}

	var body userResponse
	if err := json.Unmarshal(response.Body.Bytes(), &body); err != nil {
		t.Fatalf("decode response body: %v", err)
	}

	if body.ID != 42 {
		t.Errorf("response id = %d, want 42", body.ID)
	}
	if body.Login != "alice" {
		t.Errorf("response login = %q, want alice", body.Login)
	}
	if !body.CreatedAt.Equal(createdAt) {
		t.Errorf("response created_at = %s, want %s", body.CreatedAt, createdAt)
	}
	if strings.Contains(response.Body.String(), testRegistrationPassword) {
		t.Error("response body contains password")
	}
}

func TestRegisterHandler_MapsServiceErrors(t *testing.T) {
	internalError := errors.New("database connection details")

	tests := []struct {
		name        string
		serviceErr  error
		wantStatus  int
		wantCode    string
		wantMessage string
	}{
		{
			name:        "invalid login",
			serviceErr:  service.ErrInvalidLogin,
			wantStatus:  http.StatusBadRequest,
			wantCode:    errorCodeInvalidRequest,
			wantMessage: errorMessageInvalidRequest,
		},
		{
			name:        "invalid password",
			serviceErr:  service.ErrInvalidPassword,
			wantStatus:  http.StatusBadRequest,
			wantCode:    errorCodeInvalidRequest,
			wantMessage: errorMessageInvalidRequest,
		},
		{
			name:        "password too short",
			serviceErr:  service.ErrPasswordTooShort,
			wantStatus:  http.StatusBadRequest,
			wantCode:    errorCodeInvalidRequest,
			wantMessage: errorMessageInvalidRequest,
		},
		{
			name:        "password too long",
			serviceErr:  service.ErrPasswordTooLong,
			wantStatus:  http.StatusBadRequest,
			wantCode:    errorCodeInvalidRequest,
			wantMessage: errorMessageInvalidRequest,
		},
		{
			name:        "login already exists",
			serviceErr:  fmt.Errorf("repository: %w", model.ErrLoginAlreadyExists),
			wantStatus:  http.StatusConflict,
			wantCode:    errorCodeLoginAlreadyExists,
			wantMessage: errorMessageLoginAlreadyExists,
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
			registrar := userRegistererFunc(func(
				context.Context,
				string,
				string,
			) (model.User, error) {
				return model.User{}, tt.serviceErr
			})

			request := newRegistrationRequest(t, registrationRequestBody(t, "eve"))
			response := httptest.NewRecorder()

			registerHandler(registrar).ServeHTTP(response, request)

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

func TestRegisterHandler_RejectsInvalidRequest(t *testing.T) {
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
			body:        registrationRequestBody(t, "eve"),
			wantStatus:  http.StatusUnsupportedMediaType,
			wantCode:    errorCodeUnsupportedMediaType,
			wantMessage: errorMessageUnsupportedMediaType,
		},
		{
			name:        "unsupported Content-Type",
			contentType: "text/plain",
			body:        registrationRequestBody(t, "eve"),
			wantStatus:  http.StatusUnsupportedMediaType,
			wantCode:    errorCodeUnsupportedMediaType,
			wantMessage: errorMessageUnsupportedMediaType,
		},
		{
			name:        "empty body",
			contentType: "application/json",
			wantStatus:  http.StatusBadRequest,
			wantCode:    errorCodeInvalidRequest,
			wantMessage: errorMessageInvalidRequest,
		},
		{
			name:        "malformed JSON",
			contentType: "application/json",
			body:        `{"login":"eve"`,
			wantStatus:  http.StatusBadRequest,
			wantCode:    errorCodeInvalidRequest,
			wantMessage: errorMessageInvalidRequest,
		},
		{
			name:        "unknown field",
			contentType: "application/json",
			body:        `{"login":"eve","password":"` + testRegistrationPassword + `","role":"admin"}`,
			wantStatus:  http.StatusBadRequest,
			wantCode:    errorCodeInvalidRequest,
			wantMessage: errorMessageInvalidRequest,
		},
		{
			name:        "multiple JSON values",
			contentType: "application/json",
			body: registrationRequestBody(t, "eve") +
				registrationRequestBody(t, "eve"),
			wantStatus:  http.StatusBadRequest,
			wantCode:    errorCodeInvalidRequest,
			wantMessage: errorMessageInvalidRequest,
		},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			registrar := userRegistererFunc(func(
				context.Context,
				string,
				string,
			) (model.User, error) {
				t.Fatal("registrar must not be called")
				return model.User{}, nil
			})

			request := httptest.NewRequest(
				http.MethodPost,
				"/api/v1/auth/register",
				strings.NewReader(tt.body),
			)
			if tt.contentType != "" {
				request.Header.Set("Content-Type", tt.contentType)
			}
			response := httptest.NewRecorder()

			registerHandler(registrar).ServeHTTP(response, request)

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

func TestRegisterHandler_RejectsOversizedBody(t *testing.T) {
	registrar := userRegistererFunc(func(
		context.Context,
		string,
		string,
	) (model.User, error) {
		t.Fatal("registrar must not be called")
		return model.User{}, nil
	})

	body := `{"login":"eve","password":"` +
		strings.Repeat("a", int(maxRequestBodySize)) +
		`"}`
	request := newRegistrationRequest(t, body)
	response := httptest.NewRecorder()

	registerHandler(registrar).ServeHTTP(response, request)

	assertErrorResponse(
		t,
		response,
		http.StatusRequestEntityTooLarge,
		errorCodePayloadTooLarge,
		errorMessagePayloadTooLarge,
	)
}

func newRegistrationRequest(t *testing.T, body string) *http.Request {
	t.Helper()

	request := httptest.NewRequest(
		http.MethodPost,
		"/api/v1/auth/register",
		strings.NewReader(body),
	)
	request.Header.Set("Content-Type", "application/json")

	return request
}

func registrationRequestBody(t *testing.T, login string) string {
	t.Helper()

	body, err := json.Marshal(registerRequest{
		Login:    login,
		Password: testRegistrationPassword,
	})
	if err != nil {
		t.Fatalf("encode registration request: %v", err)
	}

	return string(body)
}

func assertErrorResponse(
	t *testing.T,
	response *httptest.ResponseRecorder,
	wantStatus int,
	wantCode string,
	wantMessage string,
) {
	t.Helper()

	if response.Code != wantStatus {
		t.Errorf("status code = %d, want %d", response.Code, wantStatus)
	}
	if contentType := response.Header().Get("Content-Type"); contentType != "application/json" {
		t.Errorf("Content-Type = %q, want application/json", contentType)
	}

	var body httperror.Response
	if err := json.Unmarshal(response.Body.Bytes(), &body); err != nil {
		t.Fatalf("decode error response: %v", err)
	}

	if body.Code != wantCode {
		t.Errorf("error code = %q, want %q", body.Code, wantCode)
	}
	if body.Message != wantMessage {
		t.Errorf("error message = %q, want %q", body.Message, wantMessage)
	}
	if strings.Contains(response.Body.String(), testRegistrationPassword) {
		t.Error("error response contains password")
	}
}
