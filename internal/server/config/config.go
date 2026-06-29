// Package config предоставляет конфигурацию Сервера GophKeeper.
package config

import (
	"flag"
	"fmt"
	"io"
	"os"
)

// Config содержит конфигурацию Сервера.
type Config struct {
	Address string
}

// Parse формирует конфигурацию Сервера из переменных окружения
// и аргументов командной строки.
func Parse(args []string) (Config, error) {
	cfg := Config{
		Address: "localhost:8080",
	}

	if address := os.Getenv("ADDRESS"); address != "" {
		cfg.Address = address
	}

	flags := flag.NewFlagSet("server", flag.ContinueOnError)
	flags.SetOutput(io.Discard)
	flags.StringVar(&cfg.Address, "a", cfg.Address, "server listen address")

	if err := flags.Parse(args); err != nil {
		return Config{}, fmt.Errorf("parse server flags: %w", err)
	}

	return cfg, nil
}
