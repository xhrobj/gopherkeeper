package cli

import (
	"context"
	"fmt"
	"io"

	urfavecli "github.com/urfave/cli/v3"
)

func newLoginCommand(input io.Reader, factory clientFactory) *urfavecli.Command {
	return &urfavecli.Command{
		Name:  "login",
		Usage: "authenticate user and save online session",
		Flags: loginPasswordFlags(),
		Action: func(ctx context.Context, command *urfavecli.Command) error {
			application, err := factory.NewApplication(configFromCommand(command))
			if err != nil {
				return err
			}

			return executeLogin(
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

func executeLogin(
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

	password, err := readLoginPassword(
		passwords,
		streams.input,
		streams.promptOutput,
		passwordStdin,
	)
	if err != nil {
		return err
	}

	user, err := application.Login(ctx, login, password)
	if err != nil {
		return err
	}

	if _, err := fmt.Fprintf(streams.output, "User %s logged in successfully.\n", user.Login); err != nil {
		return fmt.Errorf("write login result: %w", err)
	}

	return nil
}

func readLoginPassword(
	passwords passwordReader,
	input io.Reader,
	promptOutput io.Writer,
	passwordStdin bool,
) (string, error) {
	if passwordStdin {
		return passwords.ReadLine(input)
	}

	return passwords.ReadHidden(input, promptOutput, "Password: ")
}
