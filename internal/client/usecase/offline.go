package usecase

import (
	"context"
	"errors"

	"github.com/xhrobj/gopherkeeper/internal/model"
)

var (
	// ErrLocalCacheNotFound означает, что для выбранного аккаунта ещё нет
	// локального кеша, созданного явной синхронизацией.
	ErrLocalCacheNotFound = errors.New("local cache not found")

	// ErrCachedRecordNotFound означает, что запись отсутствует в существующем
	// локальном кеше аккаунта.
	ErrCachedRecordNotFound = errors.New("cached record not found")
)

// OfflineCacheRepository описывает только операции чтения существующего кеша.
type OfflineCacheRepository interface {
	ListMetadata(ctx context.Context) ([]model.RecordMetadata, error)
	Get(ctx context.Context, recordID string) (model.Record, error)
	Close() error
}

// OfflineCacheRepositoryProvider открывает только существующий зашифрованный
// кеш конкретного аккаунта для операций чтения.
type OfflineCacheRepositoryProvider func(
	ctx context.Context,
	serverAddress string,
	canonicalLogin string,
	password []byte,
) (OfflineCacheRepository, error)

// OfflineReadRequest содержит данные, необходимые для открытия существующего
// зашифрованного кеша без online-сессии и сетевого запроса.
type OfflineReadRequest struct {
	// Login определяет аккаунт локального кеша и канонизируется application-слоем.
	Login string

	// Password используется только в памяти текущего процесса для получения
	// локального ключа шифрования кеша.
	Password string
}

// OfflineSource описывает источник данных offline-чтения.
type OfflineSource string

const (
	// OfflineSourceLocalCache означает зашифрованный локальный кеш.
	OfflineSourceLocalCache OfflineSource = "local_cache"
)

// OfflineListResult содержит metadata записей существующего локального кеша
// и семантику источника данных.
type OfflineListResult struct {
	Records    []model.RecordMetadata
	Source     OfflineSource
	MayBeStale bool
}

// OfflineGetResult содержит полную расшифрованную запись локального кеша
// и семантику источника данных.
type OfflineGetResult struct {
	Record     model.Record
	Source     OfflineSource
	MayBeStale bool
}
