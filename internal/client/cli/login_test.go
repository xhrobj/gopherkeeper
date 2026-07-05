package cli

import (
	"bytes"
	"context"
	"errors"
	"io"
	"net/http"
	"strings"
	"testing"
	"time"

	"github.com/xhrobj/gopherkeeper/internal/client/httpclient"
	"github.com/xhrobj/gopherkeeper/internal/client/session"
	"github.com/xhrobj/gopherkeeper/internal/model"
)

type userLoggerFunc func(context.Context, string, string) (httpclient.LoginResult, error)

func (f userLoggerFunc) Login(
	ctx context.Context,
	login string,
	password string,
) (httpclient.LoginResult, error) {
	return f(ctx, login, password)
}

type sessionSaverFunc func(session.Session) error

func (f sessionSaverFunc) Save(session session.Session) error {
	return f(session)
}

func TestExecuteLogin_Interactive(t *testing.T) {
	createdAt := time.Date(2026, time.July, 4, 12, 0, 0, 0, time.UTC)
	expiresAt := time.Date(2026, time.July, 4, 12, 15, 0, 0, time.UTC)
	passwords := &passwordReaderStub{
		hiddenValues: []string{testRegistrationPassword},
	}

	var savedSession session.Session
	var output bytes.Buffer

	err := executeLogin(
		context.Background(),
		userLoggerFunc(func(_ context.Context, login, password string) (httpclient.LoginResult, error) {
			if login != " Alice " {
				t.Errorf("login = %q, want %q", login, " Alice ")
			}
			if password != testRegistrationPassword {
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
		}),
		sessionSaverFunc(func(session session.Session) error {
			savedSession = session
			return nil
		}),
		passwords,
		loginStreams{
			input:        strings.NewReader(""),
			output:       &output,
			promptOutput: io.Discard,
		},
		"localhost:8080",
		" Alice ",
		false,
	)
	if err != nil {
		t.Fatalf("executeLogin() error = %v", err)
	}

	if passwords.hiddenCalls != 1 {
		t.Errorf("hidden password reads = %d, want 1", passwords.hiddenCalls)
	}
	if passwords.lineCalls != 0 {
		t.Errorf("stdin password reads = %d, want 0", passwords.lineCalls)
	}
	if got := output.String(); got != "User alice logged in successfully.\n" {
		t.Errorf("output = %q, want success message", got)
	}
	if strings.Contains(output.String(), testRegistrationPassword) {
		t.Error("login output contains password")
	}
	if strings.Contains(output.String(), "test.jwt.token") {
		t.Error("login output contains access token")
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

func TestExecuteLogin_PasswordStdin(t *testing.T) {
	passwords := &passwordReaderStub{lineValue: testRegistrationPassword}

	err := executeLogin(
		context.Background(),
		userLoggerFunc(func(_ context.Context, login, password string) (httpclient.LoginResult, error) {
			if login != "bob" {
				t.Errorf("login = %q, want bob", login)
			}
			if password != testRegistrationPassword {
				t.Error("login client received unexpected password")
			}

			return httpclient.LoginResult{
				AccessToken: "test.jwt.token",
				TokenType:   "Bearer",
				ExpiresAt:   time.Date(2026, time.July, 4, 12, 15, 0, 0, time.UTC),
				User: model.User{
					ID:        42,
					Login:     "bob",
					CreatedAt: time.Date(2026, time.July, 4, 12, 0, 0, 0, time.UTC),
				},
			}, nil
		}),
		sessionSaverFunc(func(session.Session) error { return nil }),
		passwords,
		loginStreams{
			input:        strings.NewReader(testRegistrationPassword + "\n"),
			output:       io.Discard,
			promptOutput: io.Discard,
		},
		"localhost:8080",
		"bob",
		true,
	)
	if err != nil {
		t.Fatalf("executeLogin() error = %v", err)
	}

	if passwords.lineCalls != 1 {
		t.Errorf("stdin password reads = %d, want 1", passwords.lineCalls)
	}
	if passwords.hiddenCalls != 0 {
		t.Errorf("hidden password reads = %d, want 0", passwords.hiddenCalls)
	}
}

func TestExecuteLogin_ReturnsReadableInvalidCredentialsError(t *testing.T) {
	apiError := &httpclient.APIError{
		StatusCode: http.StatusUnauthorized,
		Code:       "invalid_credentials",
		Message:    "invalid login or password",
	}

	err := executeLogin(
		context.Background(),
		userLoggerFunc(func(context.Context, string, string) (httpclient.LoginResult, error) {
			return httpclient.LoginResult{}, apiError
		}),
		sessionSaverFunc(func(session.Session) error {
			t.Fatal("session must not be saved after invalid credentials")
			return nil
		}),
		&passwordReaderStub{lineValue: testRegistrationPassword},
		loginStreams{
			input:        strings.NewReader(testRegistrationPassword + "\n"),
			output:       io.Discard,
			promptOutput: io.Discard,
		},
		"localhost:8080",
		"eve",
		true,
	)
	if err == nil {
		t.Fatal("executeLogin() error = nil, want invalid credentials")
	}
	if !strings.Contains(err.Error(), "invalid login or password") {
		t.Errorf("error = %q, want readable invalid credentials message", err)
	}
	if !errors.Is(err, apiError) {
		t.Error("login error does not preserve API error")
	}
	if strings.Contains(err.Error(), testRegistrationPassword) {
		t.Error("invalid credentials error contains password")
	}
}

func TestExecuteLogin_DoesNotLeakPasswordInNetworkError(t *testing.T) {
	networkError := errors.New("connection refused")

	err := executeLogin(
		context.Background(),
		userLoggerFunc(func(context.Context, string, string) (httpclient.LoginResult, error) {
			return httpclient.LoginResult{}, networkError
		}),
		sessionSaverFunc(func(session.Session) error {
			t.Fatal("session must not be saved after network error")
			return nil
		}),
		&passwordReaderStub{lineValue: testRegistrationPassword},
		loginStreams{
			input:        strings.NewReader(testRegistrationPassword + "\n"),
			output:       io.Discard,
			promptOutput: io.Discard,
		},
		"localhost:8080",
		"eve",
		true,
	)
	if err == nil {
		t.Fatal("executeLogin() error = nil, want network error")
	}
	if !errors.Is(err, networkError) {
		t.Error("login error does not preserve network error")
	}
	if strings.Contains(err.Error(), testRegistrationPassword) {
		t.Error("network error contains password")
	}
}

func TestExecuteLogin_DoesNotLeakTokenInSaveError(t *testing.T) {
	saveError := errors.New("permission denied")

	err := executeLogin(
		context.Background(),
		userLoggerFunc(func(context.Context, string, string) (httpclient.LoginResult, error) {
			return httpclient.LoginResult{
				AccessToken: "test.jwt.token",
				TokenType:   "Bearer",
				ExpiresAt:   time.Date(2026, time.July, 4, 12, 15, 0, 0, time.UTC),
				User: model.User{
					ID:        42,
					Login:     "eve",
					CreatedAt: time.Date(2026, time.July, 4, 12, 0, 0, 0, time.UTC),
				},
			}, nil
		}),
		sessionSaverFunc(func(session.Session) error { return saveError }),
		&passwordReaderStub{lineValue: testRegistrationPassword},
		loginStreams{
			input:        strings.NewReader(testRegistrationPassword + "\n"),
			output:       io.Discard,
			promptOutput: io.Discard,
		},
		"localhost:8080",
		"eve",
		true,
	)
	if err == nil {
		t.Fatal("executeLogin() error = nil, want save error")
	}
	if !errors.Is(err, saveError) {
		t.Error("login error does not preserve save error")
	}
	if strings.Contains(err.Error(), "test.jwt.token") {
		t.Error("save error contains access token")
	}
}
