package cli

import (
	"context"
	"errors"
	"fmt"
	"io"

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
			application, err := applicationFromCommand(command, factory)
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

			application, err := applicationFromCommand(command, factory)
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

			application, err := applicationFromCommand(command, factory)
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

	return writeRecordList(output, records)
}
func executeGetRecord(
	ctx context.Context,
	application application,
	output io.Writer,
	recordID, outputPath string,
) error {
	record, err := application.GetRecord(ctx, recordID)
	if err != nil {
		return err
	}

	return writeRecord(output, record, outputPath)
}

func writeBinaryRecord(output io.Writer, record model.Record, outputPath string) error {
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

func writeNonBinaryRecord(output io.Writer, record model.Record) error {
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
