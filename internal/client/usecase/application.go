package usecase

import (
	"context"
	"errors"

	"github.com/xhrobj/gopherkeeper/internal/client/config"
	"github.com/xhrobj/gopherkeeper/internal/client/httpclient"
	"github.com/xhrobj/gopherkeeper/internal/client/session"
	"github.com/xhrobj/gopherkeeper/internal/model"
)

// Application выполняет клиентские сценарии поверх HTTP client'а и локальной session.
type Application struct {
	users         userClient
	records       recordClient
	sessions      sessionStorage
	newSessions   sessionStorageFactory
	serverAddress string
}

type userClient interface {
	Register(ctx context.Context, login, password string) (model.User, error)
	Login(ctx context.Context, login, password string) (httpclient.LoginResult, error)
	CurrentUser(ctx context.Context, accessToken string) (model.User, error)
}

type sessionStorage interface {
	Save(stored session.Session) error
	Load(expectedServerAddress string) (session.Session, error)
	Delete() error
}

type sessionStorageFactory func() (sessionStorage, error)

// NewLocal создаёт клиентское application-приложение только для локальных session-операций.
func NewLocal(cfg config.Config) *Application {
	return newApplicationWithSessionFactory(
		nil,
		func() (sessionStorage, error) {
			return session.NewFileStorage(cfg.SessionFile)
		},
		cfg.Address,
	)
}

// New создаёт клиентское application-приложение из конфигурации CLI.
func New(cfg config.Config) (*Application, error) {
	client, err := httpclient.New(cfg.Address, cfg.CACertFile)
	if err != nil {
		return nil, err
	}

	return newApplicationWithRecordsAndSessionFactory(
		client,
		client,
		func() (sessionStorage, error) {
			return session.NewFileStorage(cfg.SessionFile)
		},
		cfg.Address,
	), nil
}

func newApplication(users userClient, sessions sessionStorage, serverAddress string) *Application {
	return newApplicationWithRecords(users, nil, sessions, serverAddress)
}

func newApplicationWithRecords(
	users userClient,
	records recordClient,
	sessions sessionStorage,
	serverAddress string,
) *Application {
	return &Application{
		users:         users,
		records:       records,
		sessions:      sessions,
		serverAddress: serverAddress,
	}
}

func newApplicationWithSessionFactory(
	users userClient,
	newSessions sessionStorageFactory,
	serverAddress string,
) *Application {
	return newApplicationWithRecordsAndSessionFactory(users, nil, newSessions, serverAddress)
}

func newApplicationWithRecordsAndSessionFactory(
	users userClient,
	records recordClient,
	newSessions sessionStorageFactory,
	serverAddress string,
) *Application {
	return &Application{
		users:         users,
		records:       records,
		newSessions:   newSessions,
		serverAddress: serverAddress,
	}
}

func (a *Application) sessionStorage() (sessionStorage, error) {
	if a.sessions != nil {
		return a.sessions, nil
	}
	if a.newSessions == nil {
		return nil, errors.New("session storage factory is not configured")
	}

	sessions, err := a.newSessions()
	if err != nil {
		return nil, err
	}
	if sessions == nil {
		return nil, errors.New("session storage factory returned nil")
	}
	a.sessions = sessions

	return sessions, nil
}
