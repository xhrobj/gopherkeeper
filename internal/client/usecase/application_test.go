package usecase

import (
	"context"
	"testing"
	"time"

	"github.com/xhrobj/gopherkeeper/internal/client/session"
	"github.com/xhrobj/gopherkeeper/internal/model"
)

const testPassword = "correct-horse-battery-staple"

type healthGatewayStub struct {
	health func(context.Context) (string, error)
}

func (stub healthGatewayStub) Health(ctx context.Context) (string, error) {
	if stub.health == nil {
		return "", nil
	}
	return stub.health(ctx)
}

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
	save   func(session.Session) error
	load   func() (session.Session, error)
	delete func() error
}

func (s sessionStorageStub) Save(stored session.Session) error {
	return s.save(stored)
}

func (s sessionStorageStub) Load() (session.Session, error) {
	return s.load()
}

func (s sessionStorageStub) Delete() error {
	if s.delete == nil {
		return nil
	}
	return s.delete()
}

func newTestApplication(users UserGateway, sessions SessionStorage) *Application {
	return newTestApplicationWithRecords(users, recordGatewayStub{}, sessions)
}

func newTestApplicationWithRecords(
	users UserGateway,
	records RecordGateway,
	sessions SessionStorage,
) *Application {
	return &Application{
		users:   users,
		records: records,
		sessions: func() (SessionStorage, error) {
			return sessions, nil
		},
	}
}

func testOnlineSession() session.Session {
	return session.Session{
		AccessToken: "test.jwt.token",
		ExpiresAt:   time.Date(2026, time.July, 6, 12, 15, 0, 0, time.UTC),
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
		healthGatewayStub{},
		userGatewayStub{},
		recordGatewayStub{},
		func() (SessionStorage, error) { return sessionStorageStub{}, nil },
		func(context.Context, string, []byte) (SyncCacheRepository, error) { return nil, nil },
		func(context.Context, string, []byte) (OfflineCacheRepository, error) { return nil, nil },
	)
	if application.health == nil {
		t.Error("New() health gateway = nil")
	}
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
}

func TestNewOffline(t *testing.T) {
	provider := func(
		context.Context,
		string,
		[]byte,
	) (OfflineCacheRepository, error) {
		return nil, nil
	}

	application := NewOffline(provider)

	if application.offlineCaches == nil {
		t.Error("NewOffline() offline cache repository provider = nil")
	}
}
