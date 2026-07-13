package cli

import (
	"errors"
	"os"

	urfavecli "github.com/urfave/cli/v3"
	"github.com/xhrobj/gopherkeeper/internal/client/config"
)

const (
	configFlag      = "config"
	addressFlag     = "address"
	caCertFlag      = "ca-cert"
	sessionFileFlag = "session-file"
	cacheDirFlag    = "cache-dir"

	clientConfigMetadataKey = "client-config"
)

func resolveClientConfig(command *urfavecli.Command) (config.Config, error) {
	configFile := nonEmptyEnvironmentValue("CONFIG")
	overrides := config.Overrides{
		Address:     nonEmptyEnvironmentValue("ADDRESS"),
		CACertFile:  nonEmptyEnvironmentValue("CA_CERT_FILE"),
		SessionFile: nonEmptyEnvironmentValue("SESSION_FILE"),
		CacheDir:    nonEmptyEnvironmentValue("CACHE_DIR"),
	}

	if value := explicitStringFlag(command, configFlag); value != nil {
		configFile = value
	}
	if value := explicitStringFlag(command, addressFlag); value != nil {
		overrides.Address = value
	}
	if value := explicitStringFlag(command, caCertFlag); value != nil {
		overrides.CACertFile = value
	}
	if value := explicitStringFlag(command, sessionFileFlag); value != nil {
		overrides.SessionFile = value
	}
	if value := explicitStringFlag(command, cacheDirFlag); value != nil {
		overrides.CacheDir = value
	}

	var configFilePath string
	if configFile != nil {
		configFilePath = *configFile
	}

	return config.Resolve(configFilePath, overrides)
}

func explicitStringFlag(command *urfavecli.Command, name string) *string {
	if !command.IsSet(name) {
		return nil
	}

	value := command.String(name)
	return &value
}

func nonEmptyEnvironmentValue(name string) *string {
	value := os.Getenv(name)
	if value == "" {
		return nil
	}

	return &value
}

func configFromCommand(command *urfavecli.Command) (config.Config, error) {
	value, ok := command.Root().Metadata[clientConfigMetadataKey]
	if !ok {
		return config.Config{}, errors.New("client config is missing")
	}

	cfg, ok := value.(config.Config)
	if !ok {
		return config.Config{}, errors.New("client config has unexpected type")
	}

	return cfg, nil
}
