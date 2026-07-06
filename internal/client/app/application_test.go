package app

import (
	"context"
	"errors"
	"net/http"
	"strings"
	"testing"
	"time"

	"github.com/xhrobj/gopherkeeper/internal/client/httpclient"
	"github.com/xhrobj/gopherkeeper/internal/client/session"
	"github.com/xhrobj/gopherkeeper/internal/model"
)

const testPassword = "correct-horse-battery-staple"

type userClientStub struct {
	register func(context.Context, string, string) (model.User, error)
	login    func(context.Context, string, string) (httpclient.LoginResult, error)
	whoami   func(context.Context, string) (model.User, error)
}

func (s userClientStub) Register(ctx context.Context, login, password string) (model.User, error) {
	return s.register(ctx, login, password)
}

func (s userClientStub) Login(ctx context.Context, login, password string) (httpclient.LoginResult, error) {
	return s.login(ctx, login, password)
}

func (s userClientStub) CurrentUser(ctx context.Context, accessToken string) (model.User, error) {
	return s.whoami(ctx, accessToken)
}

type sessionStorageStub struct {
	save func(session.Session) error
	load func(string) (session.Session, error)
}

func (s sessionStorageStub) Save(stored session.Session) error {
	return s.save(stored)
}

func (s sessionStorageStub) Load(expectedServerAddress string) (session.Session, error) {
	return s.load(expectedServerAddress)
}

func testOnlineSession() session.Session {
	return session.Session{
		ServerAddress: "localhost:8080",
		AccessToken:   "test.jwt.token",
		TokenType:     "Bearer",
		ExpiresAt:     time.Date(2026, time.July, 6, 12, 15, 0, 0, time.UTC),
		User: model.User{
			ID:        42,
			Login:     "alice",
			CreatedAt: time.Date(2026, time.July, 6, 12, 0, 0, 0, time.UTC),
		},
	}
}

func TestApplication_Register(t *testing.T) {
	application := newApplication(
		userClientStub{
			register: func(_ context.Context, login, password string) (model.User, error) {
				if login != " Alice " {
					t.Errorf("login = %q, want %q", login, " Alice ")
				}
				if password != testPassword {
					t.Error("register client received unexpected password")
				}

				return model.User{Login: "alice"}, nil
			},
		},
		sessionStorageStub{},
		"localhost:8080",
	)

	user, err := application.Register(context.Background(), " Alice ", testPassword)
	if err != nil {
		t.Fatalf("Register() error = %v", err)
	}

	if user.Login != "alice" {
		t.Errorf("registered login = %q, want alice", user.Login)
	}
}

func TestApplication_Register_ReturnsReadableDuplicateError(t *testing.T) {
	apiError := &httpclient.APIError{
		StatusCode: http.StatusConflict,
		Code:       "login_already_exists",
		Message:    "login is already registered",
	}
	application := newApplication(
		userClientStub{
			register: func(context.Context, string, string) (model.User, error) {
				return model.User{}, apiError
			},
		},
		sessionStorageStub{},
		"localhost:8080",
	)

	_, err := application.Register(context.Background(), "ALICE", testPassword)
	if err == nil {
		t.Fatal("Register() error = nil, want duplicate error")
	}
	if !strings.Contains(err.Error(), `login "ALICE" is already registered`) {
		t.Errorf("error = %q, want readable duplicate message", err)
	}
	if !errors.Is(err, apiError) {
		t.Error("duplicate error does not preserve API error")
	}
	if strings.Contains(err.Error(), testPassword) {
		t.Error("duplicate error contains password")
	}
}

func TestApplication_Register_DoesNotLeakPasswordInNetworkError(t *testing.T) {
	networkError := errors.New("connection refused")
	application := newApplication(
		userClientStub{
			register: func(context.Context, string, string) (model.User, error) {
				return model.User{}, networkError
			},
		},
		sessionStorageStub{},
		"localhost:8080",
	)

	_, err := application.Register(context.Background(), "eve", testPassword)
	if err == nil {
		t.Fatal("Register() error = nil, want network error")
	}
	if !errors.Is(err, networkError) {
		t.Error("registration error does not preserve network error")
	}
	if strings.Contains(err.Error(), testPassword) {
		t.Error("network error contains password")
	}
}

func TestApplication_Register_DoesNotCreateSessionStorage(t *testing.T) {
	application := newApplicationWithSessionFactory(
		userClientStub{
			register: func(context.Context, string, string) (model.User, error) {
				return model.User{Login: "alice"}, nil
			},
		},
		func() (sessionStorage, error) {
			t.Fatal("session storage must not be created for registration")
			return nil, nil
		},
		"localhost:8080",
	)

	user, err := application.Register(context.Background(), "alice", testPassword)
	if err != nil {
		t.Fatalf("Register() error = %v", err)
	}
	if user.Login != "alice" {
		t.Errorf("registered login = %q, want alice", user.Login)
	}
}

func TestApplication_Login_ReturnsSessionStorageCreationError(t *testing.T) {
	storageError := errors.New("cache directory is unavailable")
	loginCalled := false
	application := newApplicationWithSessionFactory(
		userClientStub{
			login: func(context.Context, string, string) (httpclient.LoginResult, error) {
				loginCalled = true
				return httpclient.LoginResult{}, nil
			},
		},
		func() (sessionStorage, error) {
			return nil, storageError
		},
		"localhost:8080",
	)

	_, err := application.Login(context.Background(), "alice", testPassword)
	if err == nil {
		t.Fatal("Login() error = nil, want session storage error")
	}
	if !errors.Is(err, storageError) {
		t.Error("login error does not preserve session storage error")
	}
	if loginCalled {
		t.Error("login client was called after session storage creation error")
	}
	if strings.Contains(err.Error(), testPassword) {
		t.Error("session storage error contains password")
	}
}

func TestApplication_Login_SavesSession(t *testing.T) {
	createdAt := time.Date(2026, time.July, 5, 12, 0, 0, 0, time.UTC)
	expiresAt := time.Date(2026, time.July, 5, 12, 15, 0, 0, time.UTC)
	var savedSession session.Session
	application := newApplication(
		userClientStub{
			login: func(_ context.Context, login, password string) (httpclient.LoginResult, error) {
				if login != "alice" {
					t.Errorf("login = %q, want alice", login)
				}
				if password != testPassword {
					t.Error("login client received unexpected password")
				}

				return httpclient.LoginResult{
					AccessToken: "test.jwt.token",
					TokenType:   "Bearer",
					ExpiresAt:   expiresAt,
					User: model.User{
						ID:        42,
						Login:     "alice",
						CreatedAt: createdAt,
					},
				}, nil
			},
		},
		sessionStorageStub{
			save: func(stored session.Session) error {
				savedSession = stored
				return nil
			},
		},
		"localhost:8080",
	)

	user, err := application.Login(context.Background(), "alice", testPassword)
	if err != nil {
		t.Fatalf("Login() error = %v", err)
	}

	if user.Login != "alice" {
		t.Errorf("login result user = %q, want alice", user.Login)
	}
	wantSession := session.Session{
		ServerAddress: "localhost:8080",
		AccessToken:   "test.jwt.token",
		TokenType:     "Bearer",
		ExpiresAt:     expiresAt,
		User: model.User{
			ID:        42,
			Login:     "alice",
			CreatedAt: createdAt,
		},
	}
	if savedSession != wantSession {
		t.Errorf("saved session = %+v, want %+v", savedSession, wantSession)
	}
}

func TestApplication_Login_ReturnsReadableInvalidCredentialsError(t *testing.T) {
	apiError := &httpclient.APIError{
		StatusCode: http.StatusUnauthorized,
		Code:       "invalid_credentials",
		Message:    "invalid login or password",
	}
	application := newApplication(
		userClientStub{
			login: func(context.Context, string, string) (httpclient.LoginResult, error) {
				return httpclient.LoginResult{}, apiError
			},
		},
		sessionStorageStub{
			save: func(session.Session) error {
				t.Fatal("session must not be saved after invalid credentials")
				return nil
			},
		},
		"localhost:8080",
	)

	_, err := application.Login(context.Background(), "eve", testPassword)
	if err == nil {
		t.Fatal("Login() error = nil, want invalid credentials")
	}
	if !strings.Contains(err.Error(), "invalid login or password") {
		t.Errorf("error = %q, want readable invalid credentials message", err)
	}
	if !errors.Is(err, apiError) {
		t.Error("login error does not preserve API error")
	}
	if strings.Contains(err.Error(), testPassword) {
		t.Error("invalid credentials error contains password")
	}
}

func TestApplication_Login_DoesNotLeakPasswordInNetworkError(t *testing.T) {
	networkError := errors.New("connection refused")
	application := newApplication(
		userClientStub{
			login: func(context.Context, string, string) (httpclient.LoginResult, error) {
				return httpclient.LoginResult{}, networkError
			},
		},
		sessionStorageStub{
			save: func(session.Session) error {
				t.Fatal("session must not be saved after network error")
				return nil
			},
		},
		"localhost:8080",
	)

	_, err := application.Login(context.Background(), "eve", testPassword)
	if err == nil {
		t.Fatal("Login() error = nil, want network error")
	}
	if !errors.Is(err, networkError) {
		t.Error("login error does not preserve network error")
	}
	if strings.Contains(err.Error(), testPassword) {
		t.Error("network error contains password")
	}
}

func TestApplication_Login_DoesNotLeakTokenInSaveError(t *testing.T) {
	saveError := errors.New("permission denied")
	application := newApplication(
		userClientStub{
			login: func(context.Context, string, string) (httpclient.LoginResult, error) {
				return httpclient.LoginResult{
					AccessToken: "test.jwt.token",
					TokenType:   "Bearer",
					ExpiresAt:   time.Date(2026, time.July, 5, 12, 15, 0, 0, time.UTC),
					User: model.User{
						ID:        42,
						Login:     "eve",
						CreatedAt: time.Date(2026, time.July, 5, 12, 0, 0, 0, time.UTC),
					},
				}, nil
			},
		},
		sessionStorageStub{
			save: func(session.Session) error { return saveError },
		},
		"localhost:8080",
	)

	_, err := application.Login(context.Background(), "eve", testPassword)
	if err == nil {
		t.Fatal("Login() error = nil, want save error")
	}
	if !errors.Is(err, saveError) {
		t.Error("login error does not preserve save error")
	}
	if strings.Contains(err.Error(), "test.jwt.token") {
		t.Error("save error contains access token")
	}
}

func TestApplication_Whoami(t *testing.T) {
	currentUser := testOnlineSession().User
	application := newApplication(
		userClientStub{
			whoami: func(_ context.Context, accessToken string) (model.User, error) {
				if accessToken != "test.jwt.token" {
					t.Errorf("access token = %q, want test.jwt.token", accessToken)
				}

				return currentUser, nil
			},
		},
		sessionStorageStub{
			load: func(expectedServerAddress string) (session.Session, error) {
				if expectedServerAddress != "localhost:8080" {
					t.Errorf("expected server address = %q, want localhost:8080", expectedServerAddress)
				}

				return testOnlineSession(), nil
			},
		},
		"localhost:8080",
	)

	user, err := application.Whoami(context.Background())
	if err != nil {
		t.Fatalf("Whoami() error = %v", err)
	}

	if user.Login != "alice" {
		t.Errorf("current user = %q, want alice", user.Login)
	}
}

func TestApplication_Whoami_MapsSessionErrors(t *testing.T) {
	tests := []struct {
		name    string
		loadErr error
		want    string
	}{
		{
			name:    "not found",
			loadErr: session.ErrNotFound,
			want:    "online session not found: run gkeep login",
		},
		{
			name:    "expired",
			loadErr: session.ErrExpired,
			want:    "online session expired: run gkeep login",
		},
		{
			name:    "server mismatch",
			loadErr: session.ErrServerMismatch,
			want:    "online session belongs to another server: run gkeep login",
		},
		{
			name:    "invalid",
			loadErr: session.ErrInvalid,
			want:    "online session is invalid: run gkeep login",
		},
		{
			name:    "filesystem",
			loadErr: errors.New("permission denied"),
			want:    "load online session",
		},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			application := newApplication(
				userClientStub{
					whoami: func(context.Context, string) (model.User, error) {
						t.Fatal("current user client must not be called after session load error")
						return model.User{}, nil
					},
				},
				sessionStorageStub{
					load: func(string) (session.Session, error) {
						return session.Session{}, tt.loadErr
					},
				},
				"localhost:8080",
			)

			_, err := application.Whoami(context.Background())
			if err == nil {
				t.Fatal("Whoami() error = nil, want session error")
			}
			if !errors.Is(err, tt.loadErr) {
				t.Error("whoami error does not preserve session error")
			}
			if !strings.Contains(err.Error(), tt.want) {
				t.Errorf("error = %q, want %q", err, tt.want)
			}
			if strings.Contains(err.Error(), "test.jwt.token") {
				t.Error("session error contains access token")
			}
		})
	}
}

func TestApplication_Whoami_MapsUnauthorizedAPIError(t *testing.T) {
	apiError := &httpclient.APIError{
		StatusCode: http.StatusUnauthorized,
		Code:       "unauthorized",
		Message:    "missing or invalid bearer token",
	}
	application := newApplication(
		userClientStub{
			whoami: func(context.Context, string) (model.User, error) {
				return model.User{}, apiError
			},
		},
		sessionStorageStub{
			load: func(string) (session.Session, error) {
				return testOnlineSession(), nil
			},
		},
		"localhost:8080",
	)

	_, err := application.Whoami(context.Background())
	if err == nil {
		t.Fatal("Whoami() error = nil, want unauthorized error")
	}
	if !errors.Is(err, apiError) {
		t.Error("whoami error does not preserve API error")
	}
	if !strings.Contains(err.Error(), "online session is invalid or expired: run gkeep login") {
		t.Errorf("error = %q, want readable session error", err)
	}
	if strings.Contains(err.Error(), "test.jwt.token") {
		t.Error("whoami error contains access token")
	}
}

func TestApplication_Whoami_DoesNotLeakTokenInNetworkError(t *testing.T) {
	networkError := errors.New("connection refused")
	application := newApplication(
		userClientStub{
			whoami: func(context.Context, string) (model.User, error) {
				return model.User{}, networkError
			},
		},
		sessionStorageStub{
			load: func(string) (session.Session, error) {
				return testOnlineSession(), nil
			},
		},
		"localhost:8080",
	)

	_, err := application.Whoami(context.Background())
	if err == nil {
		t.Fatal("Whoami() error = nil, want network error")
	}
	if !errors.Is(err, networkError) {
		t.Error("whoami error does not preserve network error")
	}
	if strings.Contains(err.Error(), "test.jwt.token") {
		t.Error("network error contains access token")
	}
}
