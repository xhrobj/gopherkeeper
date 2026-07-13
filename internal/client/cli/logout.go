package cli

import (
	"context"
	"fmt"
	"io"

	urfavecli "github.com/urfave/cli/v3"
)

func newLogoutCommand(factory clientFactory) *urfavecli.Command {
	return &urfavecli.Command{
		Name:  "logout",
		Usage: "clear local online session",
		Action: func(ctx context.Context, command *urfavecli.Command) error {
			cfg, err := configFromCommand(command)
			if err != nil {
				return err
			}

			application, err := factory.NewLogoutApplication(cfg)
			if err != nil {
				return err
			}

			return executeLogout(ctx, application, command.Root().Writer)
		},
	}
}

func executeLogout(ctx context.Context, application logoutApplication, output io.Writer) error {
	if err := application.Logout(ctx); err != nil {
		return err
	}

	if _, err := fmt.Fprintln(output, "logged out"); err != nil {
		return fmt.Errorf("write logout status: %w", err)
	}

	return nil
}
