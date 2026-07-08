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
	defaultAddress = "localhost:8080"
	defaultJWTTTL  = 15 * time.Minute
	jwtSecretSize  = 32
)

// Config содержит конфигурацию Сервера.
type Config struct {
	// Address задаёт address HTTPS listener'а Сервера.
	Address string

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
	cfg := Config{
		Address:     defaultAddress,
		DatabaseDSN: os.Getenv("DATABASE_DSN"),
		TLSCertFile: os.Getenv("TLS_CERT_FILE"),
		TLSKeyFile:  os.Getenv("TLS_KEY_FILE"),
		JWTTTL:      defaultJWTTTL,
		RecordKeyID: recordcrypto.DefaultKeyID,
	}
	jwtSecretRaw := os.Getenv("JWT_SECRET")
	recordMasterKeyRaw := os.Getenv("RECORD_MASTER_KEY")

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

	if recordKeyID := os.Getenv("RECORD_KEY_ID"); recordKeyID != "" {
		cfg.RecordKeyID = recordKeyID
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

	jwtSecret, err := decodeFixedBase64Secret(jwtSecretRaw, jwtSecretSize, "JWT secret")
	if err != nil {
		return Config{}, err
	}
	cfg.JWTSecret = jwtSecret

	if cfg.JWTTTL <= 0 {
		return Config{}, errors.New("JWT TTL must be positive")
	}

	if recordMasterKeyRaw == "" {
		return Config{}, errors.New("record master key is required")
	}

	recordMasterKey, err := decodeFixedBase64Secret(
		recordMasterKeyRaw,
		recordcrypto.MasterKeySize,
		"record master key",
	)
	if err != nil {
		return Config{}, err
	}
	cfg.RecordMasterKey = recordMasterKey

	cfg.RecordKeyID = strings.TrimSpace(cfg.RecordKeyID)
	if cfg.RecordKeyID == "" {
		return Config{}, errors.New("record key ID must not be empty")
	}

	return cfg, nil
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
