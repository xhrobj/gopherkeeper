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

const notAvailable = "¯\\_(ツ)_/¯"

type runOptions struct {
	input       io.Reader
	output      io.Writer
	errorOutput io.Writer
	info        buildinfo.Info
	factory     clientFactory
	passwords   passwordReader
}

// Run запускает командный интерфейс Клиента.
func Run(
	ctx context.Context,
	args []string,
	output io.Writer,
	errorOutput io.Writer,
	info buildinfo.Info,
) error {
	return run(ctx, args, runOptions{
		input:       os.Stdin,
		output:      output,
		errorOutput: errorOutput,
		info:        info,
		factory:     defaultClientFactory{},
		passwords:   terminalPasswordReader{},
	})
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
		factory:     defaultClientFactory{},
		passwords:   streamPasswordReader{},
	})
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
		options.factory,
		options.passwords,
	)

	return command.Run(ctx, args)
}

func newRootCommand(
	input io.Reader,
	output io.Writer,
	errorOutput io.Writer,
	info buildinfo.Info,
	factory clientFactory,
	passwords passwordReader,
) *urfavecli.Command {
	defaults := config.Default()
	version := info.Version
	if version == "" {
		version = notAvailable
	}

	return &urfavecli.Command{
		Metadata: map[string]any{
			clientConfigMetadataKey: defaults,
		},
		Usage:                         "securely store and access private data",
		Version:                       version,
		Writer:                        output,
		ErrWriter:                     errorOutput,
		CustomRootCommandHelpTemplate: banner + urfavecli.RootCommandHelpTemplate,
		Flags: []urfavecli.Flag{
			&urfavecli.StringFlag{
				Name:    configFlag,
				Aliases: []string{"c"},
				Usage:   "path to JSON client config file",
			},
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
			&urfavecli.StringFlag{
				Name:  cacheDirFlag,
				Usage: "base directory for encrypted local cache",
				Value: defaults.CacheDir,
			},
		},
		Before: func(ctx context.Context, command *urfavecli.Command) (context.Context, error) {
			cfg, err := resolveClientConfig(command)
			if err != nil {
				return ctx, err
			}

			command.Root().Metadata[clientConfigMetadataKey] = cfg
			return ctx, nil
		},
		Commands: []*urfavecli.Command{
			newHealthCommand(factory),
			newRegisterCommand(input, factory, passwords),
			newLoginCommand(input, factory, passwords),
			newLogoutCommand(factory),
			newWhoamiCommand(factory),
			newSyncCommand(input, factory, passwords),
			newRecordsCommand(input, factory, passwords),
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
