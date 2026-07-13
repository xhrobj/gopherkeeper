package cli

import (
	"context"
	"errors"
	"fmt"
	"io"
	"text/tabwriter"

	urfavecli "github.com/urfave/cli/v3"
	"github.com/xhrobj/gopherkeeper/internal/client/usecase"
	"github.com/xhrobj/gopherkeeper/internal/model"
)

const (
	titleFlag         = "title"
	metadataFileFlag  = "metadata-file"
	outputFlag        = "output"
	revisionFlag      = "revision"
	recordIDArgsUsage = "<record-id>"
)

func newRecordsCommand(
	input io.Reader,
	factory clientFactory,
	passwords passwordReader,
) *urfavecli.Command {
	return &urfavecli.Command{
		Name:  "records",
		Usage: "manage private records",
		Commands: []*urfavecli.Command{
			newCreateTextRecordCommand(factory),
			newCreateCredentialsRecordCommand(input, factory, passwords),
			newCreateCardRecordCommand(input, factory, passwords),
			newCreateBinaryRecordCommand(factory),
			newUpdateTextRecordCommand(factory),
			newUpdateCredentialsRecordCommand(input, factory, passwords),
			newUpdateCardRecordCommand(input, factory, passwords),
			newUpdateBinaryRecordCommand(factory),
			newListRecordsCommand(factory),
			newGetRecordCommand(factory),
			newDeleteRecordCommand(factory),
		},
	}
}
func newListRecordsCommand(factory clientFactory) *urfavecli.Command {
	return &urfavecli.Command{
		Name:  "list",
		Usage: "list private record metadata",
		Action: func(ctx context.Context, command *urfavecli.Command) error {
			cfg, err := configFromCommand(command)
			if err != nil {
				return err
			}

			application, err := factory.NewApplication(cfg)
			if err != nil {
				return err
			}

			return executeListRecords(ctx, application, command.Root().Writer)
		},
	}
}
func newGetRecordCommand(factory clientFactory) *urfavecli.Command {
	return &urfavecli.Command{
		Name:      "get",
		Usage:     "get a private record",
		ArgsUsage: recordIDArgsUsage,
		Flags: []urfavecli.Flag{
			&urfavecli.StringFlag{
				Name:  outputFlag,
				Usage: "path for saving a binary record",
			},
		},
		Action: func(ctx context.Context, command *urfavecli.Command) error {
			recordID := command.Args().First()
			if recordID == "" {
				return errors.New("record id is required")
			}

			cfg, err := configFromCommand(command)
			if err != nil {
				return err
			}

			application, err := factory.NewApplication(cfg)
			if err != nil {
				return err
			}

			return executeGetRecord(
				ctx,
				application,
				command.Root().Writer,
				recordID,
				command.String(outputFlag),
			)
		},
	}
}
func newDeleteRecordCommand(factory clientFactory) *urfavecli.Command {
	return &urfavecli.Command{
		Name:      "delete",
		Usage:     "delete a private record",
		ArgsUsage: recordIDArgsUsage,
		Flags: []urfavecli.Flag{
			&urfavecli.Int64Flag{
				Name:     revisionFlag,
				Aliases:  []string{"r"},
				Usage:    "expected record revision",
				Required: true,
			},
		},
		Action: func(ctx context.Context, command *urfavecli.Command) error {
			recordID := command.Args().First()
			if recordID == "" {
				return errors.New("record id is required")
			}

			cfg, err := configFromCommand(command)
			if err != nil {
				return err
			}

			application, err := factory.NewApplication(cfg)
			if err != nil {
				return err
			}

			return executeDeleteRecord(
				ctx,
				application,
				command.Root().Writer,
				recordID,
				command.Int64(revisionFlag),
			)
		},
	}
}
func executeDeleteRecord(
	ctx context.Context,
	application application,
	output io.Writer,
	recordID string,
	expectedRevision int64,
) error {
	if err := application.DeleteRecord(ctx, usecase.DeleteRecordRequest{
		RecordID:         recordID,
		ExpectedRevision: expectedRevision,
	}); err != nil {
		return err
	}

	if _, err := fmt.Fprintf(output, "Deleted record %s.\n", recordID); err != nil {
		return fmt.Errorf("write deleted record: %w", err)
	}

	return nil
}
func executeListRecords(ctx context.Context, application application, output io.Writer) error {
	records, err := application.ListRecords(ctx)
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
func executeGetRecord(
	ctx context.Context,
	application application,
	output io.Writer,
	recordID string,
	outputPath string,
) error {
	record, err := application.GetRecord(ctx, recordID)
	if err != nil {
		return err
	}

	if record.Metadata.Type == model.RecordTypeBinary {
		payload, ok := record.Payload.(*model.BinaryPayload)
		if !ok || payload == nil {
			return errors.New("unexpected binary payload")
		}
		if outputPath == "" {
			return errors.New("output path is required for binary record")
		}
		if err := writeBinaryFile(outputPath, payload.Data); err != nil {
			return err
		}

		if err := writeRecordHeader(output, record.Metadata); err != nil {
			return err
		}

		return writeBinaryRecordPayload(output, payload, outputPath)
	}

	if outputPath != "" {
		return errors.New("--output can only be used with binary records")
	}
	if err := writeRecordHeader(output, record.Metadata); err != nil {
		return err
	}

	switch payload := record.Payload.(type) {
	case *model.TextPayload:
		if payload == nil {
			return errors.New("unexpected nil text payload")
		}

		return writeTextRecordPayload(output, payload)
	case *model.CredentialsPayload:
		if payload == nil {
			return errors.New("unexpected nil credentials payload")
		}

		return writeCredentialsRecordPayload(output, payload)
	case *model.CardPayload:
		if payload == nil {
			return errors.New("unexpected nil card payload")
		}

		return writeCardRecordPayload(output, payload)
	default:
		return errors.New("unsupported record payload")
	}
}
