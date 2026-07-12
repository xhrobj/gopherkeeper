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
	"github.com/xhrobj/gopherkeeper/internal/client/usecase"
	"github.com/xhrobj/gopherkeeper/internal/model"
)

const (
	titleFlag         = "title"
	textFileFlag      = "text-file"
	metadataFileFlag  = "metadata-file"
	revisionFlag      = "revision"
	recordIDArgsUsage = "<record-id>"
)

type textRecordUpdateCommandRequest struct {
	recordID         string
	expectedRevision int64
	title            string
	textFile         string
	metadataFile     string
}

func newRecordsCommand(input io.Reader, factory clientFactory) *urfavecli.Command {
	return &urfavecli.Command{
		Name:  "records",
		Usage: "manage private records",
		Commands: []*urfavecli.Command{
			newCreateTextRecordCommand(factory),
			newCreateCredentialsRecordCommand(input, factory),
			newCreateCardRecordCommand(input, factory),
			newUpdateTextRecordCommand(factory),
			newUpdateCredentialsRecordCommand(input, factory),
			newUpdateCardRecordCommand(input, factory),
			newListRecordsCommand(factory),
			newGetRecordCommand(factory),
			newDeleteRecordCommand(factory),
		},
	}
}

func newCreateTextRecordCommand(factory clientFactory) *urfavecli.Command {
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
			application, err := factory.NewApplication(configFromCommand(command))
			if err != nil {
				return err
			}

			return executeCreateTextRecord(
				ctx,
				application,
				command.Root().Writer,
				command.String(titleFlag),
				command.String(textFileFlag),
				command.String(metadataFileFlag),
			)
		},
	}
}

func newUpdateTextRecordCommand(factory clientFactory) *urfavecli.Command {
	return &urfavecli.Command{
		Name:      "update-text",
		Usage:     "update a private text record",
		ArgsUsage: recordIDArgsUsage,
		Flags: []urfavecli.Flag{
			&urfavecli.Int64Flag{
				Name:     revisionFlag,
				Aliases:  []string{"r"},
				Usage:    "expected record revision",
				Required: true,
			},
			&urfavecli.StringFlag{
				Name:     titleFlag,
				Usage:    "new record title",
				Required: true,
			},
			&urfavecli.StringFlag{
				Name:     textFileFlag,
				Usage:    "path to file with new private text payload",
				Required: true,
			},
			&urfavecli.StringFlag{
				Name:  metadataFileFlag,
				Usage: "path to optional file with new private metadata",
			},
		},
		Action: func(ctx context.Context, command *urfavecli.Command) error {
			recordID := command.Args().First()
			if recordID == "" {
				return errors.New("record id is required")
			}

			application, err := factory.NewApplication(configFromCommand(command))
			if err != nil {
				return err
			}

			return executeUpdateTextRecord(
				ctx,
				application,
				command.Root().Writer,
				textRecordUpdateCommandRequest{
					recordID:         recordID,
					expectedRevision: command.Int64(revisionFlag),
					title:            command.String(titleFlag),
					textFile:         command.String(textFileFlag),
					metadataFile:     command.String(metadataFileFlag),
				},
			)
		},
	}
}

func newListRecordsCommand(factory clientFactory) *urfavecli.Command {
	return &urfavecli.Command{
		Name:  "list",
		Usage: "list private record metadata",
		Action: func(ctx context.Context, command *urfavecli.Command) error {
			application, err := factory.NewApplication(configFromCommand(command))
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
		Action: func(ctx context.Context, command *urfavecli.Command) error {
			recordID := command.Args().First()
			if recordID == "" {
				return errors.New("record id is required")
			}

			application, err := factory.NewApplication(configFromCommand(command))
			if err != nil {
				return err
			}

			return executeGetRecord(ctx, application, command.Root().Writer, recordID)
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

			application, err := factory.NewApplication(configFromCommand(command))
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

func executeCreateTextRecord(
	ctx context.Context,
	application application,
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

	record, err := application.CreateRecord(ctx, usecase.CreateRecordRequest{
		Title: title,
		Payload: &model.TextPayload{
			Text:     text,
			Metadata: metadata,
		},
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

func executeUpdateTextRecord(
	ctx context.Context,
	application application,
	output io.Writer,
	request textRecordUpdateCommandRequest,
) error {
	text, err := readRequiredTextFile(request.textFile)
	if err != nil {
		return err
	}

	metadata, err := readOptionalTextFile(request.metadataFile)
	if err != nil {
		return err
	}

	record, err := application.UpdateRecord(ctx, usecase.UpdateRecordRequest{
		RecordID:         request.recordID,
		ExpectedRevision: request.expectedRevision,
		Title:            request.title,
		Payload: &model.TextPayload{
			Text:     text,
			Metadata: metadata,
		},
	})
	if err != nil {
		return err
	}

	if _, err := fmt.Fprintf(
		output,
		"Updated text record %s to revision %d.\n",
		record.Metadata.ID,
		record.Metadata.Revision,
	); err != nil {
		return fmt.Errorf("write updated text record: %w", err)
	}

	return nil
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

func executeGetRecord(ctx context.Context, application application, output io.Writer, recordID string) error {
	record, err := application.GetRecord(ctx, recordID)
	if err != nil {
		return err
	}

	if _, err := fmt.Fprintf(
		output,
		"ID: %s\nType: %s\nTitle: %s\nRevision: %d\nCreated at: %s\nUpdated at: %s\n",
		record.Metadata.ID,
		record.Metadata.Type,
		record.Metadata.Title,
		record.Metadata.Revision,
		formatRecordTime(record.Metadata.CreatedAt),
		formatRecordTime(record.Metadata.UpdatedAt),
	); err != nil {
		return fmt.Errorf("write record metadata: %w", err)
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

func writeTextRecordPayload(output io.Writer, payload *model.TextPayload) error {
	if _, err := fmt.Fprintf(output, "\nText:\n%s\n", payload.Text); err != nil {
		return fmt.Errorf("write text record: %w", err)
	}

	return writeRecordMetadataPayload(output, payload.Metadata, "text record")
}

func writeCredentialsRecordPayload(output io.Writer, payload *model.CredentialsPayload) error {
	if _, err := fmt.Fprintf(
		output,
		"\nLogin: %s\nPassword: %s\n",
		payload.Login,
		payload.Password,
	); err != nil {
		return fmt.Errorf("write credentials record: %w", err)
	}

	if payload.URL != "" {
		if _, err := fmt.Fprintf(output, "URL: %s\n", payload.URL); err != nil {
			return fmt.Errorf("write credentials record URL: %w", err)
		}
	}

	return writeRecordMetadataPayload(output, payload.Metadata, "credentials record")
}

func writeCardRecordPayload(output io.Writer, payload *model.CardPayload) error {
	if _, err := fmt.Fprintf(output, "\nNumber: %s\n", payload.Number); err != nil {
		return fmt.Errorf("write card record: %w", err)
	}

	if payload.Cardholder != "" {
		if _, err := fmt.Fprintf(output, "Cardholder: %s\n", payload.Cardholder); err != nil {
			return fmt.Errorf("write cardholder: %w", err)
		}
	}

	if payload.ExpiryMonth != nil && payload.ExpiryYear != nil {
		if _, err := fmt.Fprintf(
			output,
			"Expiry: %02d/%d\n",
			*payload.ExpiryMonth,
			*payload.ExpiryYear,
		); err != nil {
			return fmt.Errorf("write card expiry: %w", err)
		}
	}

	if payload.CVV != "" {
		if _, err := fmt.Fprintf(output, "CVV: %s\n", payload.CVV); err != nil {
			return fmt.Errorf("write card CVV: %w", err)
		}
	}

	return writeRecordMetadataPayload(output, payload.Metadata, "card record")
}

func writeRecordMetadataPayload(output io.Writer, metadata string, description string) error {
	if metadata == "" {
		return nil
	}

	if _, err := fmt.Fprintf(output, "\nMetadata:\n%s\n", metadata); err != nil {
		return fmt.Errorf("write %s metadata: %w", description, err)
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
