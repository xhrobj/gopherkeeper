package usecase

import (
	"context"
	"testing"
	"time"

	"github.com/xhrobj/gopherkeeper/internal/client/config"
	"github.com/xhrobj/gopherkeeper/internal/client/session"
	"github.com/xhrobj/gopherkeeper/internal/model"
)

const testPassword = "correct-horse-battery-staple"

type userClientStub struct {
	register func(context.Context, string, string) (model.User, error)
	login    func(context.Context, string, string) (model.Authentication, error)
	whoami   func(context.Context, string) (model.User, error)
}

func (s userClientStub) Register(ctx context.Context, login, password string) (model.User, error) {
	return s.register(ctx, login, password)
}

func (s userClientStub) Login(ctx context.Context, login, password string) (model.Authentication, error) {
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

func newTestApplication(users userGateway, sessions sessionStorage, serverAddress string) *Application {
	return newTestApplicationWithRecords(users, recordGatewayStub{}, sessions, serverAddress)
}

func newTestApplicationWithRecords(
	users userGateway,
	records recordGateway,
	sessions sessionStorage,
	serverAddress string,
) *Application {
	return &Application{
		users:   users,
		records: records,
		sessions: func() (sessionStorage, error) {
			return sessions, nil
		},
		serverAddress: serverAddress,
	}
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

func TestNew(t *testing.T) {
	application, err := New(config.Config{
		Address:     "localhost:8080",
		SessionFile: t.TempDir() + "/session.json",
	})
	if err != nil {
		t.Fatalf("New() error = %v", err)
	}
	if application.users == nil {
		t.Error("New() user client = nil")
	}
	if application.records == nil {
		t.Error("New() record client = nil")
	}
	if application.sessions == nil {
		t.Error("New() session storage = nil")
	}
	if application.serverAddress != "localhost:8080" {
		t.Errorf("New() server address = %q, want localhost:8080", application.serverAddress)
	}
}
