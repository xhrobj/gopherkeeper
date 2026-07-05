package cli

import (
	"context"
	"errors"
	"fmt"
	"io"
	"net/http"

	urfavecli "github.com/urfave/cli/v3"
	"github.com/xhrobj/gopherkeeper/internal/client/config"
	"github.com/xhrobj/gopherkeeper/internal/client/httpclient"
	"github.com/xhrobj/gopherkeeper/internal/client/session"
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
	Login(ctx context.Context, login, password string) (httpclient.LoginResult, error)
}

type sessionSaver interface {
	Save(stored session.Session) error
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
	client, err := httpclient.New(cfg.Address, cfg.CACertFile)
	if err != nil {
		return err
	}

	storage, err := session.NewFileStorage(cfg.SessionFile)
	if err != nil {
		return err
	}

	return executeLogin(
		ctx,
		client,
		storage,
		terminalPasswordReader{},
		loginStreams{
			input:        input,
			output:       output,
			promptOutput: promptOutput,
		},
		cfg.Address,
		login,
		passwordStdin,
	)
}

func executeLogin(
	ctx context.Context,
	logger userLogger,
	sessions sessionSaver,
	passwords passwordReader,
	streams loginStreams,
	serverAddress string,
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

	result, err := logger.Login(ctx, login, password)
	if err != nil {
		var apiError *httpclient.APIError
		if errors.As(err, &apiError) && apiError.StatusCode == http.StatusUnauthorized && apiError.Code == "invalid_credentials" {
			return fmt.Errorf("invalid login or password: %w", err)
		}

		return fmt.Errorf("login user: %w", err)
	}

	if err := sessions.Save(session.Session{
		ServerAddress: serverAddress,
		AccessToken:   result.AccessToken,
		TokenType:     result.TokenType,
		ExpiresAt:     result.ExpiresAt,
		User:          result.User,
	}); err != nil {
		return fmt.Errorf("save online session: %w", err)
	}

	if _, err := fmt.Fprintf(streams.output, "User %s logged in successfully.\n", result.User.Login); err != nil {
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
