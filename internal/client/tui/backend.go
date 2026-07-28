package tui

import (
	"context"
	"errors"

	"github.com/xhrobj/gopherkeeper/internal/client/config"
	"github.com/xhrobj/gopherkeeper/internal/client/failure"
	recordmodel "github.com/xhrobj/gopherkeeper/internal/model"
)

// SyncSummary содержит безопасные счётчики завершённой синхронизации.
type SyncSummary struct {
	// Added — количество новых записей в кеше.
	Added int

	// Updated — количество обновлённых записей в кеше.
	Updated int

	// Removed — количество удалённых из кеша записей.
	Removed int

	// Unchanged — количество неизменившихся записей.
	Unchanged int
}

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

	OpenCache(context.Context, string, string) ([]recordmodel.RecordMetadata, error)
	GetCachedRecord(context.Context, string) (recordmodel.Record, error)
	CloseCache()

	CreateRecord(context.Context, string, recordmodel.RecordPayload) (recordmodel.Record, error)
	UpdateRecord(context.Context, string, int64, string, recordmodel.RecordPayload) (recordmodel.Record, error)
	DeleteRecord(context.Context, string, int64) error

	Sync(context.Context, string) (SyncSummary, error)
}

// backendTransportCloser закрывает только сетевой transport Backend, не затрагивая локальный кеш.
type backendTransportCloser interface {
	CloseTransport() error
}

// BackendFactory создаёт runtime-зависимости TUI для переданной конфигурации.
type BackendFactory func(config.Config) (Backend, error)

func closeBackendTransport(backend Backend) {
	closer, ok := backend.(backendTransportCloser)
	if !ok || closer == nil {
		return
	}

	_ = closer.CloseTransport()
}

func createBackend(factory BackendFactory, cfg config.Config) (Backend, error) {
	if factory == nil {
		return nil, errors.New("TUI backend factory is required")
	}

	backend, err := factory(cfg)
	if err != nil {
		return nil, failure.Wrap(
			failure.KindOf(err),
			"create TUI backend",
			"Backend creation failed",
			err,
		)
	}

	if backend == nil {
		return nil, failure.Wrap(
			failure.Unknown,
			"create TUI backend: factory returned nil backend",
			"Backend creation failed",
			nil,
		)
	}

	return backend, nil
}
