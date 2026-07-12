package usecase

import (
	"context"
	"fmt"

	"github.com/xhrobj/gopherkeeper/internal/client/config"
	"github.com/xhrobj/gopherkeeper/internal/client/httpclient"
	"github.com/xhrobj/gopherkeeper/internal/client/session"
	"github.com/xhrobj/gopherkeeper/internal/model"
)

// Application выполняет клиентские online-сценарии поверх удалённых gateway
// и provider'а локального хранилища session.
type Application struct {
	users         userGateway
	records       recordGateway
	sessions      sessionStorageProvider
	serverAddress string
}

type userGateway interface {
	Register(ctx context.Context, login, password string) (model.User, error)
	Login(ctx context.Context, login, password string) (model.Authentication, error)
	CurrentUser(ctx context.Context, accessToken string) (model.User, error)
}

type sessionStorage interface {
	Save(stored session.Session) error
	Load(expectedServerAddress string) (session.Session, error)
}

type sessionStorageProvider func() (sessionStorage, error)

// New создаёт полностью сконфигурированное клиентское application-приложение.
func New(cfg config.Config) (*Application, error) {
	client, err := httpclient.New(cfg.Address, cfg.CACertFile)
	if err != nil {
		return nil, err
	}

	return &Application{
		users:         client,
		records:       client,
		sessions:      fileSessionStorageProvider(cfg.SessionFile),
		serverAddress: cfg.Address,
	}, nil
}

func fileSessionStorageProvider(path string) sessionStorageProvider {
	return func() (sessionStorage, error) {
		storage, err := session.NewFileStorage(path)
		if err != nil {
			return nil, fmt.Errorf("create online session storage: %w", err)
		}

		return storage, nil
	}
}
