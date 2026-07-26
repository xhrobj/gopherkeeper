package config

import (
	"bytes"
	"encoding/json"
	"errors"
	"fmt"
	"io"
	"os"
	"strings"
)

const (
	defaultAddress     = "localhost:8080"
	defaultGRPCAddress = "localhost:50051"
)

// Transport задаёт сетевой протокол Клиента для обращения к Серверу.
type Transport string

const (
	// TransportHTTPS выбирает HTTPS API.
	TransportHTTPS Transport = "https"
	// TransportGRPC выбирает gRPC API.
	TransportGRPC Transport = "grpc"
)

// Config содержит настройки командного Клиента.
type Config struct {
	// Transport задаёт активный сетевой протокол Клиента.
	Transport Transport

	// Address задаёт адрес HTTPS API Сервера в формате host:port.
	Address string

	// GRPCAddress задаёт адрес gRPC API Сервера в формате host:port.
	GRPCAddress string

	// CACertFile задаёт путь к PEM-файлу доверенного CA для TLS-подключений к Серверу.
	CACertFile string

	// SessionDir задаёт каталог локального хранения online-сессии Клиента.
	// Файл внутри каталога всегда называется session.json.
	SessionDir string

	// CacheDir задаёт базовый каталог локального зашифрованного кеша.
	CacheDir string
}

// Overrides содержит значения конфигурации, заданные через источники с более
// высоким приоритетом, чем JSON-файл.
type Overrides struct {
	// Transport переопределяет активный сетевой протокол.
	Transport *Transport

	// Address переопределяет адрес HTTPS API Сервера.
	Address *string

	// GRPCAddress переопределяет адрес gRPC API Сервера.
	GRPCAddress *string

	// CACertFile переопределяет путь к дополнительному CA certificate.
	CACertFile *string

	// SessionDir переопределяет каталог online-сессии.
	SessionDir *string

	// CacheDir переопределяет базовый каталог локального зашифрованного кеша.
	CacheDir *string
}

type fileConfig struct {
	Transport   *Transport `json:"transport"`
	Address     *string    `json:"address"`
	GRPCAddress *string    `json:"grpc_address"`
	CACertFile  *string    `json:"ca_cert_file"`
	SessionDir  *string    `json:"session_dir"`
	CacheDir    *string    `json:"cache_dir"`
}

// Default возвращает конфигурацию Клиента со значениями по умолчанию.
func Default() Config {
	return Config{
		Transport:   TransportHTTPS,
		Address:     defaultAddress,
		GRPCAddress: defaultGRPCAddress,
	}
}

// Resolve формирует конфигурацию Клиента из JSON-файла и значений с более
// высоким приоритетом.
//
// JSON-файл читается только при непустом пути. Приоритет источников:
// overrides > config file > default.
func Resolve(configFile string, overrides Overrides) (Config, error) {
	cfg := Default()

	if configFile != "" {
		if err := applyFile(&cfg, configFile); err != nil {
			return Config{}, err
		}
	}

	applyOverrides(&cfg, overrides)
	normalizeNetworkAddresses(&cfg)

	if err := validate(cfg); err != nil {
		return Config{}, err
	}

	return cfg, nil
}

// Save записывает конфигурацию Клиента в JSON-файл.
func Save(path string, cfg Config) error {
	if strings.TrimSpace(path) == "" {
		return errors.New("client config file path is required")
	}

	normalizeNetworkAddresses(&cfg)
	if err := validate(cfg); err != nil {
		return err
	}

	data, err := json.MarshalIndent(struct {
		Transport   Transport `json:"transport"`
		Address     string    `json:"address"`
		GRPCAddress string    `json:"grpc_address"`
		CACertFile  string    `json:"ca_cert_file"`
		SessionDir  string    `json:"session_dir"`
		CacheDir    string    `json:"cache_dir"`
	}{
		Transport:   cfg.Transport,
		Address:     cfg.Address,
		GRPCAddress: cfg.GRPCAddress,
		CACertFile:  cfg.CACertFile,
		SessionDir:  cfg.SessionDir,
		CacheDir:    cfg.CacheDir,
	}, "", "  ")
	if err != nil {
		return fmt.Errorf("encode client config file: %w", err)
	}

	data = append(data, '\n')

	if err := os.WriteFile(path, data, 0o600); err != nil {
		return fmt.Errorf("write client config file: %w", err)
	}

	return nil
}

// ActiveAddress возвращает адрес, соответствующий выбранному transport'у.
func (cfg Config) ActiveAddress() string {
	if cfg.Transport == TransportGRPC {
		return cfg.GRPCAddress
	}

	return cfg.Address
}

func normalizeNetworkAddresses(cfg *Config) {
	cfg.Address = strings.TrimSpace(cfg.Address)
	cfg.GRPCAddress = strings.TrimSpace(cfg.GRPCAddress)
}

func validate(cfg Config) error {
	switch cfg.Transport {
	case TransportHTTPS:
		if strings.TrimSpace(cfg.Address) == "" {
			return errors.New("HTTPS server address is required")
		}
	case TransportGRPC:
		if strings.TrimSpace(cfg.GRPCAddress) == "" {
			return errors.New("gRPC server address is required")
		}
	default:
		return fmt.Errorf("unsupported client transport %q", cfg.Transport)
	}

	return nil
}

func applyFile(cfg *Config, path string) error {
	data, err := os.ReadFile(path)
	if err != nil {
		return fmt.Errorf("read client config file: %w", err)
	}

	decoder := json.NewDecoder(bytes.NewReader(data))
	decoder.DisallowUnknownFields()

	var file fileConfig
	if err := decoder.Decode(&file); err != nil {
		return fmt.Errorf("decode client config file: %w", err)
	}

	if err := ensureSingleJSONValue(decoder); err != nil {
		return fmt.Errorf("decode client config file: %w", err)
	}

	if file.Transport != nil {
		cfg.Transport = *file.Transport
	}
	if file.Address != nil {
		cfg.Address = *file.Address
	}
	if file.GRPCAddress != nil {
		cfg.GRPCAddress = *file.GRPCAddress
	}
	if file.CACertFile != nil {
		cfg.CACertFile = *file.CACertFile
	}
	if file.SessionDir != nil {
		cfg.SessionDir = *file.SessionDir
	}
	if file.CacheDir != nil {
		cfg.CacheDir = *file.CacheDir
	}

	return nil
}

func ensureSingleJSONValue(decoder *json.Decoder) error {
	var extra any
	if err := decoder.Decode(&extra); err == nil {
		return errors.New("multiple JSON values")
	} else if !errors.Is(err, io.EOF) {
		return err
	}

	return nil
}

func applyOverrides(cfg *Config, overrides Overrides) {
	if overrides.Transport != nil {
		cfg.Transport = *overrides.Transport
	}
	if overrides.Address != nil {
		cfg.Address = *overrides.Address
	}
	if overrides.GRPCAddress != nil {
		cfg.GRPCAddress = *overrides.GRPCAddress
	}
	if overrides.CACertFile != nil {
		cfg.CACertFile = *overrides.CACertFile
	}
	if overrides.SessionDir != nil {
		cfg.SessionDir = *overrides.SessionDir
	}
	if overrides.CacheDir != nil {
		cfg.CacheDir = *overrides.CacheDir
	}
}
