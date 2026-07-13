package cli

import (
	"context"
	"fmt"
	"io"

	urfavecli "github.com/urfave/cli/v3"
)

func newLoginCommand(
	input io.Reader,
	factory clientFactory,
	passwords passwordReader,
) *urfavecli.Command {
	return &urfavecli.Command{
		Name:  "login",
		Usage: "authenticate user and save online session",
		Flags: loginFlags(),
		Action: func(ctx context.Context, command *urfavecli.Command) error {
			application, err := applicationFromCommand(command, factory)
			if err != nil {
				return err
			}

			return executeLogin(
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

func executeLogin(
	ctx context.Context,
	application application,
	passwords passwordReader,
	streams passwordStreams,
	login string,
) error {
	if err := validateLoginArgument(login); err != nil {
		return err
	}

	password, err := passwords.ReadHidden(streams.input, streams.promptOutput, "Password: ")
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
