package cli

import (
	"bytes"
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

type currentUserGetterFunc func(context.Context, string) (model.User, error)

func (f currentUserGetterFunc) CurrentUser(ctx context.Context, accessToken string) (model.User, error) {
	return f(ctx, accessToken)
}

type sessionLoaderFunc func(string) (session.Session, error)

func (f sessionLoaderFunc) Load(expectedServerAddress string) (session.Session, error) {
	return f(expectedServerAddress)
}

func TestExecuteWhoami(t *testing.T) {
	createdAt := time.Date(2026, time.July, 4, 12, 0, 0, 0, time.UTC)
	expiresAt := time.Date(2026, time.July, 4, 12, 15, 0, 0, time.UTC)
	var output bytes.Buffer

	err := executeWhoami(
		context.Background(),
		currentUserGetterFunc(func(_ context.Context, accessToken string) (model.User, error) {
			if accessToken != "test.jwt.token" {
				t.Errorf("access token = %q, want test.jwt.token", accessToken)
			}

			return model.User{
				ID:        42,
				Login:     "alice",
				CreatedAt: createdAt,
			}, nil
		}),
		sessionLoaderFunc(func(expectedServerAddress string) (session.Session, error) {
			if expectedServerAddress != "localhost:8080" {
				t.Errorf("expected server address = %q, want localhost:8080", expectedServerAddress)
			}

			return session.Session{
				ServerAddress: "localhost:8080",
				AccessToken:   "test.jwt.token",
				TokenType:     "Bearer",
				ExpiresAt:     expiresAt,
				User: model.User{
					ID:        42,
					Login:     "alice",
					CreatedAt: createdAt,
				},
			}, nil
		}),
		&output,
		"localhost:8080",
	)
	if err != nil {
		t.Fatalf("executeWhoami() error = %v", err)
	}

	if got := output.String(); got != "alice\n" {
		t.Errorf("output = %q, want alice", got)
	}
	if strings.Contains(output.String(), "test.jwt.token") {
		t.Error("whoami output contains access token")
	}
}

func TestExecuteWhoami_MapsSessionErrors(t *testing.T) {
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
			err := executeWhoami(
				context.Background(),
				currentUserGetterFunc(func(context.Context, string) (model.User, error) {
					t.Fatal("current user client must not be called after session load error")
					return model.User{}, nil
				}),
				sessionLoaderFunc(func(string) (session.Session, error) {
					return session.Session{}, tt.loadErr
				}),
				&bytes.Buffer{},
				"localhost:8080",
			)
			if err == nil {
				t.Fatal("executeWhoami() error = nil, want session error")
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

func TestExecuteWhoami_MapsUnauthorizedAPIError(t *testing.T) {
	apiError := &httpclient.APIError{
		StatusCode: http.StatusUnauthorized,
		Code:       "unauthorized",
		Message:    "missing or invalid bearer token",
	}

	err := executeWhoami(
		context.Background(),
		currentUserGetterFunc(func(context.Context, string) (model.User, error) {
			return model.User{}, apiError
		}),
		sessionLoaderFunc(func(string) (session.Session, error) {
			return session.Session{
				ServerAddress: "localhost:8080",
				AccessToken:   "test.jwt.token",
				TokenType:     "Bearer",
				ExpiresAt:     time.Date(2026, time.July, 4, 12, 15, 0, 0, time.UTC),
				User: model.User{
					ID:        42,
					Login:     "alice",
					CreatedAt: time.Date(2026, time.July, 4, 12, 0, 0, 0, time.UTC),
				},
			}, nil
		}),
		&bytes.Buffer{},
		"localhost:8080",
	)
	if err == nil {
		t.Fatal("executeWhoami() error = nil, want unauthorized error")
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

func TestExecuteWhoami_DoesNotLeakTokenInNetworkError(t *testing.T) {
	networkError := errors.New("connection refused")

	err := executeWhoami(
		context.Background(),
		currentUserGetterFunc(func(context.Context, string) (model.User, error) {
			return model.User{}, networkError
		}),
		sessionLoaderFunc(func(string) (session.Session, error) {
			return session.Session{
				ServerAddress: "localhost:8080",
				AccessToken:   "test.jwt.token",
				TokenType:     "Bearer",
				ExpiresAt:     time.Date(2026, time.July, 4, 12, 15, 0, 0, time.UTC),
				User: model.User{
					ID:        42,
					Login:     "alice",
					CreatedAt: time.Date(2026, time.July, 4, 12, 0, 0, 0, time.UTC),
				},
			}, nil
		}),
		&bytes.Buffer{},
		"localhost:8080",
	)
	if err == nil {
		t.Fatal("executeWhoami() error = nil, want network error")
	}
	if !errors.Is(err, networkError) {
		t.Error("whoami error does not preserve network error")
	}
	if strings.Contains(err.Error(), "test.jwt.token") {
		t.Error("network error contains access token")
	}
}
