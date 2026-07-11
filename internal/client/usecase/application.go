package usecase

import (
	"context"
	"fmt"

	"github.com/xhrobj/gopherkeeper/internal/client/config"
	"github.com/xhrobj/gopherkeeper/internal/client/httpclient"
	"github.com/xhrobj/gopherkeeper/internal/client/session"
	"github.com/xhrobj/gopherkeeper/internal/model"
)

// Application выполняет клиентские online-сценарии поверх HTTP client'а
// и локального хранилища session.
type Application struct {
	users         userClient
	records       recordClient
	sessions      sessionStorage
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
}

// New создаёт полностью сконфигурированное клиентское application-приложение.
func New(cfg config.Config) (*Application, error) {
	client, err := httpclient.New(cfg.Address, cfg.CACertFile)
	if err != nil {
		return nil, err
	}

	sessions, err := session.NewFileStorage(cfg.SessionFile)
	if err != nil {
		return nil, fmt.Errorf("create online session storage: %w", err)
	}

	return &Application{
		users:         client,
		records:       client,
		sessions:      sessions,
		serverAddress: cfg.Address,
	}, nil
}
