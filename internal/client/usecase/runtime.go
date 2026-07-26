package usecase

import (
	"context"
	"fmt"

	"github.com/xhrobj/gopherkeeper/internal/client/failure"
)

// Health проверяет доступность настроенного Сервера.
func (a *Application) Health(ctx context.Context) (string, error) {
	if a.health == nil {
		return "", failure.Wrap(
			failure.Unknown,
			"health gateway is unavailable",
			"Server health check is unavailable",
			nil,
		)
	}

	status, err := a.health.Health(ctx)
	if err != nil {
		return "", err
	}

	return status, nil
}

// Logout удаляет локальную online-сессию Клиента.
func (a *Application) Logout(_ context.Context) error {
	if a.sessions == nil {
		return failure.Wrap(
			failure.Unknown,
			"session storage is unavailable",
			"Unable to log out",
			nil,
		)
	}

	sessions, err := a.sessions()
	if err != nil {
		return err
	}

	if err := sessions.Delete(); err != nil {
		return fmt.Errorf("delete online session: %w", err)
	}

	return nil
}
