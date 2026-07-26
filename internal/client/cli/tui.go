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

type tuiCacheState struct {
	mu      sync.Mutex
	session *usecase.CacheSession
}

type tuiBackend struct {
	runtime *app.Runtime
	cache   *tuiCacheState
}

var _ tui.Backend = (*tuiBackend)(nil)

func newTUIBackend(cfg config.Config) (tui.Backend, error) {
	return newTUIBackendWithCache(cfg, &tuiCacheState{})
}

func newTUIBackendFactory() tui.BackendFactory {
	cache := &tuiCacheState{}

	return func(cfg config.Config) (tui.Backend, error) {
		return newTUIBackendWithCache(cfg, cache)
	}
}

func newTUIBackendWithCache(cfg config.Config, cache *tuiCacheState) (tui.Backend, error) {
	if cache == nil {
		cache = &tuiCacheState{}
	}

	if cfg.Transport == "" {
		cfg.Transport = config.TransportHTTPS
	}

	runtime, err := app.NewRuntime(cfg)
	if err != nil {
		return nil, failure.Context("create client application", err)
	}

	return &tuiBackend{runtime: runtime, cache: cache}, nil
}

func (backend *tuiBackend) Health(ctx context.Context) (string, error) {
	return backend.runtime.Health(ctx)
}

func (backend *tuiBackend) Register(ctx context.Context, login, password string) (string, error) {
	user, err := backend.runtime.Register(ctx, login, password)
	if err != nil {
		return "", err
	}

	return user.Login, nil
}

func (backend *tuiBackend) Login(ctx context.Context, login, password string) (string, error) {
	user, err := backend.runtime.Login(ctx, login, password)
	if err != nil {
		return "", err
	}

	return user.Login, nil
}

func (backend *tuiBackend) CurrentUser(ctx context.Context) (string, error) {
	user, err := backend.runtime.Whoami(ctx)
	if err != nil {
		return "", err
	}

	return user.Login, nil
}

func (backend *tuiBackend) Logout(ctx context.Context) error {
	return backend.runtime.Logout(ctx)
}

func (backend *tuiBackend) ListRecords(ctx context.Context) ([]model.RecordMetadata, error) {
	return backend.runtime.ListRecords(ctx)
}

func (backend *tuiBackend) GetRecord(ctx context.Context, recordID string) (model.Record, error) {
	return backend.runtime.GetRecord(ctx, recordID)
}

func (backend *tuiBackend) OpenCache(
	ctx context.Context,
	login string,
	password string,
) ([]model.RecordMetadata, error) {
	backend.cache.mu.Lock()
	defer backend.cache.mu.Unlock()

	backend.closeCacheLocked()

	session, err := backend.runtime.OpenCacheSession(ctx, usecase.OfflineReadRequest{
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

	backend.cache.session = session
	return records, nil
}

func (backend *tuiBackend) GetCachedRecord(ctx context.Context, recordID string) (model.Record, error) {
	backend.cache.mu.Lock()
	defer backend.cache.mu.Unlock()

	if backend.cache.session == nil {
		return model.Record{}, fmt.Errorf("local cache is not open")
	}

	return backend.cache.session.GetRecord(ctx, recordID)
}

func (backend *tuiBackend) CloseCache() {
	backend.cache.mu.Lock()
	defer backend.cache.mu.Unlock()

	backend.closeCacheLocked()
}

func (backend *tuiBackend) closeCacheLocked() {
	if backend.cache.session == nil {
		return
	}

	_ = backend.cache.session.Close()
	backend.cache.session = nil
}

func (backend *tuiBackend) CreateRecord(ctx context.Context, title string, payload model.RecordPayload) (model.Record, error) {
	return backend.runtime.CreateRecord(ctx, usecase.CreateRecordRequest{Title: title, Payload: payload})
}

func (backend *tuiBackend) UpdateRecord(
	ctx context.Context,
	recordID string,
	expectedRevision int64,
	title string,
	payload model.RecordPayload,
) (model.Record, error) {
	return backend.runtime.UpdateRecord(ctx, usecase.UpdateRecordRequest{
		RecordID: recordID, ExpectedRevision: expectedRevision, Title: title, Payload: payload,
	})
}

func (backend *tuiBackend) DeleteRecord(ctx context.Context, recordID string, expectedRevision int64) error {
	return backend.runtime.DeleteRecord(ctx, usecase.DeleteRecordRequest{
		RecordID: recordID, ExpectedRevision: expectedRevision,
	})
}

func (backend *tuiBackend) Sync(ctx context.Context, password string) (tui.SyncSummary, error) {
	result, err := backend.runtime.Sync(ctx, usecase.SyncRequest{Password: password})
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

func (backend *tuiBackend) CloseTransport() error {
	if backend == nil || backend.runtime == nil {
		return nil
	}

	return backend.runtime.Close()
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
		BackendFactory: newTUIBackendFactory(),
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
