package cli

import (
	"context"
	"fmt"
	"io"

	"github.com/urfave/cli/v3"
	"github.com/xhrobj/gopherkeeper/internal/client/config"
	"github.com/xhrobj/gopherkeeper/internal/client/httpclient"
)

func newHealthCommand(health healthRunner) *cli.Command {
	return &cli.Command{
		Name:  "health",
		Usage: "check Server availability",
		Action: func(ctx context.Context, command *cli.Command) error {
			return health(ctx, configFromCommand(command), command.Root().Writer)
		},
	}
}

func runHealth(
	ctx context.Context,
	cfg config.Config,
	output io.Writer,
) error {
	client, err := httpclient.New(cfg.Address, cfg.CACertFile)
	if err != nil {
		return err
	}

	status, err := client.Health(ctx)
	if err != nil {
		return err
	}

	if _, err := fmt.Fprintf(output, "Server status: %s\n", status); err != nil {
		return fmt.Errorf("write health status: %w", err)
	}

	return nil
}
