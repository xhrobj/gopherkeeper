package app

import (
	"fmt"

	"github.com/xhrobj/gopherkeeper/internal/client/config"
	"github.com/xhrobj/gopherkeeper/internal/client/httpclient"
	"github.com/xhrobj/gopherkeeper/internal/client/session"
	"github.com/xhrobj/gopherkeeper/internal/client/usecase"
)

// New создаёт клиентское application-приложение из runtime-конфигурации.
func New(cfg config.Config) (*usecase.Application, error) {
	client, err := httpclient.New(cfg.Address, cfg.CACertFile)
	if err != nil {
		return nil, err
	}

	return usecase.New(
		client,
		client,
		fileSessionStorageProvider(cfg.SessionFile),
		encryptedSyncCacheRepositoryProvider(cfg.CacheDir),
		encryptedOfflineCacheRepositoryProvider(cfg.CacheDir),
		cfg.Address,
	), nil
}

// NewLogout создаёт application-сценарий локального выхода.
func NewLogout(cfg config.Config) (*usecase.LogoutApplication, error) {
	storage, err := session.NewFileStorage(cfg.SessionFile)
	if err != nil {
		return nil, fmt.Errorf("create online session storage: %w", err)
	}

	return usecase.NewLogout(storage), nil
}

func fileSessionStorageProvider(path string) usecase.SessionStorageProvider {
	return func() (usecase.SessionStorage, error) {
		storage, err := session.NewFileStorage(path)
		if err != nil {
			return nil, fmt.Errorf("create online session storage: %w", err)
		}

		return storage, nil
	}
}
