package logger

import (
	"fmt"
	"os"

	"go.uber.org/zap"
)

const defaultLevel = "info"

// New создаёт production-логгер приложения с уровнем из переменной окружения LOG_LEVEL.
// Если переменная не задана, используется уровень info.
func New() (*zap.Logger, error) {
	cfg := zap.NewProductionConfig()

	levelName := os.Getenv("LOG_LEVEL")
	if levelName == "" {
		levelName = defaultLevel
	}

	level, err := zap.ParseAtomicLevel(levelName)
	if err != nil {
		return nil, fmt.Errorf("parse LOG_LEVEL %q: %w", levelName, err)
	}

	cfg.Level = level

	return cfg.Build()
}
