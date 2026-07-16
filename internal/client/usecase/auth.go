package usecase

import (
	"context"
	"errors"
	"fmt"

	"github.com/xhrobj/gopherkeeper/internal/client/session"
	"github.com/xhrobj/gopherkeeper/internal/model"
)

// Register регистрирует нового пользователя.
func (a *Application) Register(ctx context.Context, login, password string) (model.User, error) {
	user, err := a.users.Register(ctx, login, password)
	if err != nil {
		if errors.Is(err, model.ErrLoginAlreadyExists) {
			return model.User{}, newUserError(fmt.Sprintf("login %q is already registered", login), err)
		}

		return model.User{}, fmt.Errorf("register user: %w", err)
	}

	return user, nil
}

// Login аутентифицирует пользователя и сохраняет online-сессию локально.
func (a *Application) Login(ctx context.Context, login, password string) (model.User, error) {
	sessions, err := a.sessions()
	if err != nil {
		return model.User{}, err
	}

	result, err := a.users.Login(ctx, login, password)
	if err != nil {
		return model.User{}, mapLoginGatewayError(err)
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
	sessions, err := a.sessions()
	if err != nil {
		return session.Session{}, err
	}

	storedSession, err := sessions.Load(a.serverAddress)
	if err != nil {
		return session.Session{}, mapSessionLoadError(err)
	}

	return storedSession, nil
}

func mapSessionLoadError(err error) error {
	switch {
	case errors.Is(err, session.ErrExpired):
		return newUserError("session expired, please login again", errors.Join(ErrNotLoggedIn, err))
	case errors.Is(err, session.ErrNotFound),
		errors.Is(err, session.ErrServerMismatch),
		errors.Is(err, session.ErrInvalid):
		return newUserError("not logged in", errors.Join(ErrNotLoggedIn, err))
	default:
		return fmt.Errorf("load online session: %w", err)
	}
}

func mapLoginGatewayError(err error) error {
	if errors.Is(err, model.ErrInvalidCredentials) {
		return newUserError("invalid login or password", err)
	}

	return fmt.Errorf("login user: %w", err)
}

func mapCurrentUserError(err error) error {
	if errors.Is(err, model.ErrUnauthorized) {
		return newUserError("not logged in", errors.Join(ErrNotLoggedIn, err))
	}

	return fmt.Errorf("get current user: %w", err)
}

// ErrNotLoggedIn означает, что Клиент не имеет действующей online-сессии.
var ErrNotLoggedIn = errors.New("not logged in")

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

// Error возвращает безопасное сообщение для пользователя.
func (e *userError) Error() string {
	return e.message
}

// Unwrap возвращает исходную ошибку.
func (e *userError) Unwrap() error {
	return e.cause
}
