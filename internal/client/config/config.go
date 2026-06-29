// Package config предоставляет конфигурацию Клиента GophKeeper.
package config

import (
	"flag"
	"fmt"
	"io"
	"os"
)

const defaultAddress = "localhost:8080"

// Config содержит конфигурацию Клиента.
type Config struct {
	Address    string
	CACertFile string
}

// Parse формирует конфигурацию Клиента из переменных окружения
// и аргументов командной строки.
func Parse(args []string) (Config, error) {
	cfg := Config{
		Address:    defaultAddress,
		CACertFile: os.Getenv("CA_CERT_FILE"),
	}

	if address := os.Getenv("ADDRESS"); address != "" {
		cfg.Address = address
	}

	flags := flag.NewFlagSet("client", flag.ContinueOnError)
	flags.SetOutput(io.Discard)

	flags.StringVar(&cfg.Address, "a", cfg.Address, "server address")
	flags.StringVar(&cfg.CACertFile, "ca-cert", cfg.CACertFile, "path to an additional trusted CA certificate")

	if err := flags.Parse(args); err != nil {
		return Config{}, fmt.Errorf("parse client flags: %w", err)
	}

	return cfg, nil
}
