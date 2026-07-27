package cli

import (
	"context"
	"errors"
	"io"

	urfavecli "github.com/urfave/cli/v3"
	"github.com/xhrobj/gopherkeeper/internal/model"
)

const binaryFileFlag = "binary-file"

type binaryRecordCreateCommandRequest struct {
	title        string
	binaryFile   string
	metadataFile string
}

type binaryRecordUpdateCommandRequest struct {
	recordID         string
	expectedRevision int64
	title            string
	binaryFile       string
	metadataFile     string
}

func newCreateBinaryRecordCommand(factory clientFactory) *urfavecli.Command {
	return &urfavecli.Command{
		Name:  "create-binary",
		Usage: "create a private binary record",
		Flags: recordMutationFlags(
			false,
			recordTitleUsage,
			recordMetadataFileUsage,
			&urfavecli.StringFlag{
				Name:     binaryFileFlag,
				Usage:    "path to private binary payload",
				Required: true,
			},
		),
		Action: func(ctx context.Context, command *urfavecli.Command) error {
			application, err := applicationFromCommand(command, factory)
			if err != nil {
				return err
			}

			return executeCreateBinaryRecord(
				ctx,
				application,
				command.Root().Writer,
				binaryRecordCreateCommandRequest{
					title:        command.String(titleFlag),
					binaryFile:   command.String(binaryFileFlag),
					metadataFile: command.String(metadataFileFlag),
				},
			)
		},
	}
}

func newUpdateBinaryRecordCommand(factory clientFactory) *urfavecli.Command {
	return &urfavecli.Command{
		Name:      "update-binary",
		Usage:     "update a private binary record",
		ArgsUsage: recordIDArgsUsage,
		Flags: recordMutationFlags(
			true,
			recordTitleUsage,
			recordMetadataFileUsage,
			&urfavecli.StringFlag{
				Name:     binaryFileFlag,
				Usage:    "path to private binary payload",
				Required: true,
			},
		),
		Action: func(ctx context.Context, command *urfavecli.Command) error {
			recordID := command.Args().First()
			if recordID == "" {
				return errors.New("record id is required")
			}

			application, err := applicationFromCommand(command, factory)
			if err != nil {
				return err
			}

			return executeUpdateBinaryRecord(
				ctx,
				application,
				command.Root().Writer,
				binaryRecordUpdateCommandRequest{
					recordID:         recordID,
					expectedRevision: command.Int64(revisionFlag),
					title:            command.String(titleFlag),
					binaryFile:       command.String(binaryFileFlag),
					metadataFile:     command.String(metadataFileFlag),
				},
			)
		},
	}
}

func executeCreateBinaryRecord(
	ctx context.Context,
	application application,
	output io.Writer,
	request binaryRecordCreateCommandRequest,
) error {
	payload, err := readBinaryPayload(request.binaryFile, request.metadataFile)
	if err != nil {
		return err
	}

	return executeCreateRecord(ctx, application, output, request.title, &payload)
}

func executeUpdateBinaryRecord(
	ctx context.Context,
	application application,
	output io.Writer,
	request binaryRecordUpdateCommandRequest,
) error {
	payload, err := readBinaryPayload(request.binaryFile, request.metadataFile)
	if err != nil {
		return err
	}

	return executeUpdateRecord(ctx, application, output, recordUpdateCommandRequest{
		recordID:         request.recordID,
		expectedRevision: request.expectedRevision,
		title:            request.title,
		payload:          &payload,
	})
}

func readBinaryPayload(binaryFile, metadataFile string) (model.BinaryPayload, error) {
	filename, data, err := readBinaryFile(binaryFile)
	if err != nil {
		return model.BinaryPayload{}, err
	}

	metadata, err := readOptionalTextFile(metadataFile)
	if err != nil {
		return model.BinaryPayload{}, err
	}

	payload := model.BinaryPayload{
		Filename: filename,
		Data:     data,
		Metadata: metadata,
	}
	if err := payload.Validate(); err != nil {
		return model.BinaryPayload{}, err
	}

	return payload, nil
}
