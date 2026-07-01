package main

import (
	"context"
	"fmt"
	"os"

	"github.com/xhrobj/gopherkeeper/internal/buildinfo"
	clientcli "github.com/xhrobj/gopherkeeper/internal/client/cli"
)

var (
	buildVersion = ""
	buildDate    = ""
	buildCommit  = ""
)

func main() {
	if err := clientcli.Run(
		context.Background(),
		os.Args,
		os.Stdout,
		os.Stderr,
		buildinfo.Info{
			Version: buildVersion,
			Date:    buildDate,
			Commit:  buildCommit,
		},
	); err != nil {
		fmt.Fprintln(os.Stderr, err)
		os.Exit(1)
	}
}
