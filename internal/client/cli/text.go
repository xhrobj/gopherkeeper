package cli

import (
	"context"
	"errors"
	"io"

	urfavecli "github.com/urfave/cli/v3"
	"github.com/xhrobj/gopherkeeper/internal/model"
)

const textFileFlag = "text-file"

type textRecordUpdateCommandRequest struct {
	recordID         string
	expectedRevision int64
	title            string
	textFile         string
	metadataFile     string
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

	return executeCreateRecord(ctx, application, output, title, &model.TextPayload{
		Text:     text,
		Metadata: metadata,
	})
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

	return executeUpdateRecord(ctx, application, output, recordUpdateCommandRequest{
		recordID:         request.recordID,
		expectedRevision: request.expectedRevision,
		title:            request.title,
		payload: &model.TextPayload{
			Text:     text,
			Metadata: metadata,
		},
	})
}
