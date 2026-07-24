package cli

import (
	"context"
	"fmt"
	"io"

	urfavecli "github.com/urfave/cli/v3"
	"github.com/xhrobj/gopherkeeper/internal/buildinfo"
	"github.com/xhrobj/gopherkeeper/internal/client/app"
	"github.com/xhrobj/gopherkeeper/internal/client/config"
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
	application *usecase.Application
}

var _ tui.Backend = (*tuiBackend)(nil)

func newTUIBackend(cfg config.Config) (tui.Backend, error) {
	application, err := app.New(cfg)
	if err != nil {
		return nil, fmt.Errorf("create client application: %w", err)
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
