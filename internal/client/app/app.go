package app

import (
	"fmt"

	"github.com/xhrobj/gopherkeeper/internal/client/config"
	"github.com/xhrobj/gopherkeeper/internal/client/grpcclient"
	"github.com/xhrobj/gopherkeeper/internal/client/httpclient"
	"github.com/xhrobj/gopherkeeper/internal/client/session"
	"github.com/xhrobj/gopherkeeper/internal/client/usecase"
)

// NewRuntime создаёт online application и выбранный удалённый transport.
func NewRuntime(cfg config.Config) (*Runtime, error) {
	switch cfg.Transport {
	case config.TransportHTTPS:
		client, err := httpclient.New(cfg.Address, cfg.CACertFile)
		if err != nil {
			return nil, err
		}
		return newRuntime(newApplication(cfg, client, client, client), client)
	case config.TransportGRPC:
		client, err := grpcclient.New(cfg.GRPCAddress, cfg.CACertFile)
		if err != nil {
			return nil, err
		}
		return newRuntime(newApplication(cfg, client, client, client), client)
	default:
		return nil, fmt.Errorf("unsupported client transport %q", cfg.Transport)
	}
}

func newApplication(
	cfg config.Config,
	health usecase.HealthChecker,
	users usecase.UserGateway,
	records usecase.RecordGateway,
) *usecase.Application {
	return usecase.New(
		health,
		users,
		records,
		fileSessionStorageProvider(cfg.SessionDir),
		encryptedSyncCacheRepositoryProvider(cfg.CacheDir),
		encryptedOfflineCacheRepositoryProvider(cfg.CacheDir),
	)
}

// NewOffline создаёт клиентское application-приложение только для чтения
// существующего зашифрованного локального кеша.
func NewOffline(cfg config.Config) *usecase.OfflineApplication {
	return usecase.NewOffline(
		encryptedOfflineCacheRepositoryProvider(cfg.CacheDir),
	)
}

// NewLogout создаёт application-сценарий локального выхода.
func NewLogout(cfg config.Config) (*usecase.LogoutApplication, error) {
	storage, err := session.NewFileStorage(cfg.SessionDir)
	if err != nil {
		return nil, fmt.Errorf("create online session storage: %w", err)
	}

	return usecase.NewLogout(storage), nil
}

func fileSessionStorageProvider(directory string) usecase.SessionStorageProvider {
	return func() (usecase.SessionStorage, error) {
		storage, err := session.NewFileStorage(directory)
		if err != nil {
			return nil, fmt.Errorf("create online session storage: %w", err)
		}

		return storage, nil
	}
}
