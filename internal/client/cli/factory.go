package cli

import (
	"context"

	urfavecli "github.com/urfave/cli/v3"
	"github.com/xhrobj/gopherkeeper/internal/client/app"
	"github.com/xhrobj/gopherkeeper/internal/client/config"
	"github.com/xhrobj/gopherkeeper/internal/client/httpclient"
	"github.com/xhrobj/gopherkeeper/internal/client/usecase"
	"github.com/xhrobj/gopherkeeper/internal/model"
)

type application interface {
	Register(ctx context.Context, login, password string) (model.User, error)
	Login(ctx context.Context, login, password string) (model.User, error)
	Whoami(ctx context.Context) (model.User, error)
	CreateRecord(ctx context.Context, request usecase.CreateRecordRequest) (model.Record, error)
	UpdateRecord(ctx context.Context, request usecase.UpdateRecordRequest) (model.Record, error)
	ListRecords(ctx context.Context) ([]model.RecordMetadata, error)
	GetRecord(ctx context.Context, recordID string) (model.Record, error)
	ListCachedRecords(ctx context.Context, request usecase.OfflineReadRequest) (usecase.OfflineListResult, error)
	GetCachedRecord(
		ctx context.Context,
		request usecase.OfflineReadRequest,
		recordID string,
	) (usecase.OfflineGetResult, error)
	DeleteRecord(ctx context.Context, request usecase.DeleteRecordRequest) error
	Sync(ctx context.Context, request usecase.SyncRequest) (usecase.SyncResult, error)
}

type userLogoutter interface {
	Logout(ctx context.Context) error
}

type healthChecker interface {
	Health(ctx context.Context) (string, error)
}

type clientFactory interface {
	NewApplication(cfg config.Config) (application, error)
	NewOfflineApplication(cfg config.Config) (application, error)
	NewLogoutApplication(cfg config.Config) (userLogoutter, error)
	NewHealthClient(cfg config.Config) (healthChecker, error)
}

type defaultClientFactory struct{}

// NewApplication создаёт application для online-сценариев Клиента.
func (defaultClientFactory) NewApplication(cfg config.Config) (application, error) {
	return app.New(cfg)
}

// NewOfflineApplication создаёт application для offline read-only сценариев.
func (defaultClientFactory) NewOfflineApplication(cfg config.Config) (application, error) {
	return app.NewOffline(cfg), nil
}

// NewLogoutApplication создаёт application для локального выхода из online-сессии.
func (defaultClientFactory) NewLogoutApplication(cfg config.Config) (userLogoutter, error) {
	return app.NewLogout(cfg)
}

// NewHealthClient создаёт Клиент для проверки доступности Сервера.
func (defaultClientFactory) NewHealthClient(cfg config.Config) (healthChecker, error) {
	return httpclient.New(cfg.Address, cfg.CACertFile)
}

func applicationFromCommand(command *urfavecli.Command, factory clientFactory) (application, error) {
	cfg, err := configFromCommand(command)
	if err != nil {
		return nil, err
	}

	return factory.NewApplication(cfg)
}

func offlineApplicationFromCommand(command *urfavecli.Command, factory clientFactory) (application, error) {
	cfg, err := configFromCommand(command)
	if err != nil {
		return nil, err
	}

	return factory.NewOfflineApplication(cfg)
}
