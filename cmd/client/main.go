package main

import (
	"context"
	"fmt"
	"io"
	"log"
	"os"

	"github.com/xhrobj/gopherkeeper/internal/buildinfo"
	"github.com/xhrobj/gopherkeeper/internal/client/config"
	"github.com/xhrobj/gopherkeeper/internal/client/httpclient"
	"github.com/xhrobj/gopherkeeper/internal/logger"
	"go.uber.org/zap"
)

var (
	buildVersion = ""
	buildDate    = ""
	buildCommit  = ""
)

func main() {
	if err := buildinfo.Print(os.Stdout, buildinfo.Info{
		Version: buildVersion,
		Date:    buildDate,
		Commit:  buildCommit,
	}); err != nil {
		log.Fatal(err)
	}

	if err := printBanner(os.Stdout); err != nil {
		log.Fatal(err)
	}

	ctx := context.Background()

	if err := run(ctx, os.Args[1:], os.Stdout); err != nil {
		log.Fatal(err)
	}
}

func run(ctx context.Context, args []string, output io.Writer) error {
	if len(args) == 0 {
		return fmt.Errorf("command is required")
	}

	command := args[0]
	if command != "health" {
		return fmt.Errorf("unknown command %q", command)
	}

	cfg, err := config.Parse(args[1:])
	if err != nil {
		return err
	}

	lg, err := logger.New()
	if err != nil {
		return err
	}
	defer func() {
		_ = lg.Sync()
	}()

	lg.Info(
		"client initialized",
		zap.String("server_address", cfg.Address),
	)

	client, err := httpclient.New(cfg.Address, cfg.CACertFile)
	if err != nil {
		return err
	}

	status, err := client.Health(ctx)
	if err != nil {
		return err
	}

	if _, err := fmt.Fprintf(output, "Server status: %s\n", status); err != nil {
		return fmt.Errorf("write health status: %w", err)
	}

	return nil
}

func printBanner(output io.Writer) error {
	const banner = `
  ________              .__     ____  __.
 /  _____/  ____ ______ |  |__ |    |/ _|____   ____ ______   ___________
/   \  ___ /  _ \\____ \|  |  \|      <_/ __ \_/ __ \\____ \_/ __ \_  __ \
\    \_\  (  <_> )  |_> >   Y  \    |  \  ___/\  ___/|  |_> >  ___/|  | \/
 \______  /\____/|   __/|___|  /____|__ \___  >\___  >   __/ \___  >__|
        \/       |__|        \/        \/   \/     \/|__|        \/
         -= Client: Access your secrets securely. =-

`
	_, err := fmt.Fprint(output, banner)

	return err
}
