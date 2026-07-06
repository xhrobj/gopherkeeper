package config

import "os"

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

// Load формирует базовую конфигурацию Клиента из переменных окружения
// и значений по умолчанию.
func Load() Config {
	cfg := Config{
		Address:     defaultAddress,
		CACertFile:  os.Getenv("CA_CERT_FILE"),
		SessionFile: os.Getenv("SESSION_FILE"),
	}

	if address := os.Getenv("ADDRESS"); address != "" {
		cfg.Address = address
	}

	return cfg
}
