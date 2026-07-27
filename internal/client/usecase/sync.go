package usecase

import (
	"context"
	"errors"

	"github.com/xhrobj/gopherkeeper/internal/model"
)

// ErrLocalCacheRecordsUnreadable означает, что metadata кеша открыта, но одна или несколько
// зашифрованных записей не читаются текущей версией клиента и требуют перезаписи.
var ErrLocalCacheRecordsUnreadable = errors.New("local cache records are unreadable")

// SyncCacheRepository описывает операции зашифрованного локального кеша,
// необходимые application-сценарию синхронизации.
type SyncCacheRepository interface {
	ListState(ctx context.Context) ([]RecordState, error)
	ValidateRecords(ctx context.Context) error
	ApplyChanges(ctx context.Context, upserts []model.Record, deleteIDs []string) error
	Close() error
}

// SyncCacheRepositoryProvider лениво открывает зашифрованный кеш конкретного
// аккаунта только после успешной повторной аутентификации пользователя.
type SyncCacheRepositoryProvider func(
	ctx context.Context,
	canonicalLogin string,
	password []byte,
) (SyncCacheRepository, error)

// SyncRequest содержит параметры явной синхронизации локального кеша.
type SyncRequest struct {
	// Password содержит password текущего пользователя для повторной online-аутентификации
	// и получения локального ключа шифрования кеша.
	Password string
}

// SyncResult содержит безопасный отчёт application-сценария синхронизации.
type SyncResult struct {
	// Added содержит metadata новых записей, добавленных в локальный кеш.
	Added []model.RecordMetadata

	// Updated содержит записи, заменённые актуальными версиями с Сервера.
	Updated []RevisionChange

	// Removed содержит локальные записи, удалённые из кеша как отсутствующие на Сервере.
	Removed []RecordState

	// Unchanged содержит количество записей с одинаковой server/local revision.
	Unchanged int
}
