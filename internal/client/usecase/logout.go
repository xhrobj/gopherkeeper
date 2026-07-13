package usecase

import (
	"context"
	"fmt"
)

type sessionDeleter interface {
	Delete() error
}

// LogoutApplication выполняет локальный сценарий удаления online-сессии.
type LogoutApplication struct {
	sessions sessionDeleter
}

// NewLogout создаёт application use case локального выхода.
func NewLogout(sessions sessionDeleter) *LogoutApplication {
	return &LogoutApplication{sessions: sessions}
}

// Logout удаляет локальную online-сессию Клиента.
func (a *LogoutApplication) Logout(_ context.Context) error {
	if err := a.sessions.Delete(); err != nil {
		return fmt.Errorf("delete online session: %w", err)
	}

	return nil
}
