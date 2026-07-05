package config

import (
	"encoding/base64"
	"errors"
	"flag"
	"fmt"
	"io"
	"os"
	"time"
)

const (
	defaultAddress = "localhost:8080"
	defaultJWTTTL  = 15 * time.Minute
	jwtSecretSize  = 32
)

// Config содержит конфигурацию Сервера.
type Config struct {
	Address     string
	DatabaseDSN string
	TLSCertFile string
	TLSKeyFile  string
	JWTSecret   []byte
	JWTTTL      time.Duration
}

// Parse формирует конфигурацию Сервера из переменных окружения
// и аргументов командной строки.
func Parse(args []string) (Config, error) {
	cfg := Config{
		Address:     defaultAddress,
		DatabaseDSN: os.Getenv("DATABASE_DSN"),
		TLSCertFile: os.Getenv("TLS_CERT_FILE"),
		TLSKeyFile:  os.Getenv("TLS_KEY_FILE"),
		JWTTTL:      defaultJWTTTL,
	}
	jwtSecretRaw := os.Getenv("JWT_SECRET")

	if address := os.Getenv("ADDRESS"); address != "" {
		cfg.Address = address
	}

	if jwtTTL := os.Getenv("JWT_TTL"); jwtTTL != "" {
		duration, err := time.ParseDuration(jwtTTL)
		if err != nil {
			return Config{}, fmt.Errorf("parse JWT TTL: %w", err)
		}

		cfg.JWTTTL = duration
	}

	flags := flag.NewFlagSet("server", flag.ContinueOnError)
	flags.SetOutput(io.Discard)

	flags.StringVar(&cfg.Address, "a", cfg.Address, "server listen address")
	flags.StringVar(&cfg.DatabaseDSN, "database-dsn", cfg.DatabaseDSN, "PostgreSQL connection string")
	flags.StringVar(&cfg.TLSCertFile, "tls-cert", cfg.TLSCertFile, "path to TLS certificate file")
	flags.StringVar(&cfg.TLSKeyFile, "tls-key", cfg.TLSKeyFile, "path to TLS private key file")
	flags.DurationVar(&cfg.JWTTTL, "jwt-ttl", cfg.JWTTTL, "JWT access token TTL")

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

	if jwtSecretRaw == "" {
		return Config{}, errors.New("JWT secret is required")
	}

	jwtSecret, err := decodeJWTSecret(jwtSecretRaw)
	if err != nil {
		return Config{}, err
	}
	cfg.JWTSecret = jwtSecret

	if cfg.JWTTTL <= 0 {
		return Config{}, errors.New("JWT TTL must be positive")
	}

	return cfg, nil
}

func decodeJWTSecret(value string) ([]byte, error) {
	secret, err := base64.StdEncoding.DecodeString(value)
	if err != nil {
		return nil, fmt.Errorf("decode JWT secret: %w", err)
	}

	if len(secret) != jwtSecretSize {
		return nil, fmt.Errorf("JWT secret must decode to %d bytes", jwtSecretSize)
	}

	return secret, nil
}
