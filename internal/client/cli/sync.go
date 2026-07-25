package cli

import (
	"context"
	"fmt"
	"io"

	urfavecli "github.com/urfave/cli/v3"
	"github.com/xhrobj/gopherkeeper/internal/client/usecase"
)

func newSyncCommand(
	input io.Reader,
	factory clientFactory,
	passwords passwordReader,
) *urfavecli.Command {
	return &urfavecli.Command{
		Name:  "sync",
		Usage: "synchronize the encrypted local cache with the Server",
		Action: func(ctx context.Context, command *urfavecli.Command) error {
			application, err := applicationFromCommand(command, factory)
			if err != nil {
				return err
			}

			return executeSync(
				ctx,
				application,
				passwords,
				passwordStreams{
					input:        input,
					output:       command.Root().Writer,
					promptOutput: command.Root().ErrWriter,
				},
			)
		},
	}
}

func executeSync(
	ctx context.Context,
	application application,
	passwords passwordReader,
	streams passwordStreams,
) error {
	password, err := passwords.ReadHidden(streams.input, streams.promptOutput, "Password: ")
	if err != nil {
		return err
	}

	result, err := application.Sync(ctx, usecase.SyncRequest{Password: password})
	if err != nil {
		return err
	}

	return writeSyncResult(streams.output, result)
}

func writeSyncResult(output io.Writer, result usecase.SyncResult) error {
	if _, err := fmt.Fprintln(output, "Cache synchronization completed."); err != nil {
		return fmt.Errorf("write synchronization result: %w", err)
	}
	if _, err := fmt.Fprintf(
		output,
		"Added: %d\nUpdated: %d\nRemoved: %d\nUnchanged: %d\n",
		len(result.Added),
		len(result.Updated),
		len(result.Removed),
		result.Unchanged,
	); err != nil {
		return fmt.Errorf("write synchronization summary: %w", err)
	}

	return nil
}
