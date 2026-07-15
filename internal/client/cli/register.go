package cli

import (
	"context"
	"errors"
	"fmt"
	"io"

	urfavecli "github.com/urfave/cli/v3"
)

func newRegisterCommand(
	input io.Reader,
	factory clientFactory,
	passwords passwordReader,
) *urfavecli.Command {
	return &urfavecli.Command{
		Name:  "register",
		Usage: "register a new user",
		Flags: loginFlags(),
		Action: func(ctx context.Context, command *urfavecli.Command) error {
			application, err := applicationFromCommand(command, factory)
			if err != nil {
				return err
			}

			return executeRegistration(
				ctx,
				application,
				passwords,
				passwordStreams{
					input:        input,
					output:       command.Root().Writer,
					promptOutput: command.Root().ErrWriter,
				},
				command.String("login"),
			)
		},
	}
}

func executeRegistration(
	ctx context.Context,
	application application,
	passwords passwordReader,
	streams passwordStreams,
	login string,
) error {
	canonicalLogin, err := canonicalizeLoginArgument(login)
	if err != nil {
		return err
	}

	password, err := readRegistrationPassword(
		passwords,
		streams.input,
		streams.promptOutput,
	)
	if err != nil {
		return err
	}

	user, err := application.Register(ctx, canonicalLogin, password)
	if err != nil {
		return err
	}

	if _, err := fmt.Fprintf(streams.output, "User %s registered successfully.\n", user.Login); err != nil {
		return fmt.Errorf("write registration result: %w", err)
	}

	return nil
}

func readRegistrationPassword(
	passwords passwordReader,
	input io.Reader,
	promptOutput io.Writer,
) (string, error) {
	password, err := passwords.ReadHidden(input, promptOutput, "Password: ")
	if err != nil {
		return "", err
	}

	repeatedPassword, err := passwords.ReadHidden(input, promptOutput, "Repeat password: ")
	if err != nil {
		return "", err
	}

	if password != repeatedPassword {
		return "", errors.New("passwords do not match")
	}

	return password, nil
}
