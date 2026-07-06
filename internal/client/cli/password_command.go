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

func loginPasswordFlags() []urfavecli.Flag {
	return []urfavecli.Flag{
		&urfavecli.StringFlag{
			Name:     "login",
			Usage:    "user login",
			Required: true,
		},
		&urfavecli.BoolFlag{
			Name:  "password-stdin",
			Usage: "read password from standard input",
		},
	}
}
