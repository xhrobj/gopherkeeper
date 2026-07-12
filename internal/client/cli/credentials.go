package cli

import (
	"context"
	"errors"
	"fmt"
	"io"

	urfavecli "github.com/urfave/cli/v3"
	"github.com/xhrobj/gopherkeeper/internal/model"
)

type credentialsRecordCreateCommandRequest struct {
	title        string
	metadataFile string
}

type credentialsRecordUpdateCommandRequest struct {
	recordID         string
	expectedRevision int64
	title            string
	metadataFile     string
}

func newCreateCredentialsRecordCommand(
	input io.Reader,
	factory clientFactory,
	passwords passwordReader,
) *urfavecli.Command {
	return &urfavecli.Command{
		Name:  "create-credentials",
		Usage: "create a private credentials record",
		Flags: credentialsRecordFlags(false),
		Action: func(ctx context.Context, command *urfavecli.Command) error {
			application, err := factory.NewApplication(configFromCommand(command))
			if err != nil {
				return err
			}

			return executeCreateCredentialsRecord(
				ctx,
				application,
				passwords,
				passwordStreams{
					input:        input,
					output:       command.Root().Writer,
					promptOutput: command.Root().ErrWriter,
				},
				credentialsRecordCreateCommandRequest{
					title:        command.String(titleFlag),
					metadataFile: command.String(metadataFileFlag),
				},
			)
		},
	}
}

func newUpdateCredentialsRecordCommand(
	input io.Reader,
	factory clientFactory,
	passwords passwordReader,
) *urfavecli.Command {
	return &urfavecli.Command{
		Name:      "update-credentials",
		Usage:     "update a private credentials record",
		ArgsUsage: recordIDArgsUsage,
		Flags:     credentialsRecordFlags(true),
		Action: func(ctx context.Context, command *urfavecli.Command) error {
			recordID := command.Args().First()
			if recordID == "" {
				return errors.New("record id is required")
			}

			application, err := factory.NewApplication(configFromCommand(command))
			if err != nil {
				return err
			}

			return executeUpdateCredentialsRecord(
				ctx,
				application,
				passwords,
				passwordStreams{
					input:        input,
					output:       command.Root().Writer,
					promptOutput: command.Root().ErrWriter,
				},
				credentialsRecordUpdateCommandRequest{
					recordID:         recordID,
					expectedRevision: command.Int64(revisionFlag),
					title:            command.String(titleFlag),
					metadataFile:     command.String(metadataFileFlag),
				},
			)
		},
	}
}

func credentialsRecordFlags(withRevision bool) []urfavecli.Flag {
	flags := make([]urfavecli.Flag, 0, 3)
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
			Name:  metadataFileFlag,
			Usage: "path to optional file with private metadata",
		},
	)
}

func executeCreateCredentialsRecord(
	ctx context.Context,
	application application,
	reader passwordReader,
	streams passwordStreams,
	request credentialsRecordCreateCommandRequest,
) error {
	payload, err := readCredentialsPayload(
		reader,
		streams.input,
		streams.promptOutput,
		request.metadataFile,
	)
	if err != nil {
		return err
	}

	return executeCreateRecord(ctx, application, streams.output, request.title, &payload)
}

func executeUpdateCredentialsRecord(
	ctx context.Context,
	application application,
	reader passwordReader,
	streams passwordStreams,
	request credentialsRecordUpdateCommandRequest,
) error {
	payload, err := readCredentialsPayload(
		reader,
		streams.input,
		streams.promptOutput,
		request.metadataFile,
	)
	if err != nil {
		return err
	}

	return executeUpdateRecord(ctx, application, streams.output, recordUpdateCommandRequest{
		recordID:         request.recordID,
		expectedRevision: request.expectedRevision,
		title:            request.title,
		payload:          &payload,
	})
}

func readCredentialsPayload(
	reader passwordReader,
	input io.Reader,
	promptOutput io.Writer,
	metadataFile string,
) (model.CredentialsPayload, error) {
	login, err := readPromptedLine(reader, input, promptOutput, "Login: ")
	if err != nil {
		return model.CredentialsPayload{}, fmt.Errorf("read credentials login: %w", err)
	}

	password, err := reader.ReadHidden(input, promptOutput, "Password: ")
	if err != nil {
		return model.CredentialsPayload{}, err
	}

	resourceURL, err := readPromptedLine(reader, input, promptOutput, "URL (optional): ")
	if err != nil {
		return model.CredentialsPayload{}, fmt.Errorf("read credentials URL: %w", err)
	}

	metadata, err := readOptionalTextFile(metadataFile)
	if err != nil {
		return model.CredentialsPayload{}, err
	}

	payload := model.CredentialsPayload{
		Login:    login,
		Password: password,
		URL:      resourceURL,
		Metadata: metadata,
	}
	if err := payload.Validate(); err != nil {
		return model.CredentialsPayload{}, err
	}

	return payload, nil
}

func readPromptedLine(
	reader passwordReader,
	input io.Reader,
	promptOutput io.Writer,
	prompt string,
) (string, error) {
	return reader.ReadLine(input, promptOutput, prompt)
}
