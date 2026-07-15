package cli

import (
	"context"
	"fmt"
	"io"
	"text/tabwriter"

	urfavecli "github.com/urfave/cli/v3"
	"github.com/xhrobj/gopherkeeper/internal/client/usecase"
)

const refreshFlag = "refresh"

func newSyncCommand(
	input io.Reader,
	factory clientFactory,
	passwords passwordReader,
) *urfavecli.Command {
	return &urfavecli.Command{
		Name:  "sync",
		Usage: "synchronize the encrypted local cache with the Server",
		Flags: []urfavecli.Flag{
			&urfavecli.BoolFlag{
				Name:  refreshFlag,
				Usage: "replace stale cached records with current Server versions",
			},
		},
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
				command.Bool(refreshFlag),
			)
		},
	}
}

func executeSync(
	ctx context.Context,
	application application,
	passwords passwordReader,
	streams passwordStreams,
	refreshStale bool,
) error {
	password, err := passwords.ReadHidden(streams.input, streams.promptOutput, "Password: ")
	if err != nil {
		return err
	}

	result, err := application.Sync(ctx, usecase.SyncRequest{
		Password:     password,
		RefreshStale: refreshStale,
	})
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
		"Added: %d\nUpdated: %d\nRemoved: %d\nUnchanged: %d\nStale: %d\n",
		len(result.Added),
		len(result.Updated),
		len(result.Removed),
		result.Unchanged,
		len(result.Stale),
	); err != nil {
		return fmt.Errorf("write synchronization summary: %w", err)
	}

	if len(result.Stale) == 0 {
		return nil
	}

	if _, err := fmt.Fprintln(output, "\nStale records:"); err != nil {
		return fmt.Errorf("write stale record heading: %w", err)
	}

	writer := tabwriter.NewWriter(output, 0, 0, 2, ' ', 0)
	if _, err := fmt.Fprintln(
		writer,
		"ID\tTYPE\tTITLE\tLOCAL REVISION\tSERVER REVISION",
	); err != nil {
		return fmt.Errorf("write stale record header: %w", err)
	}
	for _, change := range result.Stale {
		if _, err := fmt.Fprintf(
			writer,
			"%s\t%s\t%s\t%d\t%d\n",
			change.Metadata.ID,
			change.Metadata.Type,
			change.Metadata.Title,
			change.LocalRevision,
			change.Metadata.Revision,
		); err != nil {
			return fmt.Errorf("write stale record: %w", err)
		}
	}
	if err := writer.Flush(); err != nil {
		return fmt.Errorf("flush stale records: %w", err)
	}

	if _, err := fmt.Fprintln(
		output,
		"Run `gkeep sync --refresh` to update stale records.",
	); err != nil {
		return fmt.Errorf("write stale record hint: %w", err)
	}

	return nil
}
