package cli

import (
	"context"
	"fmt"
	"io"
	"os"

	urfavecli "github.com/urfave/cli/v3"
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

type commandRunners struct {
	health   outputRunner
	register passwordRunner
	login    passwordRunner
	whoami   outputRunner
}

type runOptions struct {
	input       io.Reader
	output      io.Writer
	errorOutput io.Writer
	info        buildinfo.Info
	runners     commandRunners
}

// Run запускает командный интерфейс Клиента.
func Run(
	ctx context.Context,
	args []string,
	output io.Writer,
	errorOutput io.Writer,
	info buildinfo.Info,
) error {
	return RunWithInput(ctx, args, os.Stdin, output, errorOutput, info)
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
	return run(ctx, args, runOptions{
		input:       input,
		output:      output,
		errorOutput: errorOutput,
		info:        info,
		runners:     defaultCommandRunners(),
	})
}

func defaultCommandRunners() commandRunners {
	return commandRunners{
		health:   runHealth,
		register: runRegister,
		login:    runLogin,
		whoami:   runWhoami,
	}
}

func run(ctx context.Context, args []string, options runOptions) error {
	previousVersionPrinter := urfavecli.VersionPrinter
	urfavecli.VersionPrinter = func(command *urfavecli.Command) {
		_ = printVersion(command.Root().Writer, options.info)
	}
	defer func() {
		urfavecli.VersionPrinter = previousVersionPrinter
	}()

	command := newRootCommand(
		options.input,
		options.output,
		options.errorOutput,
		options.info,
		options.runners,
	)

	return command.Run(ctx, args)
}

func newRootCommand(
	input io.Reader,
	output io.Writer,
	errorOutput io.Writer,
	info buildinfo.Info,
	runners commandRunners,
) *urfavecli.Command {
	defaults := config.Load()

	version := info.Version
	if version == "" {
		version = "¯\\_(ツ)_/¯"
	}

	return &urfavecli.Command{
		Usage:                         "securely store and access private data",
		Version:                       version,
		Writer:                        output,
		ErrWriter:                     errorOutput,
		CustomRootCommandHelpTemplate: banner + urfavecli.RootCommandHelpTemplate,
		Flags: []urfavecli.Flag{
			&urfavecli.StringFlag{
				Name:    addressFlag,
				Aliases: []string{"a"},
				Usage:   "Server address",
				Value:   defaults.Address,
			},
			&urfavecli.StringFlag{
				Name:  caCertFlag,
				Usage: "path to an additional trusted CA certificate",
				Value: defaults.CACertFile,
			},
			&urfavecli.StringFlag{
				Name:  sessionFileFlag,
				Usage: "path to online session file",
				Value: defaults.SessionFile,
			},
		},
		Commands: []*urfavecli.Command{
			newHealthCommand(runners.health),
			newRegisterCommand(input, runners.register),
			newLoginCommand(input, runners.login),
			newWhoamiCommand(runners.whoami),
		},
		Action: func(_ context.Context, command *urfavecli.Command) error {
			return urfavecli.ShowRootCommandHelp(command)
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
