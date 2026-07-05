package app

import (
	"context"
	"errors"
	"fmt"
	"net/http"

	"github.com/xhrobj/gopherkeeper/internal/client/config"
	"github.com/xhrobj/gopherkeeper/internal/client/httpclient"
	"github.com/xhrobj/gopherkeeper/internal/client/session"
	"github.com/xhrobj/gopherkeeper/internal/model"
)

// Application выполняет клиентские сценарии поверх HTTP client'а и локальной session.
type Application struct {
	users         userClient
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
}

type sessionStorageFactory func() (sessionStorage, error)

// New создаёт клиентское application-приложение из конфигурации CLI.
func New(cfg config.Config) (*Application, error) {
	users, err := httpclient.New(cfg.Address, cfg.CACertFile)
	if err != nil {
		return nil, err
	}

	return newApplicationWithSessionFactory(
		users,
		func() (sessionStorage, error) {
			return session.NewFileStorage(cfg.SessionFile)
		},
		cfg.Address,
	), nil
}

func newApplication(users userClient, sessions sessionStorage, serverAddress string) *Application {
	return &Application{
		users:         users,
		sessions:      sessions,
		serverAddress: serverAddress,
	}
}

func newApplicationWithSessionFactory(
	users userClient,
	newSessions sessionStorageFactory,
	serverAddress string,
) *Application {
	return &Application{
		users:         users,
		newSessions:   newSessions,
		serverAddress: serverAddress,
	}
}

// Register регистрирует нового пользователя.
func (a *Application) Register(ctx context.Context, login, password string) (model.User, error) {
	user, err := a.users.Register(ctx, login, password)
	if err != nil {
		var apiError *httpclient.APIError
		if errors.As(err, &apiError) && apiError.Code == "login_already_exists" {
			return model.User{}, fmt.Errorf("login %q is already registered: %w", login, err)
		}

		return model.User{}, fmt.Errorf("register user: %w", err)
	}

	return user, nil
}

// Login аутентифицирует пользователя и сохраняет online-сессию локально.
func (a *Application) Login(ctx context.Context, login, password string) (model.User, error) {
	sessions, err := a.sessionStorage()
	if err != nil {
		return model.User{}, fmt.Errorf("create online session storage: %w", err)
	}

	result, err := a.users.Login(ctx, login, password)
	if err != nil {
		var apiError *httpclient.APIError
		if errors.As(err, &apiError) && apiError.StatusCode == http.StatusUnauthorized && apiError.Code == "invalid_credentials" {
			return model.User{}, fmt.Errorf("invalid login or password: %w", err)
		}

		return model.User{}, fmt.Errorf("login user: %w", err)
	}

	if err := sessions.Save(session.Session{
		ServerAddress: a.serverAddress,
		AccessToken:   result.AccessToken,
		TokenType:     result.TokenType,
		ExpiresAt:     result.ExpiresAt,
		User:          result.User,
	}); err != nil {
		return model.User{}, fmt.Errorf("save online session: %w", err)
	}

	return result.User, nil
}

// Whoami возвращает пользователя из текущей online-сессии.
func (a *Application) Whoami(ctx context.Context) (model.User, error) {
	sessions, err := a.sessionStorage()
	if err != nil {
		return model.User{}, fmt.Errorf("create online session storage: %w", err)
	}

	storedSession, err := sessions.Load(a.serverAddress)
	if err != nil {
		return model.User{}, mapSessionLoadError(err)
	}

	user, err := a.users.CurrentUser(ctx, storedSession.AccessToken)
	if err != nil {
		return model.User{}, mapCurrentUserError(err)
	}

	return user, nil
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

func mapSessionLoadError(err error) error {
	switch {
	case errors.Is(err, session.ErrNotFound):
		return fmt.Errorf("online session not found: run gkeep login: %w", err)
	case errors.Is(err, session.ErrExpired):
		return fmt.Errorf("online session expired: run gkeep login: %w", err)
	case errors.Is(err, session.ErrServerMismatch):
		return fmt.Errorf("online session belongs to another server: run gkeep login: %w", err)
	case errors.Is(err, session.ErrInvalid):
		return fmt.Errorf("online session is invalid: run gkeep login: %w", err)
	default:
		return fmt.Errorf("load online session: %w", err)
	}
}

func mapCurrentUserError(err error) error {
	var apiError *httpclient.APIError
	if errors.As(err, &apiError) && apiError.StatusCode == http.StatusUnauthorized && apiError.Code == "unauthorized" {
		return fmt.Errorf("online session is invalid or expired: run gkeep login: %w", err)
	}

	return fmt.Errorf("get current user: %w", err)
}
