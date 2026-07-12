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

const defaultAddress = "localhost:8080"

// Config содержит настройки командного Клиента.
type Config struct {
	// Address задаёт адрес Сервера в формате host:port.
	Address string

	// CACertFile задаёт путь к PEM-файлу доверенного CA для HTTPS-подключений к Серверу.
	CACertFile string

	// SessionFile задаёт путь к файлу локального хранения online-сессии Клиента.
	SessionFile string
}

// Overrides содержит значения конфигурации, заданные через источники с более
// высоким приоритетом, чем JSON-файл.
type Overrides struct {
	// Address переопределяет адрес Сервера.
	Address *string

	// CACertFile переопределяет путь к дополнительному CA certificate.
	CACertFile *string

	// SessionFile переопределяет путь к файлу online-сессии.
	SessionFile *string
}

type fileConfig struct {
	Address     *string `json:"address"`
	CACertFile  *string `json:"ca_cert_file"`
	SessionFile *string `json:"session_file"`
}

// Default возвращает конфигурацию Клиента со значениями по умолчанию.
func Default() Config {
	return Config{Address: defaultAddress}
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

	if strings.TrimSpace(cfg.Address) == "" {
		return Config{}, errors.New("server address is required")
	}

	return cfg, nil
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

	if file.Address != nil {
		cfg.Address = *file.Address
	}
	if file.CACertFile != nil {
		cfg.CACertFile = *file.CACertFile
	}
	if file.SessionFile != nil {
		cfg.SessionFile = *file.SessionFile
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
	if overrides.Address != nil {
		cfg.Address = *overrides.Address
	}
	if overrides.CACertFile != nil {
		cfg.CACertFile = *overrides.CACertFile
	}
	if overrides.SessionFile != nil {
		cfg.SessionFile = *overrides.SessionFile
	}
}
