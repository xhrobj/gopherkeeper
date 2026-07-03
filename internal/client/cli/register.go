package cli

import (
	"context"
	"errors"
	"fmt"
	"io"

	urfavecli "github.com/urfave/cli/v3"
	"github.com/xhrobj/gopherkeeper/internal/client/config"
	"github.com/xhrobj/gopherkeeper/internal/client/httpclient"
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

type userRegistrar interface {
	Register(ctx context.Context, login, password string) (model.User, error)
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
			cfg := config.Config{
				Address:    command.String("address"),
				CACertFile: command.String("ca-cert"),
			}

			return register(
				ctx,
				cfg,
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
	client, err := httpclient.New(cfg.Address, cfg.CACertFile)
	if err != nil {
		return err
	}

	return executeRegistration(
		ctx,
		client,
		terminalPasswordReader{},
		input,
		output,
		promptOutput,
		login,
		passwordStdin,
	)
}

func executeRegistration(
	ctx context.Context,
	registrar userRegistrar,
	passwords passwordReader,
	input io.Reader,
	output io.Writer,
	promptOutput io.Writer,
	login string,
	passwordStdin bool,
) error {
	password, err := readRegistrationPassword(
		passwords,
		input,
		promptOutput,
		passwordStdin,
	)
	if err != nil {
		return err
	}

	user, err := registrar.Register(ctx, login, password)
	if err != nil {
		var apiError *httpclient.APIError
		if errors.As(err, &apiError) && apiError.Code == "login_already_exists" {
			return fmt.Errorf("login %q is already registered: %w", login, err)
		}

		return fmt.Errorf("register user: %w", err)
	}

	if _, err := fmt.Fprintf(output, "User %s registered successfully.\n", user.Login); err != nil {
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
