package tui

import (
	"context"
	"errors"
	"fmt"

	"github.com/xhrobj/gopherkeeper/internal/client/config"
	recordmodel "github.com/xhrobj/gopherkeeper/internal/model"
)

// Backend описывает пользовательские сценарии, доступные терминальному интерфейсу.
// Экземпляр Backend привязан к одной runtime-конфигурации.
type Backend interface {
	Health(context.Context) (string, error)
	Register(context.Context, string, string) (string, error)
	Login(context.Context, string, string) (string, error)
	CurrentUser(context.Context) (string, error)
	Logout(context.Context) error
	ListRecords(context.Context) ([]recordmodel.RecordMetadata, error)
	GetRecord(context.Context, string) (recordmodel.Record, error)
	CreateRecord(context.Context, string, recordmodel.RecordPayload) (recordmodel.Record, error)
	UpdateRecord(context.Context, string, int64, string, recordmodel.RecordPayload) (recordmodel.Record, error)
	DeleteRecord(context.Context, string, int64) error
}

// BackendFactory создаёт runtime-зависимости TUI для переданной конфигурации.
type BackendFactory func(config.Config) (Backend, error)

func createBackend(factory BackendFactory, cfg config.Config) (Backend, error) {
	if factory == nil {
		return nil, errors.New("TUI backend factory is required")
	}

	backend, err := factory(cfg)
	if err != nil {
		return nil, fmt.Errorf("create TUI backend: %w", err)
	}

	if backend == nil {
		return nil, errors.New("create TUI backend: factory returned nil backend")
	}

	return backend, nil
}
