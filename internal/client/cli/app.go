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

type runOptions struct {
	input       io.Reader
	output      io.Writer
	errorOutput io.Writer
	info        buildinfo.Info
	factory     clientFactory
	passwords   passwordReader
	tui         tuiRunner
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
		tui:         defaultTUIRunner{},
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
		tui:         defaultTUIRunner{},
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
		options.tui,
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
	tui tuiRunner,
) *urfavecli.Command {
	defaults := config.Default()
	version := buildinfo.Value(info.Version)

	return &urfavecli.Command{
		Metadata: map[string]any{
			clientConfigMetadataKey:     defaults,
			clientConfigFileMetadataKey: "",
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
				Name:    transportFlag,
				Aliases: []string{"t"},
				Usage:   "client transport: https or grpc",
				Value:   string(defaults.Transport),
			},
			&urfavecli.StringFlag{
				Name:    addressFlag,
				Aliases: []string{"a"},
				Usage:   "HTTPS Server address",
				Value:   defaults.Address,
			},
			&urfavecli.StringFlag{
				Name:    grpcAddressFlag,
				Aliases: []string{"g"},
				Usage:   "gRPC Server address",
				Value:   defaults.GRPCAddress,
			},
			&urfavecli.StringFlag{
				Name:  caCertFlag,
				Usage: "path to an additional trusted CA certificate",
				Value: defaults.CACertFile,
			},
			&urfavecli.StringFlag{
				Name:  sessionDirFlag,
				Usage: "directory for online session file session.json",
				Value: defaults.SessionDir,
			},
			&urfavecli.StringFlag{
				Name:  cacheDirFlag,
				Usage: "base directory for encrypted local cache",
				Value: defaults.CacheDir,
			},
		},
		Before: func(ctx context.Context, command *urfavecli.Command) (context.Context, error) {
			cfg, configFile, err := resolveClientConfigWithFile(command)
			if err != nil {
				return ctx, err
			}

			command.Root().Metadata[clientConfigMetadataKey] = cfg
			command.Root().Metadata[clientConfigFileMetadataKey] = configFile
			return ctx, nil
		},
		After: func(_ context.Context, command *urfavecli.Command) error {
			return closeApplicationFromCommand(command)
		},
		Commands: []*urfavecli.Command{
			newTUICommand(input, output, info, tui),
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
