package middleware

import (
	"context"
	"encoding/json"
	"errors"
	"net/http"
	"net/http/httptest"
	"testing"

	serverauth "github.com/xhrobj/gopherkeeper/internal/server/auth"
	"github.com/xhrobj/gopherkeeper/internal/server/httperror"
)

type tokenValidatorFunc func(context.Context, string) (int64, error)

func (f tokenValidatorFunc) Validate(ctx context.Context, token string) (int64, error) {
	return f(ctx, token)
}

func TestWithAuthentication_AddsUserIDToContext(t *testing.T) {
	const token = "valid-token"
	var gotToken string

	validator := tokenValidatorFunc(func(_ context.Context, value string) (int64, error) {
		gotToken = value
		return 42, nil
	})
	next := http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		userID, ok := UserIDFromContext(r.Context())
		if !ok {
			t.Fatal("UserIDFromContext() ok = false, want true")
		}
		if userID != 42 {
			t.Fatalf("UserIDFromContext() userID = %d, want 42", userID)
		}

		w.WriteHeader(http.StatusNoContent)
	})
	request := httptest.NewRequest(http.MethodGet, "/protected", nil)
	request.Header.Set("Authorization", authorizationSchemeBearer+" "+token)
	response := httptest.NewRecorder()

	WithAuthentication(next, validator).ServeHTTP(response, request)

	if gotToken != token {
		t.Errorf("Validate() token = %q, want %q", gotToken, token)
	}
	if response.Code != http.StatusNoContent {
		t.Errorf("status code = %d, want %d", response.Code, http.StatusNoContent)
	}
}

func TestWithAuthentication_AcceptsCaseInsensitiveBearerScheme(t *testing.T) {
	validator := tokenValidatorFunc(func(context.Context, string) (int64, error) {
		return 42, nil
	})
	nextCalled := false
	next := http.HandlerFunc(func(w http.ResponseWriter, _ *http.Request) {
		nextCalled = true
		w.WriteHeader(http.StatusNoContent)
	})
	request := httptest.NewRequest(http.MethodGet, "/protected", nil)
	request.Header.Set("Authorization", "bearer valid-token")
	response := httptest.NewRecorder()

	WithAuthentication(next, validator).ServeHTTP(response, request)

	if !nextCalled {
		t.Fatal("next handler was not called")
	}
	if response.Code != http.StatusNoContent {
		t.Errorf("status code = %d, want %d", response.Code, http.StatusNoContent)
	}
}

func TestWithAuthentication_RejectsInvalidAuthorization(t *testing.T) {
	tests := []struct {
		name        string
		headerValue string
	}{
		{
			name: "missing Authorization",
		},
		{
			name:        "empty Authorization",
			headerValue: " ",
		},
		{
			name:        "missing token",
			headerValue: authorizationSchemeBearer,
		},
		{
			name:        "wrong scheme",
			headerValue: "Basic valid-token",
		},
		{
			name:        "extra fields",
			headerValue: authorizationSchemeBearer + " valid-token extra",
		},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			validator := tokenValidatorFunc(func(context.Context, string) (int64, error) {
				t.Fatal("validator must not be called")
				return 0, nil
			})
			next := http.HandlerFunc(func(http.ResponseWriter, *http.Request) {
				t.Fatal("next handler must not be called")
			})
			request := httptest.NewRequest(http.MethodGet, "/protected", nil)
			if tt.headerValue != "" {
				request.Header.Set("Authorization", tt.headerValue)
			}
			response := httptest.NewRecorder()

			WithAuthentication(next, validator).ServeHTTP(response, request)

			assertUnauthorizedResponse(t, response)
		})
	}
}

func TestWithAuthentication_RejectsInvalidToken(t *testing.T) {
	validator := tokenValidatorFunc(func(context.Context, string) (int64, error) {
		return 0, serverauth.ErrInvalidToken
	})
	next := http.HandlerFunc(func(http.ResponseWriter, *http.Request) {
		t.Fatal("next handler must not be called")
	})
	request := httptest.NewRequest(http.MethodGet, "/protected", nil)
	request.Header.Set("Authorization", authorizationSchemeBearer+" invalid-token")
	response := httptest.NewRecorder()

	WithAuthentication(next, validator).ServeHTTP(response, request)

	assertUnauthorizedResponse(t, response)
}

func TestWithAuthentication_RejectsValidatorError(t *testing.T) {
	validator := tokenValidatorFunc(func(context.Context, string) (int64, error) {
		return 0, errors.New("validator failed")
	})
	next := http.HandlerFunc(func(http.ResponseWriter, *http.Request) {
		t.Fatal("next handler must not be called")
	})
	request := httptest.NewRequest(http.MethodGet, "/protected", nil)
	request.Header.Set("Authorization", authorizationSchemeBearer+" token")
	response := httptest.NewRecorder()

	WithAuthentication(next, validator).ServeHTTP(response, request)

	assertUnauthorizedResponse(t, response)
}

func TestUserIDFromContext_ReturnsFalseForMissingUserID(t *testing.T) {
	userID, ok := UserIDFromContext(context.Background())
	if ok {
		t.Fatal("UserIDFromContext() ok = true, want false")
	}
	if userID != 0 {
		t.Fatalf("UserIDFromContext() userID = %d, want 0", userID)
	}
}

func assertUnauthorizedResponse(t *testing.T, response *httptest.ResponseRecorder) {
	t.Helper()

	if authenticate := response.Header().Get("WWW-Authenticate"); authenticate != authorizationSchemeBearer {
		t.Errorf("WWW-Authenticate = %q, want %q", authenticate, authorizationSchemeBearer)
	}
	assertErrorResponse(
		t,
		response,
		http.StatusUnauthorized,
		errorCodeUnauthorized,
		errorMessageUnauthorized,
	)
}

func assertErrorResponse(
	t *testing.T,
	response *httptest.ResponseRecorder,
	wantStatus int,
	wantCode string,
	wantMessage string,
) {
	t.Helper()

	if contentType := response.Header().Get("Content-Type"); contentType != "application/json" {
		t.Errorf("Content-Type = %q, want application/json", contentType)
	}
	if response.Code != wantStatus {
		t.Errorf("status code = %d, want %d", response.Code, wantStatus)
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
}
