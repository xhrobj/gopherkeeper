package tui

import (
	"context"

	"github.com/xhrobj/gopherkeeper/internal/client/config"
)

// Backend описывает пользовательские сценарии, доступные терминальному интерфейсу.
// Экземпляр Backend привязан к одной runtime-конфигурации.
type Backend interface {
	Health(context.Context) (string, error)
	Register(context.Context, string, string) (string, error)
	Login(context.Context, string, string) (string, error)
	CurrentUser(context.Context) (string, error)
	Logout(context.Context) error
}

// BackendFactory создаёт runtime-зависимости TUI для переданной конфигурации.
type BackendFactory func(config.Config) Backend
