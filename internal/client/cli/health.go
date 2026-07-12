package cli

import (
	"context"
	"fmt"
	"io"

	urfavecli "github.com/urfave/cli/v3"
)

func newHealthCommand(factory clientFactory) *urfavecli.Command {
	return &urfavecli.Command{
		Name:  "health",
		Usage: "check Server availability",
		Action: func(ctx context.Context, command *urfavecli.Command) error {
			client, err := factory.NewHealthClient(configFromCommand(command))
			if err != nil {
				return err
			}

			return executeHealth(ctx, client, command.Root().Writer)
		},
	}
}

func executeHealth(ctx context.Context, client healthClient, output io.Writer) error {
	status, err := client.Health(ctx)
	if err != nil {
		return err
	}

	if _, err := fmt.Fprintf(output, "Server status: %s\n", status); err != nil {
		return fmt.Errorf("write health status: %w", err)
	}

	return nil
}
