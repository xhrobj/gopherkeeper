package cli

import (
	"context"

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
	DeleteRecord(ctx context.Context, request usecase.DeleteRecordRequest) error
}

type logoutApplication interface {
	Logout(ctx context.Context) error
}

type healthClient interface {
	Health(ctx context.Context) (string, error)
}

type clientFactory interface {
	NewApplication(cfg config.Config) (application, error)
	NewLogoutApplication(cfg config.Config) (logoutApplication, error)
	NewHealthClient(cfg config.Config) (healthClient, error)
}

type defaultClientFactory struct{}

func (defaultClientFactory) NewApplication(cfg config.Config) (application, error) {
	return app.New(cfg)
}

func (defaultClientFactory) NewLogoutApplication(cfg config.Config) (logoutApplication, error) {
	return app.NewLogout(cfg)
}

func (defaultClientFactory) NewHealthClient(cfg config.Config) (healthClient, error) {
	return httpclient.New(cfg.Address, cfg.CACertFile)
}
