package usecase

import (
	"context"
	"fmt"
)

// Health проверяет доступность настроенного Сервера.
func (a *Application) Health(ctx context.Context) (string, error) {
	if a.health == nil {
		return "", fmt.Errorf("health gateway is unavailable")
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
		return fmt.Errorf("session storage is unavailable")
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
