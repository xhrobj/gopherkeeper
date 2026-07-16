package usecase

import (
	"context"
	"testing"
	"time"

	"github.com/xhrobj/gopherkeeper/internal/client/session"
	"github.com/xhrobj/gopherkeeper/internal/model"
)

const testPassword = "correct-horse-battery-staple"

type userGatewayStub struct {
	register func(context.Context, string, string) (model.User, error)
	login    func(context.Context, string, string) (model.Authentication, error)
	whoami   func(context.Context, string) (model.User, error)
}

func (s userGatewayStub) Register(ctx context.Context, login, password string) (model.User, error) {
	return s.register(ctx, login, password)
}

func (s userGatewayStub) Login(ctx context.Context, login, password string) (model.Authentication, error) {
	return s.login(ctx, login, password)
}

func (s userGatewayStub) CurrentUser(ctx context.Context, accessToken string) (model.User, error) {
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

func newTestApplication(users UserGateway, sessions SessionStorage, serverAddress string) *Application {
	return newTestApplicationWithRecords(users, recordGatewayStub{}, sessions, serverAddress)
}

func newTestApplicationWithRecords(
	users UserGateway,
	records RecordGateway,
	sessions SessionStorage,
	serverAddress string,
) *Application {
	return &Application{
		users:   users,
		records: records,
		sessions: func() (SessionStorage, error) {
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
	application := New(
		userGatewayStub{},
		recordGatewayStub{},
		func() (SessionStorage, error) { return sessionStorageStub{}, nil },
		func(context.Context, string, string, []byte) (SyncCacheRepository, error) { return nil, nil },
		func(context.Context, string, string, []byte) (OfflineCacheRepository, error) { return nil, nil },
		"localhost:8080",
	)
	if application.users == nil {
		t.Error("New() user gateway = nil")
	}
	if application.records == nil {
		t.Error("New() record gateway = nil")
	}
	if application.sessions == nil {
		t.Error("New() session storage provider = nil")
	}
	if application.syncCaches == nil {
		t.Error("New() sync cache repository provider = nil")
	}
	if application.offlineCaches == nil {
		t.Error("New() offline cache repository provider = nil")
	}
	if application.serverAddress != "localhost:8080" {
		t.Errorf("New() server address = %q, want localhost:8080", application.serverAddress)
	}
}

func TestNewOffline(t *testing.T) {
	provider := func(
		context.Context,
		string,
		string,
		[]byte,
	) (OfflineCacheRepository, error) {
		return nil, nil
	}

	application := NewOffline(provider, "localhost:8080")

	if application.users != nil {
		t.Error("NewOffline() user gateway != nil")
	}
	if application.records != nil {
		t.Error("NewOffline() record gateway != nil")
	}
	if application.sessions != nil {
		t.Error("NewOffline() session storage provider != nil")
	}
	if application.syncCaches != nil {
		t.Error("NewOffline() sync cache repository provider != nil")
	}
	if application.offlineCaches == nil {
		t.Error("NewOffline() offline cache repository provider = nil")
	}
	if application.serverAddress != "localhost:8080" {
		t.Errorf("NewOffline() server address = %q, want localhost:8080", application.serverAddress)
	}
}
