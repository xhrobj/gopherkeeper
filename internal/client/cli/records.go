package cli

import (
	"context"
	"errors"
	"fmt"
	"io"
	"os"
	"text/tabwriter"
	"time"

	urfavecli "github.com/urfave/cli/v3"
	"github.com/xhrobj/gopherkeeper/internal/client/config"
	"github.com/xhrobj/gopherkeeper/internal/client/usecase"
	"github.com/xhrobj/gopherkeeper/internal/model"
)

const (
	titleFlag        = "title"
	textFileFlag     = "text-file"
	metadataFileFlag = "metadata-file"
)

type textRecordCreator interface {
	CreateTextRecord(ctx context.Context, request usecase.CreateTextRecordRequest) (usecase.TextRecord, error)
}

type recordLister interface {
	ListRecords(ctx context.Context) ([]model.RecordMetadata, error)
}

type textRecordGetter interface {
	GetTextRecord(ctx context.Context, recordID string) (usecase.TextRecord, error)
}

func newRecordsCommand(runners commandRunners) *urfavecli.Command {
	return &urfavecli.Command{
		Name:  "records",
		Usage: "manage private records",
		Commands: []*urfavecli.Command{
			newCreateTextRecordCommand(runners.createTextRecord),
			newListRecordsCommand(runners.listRecords),
			newGetRecordCommand(runners.getRecord),
		},
	}
}

func newCreateTextRecordCommand(create textRecordCreateRunner) *urfavecli.Command {
	return &urfavecli.Command{
		Name:  "create-text",
		Usage: "create a private text record",
		Flags: []urfavecli.Flag{
			&urfavecli.StringFlag{
				Name:     titleFlag,
				Usage:    "record title",
				Required: true,
			},
			&urfavecli.StringFlag{
				Name:     textFileFlag,
				Usage:    "path to file with private text payload",
				Required: true,
			},
			&urfavecli.StringFlag{
				Name:  metadataFileFlag,
				Usage: "path to optional file with private metadata",
			},
		},
		Action: func(ctx context.Context, command *urfavecli.Command) error {
			return create(
				ctx,
				configFromCommand(command),
				command.Root().Writer,
				command.String(titleFlag),
				command.String(textFileFlag),
				command.String(metadataFileFlag),
			)
		},
	}
}

func newListRecordsCommand(list outputRunner) *urfavecli.Command {
	return &urfavecli.Command{
		Name:  "list",
		Usage: "list private record metadata",
		Action: func(ctx context.Context, command *urfavecli.Command) error {
			return list(ctx, configFromCommand(command), command.Root().Writer)
		},
	}
}

func newGetRecordCommand(get recordGetRunner) *urfavecli.Command {
	return &urfavecli.Command{
		Name:      "get",
		Usage:     "get a private text record",
		ArgsUsage: "<record-id>",
		Action: func(ctx context.Context, command *urfavecli.Command) error {
			recordID := command.Args().First()
			if recordID == "" {
				return errors.New("record id is required")
			}

			return get(ctx, configFromCommand(command), command.Root().Writer, recordID)
		},
	}
}

func runCreateTextRecord(
	ctx context.Context,
	cfg config.Config,
	output io.Writer,
	title string,
	textFile string,
	metadataFile string,
) error {
	application, err := usecase.New(cfg)
	if err != nil {
		return err
	}

	return executeCreateTextRecord(ctx, application, output, title, textFile, metadataFile)
}

func runListRecords(
	ctx context.Context,
	cfg config.Config,
	output io.Writer,
) error {
	application, err := usecase.New(cfg)
	if err != nil {
		return err
	}

	return executeListRecords(ctx, application, output)
}

func runGetRecord(
	ctx context.Context,
	cfg config.Config,
	output io.Writer,
	recordID string,
) error {
	application, err := usecase.New(cfg)
	if err != nil {
		return err
	}

	return executeGetRecord(ctx, application, output, recordID)
}

func executeCreateTextRecord(
	ctx context.Context,
	creator textRecordCreator,
	output io.Writer,
	title string,
	textFile string,
	metadataFile string,
) error {
	text, err := readRequiredTextFile(textFile)
	if err != nil {
		return err
	}

	metadata, err := readOptionalTextFile(metadataFile)
	if err != nil {
		return err
	}

	record, err := creator.CreateTextRecord(ctx, usecase.CreateTextRecordRequest{
		Title:    title,
		Text:     text,
		Metadata: metadata,
	})
	if err != nil {
		return err
	}

	if _, err := fmt.Fprintf(
		output,
		"Created text record %s with revision %d.\n",
		record.Metadata.ID,
		record.Metadata.Revision,
	); err != nil {
		return fmt.Errorf("write created text record: %w", err)
	}

	return nil
}

func executeListRecords(ctx context.Context, lister recordLister, output io.Writer) error {
	records, err := lister.ListRecords(ctx)
	if err != nil {
		return err
	}

	if len(records) == 0 {
		if _, err := fmt.Fprintln(output, "No records found."); err != nil {
			return fmt.Errorf("write empty record list: %w", err)
		}

		return nil
	}

	writer := tabwriter.NewWriter(output, 0, 0, 2, ' ', 0)
	if _, err := fmt.Fprintln(writer, "ID\tTYPE\tTITLE\tREVISION\tUPDATED AT"); err != nil {
		return fmt.Errorf("write record list header: %w", err)
	}
	for _, record := range records {
		if _, err := fmt.Fprintf(
			writer,
			"%s\t%s\t%s\t%d\t%s\n",
			record.ID,
			record.Type,
			record.Title,
			record.Revision,
			formatRecordTime(record.UpdatedAt),
		); err != nil {
			return fmt.Errorf("write record list item: %w", err)
		}
	}
	if err := writer.Flush(); err != nil {
		return fmt.Errorf("flush record list: %w", err)
	}

	return nil
}

func executeGetRecord(ctx context.Context, getter textRecordGetter, output io.Writer, recordID string) error {
	record, err := getter.GetTextRecord(ctx, recordID)
	if err != nil {
		return err
	}

	if _, err := fmt.Fprintf(
		output,
		"ID: %s\nType: %s\nTitle: %s\nRevision: %d\nCreated at: %s\nUpdated at: %s\n\nText:\n%s\n",
		record.Metadata.ID,
		record.Metadata.Type,
		record.Metadata.Title,
		record.Metadata.Revision,
		formatRecordTime(record.Metadata.CreatedAt),
		formatRecordTime(record.Metadata.UpdatedAt),
		record.Payload.Text,
	); err != nil {
		return fmt.Errorf("write text record: %w", err)
	}

	if record.Payload.Metadata != "" {
		if _, err := fmt.Fprintf(output, "\nMetadata:\n%s\n", record.Payload.Metadata); err != nil {
			return fmt.Errorf("write text record metadata: %w", err)
		}
	}

	return nil
}

func readRequiredTextFile(path string) (string, error) {
	return readLimitedTextFile(path, "text file", model.TextPayloadMaxSize)
}

func readOptionalTextFile(path string) (string, error) {
	if path == "" {
		return "", nil
	}

	return readLimitedTextFile(path, "metadata file", model.MetadataMaxSize)
}

func readLimitedTextFile(path string, description string, maxSize int64) (string, error) {
	info, err := os.Stat(path)
	if err != nil {
		return "", fmt.Errorf("stat %s: %w", description, err)
	}
	if info.IsDir() {
		return "", fmt.Errorf("%s is a directory", description)
	}
	if info.Size() > maxSize {
		return "", fmt.Errorf("%s is too large: %w", description, model.ErrPayloadTooLarge)
	}

	data, err := os.ReadFile(path)
	if err != nil {
		return "", fmt.Errorf("read %s: %w", description, err)
	}

	return string(data), nil
}

func formatRecordTime(value time.Time) string {
	return value.UTC().Format(time.RFC3339)
}
