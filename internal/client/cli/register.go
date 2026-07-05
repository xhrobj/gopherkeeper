package cli

import (
	"context"
	"errors"
	"fmt"
	"io"

	urfavecli "github.com/urfave/cli/v3"
	clientapp "github.com/xhrobj/gopherkeeper/internal/client/app"
	"github.com/xhrobj/gopherkeeper/internal/client/config"
	"github.com/xhrobj/gopherkeeper/internal/model"
)

type registerRunner func(
	context.Context,
	config.Config,
	io.Reader,
	io.Writer,
	io.Writer,
	string,
	bool,
) error

type userRegisterer interface {
	Register(ctx context.Context, login, password string) (model.User, error)
}

type registrationStreams struct {
	input        io.Reader
	output       io.Writer
	promptOutput io.Writer
}

func newRegisterCommand(input io.Reader, register registerRunner) *urfavecli.Command {
	return &urfavecli.Command{
		Name:  "register",
		Usage: "register a new user",
		Flags: []urfavecli.Flag{
			&urfavecli.StringFlag{
				Name:     "login",
				Usage:    "user login",
				Required: true,
			},
			&urfavecli.BoolFlag{
				Name:  "password-stdin",
				Usage: "read password from standard input",
			},
		},
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
	application, err := clientapp.New(cfg)
	if err != nil {
		return err
	}

	return executeRegistration(
		ctx,
		application,
		terminalPasswordReader{},
		registrationStreams{
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
	streams registrationStreams,
	login string,
	passwordStdin bool,
) error {
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
