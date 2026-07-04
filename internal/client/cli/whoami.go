package cli

import (
	"context"
	"errors"
	"fmt"
	"io"
	"net/http"

	urfavecli "github.com/urfave/cli/v3"
	"github.com/xhrobj/gopherkeeper/internal/client/config"
	"github.com/xhrobj/gopherkeeper/internal/client/httpclient"
	"github.com/xhrobj/gopherkeeper/internal/client/session"
	"github.com/xhrobj/gopherkeeper/internal/model"
)

type whoamiRunner func(context.Context, config.Config, io.Writer) error

type currentUserGetter interface {
	CurrentUser(ctx context.Context, accessToken string) (model.User, error)
}

type sessionLoader interface {
	Load(expectedServerAddress string) (session.Session, error)
}

func newWhoamiCommand(whoami whoamiRunner) *urfavecli.Command {
	return &urfavecli.Command{
		Name:  "whoami",
		Usage: "show the authenticated user",
		Action: func(ctx context.Context, command *urfavecli.Command) error {
			return whoami(ctx, configFromCommand(command), command.Root().Writer)
		},
	}
}

func runWhoami(
	ctx context.Context,
	cfg config.Config,
	output io.Writer,
) error {
	client, err := httpclient.New(cfg.Address, cfg.CACertFile)
	if err != nil {
		return err
	}

	storage, err := session.NewFileStorage(cfg.SessionFile)
	if err != nil {
		return err
	}

	return executeWhoami(ctx, client, storage, output, cfg.Address)
}

func executeWhoami(
	ctx context.Context,
	getter currentUserGetter,
	sessions sessionLoader,
	output io.Writer,
	serverAddress string,
) error {
	storedSession, err := sessions.Load(serverAddress)
	if err != nil {
		return mapSessionLoadError(err)
	}

	user, err := getter.CurrentUser(ctx, storedSession.AccessToken)
	if err != nil {
		return mapCurrentUserError(err)
	}

	if _, err := fmt.Fprintf(output, "%s\n", user.Login); err != nil {
		return fmt.Errorf("write current user: %w", err)
	}

	return nil
}

func mapSessionLoadError(err error) error {
	switch {
	case errors.Is(err, session.ErrNotFound):
		return fmt.Errorf("online session not found: run gkeep login: %w", err)
	case errors.Is(err, session.ErrExpired):
		return fmt.Errorf("online session expired: run gkeep login: %w", err)
	case errors.Is(err, session.ErrServerMismatch):
		return fmt.Errorf("online session belongs to another server: run gkeep login: %w", err)
	case errors.Is(err, session.ErrInvalid):
		return fmt.Errorf("online session is invalid: run gkeep login: %w", err)
	default:
		return fmt.Errorf("load online session: %w", err)
	}
}

func mapCurrentUserError(err error) error {
	var apiError *httpclient.APIError
	if errors.As(err, &apiError) && apiError.StatusCode == http.StatusUnauthorized && apiError.Code == "unauthorized" {
		return fmt.Errorf("online session is invalid or expired: run gkeep login: %w", err)
	}

	return fmt.Errorf("get current user: %w", err)
}
