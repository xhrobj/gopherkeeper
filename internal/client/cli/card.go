package cli

import (
	"context"
	"errors"
	"fmt"
	"io"
	"strconv"
	"strings"

	urfavecli "github.com/urfave/cli/v3"
	"github.com/xhrobj/gopherkeeper/internal/model"
)

type cardRecordCreateCommandRequest struct {
	title        string
	metadataFile string
}

type cardRecordUpdateCommandRequest struct {
	recordID         string
	expectedRevision int64
	title            string
	metadataFile     string
}

func newCreateCardRecordCommand(
	input io.Reader,
	factory clientFactory,
	passwords passwordReader,
) *urfavecli.Command {
	return &urfavecli.Command{
		Name:  "create-card",
		Usage: "create a private card record",
		Flags: cardRecordFlags(false),
		Action: func(ctx context.Context, command *urfavecli.Command) error {
			application, err := applicationFromCommand(command, factory)
			if err != nil {
				return err
			}

			return executeCreateCardRecord(
				ctx,
				application,
				passwords,
				passwordStreams{
					input:        input,
					output:       command.Root().Writer,
					promptOutput: command.Root().ErrWriter,
				},
				cardRecordCreateCommandRequest{
					title:        command.String(titleFlag),
					metadataFile: command.String(metadataFileFlag),
				},
			)
		},
	}
}

func newUpdateCardRecordCommand(
	input io.Reader,
	factory clientFactory,
	passwords passwordReader,
) *urfavecli.Command {
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

			application, err := applicationFromCommand(command, factory)
			if err != nil {
				return err
			}

			return executeUpdateCardRecord(
				ctx,
				application,
				passwords,
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
				},
			)
		},
	}
}

func cardRecordFlags(withRevision bool) []urfavecli.Flag {
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
	)
	if err != nil {
		return err
	}

	return executeCreateRecord(ctx, application, streams.output, request.title, &payload)
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

func readCardPayload(
	reader passwordReader,
	input io.Reader,
	promptOutput io.Writer,
	metadataFile string,
) (model.CardPayload, error) {
	number, err := reader.ReadHidden(input, promptOutput, "Card number: ")
	if err != nil {
		return model.CardPayload{}, fmt.Errorf("read card number: %w", err)
	}

	cardholder, err := readPromptedLine(reader, input, promptOutput, "Cardholder (optional): ")
	if err != nil {
		return model.CardPayload{}, fmt.Errorf("read cardholder: %w", err)
	}

	expiryMonth, expiryYear, err := readOptionalCardExpiry(reader, input, promptOutput)
	if err != nil {
		return model.CardPayload{}, fmt.Errorf("read card expiry: %w", err)
	}

	cvv, err := reader.ReadHidden(input, promptOutput, "CVV (optional): ")
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

func readOptionalCardExpiry(
	reader passwordReader,
	input io.Reader,
	promptOutput io.Writer,
) (*int, *int, error) {
	for {
		value, err := readPromptedLine(reader, input, promptOutput, "Expiry (MM/YYYY, optional): ")
		if err != nil {
			return nil, nil, err
		}

		value = strings.TrimSpace(value)
		if value == "" {
			return nil, nil, nil
		}

		month, year, ok := parseCardExpiry(value)
		if ok {
			return &month, &year, nil
		}

		if _, err := fmt.Fprintln(promptOutput, "Invalid expiry. Use MM/YYYY or leave it empty."); err != nil {
			return nil, nil, fmt.Errorf("write expiry validation message: %w", err)
		}
	}
}

func parseCardExpiry(value string) (int, int, bool) {
	if len(value) != len("MM/YYYY") || value[2] != '/' ||
		!containsOnlyASCIIDigits(value[:2]) || !containsOnlyASCIIDigits(value[3:]) {
		return 0, 0, false
	}

	month, err := strconv.Atoi(value[:2])
	if err != nil || month < 1 || month > 12 {
		return 0, 0, false
	}

	year, err := strconv.Atoi(value[3:])
	if err != nil || year < 1 || year > 9999 {
		return 0, 0, false
	}

	return month, year, true
}

func containsOnlyASCIIDigits(value string) bool {
	for _, char := range value {
		if char < '0' || char > '9' {
			return false
		}
	}

	return true
}
