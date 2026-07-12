package cli

import (
	"context"
	"errors"
	"fmt"
	"io"
	"strconv"
	"strings"

	urfavecli "github.com/urfave/cli/v3"
	"github.com/xhrobj/gopherkeeper/internal/client/usecase"
	"github.com/xhrobj/gopherkeeper/internal/model"
)

const cardStdinFlag = "card-stdin"

var errMultipleCardValues = errors.New("card stdin must contain one JSON value")

type cardRecordCreateCommandRequest struct {
	title        string
	metadataFile string
	cardStdin    bool
}

type cardRecordUpdateCommandRequest struct {
	recordID         string
	expectedRevision int64
	title            string
	metadataFile     string
	cardStdin        bool
}

func newCreateCardRecordCommand(input io.Reader, factory clientFactory) *urfavecli.Command {
	return &urfavecli.Command{
		Name:  "create-card",
		Usage: "create a private card record",
		Flags: cardRecordFlags(false),
		Action: func(ctx context.Context, command *urfavecli.Command) error {
			application, err := factory.NewApplication(configFromCommand(command))
			if err != nil {
				return err
			}

			return executeCreateCardRecord(
				ctx,
				application,
				terminalPasswordReader{},
				passwordStreams{
					input:        input,
					output:       command.Root().Writer,
					promptOutput: command.Root().ErrWriter,
				},
				cardRecordCreateCommandRequest{
					title:        command.String(titleFlag),
					metadataFile: command.String(metadataFileFlag),
					cardStdin:    command.Bool(cardStdinFlag),
				},
			)
		},
	}
}

func newUpdateCardRecordCommand(input io.Reader, factory clientFactory) *urfavecli.Command {
	return &urfavecli.Command{
		Name:      "update-card",
		Usage:     "update a private card record",
		ArgsUsage: recordIDArgsUsage,
		Flags:     cardRecordFlags(true),
		Action: func(ctx context.Context, command *urfavecli.Command) error {
			recordID := command.Args().First()
			if recordID == "" {
				return errors.New("record id is required")
			}

			application, err := factory.NewApplication(configFromCommand(command))
			if err != nil {
				return err
			}

			return executeUpdateCardRecord(
				ctx,
				application,
				terminalPasswordReader{},
				passwordStreams{
					input:        input,
					output:       command.Root().Writer,
					promptOutput: command.Root().ErrWriter,
				},
				cardRecordUpdateCommandRequest{
					recordID:         recordID,
					expectedRevision: command.Int64(revisionFlag),
					title:            command.String(titleFlag),
					metadataFile:     command.String(metadataFileFlag),
					cardStdin:        command.Bool(cardStdinFlag),
				},
			)
		},
	}
}

func cardRecordFlags(withRevision bool) []urfavecli.Flag {
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
			Name:  cardStdinFlag,
			Usage: "read card payload as JSON from standard input",
		},
	)
}

func executeCreateCardRecord(
	ctx context.Context,
	application application,
	reader passwordReader,
	streams passwordStreams,
	request cardRecordCreateCommandRequest,
) error {
	payload, err := readCardPayload(
		reader,
		streams.input,
		streams.promptOutput,
		request.metadataFile,
		request.cardStdin,
	)
	if err != nil {
		return err
	}

	record, err := application.CreateRecord(ctx, usecase.CreateRecordRequest{
		Title:   request.title,
		Payload: &payload,
	})
	if err != nil {
		return err
	}

	if _, err := fmt.Fprintf(
		streams.output,
		"Created card record %s with revision %d.\n",
		record.Metadata.ID,
		record.Metadata.Revision,
	); err != nil {
		return fmt.Errorf("write created card record: %w", err)
	}

	return nil
}

func executeUpdateCardRecord(
	ctx context.Context,
	application application,
	reader passwordReader,
	streams passwordStreams,
	request cardRecordUpdateCommandRequest,
) error {
	payload, err := readCardPayload(
		reader,
		streams.input,
		streams.promptOutput,
		request.metadataFile,
		request.cardStdin,
	)
	if err != nil {
		return err
	}

	record, err := application.UpdateRecord(ctx, usecase.UpdateRecordRequest{
		RecordID:         request.recordID,
		ExpectedRevision: request.expectedRevision,
		Title:            request.title,
		Payload:          &payload,
	})
	if err != nil {
		return err
	}

	if _, err := fmt.Fprintf(
		streams.output,
		"Updated card record %s to revision %d.\n",
		record.Metadata.ID,
		record.Metadata.Revision,
	); err != nil {
		return fmt.Errorf("write updated card record: %w", err)
	}

	return nil
}

func readCardPayload(
	reader passwordReader,
	input io.Reader,
	promptOutput io.Writer,
	metadataFile string,
	cardStdin bool,
) (model.CardPayload, error) {
	if cardStdin {
		if metadataFile != "" {
			return model.CardPayload{}, errors.New("--metadata-file cannot be used with --card-stdin")
		}

		return readCardPayloadJSON(input)
	}

	number, err := readHiddenCardField(reader, input, promptOutput, "Card number: ")
	if err != nil {
		return model.CardPayload{}, fmt.Errorf("read card number: %w", err)
	}

	cardholder, err := readPromptedLine(reader, input, promptOutput, "Cardholder (optional): ")
	if err != nil {
		return model.CardPayload{}, fmt.Errorf("read cardholder: %w", err)
	}

	expiryMonth, err := readOptionalCardInteger(reader, input, promptOutput, "Expiry month (optional): ")
	if err != nil {
		return model.CardPayload{}, fmt.Errorf("read expiry month: %w", err)
	}

	expiryYear, err := readOptionalCardInteger(reader, input, promptOutput, "Expiry year (optional): ")
	if err != nil {
		return model.CardPayload{}, fmt.Errorf("read expiry year: %w", err)
	}

	cvv, err := readHiddenCardField(reader, input, promptOutput, "CVV (optional): ")
	if err != nil {
		return model.CardPayload{}, fmt.Errorf("read card CVV: %w", err)
	}

	metadata, err := readOptionalTextFile(metadataFile)
	if err != nil {
		return model.CardPayload{}, err
	}

	payload := model.CardPayload{
		Number:      number,
		Cardholder:  cardholder,
		ExpiryMonth: expiryMonth,
		ExpiryYear:  expiryYear,
		CVV:         cvv,
		Metadata:    metadata,
	}
	if err := payload.Validate(); err != nil {
		return model.CardPayload{}, err
	}

	return payload, nil
}

func readHiddenCardField(
	reader passwordReader,
	input io.Reader,
	promptOutput io.Writer,
	prompt string,
) (string, error) {
	value, err := reader.ReadHidden(input, promptOutput, prompt)
	if errors.Is(err, errPasswordInputNotTerminal) {
		return "", errors.New("card input is not a terminal; use --card-stdin")
	}
	if err != nil {
		return "", err
	}

	return value, nil
}

func readOptionalCardInteger(
	reader passwordReader,
	input io.Reader,
	promptOutput io.Writer,
	prompt string,
) (*int, error) {
	value, err := readPromptedLine(reader, input, promptOutput, prompt)
	if err != nil {
		return nil, err
	}
	value = strings.TrimSpace(value)
	if value == "" {
		return nil, nil
	}

	parsed, err := strconv.Atoi(value)
	if err != nil {
		return nil, fmt.Errorf("parse %q as integer: %w", value, err)
	}

	return &parsed, nil
}

func readCardPayloadJSON(input io.Reader) (model.CardPayload, error) {
	var payload model.CardPayload
	if err := readRecordPayloadJSON(input, "card", errMultipleCardValues, &payload); err != nil {
		return model.CardPayload{}, err
	}

	if err := payload.Validate(); err != nil {
		return model.CardPayload{}, err
	}

	return payload, nil
}
