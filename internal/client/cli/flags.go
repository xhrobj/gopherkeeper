package cli

import (
	"errors"
	"os"

	urfavecli "github.com/urfave/cli/v3"
	"github.com/xhrobj/gopherkeeper/internal/client/config"
)

const (
	configFlag      = "config"
	transportFlag   = "transport"
	addressFlag     = "address"
	grpcAddressFlag = "grpc-address"
	caCertFlag      = "ca-cert"
	sessionDirFlag  = "session-dir"
	cacheDirFlag    = "cache-dir"

	clientConfigMetadataKey      = "client-config"
	clientConfigFileMetadataKey  = "client-config-file"
	clientApplicationMetadataKey = "client-application"
)

func resolveClientConfigWithFile(command *urfavecli.Command) (config.Config, string, error) {
	configFile := nonEmptyEnvironmentValue("CONFIG")
	overrides := config.Overrides{
		Transport:   nonEmptyTransportEnvironmentValue("TRANSPORT"),
		Address:     nonEmptyEnvironmentValue("ADDRESS"),
		GRPCAddress: nonEmptyEnvironmentValue("GRPC_ADDRESS"),
		CACertFile:  nonEmptyEnvironmentValue("CA_CERT_FILE"),
		SessionDir:  nonEmptyEnvironmentValue("SESSION_DIR"),
		CacheDir:    nonEmptyEnvironmentValue("CACHE_DIR"),
	}

	if value := explicitStringFlag(command, configFlag); value != nil {
		configFile = value
	}
	if value := explicitTransportFlag(command, transportFlag); value != nil {
		overrides.Transport = value
	}
	if value := explicitStringFlag(command, addressFlag); value != nil {
		overrides.Address = value
	}
	if value := explicitStringFlag(command, grpcAddressFlag); value != nil {
		overrides.GRPCAddress = value
	}
	if value := explicitStringFlag(command, caCertFlag); value != nil {
		overrides.CACertFile = value
	}
	if value := explicitStringFlag(command, sessionDirFlag); value != nil {
		overrides.SessionDir = value
	}
	if value := explicitStringFlag(command, cacheDirFlag); value != nil {
		overrides.CacheDir = value
	}

	var configFilePath string
	if configFile != nil {
		configFilePath = *configFile
	}

	cfg, err := config.Resolve(configFilePath, overrides)
	if err != nil {
		return config.Config{}, "", err
	}

	return cfg, configFilePath, nil
}

func explicitStringFlag(command *urfavecli.Command, name string) *string {
	if !command.IsSet(name) {
		return nil
	}

	value := command.String(name)
	return &value
}

func explicitTransportFlag(command *urfavecli.Command, name string) *config.Transport {
	value := explicitStringFlag(command, name)
	if value == nil {
		return nil
	}

	transport := config.Transport(*value)
	return &transport
}

func nonEmptyEnvironmentValue(name string) *string {
	value := os.Getenv(name)
	if value == "" {
		return nil
	}

	return &value
}

func nonEmptyTransportEnvironmentValue(name string) *config.Transport {
	value := nonEmptyEnvironmentValue(name)
	if value == nil {
		return nil
	}

	transport := config.Transport(*value)
	return &transport
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

func configFileFromCommand(command *urfavecli.Command) (string, error) {
	value, ok := command.Root().Metadata[clientConfigFileMetadataKey]
	if !ok {
		return "", nil
	}

	path, ok := value.(string)
	if !ok {
		return "", errors.New("client config file has unexpected type")
	}

	return path, nil
}
