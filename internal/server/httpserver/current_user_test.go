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
	"github.com/xhrobj/gopherkeeper/internal/server/middleware"
)

type currentUserReaderFunc func(context.Context, int64) (model.User, error)

func (f currentUserReaderFunc) FindByID(ctx context.Context, id int64) (model.User, error) {
	return f(ctx, id)
}

func TestCurrentUserHandler_ReturnsCurrentUser(t *testing.T) {
	createdAt := time.Date(2026, time.July, 4, 12, 0, 0, 0, time.UTC)
	var gotID int64
	reader := currentUserReaderFunc(func(_ context.Context, id int64) (model.User, error) {
		gotID = id

		return model.User{
			ID:        id,
			Login:     "alice",
			CreatedAt: createdAt,
		}, nil
	})
	request := httptest.NewRequest(http.MethodGet, "/api/v1/users/me", nil)
	response := httptest.NewRecorder()

	serveAuthenticatedCurrentUserHandler(t, reader, response, request)

	if gotID != 42 {
		t.Errorf("FindByID() id = %d, want 42", gotID)
	}
	if response.Code != http.StatusOK {
		t.Errorf("status code = %d, want %d", response.Code, http.StatusOK)
	}
	assertUserResponse(t, response, userResponse{
		ID:        42,
		Login:     "alice",
		CreatedAt: createdAt,
	})
}

func TestCurrentUserHandler_RejectsMissingUserID(t *testing.T) {
	reader := currentUserReaderFunc(func(context.Context, int64) (model.User, error) {
		t.Fatal("user reader must not be called")
		return model.User{}, nil
	})
	request := httptest.NewRequest(http.MethodGet, "/api/v1/users/me", nil)
	response := httptest.NewRecorder()

	currentUserHandler(reader).ServeHTTP(response, request)

	assertUnauthorizedResponse(t, response)
}

func TestCurrentUserHandler_MapsReaderErrors(t *testing.T) {
	internalError := errors.New("database connection details")
	tests := []struct {
		name        string
		readerErr   error
		wantStatus  int
		wantCode    string
		wantMessage string
	}{
		{
			name:        "user not found",
			readerErr:   model.ErrUserNotFound,
			wantStatus:  http.StatusUnauthorized,
			wantCode:    "unauthorized",
			wantMessage: "missing or invalid bearer token",
		},
		{
			name:        "internal error",
			readerErr:   internalError,
			wantStatus:  http.StatusInternalServerError,
			wantCode:    errorCodeInternal,
			wantMessage: errorMessageInternal,
		},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			reader := currentUserReaderFunc(func(context.Context, int64) (model.User, error) {
				return model.User{}, tt.readerErr
			})
			request := httptest.NewRequest(http.MethodGet, "/api/v1/users/me", nil)
			response := httptest.NewRecorder()

			serveAuthenticatedCurrentUserHandler(t, reader, response, request)

			assertErrorResponse(t, response, tt.wantStatus, tt.wantCode, tt.wantMessage)
			if strings.Contains(response.Body.String(), internalError.Error()) {
				t.Error("response body contains internal error details")
			}
		})
	}
}

func serveAuthenticatedCurrentUserHandler(
	t *testing.T,
	reader CurrentUserReader,
	response *httptest.ResponseRecorder,
	request *http.Request,
) {
	t.Helper()

	validator := tokenValidatorFunc(func(_ context.Context, token string) (int64, error) {
		if token != "valid-token" {
			t.Fatalf("Validate() token = %q, want valid-token", token)
		}

		return 42, nil
	})
	request.Header.Set("Authorization", "Bearer valid-token")

	middleware.WithAuthentication(currentUserHandler(reader), validator).ServeHTTP(response, request)
}

func assertUnauthorizedResponse(t *testing.T, response *httptest.ResponseRecorder) {
	t.Helper()

	if authenticate := response.Header().Get("WWW-Authenticate"); authenticate != "Bearer" {
		t.Errorf("WWW-Authenticate = %q, want Bearer", authenticate)
	}
	assertErrorResponse(
		t,
		response,
		http.StatusUnauthorized,
		"unauthorized",
		"missing or invalid bearer token",
	)
}

func assertUserResponse(t *testing.T, response *httptest.ResponseRecorder, want userResponse) {
	t.Helper()

	if contentType := response.Header().Get("Content-Type"); contentType != "application/json" {
		t.Errorf("Content-Type = %q, want application/json", contentType)
	}

	var body userResponse
	if err := json.Unmarshal(response.Body.Bytes(), &body); err != nil {
		t.Fatalf("decode user response: %v", err)
	}

	if body.ID != want.ID {
		t.Errorf("response id = %d, want %d", body.ID, want.ID)
	}
	if body.Login != want.Login {
		t.Errorf("response login = %q, want %q", body.Login, want.Login)
	}
	if !body.CreatedAt.Equal(want.CreatedAt) {
		t.Errorf("response created_at = %s, want %s", body.CreatedAt, want.CreatedAt)
	}
}
