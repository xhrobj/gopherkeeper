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

type fileConfig struct {
	Address     *string `json:"address"`
	CACertFile  *string `json:"ca_cert_file"`
	SessionFile *string `json:"session_file"`
}

type explicitFlags struct {
	ConfigFile  *string
	Address     *string
	CACertFile  *string
	SessionFile *string
}

// Load формирует базовую конфигурацию Клиента из переменных окружения
// и значений по умолчанию.
func Load() Config {
	cfg, err := Parse(nil)
	if err != nil {
		return Config{Address: defaultAddress}
	}

	return cfg
}

// Parse формирует конфигурацию Клиента из явно указанного JSON-файла,
// переменных окружения и аргументов командной строки.
//
// JSON-файл читается только когда путь задан флагом --config / -c
// или переменной окружения CONFIG.
//
// Приоритет источников: flag > env > config file > default.
func Parse(args []string) (Config, error) {
	flags, err := scanExplicitFlags(args)
	if err != nil {
		return Config{}, err
	}

	cfg := Config{Address: defaultAddress}

	configFile, configFileExplicit := resolveConfigFile(flags.ConfigFile)

	if configFileExplicit && configFile != "" {
		if err := applyFile(&cfg, configFile); err != nil {
			return Config{}, err
		}
	}

	applyEnvironment(&cfg)
	applyFlags(&cfg, flags)

	if strings.TrimSpace(cfg.Address) == "" {
		return Config{}, errors.New("server address is required")
	}

	return cfg, nil
}

func resolveConfigFile(flagValue *string) (string, bool) {
	if flagValue != nil {
		return *flagValue, true
	}

	if envValue := os.Getenv("CONFIG"); envValue != "" {
		return envValue, true
	}

	return "", false
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

func applyEnvironment(cfg *Config) {
	if address := os.Getenv("ADDRESS"); address != "" {
		cfg.Address = address
	}
	if caCertFile := os.Getenv("CA_CERT_FILE"); caCertFile != "" {
		cfg.CACertFile = caCertFile
	}
	if sessionFile := os.Getenv("SESSION_FILE"); sessionFile != "" {
		cfg.SessionFile = sessionFile
	}
}

func applyFlags(cfg *Config, flags explicitFlags) {
	if flags.Address != nil {
		cfg.Address = *flags.Address
	}
	if flags.CACertFile != nil {
		cfg.CACertFile = *flags.CACertFile
	}
	if flags.SessionFile != nil {
		cfg.SessionFile = *flags.SessionFile
	}
}

func scanExplicitFlags(args []string) (explicitFlags, error) {
	var flags explicitFlags

	for i := 0; i < len(args); i++ {
		arg := args[i]
		if arg == "--" {
			break
		}

		name, value, hasInlineValue, ok := splitFlag(arg)
		if !ok {
			continue
		}

		setter := flagSetter(&flags, name)
		if setter == nil {
			continue
		}

		if !hasInlineValue {
			if i+1 >= len(args) {
				return explicitFlags{}, fmt.Errorf("flag %s requires a value", arg)
			}
			i++
			value = args[i]
		}

		setter(value)
	}

	return flags, nil
}

func splitFlag(arg string) (name string, value string, hasInlineValue bool, ok bool) {
	if strings.HasPrefix(arg, "--") {
		body := strings.TrimPrefix(arg, "--")
		if body == "" {
			return "", "", false, false
		}

		name, value, hasInlineValue = strings.Cut(body, "=")
		return name, value, hasInlineValue, true
	}

	if strings.HasPrefix(arg, "-") && len(arg) > 1 {
		body := strings.TrimPrefix(arg, "-")
		name, value, hasInlineValue = strings.Cut(body, "=")
		return name, value, hasInlineValue, true
	}

	return "", "", false, false
}

func flagSetter(flags *explicitFlags, name string) func(string) {
	switch name {
	case "c", "config":
		return func(value string) { flags.ConfigFile = &value }
	case "a", "address":
		return func(value string) { flags.Address = &value }
	case "ca-cert":
		return func(value string) { flags.CACertFile = &value }
	case "session-file":
		return func(value string) { flags.SessionFile = &value }
	default:
		return nil
	}
}
