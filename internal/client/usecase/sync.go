package usecase

import (
	"context"

	"github.com/xhrobj/gopherkeeper/internal/model"
)

// CacheRepository описывает операции зашифрованного локального кеша,
// необходимые application-сценарию синхронизации.
type CacheRepository interface {
	ListState(ctx context.Context) ([]RecordState, error)
	ApplyChanges(ctx context.Context, upserts []model.Record, deleteIDs []string) error
	Close() error
}

// CacheRepositoryProvider лениво открывает зашифрованный кеш конкретного
// аккаунта только после успешной повторной аутентификации пользователя.
type CacheRepositoryProvider func(
	ctx context.Context,
	serverAddress string,
	canonicalLogin string,
	password []byte,
) (CacheRepository, error)

// SyncRequest содержит параметры явной синхронизации локального кеша.
type SyncRequest struct {
	// Password содержит password текущего пользователя для повторной online-аутентификации
	// и получения локального ключа шифрования кеша.
	Password string

	// RefreshStale разрешает заменить устаревшие локальные записи актуальными
	// версиями с Сервера.
	RefreshStale bool
}

// SyncResult содержит безопасный отчёт application-сценария синхронизации.
type SyncResult struct {
	// Added содержит metadata новых записей, добавленных в локальный кеш.
	Added []model.RecordMetadata

	// Updated содержит записи, обновлённые после явного разрешения RefreshStale.
	Updated []RevisionChange

	// Removed содержит локальные записи, удалённые из кеша как отсутствующие на Сервере.
	Removed []RecordState

	// Stale содержит устаревшие локальные записи, не заменённые в текущем запуске.
	Stale []RevisionChange

	// Unchanged содержит количество записей с одинаковой server/local revision.
	Unchanged int
}
