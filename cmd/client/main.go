package main

import (
	"fmt"
	"io"
	"log"
	"os"

	"github.com/xhrobj/gopherkeeper/internal/buildinfo"
	"github.com/xhrobj/gopherkeeper/internal/client/config"
	"github.com/xhrobj/gopherkeeper/internal/logger"
	"go.uber.org/zap"
)

var (
	buildVersion = ""
	buildDate    = ""
	buildCommit  = ""
)

func main() {
	err := buildinfo.Print(os.Stdout, buildVersion, buildDate, buildCommit)
	if err != nil {
		log.Fatal(err)
	}

	if err := printBanner(os.Stdout); err != nil {
		log.Fatal(err)
	}

	if err := run(); err != nil {
		log.Fatal(err)
	}
}

func run() error {
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

	lg.Info(
		"client is ready",
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
         -= Client: Access your secrets securely. =-

`
	_, err := fmt.Fprint(output, banner)

	return err
}
