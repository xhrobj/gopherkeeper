package cli

import (
	"context"
	"fmt"
	"io"

	urfavecli "github.com/urfave/cli/v3"
	"github.com/xhrobj/gopherkeeper/internal/buildinfo"
	"github.com/xhrobj/gopherkeeper/internal/client/app"
	"github.com/xhrobj/gopherkeeper/internal/client/config"
	"github.com/xhrobj/gopherkeeper/internal/client/httpclient"
	"github.com/xhrobj/gopherkeeper/internal/client/tui"
	"github.com/xhrobj/gopherkeeper/internal/client/usecase"
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
	healthClient      *httpclient.Client
	healthClientError error
	application       *usecase.Application
	applicationError  error
	logoutApplication *usecase.LogoutApplication
	logoutError       error
}

func newTUIBackend(cfg config.Config) tui.Backend {
	healthClient, healthClientError := httpclient.New(cfg.Address, cfg.CACertFile)
	application, applicationError := app.New(cfg)
	logoutApplication, logoutError := app.NewLogout(cfg)

	return &tuiBackend{
		healthClient:      healthClient,
		healthClientError: healthClientError,
		application:       application,
		applicationError:  applicationError,
		logoutApplication: logoutApplication,
		logoutError:       logoutError,
	}
}

func (backend *tuiBackend) Health(ctx context.Context) (string, error) {
	client := backend.healthClient
	err := backend.healthClientError

	if err != nil {
		return "", err
	}

	return client.Health(ctx)
}

func (backend *tuiBackend) Register(
	ctx context.Context,
	login string,
	password string,
) (string, error) {
	application, err := backend.onlineApplication()
	if err != nil {
		return "", err
	}

	user, err := application.Register(ctx, login, password)
	if err != nil {
		return "", err
	}

	return user.Login, nil
}

func (backend *tuiBackend) Login(
	ctx context.Context,
	login string,
	password string,
) (string, error) {
	application, err := backend.onlineApplication()
	if err != nil {
		return "", err
	}

	user, err := application.Login(ctx, login, password)
	if err != nil {
		return "", err
	}

	return user.Login, nil
}

func (backend *tuiBackend) CurrentUser(ctx context.Context) (string, error) {
	application, err := backend.onlineApplication()
	if err != nil {
		return "", err
	}

	user, err := application.Whoami(ctx)
	if err != nil {
		return "", err
	}

	return user.Login, nil
}

func (backend *tuiBackend) Logout(ctx context.Context) error {
	application := backend.logoutApplication
	err := backend.logoutError

	if err != nil {
		return err
	}

	return application.Logout(ctx)
}

func (backend *tuiBackend) onlineApplication() (*usecase.Application, error) {
	application := backend.application
	err := backend.applicationError

	return application, err
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
