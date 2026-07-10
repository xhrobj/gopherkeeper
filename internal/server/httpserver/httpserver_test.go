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
	"github.com/xhrobj/gopherkeeper/internal/server/service"
)

type databasePingerFunc func(context.Context) error

func (f databasePingerFunc) Ping(ctx context.Context) error {
	return f(ctx)
}

type tokenValidatorFunc func(context.Context, string) (int64, error)

func (f tokenValidatorFunc) Validate(ctx context.Context, token string) (int64, error) {
	return f(ctx, token)
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
			deps := newTestDependencies(t)
			deps.Database = databasePingerFunc(func(context.Context) error {
				return tt.pingErr
			})
			handler := NewHandler(deps)

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
	handler := NewHandler(newTestDependencies(t))

	request := httptest.NewRequest(http.MethodPost, "/health", nil)
	response := httptest.NewRecorder()

	handler.ServeHTTP(response, request)

	if response.Code != http.StatusMethodNotAllowed {
		t.Errorf("status code = %d, want %d", response.Code, http.StatusMethodNotAllowed)
	}
}

func TestNewHandler_RoutesRegistration(t *testing.T) {
	registrarCalled := false
	deps := newTestDependencies(t)
	deps.Registerer = userRegistererFunc(func(
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
	})
	handler := NewHandler(deps)

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
	deps := newTestDependencies(t)
	deps.Authenticator = userAuthenticatorFunc(func(
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
	})
	handler := NewHandler(deps)

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
	deps := newTestDependencies(t)
	deps.TokenValidator = tokenValidatorFunc(func(_ context.Context, token string) (int64, error) {
		validatorCalled = true
		if token != "valid-token" {
			t.Fatalf("Validate() token = %q, want valid-token", token)
		}

		return 42, nil
	})
	deps.CurrentUserReader = currentUserReaderFunc(func(_ context.Context, id int64) (model.User, error) {
		readerCalled = true
		if id != 42 {
			t.Fatalf("FindByID() id = %d, want 42", id)
		}

		return model.User{
			ID:        42,
			Login:     "alice",
			CreatedAt: time.Date(2026, time.July, 4, 12, 0, 0, 0, time.UTC),
		}, nil
	})
	handler := NewHandler(deps)

	request := httptest.NewRequest(http.MethodGet, "/api/v1/users/me", nil)
	request.Header.Set("Authorization", "Bearer valid-token")
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

func TestNewHandler_RoutesRecords(t *testing.T) {
	createdAt := time.Date(2026, time.July, 8, 12, 0, 0, 0, time.UTC)
	tests := []struct {
		name       string
		request    *http.Request
		wantStatus int
	}{
		{
			name:       "create",
			request:    newCreateRecordRequest(t, createTextRecordRequestBody(t, "my note", "secret", "")),
			wantStatus: http.StatusCreated,
		},
		{
			name:       "list",
			request:    httptest.NewRequest(http.MethodGet, "/api/v1/records", nil),
			wantStatus: http.StatusOK,
		},
		{
			name:       "get",
			request:    httptest.NewRequest(http.MethodGet, "/api/v1/records/"+testRecordID, nil),
			wantStatus: http.StatusOK,
		},
		{
			name:       "update",
			request:    newUpdateRecordRequest(t, testRecordID, updateTextRecordRequestBody(t, "new note", "new secret", "")),
			wantStatus: http.StatusOK,
		},
		{
			name:       "delete",
			request:    newDeleteRecordRequest(testRecordID),
			wantStatus: http.StatusNoContent,
		},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			deps := newTestDependencies(t)
			deps.TokenValidator = tokenValidatorFunc(func(_ context.Context, token string) (int64, error) {
				if token != "valid-token" {
					t.Fatalf("Validate() token = %q, want valid-token", token)
				}

				return 42, nil
			})
			tt.request.Header.Set("If-Match", `"1"`)
			deps.Records = recordManagerStub{
				createText: func(_ context.Context, request service.CreateTextRecordRequest) (service.TextRecord, error) {
					return service.TextRecord{
						Metadata: model.RecordMetadata{
							ID:        testRecordID,
							Type:      model.RecordTypeText,
							Title:     request.Title,
							Revision:  model.RecordInitialRevision,
							CreatedAt: createdAt,
							UpdatedAt: createdAt,
						},
						Payload: request.Payload,
					}, nil
				},
				list: func(context.Context, int64) ([]model.RecordMetadata, error) {
					return []model.RecordMetadata{{
						ID:        testRecordID,
						Type:      model.RecordTypeText,
						Title:     "my note",
						Revision:  model.RecordInitialRevision,
						CreatedAt: createdAt,
						UpdatedAt: createdAt,
					}}, nil
				},
				get: func(context.Context, int64, string) (service.DecryptedRecord, error) {
					payload := model.TextPayload{Text: "secret"}
					return service.DecryptedRecord{
						Metadata: model.RecordMetadata{
							ID:        testRecordID,
							Type:      model.RecordTypeText,
							Title:     "my note",
							Revision:  model.RecordInitialRevision,
							CreatedAt: createdAt,
							UpdatedAt: createdAt,
						},
						Text: &payload,
					}, nil
				},
				updateText: func(_ context.Context, request service.UpdateTextRecordRequest) (service.TextRecord, error) {
					return service.TextRecord{
						Metadata: model.RecordMetadata{
							ID:        request.RecordID,
							Type:      model.RecordTypeText,
							Title:     request.Title,
							Revision:  2,
							CreatedAt: createdAt,
							UpdatedAt: createdAt,
						},
						Payload: request.Payload,
					}, nil
				},
				delete: func(context.Context, service.DeleteRecordRequest) error {
					return nil
				},
			}
			handler := NewHandler(deps)
			tt.request.Header.Set("Authorization", "Bearer valid-token")
			response := httptest.NewRecorder()

			handler.ServeHTTP(response, tt.request)

			if response.Code != tt.wantStatus {
				t.Errorf("status code = %d, want %d", response.Code, tt.wantStatus)
			}
		})
	}
}

func newTestDependencies(t *testing.T) Dependencies {
	t.Helper()

	return Dependencies{
		Database: databasePingerFunc(func(context.Context) error {
			return nil
		}),
		Registerer:        unusedUserRegisterer(t),
		Authenticator:     unusedUserAuthenticator(t),
		TokenValidator:    unusedTokenValidator(t),
		CurrentUserReader: unusedCurrentUserReader(t),
		Records:           unusedRecordManager(t),
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

func unusedTokenValidator(t *testing.T) middleware.TokenValidator {
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

func unusedRecordManager(t *testing.T) RecordManager {
	t.Helper()

	return recordManagerStub{
		createText: func(context.Context, service.CreateTextRecordRequest) (service.TextRecord, error) {
			t.Fatal("record manager must not be called")
			return service.TextRecord{}, nil
		},
		createCredentials: func(
			context.Context,
			service.CreateCredentialsRecordRequest,
		) (service.CredentialsRecord, error) {
			t.Fatal("record manager must not be called")
			return service.CredentialsRecord{}, nil
		},
		list: func(context.Context, int64) ([]model.RecordMetadata, error) {
			t.Fatal("record manager must not be called")
			return nil, nil
		},
		get: func(context.Context, int64, string) (service.DecryptedRecord, error) {
			t.Fatal("record manager must not be called")
			return service.DecryptedRecord{}, nil
		},
		updateText: func(context.Context, service.UpdateTextRecordRequest) (service.TextRecord, error) {
			t.Fatal("record manager must not be called")
			return service.TextRecord{}, nil
		},
		updateCredentials: func(
			context.Context,
			service.UpdateCredentialsRecordRequest,
		) (service.CredentialsRecord, error) {
			t.Fatal("record manager must not be called")
			return service.CredentialsRecord{}, nil
		},
		delete: func(context.Context, service.DeleteRecordRequest) error {
			t.Fatal("record manager must not be called")
			return nil
		},
	}
}
