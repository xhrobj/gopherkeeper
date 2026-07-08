package cli

import (
	"context"
	"fmt"
	"io"

	urfavecli "github.com/urfave/cli/v3"
	"github.com/xhrobj/gopherkeeper/internal/client/config"
	"github.com/xhrobj/gopherkeeper/internal/client/usecase"
)

type userLogoutter interface {
	Logout(ctx context.Context) error
}

func newLogoutCommand(logout outputRunner) *urfavecli.Command {
	return &urfavecli.Command{
		Name:  "logout",
		Usage: "clear local online session",
		Action: func(ctx context.Context, command *urfavecli.Command) error {
			return logout(ctx, configFromCommand(command), command.Root().Writer)
		},
	}
}

func runLogout(
	ctx context.Context,
	cfg config.Config,
	output io.Writer,
) error {
	application := usecase.NewLocal(cfg)

	return executeLogout(ctx, application, output)
}

func executeLogout(
	ctx context.Context,
	logoutter userLogoutter,
	output io.Writer,
) error {
	if err := logoutter.Logout(ctx); err != nil {
		return err
	}

	if _, err := fmt.Fprintln(output, "logged out"); err != nil {
		return fmt.Errorf("write logout status: %w", err)
	}

	return nil
}
