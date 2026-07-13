package cli

import (
	"context"
	"errors"
	"fmt"
	"io"

	urfavecli "github.com/urfave/cli/v3"
	"github.com/xhrobj/gopherkeeper/internal/client/usecase"
)

func newWhoamiCommand(factory clientFactory) *urfavecli.Command {
	return &urfavecli.Command{
		Name:  "whoami",
		Usage: "show the authenticated user",
		Action: func(ctx context.Context, command *urfavecli.Command) error {
			cfg, err := configFromCommand(command)
			if err != nil {
				return err
			}

			application, err := factory.NewApplication(cfg)
			if err != nil {
				return err
			}

			return executeWhoami(ctx, application, command.Root().Writer)
		},
	}
}

func executeWhoami(ctx context.Context, application application, output io.Writer) error {
	user, err := application.Whoami(ctx)
	if err != nil {
		if errors.Is(err, usecase.ErrNotLoggedIn) {
			if _, writeErr := fmt.Fprintf(output, "%s\n", usecase.ErrNotLoggedIn); writeErr != nil {
				return fmt.Errorf("write current user status: %w", writeErr)
			}

			return nil
		}

		return err
	}

	if _, err := fmt.Fprintf(output, "%s\n", user.Login); err != nil {
		return fmt.Errorf("write current user: %w", err)
	}

	return nil
}
