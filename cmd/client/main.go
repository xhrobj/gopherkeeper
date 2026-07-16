// Package main запускает CLI-клиент GophKeeper.
package main

import (
	"context"
	"fmt"
	"os"
	"os/signal"
	"syscall"

	"github.com/xhrobj/gopherkeeper/internal/buildinfo"
	"github.com/xhrobj/gopherkeeper/internal/client/cli"
)

var (
	buildVersion = ""
	buildDate    = ""
	buildCommit  = ""
)

func main() {
	ctx, stop := signal.NotifyContext(context.Background(), os.Interrupt, syscall.SIGTERM)
	defer stop()

	buildInfo := buildinfo.Info{
		Version: buildVersion,
		Date:    buildDate,
		Commit:  buildCommit,
	}

	if err := cli.Run(ctx, os.Args, os.Stdout, os.Stderr, buildInfo); err != nil {
		fmt.Fprintln(os.Stderr, err)
		os.Exit(1)
	}
}
