package config

import (
	"encoding/base64"
	"errors"
	"flag"
	"fmt"
	"io"
	"os"
	"strings"
	"time"

	"github.com/xhrobj/gopherkeeper/internal/server/recordcrypto"
)

const (
	defaultHTTPAddress = "localhost:8080"
	defaultGRPCAddress = "localhost:50051"
	defaultJWTTTL      = 15 * time.Minute
	jwtSecretSize      = 32
)

// Config содержит конфигурацию Сервера.
type Config struct {
	// HTTPAddress задаёт address HTTPS listener'а Сервера.
	HTTPAddress string

	// GRPCAddress задаёт address gRPC listener'а Сервера.
	GRPCAddress string

	// DatabaseDSN задаёт PostgreSQL DSN для подключения Сервера к базе данных.
	DatabaseDSN string

	// TLSCertFile задаёт путь к TLS certificate Сервера.
	TLSCertFile string

	// TLSKeyFile задаёт путь к TLS private key Сервера.
	TLSKeyFile string

	// JWTSecret задаёт секретный ключ для подписи и проверки JWT.
	JWTSecret []byte

	// JWTTTL задаёт время жизни JWT access token.
	JWTTTL time.Duration

	// RecordMasterKey задаёт мастер-ключ для серверного шифрования payload'ов записей.
	RecordMasterKey []byte

	// RecordKeyID задаёт идентификатор активного ключа шифрования записей.
	RecordKeyID string
}

// Parse формирует конфигурацию Сервера из переменных окружения
// и аргументов командной строки.
func Parse(args []string) (Config, error) {
	cfg, jwtSecretRaw, recordMasterKeyRaw, err := configFromEnvironment()
	if err != nil {
		return Config{}, err
	}
	if err := parseServerFlags(args, &cfg); err != nil {
		return Config{}, err
	}
	if err := validateRequiredConfig(&cfg); err != nil {
		return Config{}, err
	}
	if err := configureJWTSecret(&cfg, jwtSecretRaw); err != nil {
		return Config{}, err
	}
	if cfg.JWTTTL <= 0 {
		return Config{}, errors.New("JWT TTL must be positive")
	}
	if err := configureRecordMasterKey(&cfg, recordMasterKeyRaw); err != nil {
		return Config{}, err
	}
	if err := normalizeRecordKeyID(&cfg); err != nil {
		return Config{}, err
	}

	return cfg, nil
}

func configFromEnvironment() (Config, string, string, error) {
	cfg := Config{
		HTTPAddress: defaultHTTPAddress,
		GRPCAddress: defaultGRPCAddress,
		DatabaseDSN: os.Getenv("DATABASE_DSN"),
		TLSCertFile: os.Getenv("TLS_CERT_FILE"),
		TLSKeyFile:  os.Getenv("TLS_KEY_FILE"),
		JWTTTL:      defaultJWTTTL,
		RecordKeyID: recordcrypto.DefaultKeyID,
	}
	jwtSecretRaw := os.Getenv("JWT_SECRET")
	recordMasterKeyRaw := os.Getenv("RECORD_MASTER_KEY")

	if address := os.Getenv("ADDRESS"); address != "" {
		cfg.HTTPAddress = address
	}
	if grpcAddress := os.Getenv("GRPC_ADDRESS"); grpcAddress != "" {
		cfg.GRPCAddress = grpcAddress
	}
	if jwtTTL := os.Getenv("JWT_TTL"); jwtTTL != "" {
		duration, err := time.ParseDuration(jwtTTL)
		if err != nil {
			return Config{}, "", "", fmt.Errorf("parse JWT TTL: %w", err)
		}
		cfg.JWTTTL = duration
	}
	if recordKeyID := os.Getenv("RECORD_KEY_ID"); recordKeyID != "" {
		cfg.RecordKeyID = recordKeyID
	}

	return cfg, jwtSecretRaw, recordMasterKeyRaw, nil
}

func parseServerFlags(args []string, cfg *Config) error {
	flags := flag.NewFlagSet("server", flag.ContinueOnError)
	flags.SetOutput(io.Discard)

	flags.StringVar(&cfg.HTTPAddress, "a", cfg.HTTPAddress, "HTTPS listen address")
	flags.StringVar(&cfg.HTTPAddress, "address", cfg.HTTPAddress, "HTTPS listen address")
	flags.StringVar(&cfg.GRPCAddress, "g", cfg.GRPCAddress, "gRPC listen address")
	flags.StringVar(&cfg.GRPCAddress, "grpc-address", cfg.GRPCAddress, "gRPC listen address")
	flags.StringVar(&cfg.DatabaseDSN, "database-dsn", cfg.DatabaseDSN, "PostgreSQL connection string")
	flags.StringVar(&cfg.TLSCertFile, "tls-cert", cfg.TLSCertFile, "path to TLS certificate file")
	flags.StringVar(&cfg.TLSKeyFile, "tls-key", cfg.TLSKeyFile, "path to TLS private key file")
	flags.DurationVar(&cfg.JWTTTL, "jwt-ttl", cfg.JWTTTL, "JWT access token TTL")

	if err := flags.Parse(args); err != nil {
		return fmt.Errorf("parse server flags: %w", err)
	}

	return nil
}

func validateRequiredConfig(cfg *Config) error {
	cfg.HTTPAddress = strings.TrimSpace(cfg.HTTPAddress)
	if cfg.HTTPAddress == "" {
		return errors.New("HTTPS address must not be empty")
	}

	cfg.GRPCAddress = strings.TrimSpace(cfg.GRPCAddress)
	if cfg.GRPCAddress == "" {
		return errors.New("gRPC address must not be empty")
	}
	if cfg.GRPCAddress == cfg.HTTPAddress {
		return errors.New("HTTPS and gRPC addresses must differ")
	}
	if cfg.DatabaseDSN == "" {
		return errors.New("database DSN is required")
	}
	if cfg.TLSCertFile == "" {
		return errors.New("tls certificate file is required")
	}
	if cfg.TLSKeyFile == "" {
		return errors.New("tls private key file is required")
	}

	return nil
}

func configureJWTSecret(cfg *Config, raw string) error {
	if raw == "" {
		return errors.New("JWT secret is required")
	}

	secret, err := decodeFixedBase64Secret(raw, jwtSecretSize, "JWT secret")
	if err != nil {
		return err
	}

	cfg.JWTSecret = secret

	return nil
}

func configureRecordMasterKey(cfg *Config, raw string) error {
	if raw == "" {
		return errors.New("record master key is required")
	}

	key, err := decodeFixedBase64Secret(raw, recordcrypto.MasterKeySize, "record master key")
	if err != nil {
		return err
	}

	cfg.RecordMasterKey = key

	return nil
}

func normalizeRecordKeyID(cfg *Config) error {
	cfg.RecordKeyID = strings.TrimSpace(cfg.RecordKeyID)
	if cfg.RecordKeyID == "" {
		return errors.New("record key ID must not be empty")
	}

	return nil
}

func decodeFixedBase64Secret(value string, size int, name string) ([]byte, error) {
	secret, err := base64.StdEncoding.DecodeString(value)
	if err != nil {
		return nil, fmt.Errorf("decode %s: %w", name, err)
	}

	if len(secret) != size {
		return nil, fmt.Errorf("%s must decode to %d bytes", name, size)
	}

	return secret, nil
}
