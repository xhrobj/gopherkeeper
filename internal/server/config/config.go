// Package config предоставляет конфигурацию Сервера GophKeeper.
package config

import (
	"errors"
	"flag"
	"fmt"
	"io"
	"os"
)

const defaultAddress = "localhost:8080"

// Config содержит конфигурацию Сервера.
type Config struct {
	Address     string
	DatabaseDSN string
	TLSCertFile string
	TLSKeyFile  string
}

// Parse формирует конфигурацию Сервера из переменных окружения
// и аргументов командной строки.
func Parse(args []string) (Config, error) {
	cfg := Config{
		Address:     defaultAddress,
		DatabaseDSN: os.Getenv("DATABASE_DSN"),
		TLSCertFile: os.Getenv("TLS_CERT_FILE"),
		TLSKeyFile:  os.Getenv("TLS_KEY_FILE"),
	}

	if address := os.Getenv("ADDRESS"); address != "" {
		cfg.Address = address
	}

	flags := flag.NewFlagSet("server", flag.ContinueOnError)
	flags.SetOutput(io.Discard)

	flags.StringVar(&cfg.Address, "a", cfg.Address, "server listen address")
	flags.StringVar(&cfg.DatabaseDSN, "database-dsn", cfg.DatabaseDSN, "PostgreSQL connection string")
	flags.StringVar(&cfg.TLSCertFile, "tls-cert", cfg.TLSCertFile, "path to TLS certificate file")
	flags.StringVar(&cfg.TLSKeyFile, "tls-key", cfg.TLSKeyFile, "path to TLS private key file")

	if err := flags.Parse(args); err != nil {
		return Config{}, fmt.Errorf("parse server flags: %w", err)
	}

	if cfg.DatabaseDSN == "" {
		return Config{}, errors.New("database DSN is required")
	}

	if cfg.TLSCertFile == "" {
		return Config{}, errors.New("tls certificate file is required")
	}

	if cfg.TLSKeyFile == "" {
		return Config{}, errors.New("tls private key file is required")
	}

	return cfg, nil
}
