package cli

import (
	"context"
	"fmt"
	"io"
	"sync"

	urfavecli "github.com/urfave/cli/v3"
	"github.com/xhrobj/gopherkeeper/internal/buildinfo"
	"github.com/xhrobj/gopherkeeper/internal/client/app"
	"github.com/xhrobj/gopherkeeper/internal/client/config"
	"github.com/xhrobj/gopherkeeper/internal/client/failure"
	"github.com/xhrobj/gopherkeeper/internal/client/tui"
	"github.com/xhrobj/gopherkeeper/internal/client/usecase"
	"github.com/xhrobj/gopherkeeper/internal/model"
)

type tuiRunner interface {
	Run(
		ctx context.Context,
		cfg config.Config,
		configFile string,
		info buildinfo.Info,
		input io.Reader,
		output io.Writer,
	) error
}

type defaultTUIRunner struct{}

type tuiBackend struct {
	application  *usecase.Application
	cacheMu      sync.Mutex
	cacheSession *usecase.CacheSession
}

var _ tui.Backend = (*tuiBackend)(nil)

func newTUIBackend(cfg config.Config) (tui.Backend, error) {
	application, err := app.New(cfg)
	if err != nil {
		return nil, failure.Context("create client application", err)
	}

	return &tuiBackend{application: application}, nil
}

func (backend *tuiBackend) Health(ctx context.Context) (string, error) {
	return backend.application.Health(ctx)
}

func (backend *tuiBackend) Register(ctx context.Context, login, password string) (string, error) {
	user, err := backend.application.Register(ctx, login, password)
	if err != nil {
		return "", err
	}

	return user.Login, nil
}

func (backend *tuiBackend) Login(ctx context.Context, login, password string) (string, error) {
	user, err := backend.application.Login(ctx, login, password)
	if err != nil {
		return "", err
	}

	return user.Login, nil
}

func (backend *tuiBackend) CurrentUser(ctx context.Context) (string, error) {
	user, err := backend.application.Whoami(ctx)
	if err != nil {
		return "", err
	}

	return user.Login, nil
}

func (backend *tuiBackend) Logout(ctx context.Context) error {
	return backend.application.Logout(ctx)
}

func (backend *tuiBackend) ListRecords(ctx context.Context) ([]model.RecordMetadata, error) {
	return backend.application.ListRecords(ctx)
}

func (backend *tuiBackend) GetRecord(ctx context.Context, recordID string) (model.Record, error) {
	return backend.application.GetRecord(ctx, recordID)
}

func (backend *tuiBackend) OpenCache(
	ctx context.Context,
	login string,
	password string,
) ([]model.RecordMetadata, error) {
	backend.cacheMu.Lock()
	defer backend.cacheMu.Unlock()

	backend.closeCacheLocked()

	session, err := backend.application.OpenCacheSession(ctx, usecase.OfflineReadRequest{
		Login: login, Password: password,
	})
	if err != nil {
		return nil, err
	}

	records, err := session.ListRecords(ctx)
	if err != nil {
		_ = session.Close()
		return nil, err
	}
	if err := ctx.Err(); err != nil {
		_ = session.Close()
		return nil, err
	}

	backend.cacheSession = session
	return records, nil
}

func (backend *tuiBackend) GetCachedRecord(ctx context.Context, recordID string) (model.Record, error) {
	backend.cacheMu.Lock()
	defer backend.cacheMu.Unlock()

	if backend.cacheSession == nil {
		return model.Record{}, fmt.Errorf("local cache is not open")
	}

	return backend.cacheSession.GetRecord(ctx, recordID)
}

func (backend *tuiBackend) CloseCache() {
	backend.cacheMu.Lock()
	defer backend.cacheMu.Unlock()

	backend.closeCacheLocked()
}

func (backend *tuiBackend) closeCacheLocked() {
	if backend.cacheSession == nil {
		return
	}

	_ = backend.cacheSession.Close()
	backend.cacheSession = nil
}

func (backend *tuiBackend) CreateRecord(ctx context.Context, title string, payload model.RecordPayload) (model.Record, error) {
	return backend.application.CreateRecord(ctx, usecase.CreateRecordRequest{Title: title, Payload: payload})
}

func (backend *tuiBackend) UpdateRecord(
	ctx context.Context,
	recordID string,
	expectedRevision int64,
	title string,
	payload model.RecordPayload,
) (model.Record, error) {
	return backend.application.UpdateRecord(ctx, usecase.UpdateRecordRequest{
		RecordID: recordID, ExpectedRevision: expectedRevision, Title: title, Payload: payload,
	})
}

func (backend *tuiBackend) DeleteRecord(ctx context.Context, recordID string, expectedRevision int64) error {
	return backend.application.DeleteRecord(ctx, usecase.DeleteRecordRequest{
		RecordID: recordID, ExpectedRevision: expectedRevision,
	})
}

func (backend *tuiBackend) Sync(ctx context.Context, password string) (tui.SyncSummary, error) {
	result, err := backend.application.Sync(ctx, usecase.SyncRequest{Password: password})
	if err != nil {
		return tui.SyncSummary{}, err
	}

	return tui.SyncSummary{
		Added:     len(result.Added),
		Updated:   len(result.Updated),
		Removed:   len(result.Removed),
		Unchanged: result.Unchanged,
	}, nil
}

func (defaultTUIRunner) Run(
	ctx context.Context,
	cfg config.Config,
	configFile string,
	info buildinfo.Info,
	input io.Reader,
	output io.Writer,
) error {
	return tui.Run(ctx, tui.Options{
		Input:          input,
		Output:         output,
		Config:         cfg,
		ConfigFile:     configFile,
		Info:           info,
		BackendFactory: newTUIBackend,
	})
}

func newTUICommand(
	input io.Reader,
	output io.Writer,
	info buildinfo.Info,
	runner tuiRunner,
) *urfavecli.Command {
	return &urfavecli.Command{
		Name:  "tui",
		Usage: "open the interactive terminal interface",
		Action: func(ctx context.Context, command *urfavecli.Command) error {
			cfg, err := configFromCommand(command)
			if err != nil {
				return err
			}

			configFile, err := configFileFromCommand(command)
			if err != nil {
				return err
			}

			if err := runner.Run(ctx, cfg, configFile, info, input, output); err != nil {
				return fmt.Errorf("run TUI: %w", err)
			}

			return nil
		},
	}
}
