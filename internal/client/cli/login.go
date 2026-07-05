package cli

import (
	"context"
	"fmt"
	"io"

	urfavecli "github.com/urfave/cli/v3"
	clientapp "github.com/xhrobj/gopherkeeper/internal/client/app"
	"github.com/xhrobj/gopherkeeper/internal/client/config"
	"github.com/xhrobj/gopherkeeper/internal/model"
)

type loginRunner func(
	context.Context,
	config.Config,
	io.Reader,
	io.Writer,
	io.Writer,
	string,
	bool,
) error

type userLogger interface {
	Login(ctx context.Context, login, password string) (model.User, error)
}

type loginStreams struct {
	input        io.Reader
	output       io.Writer
	promptOutput io.Writer
}

func newLoginCommand(input io.Reader, login loginRunner) *urfavecli.Command {
	return &urfavecli.Command{
		Name:  "login",
		Usage: "authenticate user and save online session",
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
			return login(
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

func runLogin(
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

	return executeLogin(
		ctx,
		application,
		terminalPasswordReader{},
		loginStreams{
			input:        input,
			output:       output,
			promptOutput: promptOutput,
		},
		login,
		passwordStdin,
	)
}

func executeLogin(
	ctx context.Context,
	logger userLogger,
	passwords passwordReader,
	streams loginStreams,
	login string,
	passwordStdin bool,
) error {
	password, err := readLoginPassword(
		passwords,
		streams.input,
		streams.promptOutput,
		passwordStdin,
	)
	if err != nil {
		return err
	}

	user, err := logger.Login(ctx, login, password)
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
