package cli

import (
	"context"
	"fmt"
	"io"
	"os"
	"strings"

	"github.com/urfave/cli/v3"
	"github.com/xhrobj/gopherkeeper/internal/buildinfo"
	"github.com/xhrobj/gopherkeeper/internal/client/config"
)

const banner = `
  ________              .__     ____  __.
 /  _____/  ____ ______ |  |__ |    |/ _|____   ____ ______   ___________
/   \  ___ /  _ \\____ \|  |  \|      <_/ __ \_/ __ \\____ \_/ __ \_  __ \
\    \_\  (  <_> )  |_> >   Y  \    |  \  ___/\  ___/|  |_> >  ___/|  | \/
 \______  /\____/|   __/|___|  /____|__ \___  >\___  >   __/ \___  >__|
        \/       |__|        \/        \/   \/     \/|__|        \/
         -= Client: Access your secrets securely. =-

`

type healthRunner func(context.Context, config.Config, io.Writer) error

// Run запускает командный интерфейс Клиента.
func Run(
	ctx context.Context,
	args []string,
	output io.Writer,
	errorOutput io.Writer,
	info buildinfo.Info,
) error {
	return runWithInput(ctx, args, os.Stdin, output, errorOutput, info, runHealth, runRegister)
}

// RunWithInput запускает командный интерфейс с заданным стандартным вводом.
func RunWithInput(
	ctx context.Context,
	args []string,
	input io.Reader,
	output io.Writer,
	errorOutput io.Writer,
	info buildinfo.Info,
) error {
	return runWithInput(ctx, args, input, output, errorOutput, info, runHealth, runRegister)
}

func run(
	ctx context.Context,
	args []string,
	output io.Writer,
	errorOutput io.Writer,
	info buildinfo.Info,
	health healthRunner,
) error {
	return runWithInput(
		ctx,
		args,
		strings.NewReader(""),
		output,
		errorOutput,
		info,
		health,
		runRegister,
	)
}

func runWithInput(
	ctx context.Context,
	args []string,
	input io.Reader,
	output io.Writer,
	errorOutput io.Writer,
	info buildinfo.Info,
	health healthRunner,
	register registerRunner,
) error {
	previousVersionPrinter := cli.VersionPrinter
	cli.VersionPrinter = func(command *cli.Command) {
		_ = printVersion(command.Root().Writer, info)
	}
	defer func() {
		cli.VersionPrinter = previousVersionPrinter
	}()

	command := newCommand(input, output, errorOutput, info, health, register)

	return command.Run(ctx, args)
}

func newCommand(
	input io.Reader,
	output io.Writer,
	errorOutput io.Writer,
	info buildinfo.Info,
	health healthRunner,
	register registerRunner,
) *cli.Command {
	defaults := config.Load()
	version := info.Version
	if version == "" {
		version = "¯\\_(ツ)_/¯"
	}

	return &cli.Command{
		Usage:                         "securely store and access private data",
		Version:                       version,
		Writer:                        output,
		ErrWriter:                     errorOutput,
		CustomRootCommandHelpTemplate: banner + cli.RootCommandHelpTemplate,
		Flags: []cli.Flag{
			&cli.StringFlag{
				Name:    "address",
				Aliases: []string{"a"},
				Usage:   "Server address",
				Value:   defaults.Address,
			},
			&cli.StringFlag{
				Name:  "ca-cert",
				Usage: "path to an additional trusted CA certificate",
				Value: defaults.CACertFile,
			},
		},
		Commands: []*cli.Command{
			newHealthCommand(health),
			newRegisterCommand(input, register),
		},
		Action: func(_ context.Context, command *cli.Command) error {
			return cli.ShowRootCommandHelp(command)
		},
	}
}

func printVersion(output io.Writer, info buildinfo.Info) error {
	if _, err := fmt.Fprint(output, banner); err != nil {
		return fmt.Errorf("write banner: %w", err)
	}

	if err := buildinfo.Print(output, info); err != nil {
		return fmt.Errorf("write build info: %w", err)
	}

	return nil
}
