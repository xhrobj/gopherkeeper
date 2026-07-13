package cli

import (
	"io"

	urfavecli "github.com/urfave/cli/v3"
)

type passwordStreams struct {
	input        io.Reader
	output       io.Writer
	promptOutput io.Writer
}

func loginFlags() []urfavecli.Flag {
	return []urfavecli.Flag{
		&urfavecli.StringFlag{
			Name:     "login",
			Aliases:  []string{"l"},
			Usage:    "user login",
			Required: true,
		},
	}
}
