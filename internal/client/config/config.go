// Package config предоставляет конфигурацию Клиента GophKeeper.
package config

import "os"

const defaultAddress = "localhost:8080"

// Config содержит конфигурацию Клиента.
type Config struct {
	Address    string
	CACertFile string
}

// Load формирует базовую конфигурацию Клиента из переменных окружения
// и значений по умолчанию.
func Load() Config {
	cfg := Config{
		Address:    defaultAddress,
		CACertFile: os.Getenv("CA_CERT_FILE"),
	}

	if address := os.Getenv("ADDRESS"); address != "" {
		cfg.Address = address
	}

	return cfg
}
