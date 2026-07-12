package cli

import (
	"context"
	"errors"
	"io"

	urfavecli "github.com/urfave/cli/v3"
	"github.com/xhrobj/gopherkeeper/internal/model"
)

const (
	binaryFileFlag  = "binary-file"
	contentTypeFlag = "content-type"
)

type binaryRecordCreateCommandRequest struct {
	title        string
	binaryFile   string
	contentType  string
	metadataFile string
}

type binaryRecordUpdateCommandRequest struct {
	recordID         string
	expectedRevision int64
	title            string
	binaryFile       string
	contentType      string
	metadataFile     string
}

func newCreateBinaryRecordCommand(factory clientFactory) *urfavecli.Command {
	return &urfavecli.Command{
		Name:  "create-binary",
		Usage: "create a private binary record",
		Flags: binaryRecordFlags(false),
		Action: func(ctx context.Context, command *urfavecli.Command) error {
			application, err := factory.NewApplication(configFromCommand(command))
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
					contentType:  command.String(contentTypeFlag),
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
		Flags:     binaryRecordFlags(true),
		Action: func(ctx context.Context, command *urfavecli.Command) error {
			recordID := command.Args().First()
			if recordID == "" {
				return errors.New("record id is required")
			}

			application, err := factory.NewApplication(configFromCommand(command))
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
					contentType:      command.String(contentTypeFlag),
					metadataFile:     command.String(metadataFileFlag),
				},
			)
		},
	}
}

func binaryRecordFlags(withRevision bool) []urfavecli.Flag {
	flags := make([]urfavecli.Flag, 0, 5)
	if withRevision {
		flags = append(flags, &urfavecli.Int64Flag{
			Name:     revisionFlag,
			Aliases:  []string{"r"},
			Usage:    "expected record revision",
			Required: true,
		})
	}

	return append(flags,
		&urfavecli.StringFlag{
			Name:     titleFlag,
			Usage:    "record title",
			Required: true,
		},
		&urfavecli.StringFlag{
			Name:     binaryFileFlag,
			Usage:    "path to private binary payload",
			Required: true,
		},
		&urfavecli.StringFlag{
			Name:  contentTypeFlag,
			Usage: "optional content type stored with the binary payload",
		},
		&urfavecli.StringFlag{
			Name:  metadataFileFlag,
			Usage: "path to optional file with private metadata",
		},
	)
}

func executeCreateBinaryRecord(
	ctx context.Context,
	application application,
	output io.Writer,
	request binaryRecordCreateCommandRequest,
) error {
	payload, err := readBinaryPayload(request.binaryFile, request.contentType, request.metadataFile)
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
	payload, err := readBinaryPayload(request.binaryFile, request.contentType, request.metadataFile)
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

func readBinaryPayload(binaryFile string, contentType string, metadataFile string) (model.BinaryPayload, error) {
	filename, data, err := readBinaryFile(binaryFile)
	if err != nil {
		return model.BinaryPayload{}, err
	}

	metadata, err := readOptionalTextFile(metadataFile)
	if err != nil {
		return model.BinaryPayload{}, err
	}

	payload := model.BinaryPayload{
		Filename:    filename,
		Data:        data,
		ContentType: contentType,
		Metadata:    metadata,
	}
	if err := payload.Validate(); err != nil {
		return model.BinaryPayload{}, err
	}

	return payload, nil
}
