package usecase

import (
	"context"
	"errors"
	"fmt"

	"github.com/xhrobj/gopherkeeper/internal/client/httpclient"
	"github.com/xhrobj/gopherkeeper/internal/client/session"
	"github.com/xhrobj/gopherkeeper/internal/model"
)

// Register регистрирует нового пользователя.
func (a *Application) Register(ctx context.Context, login, password string) (model.User, error) {
	user, err := a.users.Register(ctx, login, password)
	if err != nil {
		var apiError *httpclient.APIError
		if errors.As(err, &apiError) && apiError.Code == "login_already_exists" {
			return model.User{}, newUserError(fmt.Sprintf("login %q is already registered", login), err)
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
		if errors.As(err, &apiError) && apiError.Code == "invalid_credentials" {
			return model.User{}, newUserError("invalid login or password", err)
		}

		return model.User{}, fmt.Errorf("login user: %w", err)
	}

	if err := sessions.Save(session.Session{
		ServerAddress: a.serverAddress,
		AccessToken:   result.AccessToken,
		ExpiresAt:     result.ExpiresAt,
	}); err != nil {
		return model.User{}, fmt.Errorf("save online session: %w", err)
	}

	return result.User, nil
}

// Whoami возвращает пользователя из текущей online-сессии.
func (a *Application) Whoami(ctx context.Context) (model.User, error) {
	storedSession, err := a.loadSession()
	if err != nil {
		return model.User{}, err
	}

	user, err := a.users.CurrentUser(ctx, storedSession.AccessToken)
	if err != nil {
		return model.User{}, mapCurrentUserError(err)
	}

	return user, nil
}

func (a *Application) loadSession() (session.Session, error) {
	sessions, err := a.sessionStorage()
	if err != nil {
		return session.Session{}, fmt.Errorf("create online session storage: %w", err)
	}

	storedSession, err := sessions.Load(a.serverAddress)
	if err != nil {
		return session.Session{}, mapSessionLoadError(err)
	}

	return storedSession, nil
}

func mapSessionLoadError(err error) error {
	switch {
	case errors.Is(err, session.ErrNotFound):
		return newUserError("online session not found: run gkeep login", err)
	case errors.Is(err, session.ErrExpired):
		return newUserError("online session expired: run gkeep login", err)
	case errors.Is(err, session.ErrServerMismatch):
		return newUserError("online session belongs to another server: run gkeep login", err)
	case errors.Is(err, session.ErrInvalid):
		return newUserError("online session is invalid: run gkeep login", err)
	default:
		return fmt.Errorf("load online session: %w", err)
	}
}

func mapCurrentUserError(err error) error {
	var apiError *httpclient.APIError
	if errors.As(err, &apiError) && apiError.Code == "unauthorized" {
		return newUserError("online session is invalid or expired: run gkeep login", err)
	}

	return fmt.Errorf("get current user: %w", err)
}

type userError struct {
	message string
	cause   error
}

func newUserError(message string, cause error) error {
	return &userError{
		message: message,
		cause:   cause,
	}
}

func (e *userError) Error() string {
	return e.message
}

func (e *userError) Unwrap() error {
	return e.cause
}
