package cli

import (
	"context"
	"errors"
	"fmt"
	"io"

	urfavecli "github.com/urfave/cli/v3"
	"github.com/xhrobj/gopherkeeper/internal/client/config"
	"github.com/xhrobj/gopherkeeper/internal/client/usecase"
	"github.com/xhrobj/gopherkeeper/internal/model"
)

const credentialsStdinFlag = "credentials-stdin"

var errMultipleCredentialsValues = errors.New("credentials stdin must contain one JSON value")

type credentialsRecordCreator interface {
	CreateCredentialsRecord(
		ctx context.Context,
		request usecase.CreateCredentialsRecordRequest,
	) (usecase.CredentialsRecord, error)
}

type credentialsRecordUpdater interface {
	UpdateCredentialsRecord(
		ctx context.Context,
		request usecase.UpdateCredentialsRecordRequest,
	) (usecase.CredentialsRecord, error)
}

func newCreateCredentialsRecordCommand(
	input io.Reader,
	create credentialsRecordCreateRunner,
) *urfavecli.Command {
	return &urfavecli.Command{
		Name:  "create-credentials",
		Usage: "create a private credentials record",
		Flags: credentialsRecordFlags(false),
		Action: func(ctx context.Context, command *urfavecli.Command) error {
			return create(
				ctx,
				configFromCommand(command),
				input,
				command.Root().Writer,
				command.Root().ErrWriter,
				credentialsRecordCreateCommandRequest{
					title:            command.String(titleFlag),
					metadataFile:     command.String(metadataFileFlag),
					credentialsStdin: command.Bool(credentialsStdinFlag),
				},
			)
		},
	}
}

func newUpdateCredentialsRecordCommand(
	input io.Reader,
	update credentialsRecordUpdateRunner,
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

			return update(
				ctx,
				configFromCommand(command),
				input,
				command.Root().Writer,
				command.Root().ErrWriter,
				credentialsRecordUpdateCommandRequest{
					recordID:         recordID,
					expectedRevision: command.Int64(revisionFlag),
					title:            command.String(titleFlag),
					metadataFile:     command.String(metadataFileFlag),
					credentialsStdin: command.Bool(credentialsStdinFlag),
				},
			)
		},
	}
}

func credentialsRecordFlags(withRevision bool) []urfavecli.Flag {
	flags := make([]urfavecli.Flag, 0, 4)
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
			Usage: "path to optional file with private metadata in interactive mode",
		},
		&urfavecli.BoolFlag{
			Name:  credentialsStdinFlag,
			Usage: "read credentials payload as JSON from standard input",
		},
	)
}

func runCreateCredentialsRecord(
	ctx context.Context,
	cfg config.Config,
	input io.Reader,
	output io.Writer,
	promptOutput io.Writer,
	request credentialsRecordCreateCommandRequest,
) error {
	application, err := usecase.New(cfg)
	if err != nil {
		return err
	}

	return executeCreateCredentialsRecord(
		ctx,
		application,
		terminalPasswordReader{},
		passwordStreams{
			input:        input,
			output:       output,
			promptOutput: promptOutput,
		},
		request,
	)
}

func runUpdateCredentialsRecord(
	ctx context.Context,
	cfg config.Config,
	input io.Reader,
	output io.Writer,
	promptOutput io.Writer,
	request credentialsRecordUpdateCommandRequest,
) error {
	application, err := usecase.New(cfg)
	if err != nil {
		return err
	}

	return executeUpdateCredentialsRecord(
		ctx,
		application,
		terminalPasswordReader{},
		passwordStreams{
			input:        input,
			output:       output,
			promptOutput: promptOutput,
		},
		request,
	)
}

func executeCreateCredentialsRecord(
	ctx context.Context,
	creator credentialsRecordCreator,
	reader passwordReader,
	streams passwordStreams,
	request credentialsRecordCreateCommandRequest,
) error {
	payload, err := readCredentialsPayload(
		reader,
		streams.input,
		streams.promptOutput,
		request.metadataFile,
		request.credentialsStdin,
	)
	if err != nil {
		return err
	}

	record, err := creator.CreateCredentialsRecord(ctx, usecase.CreateCredentialsRecordRequest{
		Title:    request.title,
		Login:    payload.Login,
		Password: payload.Password,
		URL:      payload.URL,
		Metadata: payload.Metadata,
	})
	if err != nil {
		return err
	}

	if _, err := fmt.Fprintf(
		streams.output,
		"Created credentials record %s with revision %d.\n",
		record.Metadata.ID,
		record.Metadata.Revision,
	); err != nil {
		return fmt.Errorf("write created credentials record: %w", err)
	}

	return nil
}

func executeUpdateCredentialsRecord(
	ctx context.Context,
	updater credentialsRecordUpdater,
	reader passwordReader,
	streams passwordStreams,
	request credentialsRecordUpdateCommandRequest,
) error {
	payload, err := readCredentialsPayload(
		reader,
		streams.input,
		streams.promptOutput,
		request.metadataFile,
		request.credentialsStdin,
	)
	if err != nil {
		return err
	}

	record, err := updater.UpdateCredentialsRecord(ctx, usecase.UpdateCredentialsRecordRequest{
		RecordID:         request.recordID,
		ExpectedRevision: request.expectedRevision,
		Title:            request.title,
		Login:            payload.Login,
		Password:         payload.Password,
		URL:              payload.URL,
		Metadata:         payload.Metadata,
	})
	if err != nil {
		return err
	}

	if _, err := fmt.Fprintf(
		streams.output,
		"Updated credentials record %s to revision %d.\n",
		record.Metadata.ID,
		record.Metadata.Revision,
	); err != nil {
		return fmt.Errorf("write updated credentials record: %w", err)
	}

	return nil
}

func readCredentialsPayload(
	reader passwordReader,
	input io.Reader,
	promptOutput io.Writer,
	metadataFile string,
	credentialsStdin bool,
) (model.CredentialsPayload, error) {
	if credentialsStdin {
		if metadataFile != "" {
			return model.CredentialsPayload{}, errors.New(
				"--metadata-file cannot be used with --credentials-stdin",
			)
		}

		return readCredentialsPayloadJSON(input)
	}

	login, err := readPromptedLine(reader, input, promptOutput, "Login: ")
	if err != nil {
		return model.CredentialsPayload{}, fmt.Errorf("read credentials login: %w", err)
	}

	password, err := reader.ReadHidden(input, promptOutput, "Password: ")
	if err != nil {
		if errors.Is(err, errPasswordInputNotTerminal) {
			return model.CredentialsPayload{}, errors.New(
				"password input is not a terminal; use --credentials-stdin",
			)
		}

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
	if _, err := fmt.Fprint(promptOutput, prompt); err != nil {
		return "", fmt.Errorf("write prompt: %w", err)
	}

	return reader.ReadLine(input)
}

func readCredentialsPayloadJSON(input io.Reader) (model.CredentialsPayload, error) {
	var payload model.CredentialsPayload
	if err := readRecordPayloadJSON(
		input,
		"credentials",
		errMultipleCredentialsValues,
		&payload,
	); err != nil {
		return model.CredentialsPayload{}, err
	}

	if err := payload.Validate(); err != nil {
		return model.CredentialsPayload{}, err
	}

	return payload, nil
}
