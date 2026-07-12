package cli

import (
	"context"
	"errors"
	"fmt"
	"io"

	urfavecli "github.com/urfave/cli/v3"
)

func newRegisterCommand(input io.Reader, factory clientFactory) *urfavecli.Command {
	return &urfavecli.Command{
		Name:  "register",
		Usage: "register a new user",
		Flags: loginPasswordFlags(),
		Action: func(ctx context.Context, command *urfavecli.Command) error {
			application, err := factory.NewApplication(configFromCommand(command))
			if err != nil {
				return err
			}

			return executeRegistration(
				ctx,
				application,
				terminalPasswordReader{},
				passwordStreams{
					input:        input,
					output:       command.Root().Writer,
					promptOutput: command.Root().ErrWriter,
				},
				command.String("login"),
				command.Bool("password-stdin"),
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
	passwordStdin bool,
) error {
	if err := validateLoginArgument(login); err != nil {
		return err
	}

	password, err := readRegistrationPassword(
		passwords,
		streams.input,
		streams.promptOutput,
		passwordStdin,
	)
	if err != nil {
		return err
	}

	user, err := application.Register(ctx, login, password)
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
	passwordStdin bool,
) (string, error) {
	if passwordStdin {
		return passwords.ReadLine(input)
	}

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
