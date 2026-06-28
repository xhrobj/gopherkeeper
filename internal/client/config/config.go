package config

import (
	"flag"
	"fmt"
	"io"
	"os"
)

type Config struct {
	Address string
}

func Parse(args []string) (Config, error) {
	cfg := Config{
		Address: "localhost:8080",
	}

	if address := os.Getenv("ADDRESS"); address != "" {
		cfg.Address = address
	}

	flags := flag.NewFlagSet("client", flag.ContinueOnError)
	flags.SetOutput(io.Discard)
	flags.StringVar(&cfg.Address, "a", cfg.Address, "server address")

	if err := flags.Parse(args); err != nil {
		return Config{}, fmt.Errorf("parse client flags: %w", err)
	}

	return cfg, nil
}
