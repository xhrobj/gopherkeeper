package main

import (
	"context"
	"fmt"
	"io"
	"log"
	"os"

	"github.com/xhrobj/gopherkeeper/internal/buildinfo"
	"github.com/xhrobj/gopherkeeper/internal/logger"
	"github.com/xhrobj/gopherkeeper/internal/server/config"
	"github.com/xhrobj/gopherkeeper/internal/server/postgres"
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

	if err := run(ctx); err != nil {
		log.Fatal(err)
	}
}

func run(ctx context.Context) error {
	cfg, err := config.Parse(os.Args[1:])
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

	pool, err := postgres.Open(ctx, cfg.DatabaseDSN)
	if err != nil {
		return err
	}
	defer pool.Close()

	lg.Info("postgres connection verified")

	lg.Info(
		"server initialized",
		zap.String("server_address", cfg.Address),
	)

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
         -= Server: Keeping your secrets secure. =-

`
	_, err := fmt.Fprint(output, banner)

	return err
}
