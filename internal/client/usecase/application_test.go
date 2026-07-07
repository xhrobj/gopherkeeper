package usecase

import (
	"context"
	"errors"
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
	save   func(session.Session) error
	load   func(string) (session.Session, error)
	delete func() error
}

func (s sessionStorageStub) Save(stored session.Session) error {
	return s.save(stored)
}

func (s sessionStorageStub) Load(expectedServerAddress string) (session.Session, error) {
	return s.load(expectedServerAddress)
}

func (s sessionStorageStub) Delete() error {
	return s.delete()
}

func testOnlineSession() session.Session {
	return session.Session{
		ServerAddress: "localhost:8080",
		AccessToken:   "test.jwt.token",
		ExpiresAt:     time.Date(2026, time.July, 6, 12, 15, 0, 0, time.UTC),
	}
}

func testUser() model.User {
	return model.User{
		ID:        42,
		Login:     "alice",
		CreatedAt: time.Date(2026, time.July, 6, 12, 0, 0, 0, time.UTC),
	}
}

func TestApplicationSessionStorage_UsesConfiguredStorage(t *testing.T) {
	configuredStorage := sessionStorageStub{}
	application := newApplication(userClientStub{}, configuredStorage, "localhost:8080")

	got, err := application.sessionStorage()
	if err != nil {
		t.Fatalf("sessionStorage() error = %v", err)
	}
	if _, ok := got.(sessionStorageStub); !ok {
		t.Fatalf("sessionStorage() = %T, want sessionStorageStub", got)
	}
}

func TestApplicationSessionStorage_CreatesStorageOnce(t *testing.T) {
	var calls int
	application := newApplicationWithSessionFactory(
		userClientStub{},
		func() (sessionStorage, error) {
			calls++
			return sessionStorageStub{}, nil
		},
		"localhost:8080",
	)

	if _, err := application.sessionStorage(); err != nil {
		t.Fatalf("first sessionStorage() error = %v", err)
	}
	if _, err := application.sessionStorage(); err != nil {
		t.Fatalf("second sessionStorage() error = %v", err)
	}
	if calls != 1 {
		t.Errorf("session storage factory calls = %d, want 1", calls)
	}
}

func TestApplicationSessionStorage_ReturnsFactoryErrors(t *testing.T) {
	factoryError := errors.New("permission denied")
	tests := []struct {
		name        string
		application *Application
		want        string
	}{
		{
			name:        "missing factory",
			application: &Application{},
			want:        "session storage factory is not configured",
		},
		{
			name: "factory error",
			application: newApplicationWithSessionFactory(
				userClientStub{},
				func() (sessionStorage, error) {
					return nil, factoryError
				},
				"localhost:8080",
			),
			want: factoryError.Error(),
		},
		{
			name: "nil storage",
			application: newApplicationWithSessionFactory(
				userClientStub{},
				func() (sessionStorage, error) {
					return nil, nil
				},
				"localhost:8080",
			),
			want: "session storage factory returned nil",
		},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			_, err := tt.application.sessionStorage()
			if err == nil {
				t.Fatal("sessionStorage() error = nil, want error")
			}
			if err.Error() != tt.want {
				t.Errorf("sessionStorage() error = %q, want %q", err, tt.want)
			}
		})
	}
}
