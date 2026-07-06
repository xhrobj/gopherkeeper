package cli

import (
	"context"
	"fmt"
	"io"

	urfavecli "github.com/urfave/cli/v3"
	"github.com/xhrobj/gopherkeeper/internal/client/config"
	"github.com/xhrobj/gopherkeeper/internal/client/usecase"
	"github.com/xhrobj/gopherkeeper/internal/model"
)

type currentUserGetter interface {
	Whoami(ctx context.Context) (model.User, error)
}

func newWhoamiCommand(whoami outputRunner) *urfavecli.Command {
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
	application, err := usecase.New(cfg)
	if err != nil {
		return err
	}

	return executeWhoami(ctx, application, output)
}

func executeWhoami(
	ctx context.Context,
	getter currentUserGetter,
	output io.Writer,
) error {
	user, err := getter.Whoami(ctx)
	if err != nil {
		return err
	}

	if _, err := fmt.Fprintf(output, "%s\n", user.Login); err != nil {
		return fmt.Errorf("write current user: %w", err)
	}

	return nil
}
