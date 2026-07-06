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

type userRegisterer interface {
	Register(ctx context.Context, login, password string) (model.User, error)
}

func newRegisterCommand(input io.Reader, register passwordRunner) *urfavecli.Command {
	return &urfavecli.Command{
		Name:  "register",
		Usage: "register a new user",
		Flags: loginPasswordFlags(),
		Action: func(ctx context.Context, command *urfavecli.Command) error {
			return register(
				ctx,
				configFromCommand(command),
				input,
				command.Root().Writer,
				command.Root().ErrWriter,
				command.String("login"),
				command.Bool("password-stdin"),
			)
		},
	}
}

func runRegister(
	ctx context.Context,
	cfg config.Config,
	input io.Reader,
	output io.Writer,
	promptOutput io.Writer,
	login string,
	passwordStdin bool,
) error {
	application, err := usecase.New(cfg)
	if err != nil {
		return err
	}

	return executeRegistration(
		ctx,
		application,
		terminalPasswordReader{},
		passwordStreams{
			input:        input,
			output:       output,
			promptOutput: promptOutput,
		},
		login,
		passwordStdin,
	)
}

func executeRegistration(
	ctx context.Context,
	registerer userRegisterer,
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

	user, err := registerer.Register(ctx, login, password)
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
