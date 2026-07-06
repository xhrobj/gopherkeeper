package cli

import (
	urfavecli "github.com/urfave/cli/v3"
	"github.com/xhrobj/gopherkeeper/internal/client/config"
)

const (
	addressFlag     = "address"
	caCertFlag      = "ca-cert"
	sessionFileFlag = "session-file"
)

func configFromCommand(command *urfavecli.Command) config.Config {
	return config.Config{
		Address:     command.String(addressFlag),
		CACertFile:  command.String(caCertFlag),
		SessionFile: command.String(sessionFileFlag),
	}
}
