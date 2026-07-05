package httpserver

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
	"github.com/xhrobj/gopherkeeper/internal/server/service"
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
			handler := NewHandler(
				databasePingerFunc(func(context.Context) error {
					return tt.pingErr
				}),
				unusedUserRegisterer(t),
				unusedUserAuthenticator(t),
				unusedTokenValidator(t),
				unusedCurrentUserReader(t),
			)

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

func TestHealthHandler_RejectsUnsupportedMethod(t *testing.T) {
	handler := NewHandler(
		databasePingerFunc(func(context.Context) error {
			return nil
		}),
		unusedUserRegisterer(t),
		unusedUserAuthenticator(t),
		unusedTokenValidator(t),
		unusedCurrentUserReader(t),
	)

	request := httptest.NewRequest(http.MethodPost, "/health", nil)
	response := httptest.NewRecorder()

	handler.ServeHTTP(response, request)

	if response.Code != http.StatusMethodNotAllowed {
		t.Errorf("status code = %d, want %d", response.Code, http.StatusMethodNotAllowed)
	}
}

func TestNewHandler_RoutesRegistration(t *testing.T) {
	registrarCalled := false
	handler := NewHandler(
		databasePingerFunc(func(context.Context) error {
			return nil
		}),
		userRegistererFunc(func(
			context.Context,
			string,
			string,
		) (model.User, error) {
			registrarCalled = true

			return model.User{
				ID:        42,
				Login:     "alice",
				CreatedAt: time.Date(2026, time.July, 1, 12, 0, 0, 0, time.UTC),
			}, nil
		}),
		unusedUserAuthenticator(t),
		unusedTokenValidator(t),
		unusedCurrentUserReader(t),
	)

	request := newRegistrationRequest(t, registrationRequestBody(t, "alice"))
	response := httptest.NewRecorder()

	handler.ServeHTTP(response, request)

	if !registrarCalled {
		t.Fatal("registration service was not called")
	}
	if response.Code != http.StatusCreated {
		t.Errorf("status code = %d, want %d", response.Code, http.StatusCreated)
	}
}

func TestNewHandler_RoutesLogin(t *testing.T) {
	authenticatorCalled := false
	handler := NewHandler(
		databasePingerFunc(func(context.Context) error {
			return nil
		}),
		unusedUserRegisterer(t),
		userAuthenticatorFunc(func(
			context.Context,
			string,
			string,
		) (service.AuthenticationResult, error) {
			authenticatorCalled = true

			return service.AuthenticationResult{
				AccessToken: "access-token",
				ExpiresAt:   time.Date(2026, time.July, 4, 12, 15, 0, 0, time.UTC),
				User: model.User{
					ID:        42,
					Login:     "alice",
					CreatedAt: time.Date(2026, time.July, 4, 12, 0, 0, 0, time.UTC),
				},
			}, nil
		}),
		unusedTokenValidator(t),
		unusedCurrentUserReader(t),
	)

	request := newLoginRequest(t, loginRequestBody(t, "alice"))
	response := httptest.NewRecorder()

	handler.ServeHTTP(response, request)

	if !authenticatorCalled {
		t.Fatal("authentication service was not called")
	}
	if response.Code != http.StatusOK {
		t.Errorf("status code = %d, want %d", response.Code, http.StatusOK)
	}
}

func TestNewHandler_RoutesCurrentUser(t *testing.T) {
	validatorCalled := false
	readerCalled := false

	handler := NewHandler(
		databasePingerFunc(func(context.Context) error {
			return nil
		}),
		unusedUserRegisterer(t),
		unusedUserAuthenticator(t),
		tokenValidatorFunc(func(_ context.Context, token string) (int64, error) {
			validatorCalled = true
			if token != "valid-token" {
				t.Fatalf("Validate() token = %q, want valid-token", token)
			}

			return 42, nil
		}),
		currentUserReaderFunc(func(_ context.Context, id int64) (model.User, error) {
			readerCalled = true
			if id != 42 {
				t.Fatalf("FindByID() id = %d, want 42", id)
			}

			return model.User{
				ID:        42,
				Login:     "alice",
				CreatedAt: time.Date(2026, time.July, 4, 12, 0, 0, 0, time.UTC),
			}, nil
		}),
	)

	request := httptest.NewRequest(http.MethodGet, "/api/v1/users/me", nil)
	request.Header.Set("Authorization", authorizationSchemeBearer+" valid-token")
	response := httptest.NewRecorder()

	handler.ServeHTTP(response, request)

	if !validatorCalled {
		t.Fatal("token validator was not called")
	}
	if !readerCalled {
		t.Fatal("current user reader was not called")
	}
	if response.Code != http.StatusOK {
		t.Errorf("status code = %d, want %d", response.Code, http.StatusOK)
	}
}

func unusedUserRegisterer(t *testing.T) UserRegisterer {
	t.Helper()

	return userRegistererFunc(func(
		context.Context,
		string,
		string,
	) (model.User, error) {
		t.Fatal("registration service must not be called")
		return model.User{}, nil
	})
}

func unusedUserAuthenticator(t *testing.T) UserAuthenticator {
	t.Helper()

	return userAuthenticatorFunc(func(
		context.Context,
		string,
		string,
	) (service.AuthenticationResult, error) {
		t.Fatal("authentication service must not be called")
		return service.AuthenticationResult{}, nil
	})
}

func unusedTokenValidator(t *testing.T) TokenValidator {
	t.Helper()

	return tokenValidatorFunc(func(context.Context, string) (int64, error) {
		t.Fatal("token validator must not be called")
		return 0, nil
	})
}

func unusedCurrentUserReader(t *testing.T) CurrentUserReader {
	t.Helper()

	return currentUserReaderFunc(func(context.Context, int64) (model.User, error) {
		t.Fatal("current user reader must not be called")
		return model.User{}, nil
	})
}
