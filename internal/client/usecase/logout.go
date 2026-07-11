package usecase

import (
	"context"
	"fmt"

	"github.com/xhrobj/gopherkeeper/internal/client/config"
	"github.com/xhrobj/gopherkeeper/internal/client/session"
)

type sessionDeleter interface {
	Delete() error
}

// LogoutApplication выполняет локальный сценарий удаления online-сессии.
type LogoutApplication struct {
	sessions sessionDeleter
}

// NewLogout создаёт application use case локального выхода.
func NewLogout(cfg config.Config) (*LogoutApplication, error) {
	sessions, err := session.NewFileStorage(cfg.SessionFile)
	if err != nil {
		return nil, fmt.Errorf("create online session storage: %w", err)
	}

	return &LogoutApplication{sessions: sessions}, nil
}

// Logout удаляет локальную online-сессию Клиента.
func (a *LogoutApplication) Logout(_ context.Context) error {
	if err := a.sessions.Delete(); err != nil {
		return fmt.Errorf("delete online session: %w", err)
	}

	return nil
}
