// Package cli предоставляет командный интерфейс Клиента GophKeeper.
package cli

import (
	"context"
	"fmt"
	"io"

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
	return run(ctx, args, output, errorOutput, info, runHealth)
}

func run(
	ctx context.Context,
	args []string,
	output io.Writer,
	errorOutput io.Writer,
	info buildinfo.Info,
	health healthRunner,
) error {
	previousVersionPrinter := cli.VersionPrinter
	cli.VersionPrinter = func(command *cli.Command) {
		_ = printVersion(command.Root().Writer, info)
	}
	defer func() {
		cli.VersionPrinter = previousVersionPrinter
	}()

	command := newCommand(output, errorOutput, info, health)

	return command.Run(ctx, args)
}

func newCommand(
	output io.Writer,
	errorOutput io.Writer,
	info buildinfo.Info,
	health healthRunner,
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
