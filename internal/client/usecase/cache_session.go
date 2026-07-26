package usecase

import (
	"context"
	"errors"

	"github.com/xhrobj/gopherkeeper/internal/model"
)

// CacheSession удерживает открытый зашифрованный локальный кеш для серии
// read-only операций одного интерактивного клиентского сценария.
type CacheSession struct {
	repository OfflineCacheRepository
}

// OpenCacheSession открывает существующий зашифрованный кеш аккаунта и
// возвращает сессию, которую вызывающая сторона обязана закрыть.
func (a *Application) OpenCacheSession(
	ctx context.Context,
	request OfflineReadRequest,
) (*CacheSession, error) {
	return openCacheSession(ctx, a.offlineCaches, a.serverAddress, request)
}

// OpenCacheSession открывает существующий зашифрованный кеш аккаунта и
// возвращает сессию, которую вызывающая сторона обязана закрыть.
func (a *OfflineApplication) OpenCacheSession(
	ctx context.Context,
	request OfflineReadRequest,
) (*CacheSession, error) {
	return openCacheSession(ctx, a.offlineCaches, a.serverAddress, request)
}

func openCacheSession(
	ctx context.Context,
	offlineCaches OfflineCacheRepositoryProvider,
	serverAddress string,
	request OfflineReadRequest,
) (*CacheSession, error) {
	repository, err := openOfflineCache(ctx, offlineCaches, serverAddress, request)
	if err != nil {
		return nil, err
	}

	return &CacheSession{repository: repository}, nil
}

// ListRecords возвращает metadata записей из уже открытого локального кеша.
func (session *CacheSession) ListRecords(ctx context.Context) ([]model.RecordMetadata, error) {
	if session == nil || session.repository == nil {
		return nil, newUserError("local cache session is closed", errors.New("cache session is closed"))
	}

	records, err := session.repository.ListMetadata(ctx)
	if err != nil {
		return nil, newUserError("failed to read encrypted local cache", err)
	}

	return records, nil
}

// GetRecord возвращает полную запись из уже открытого локального кеша.
func (session *CacheSession) GetRecord(ctx context.Context, recordID string) (model.Record, error) {
	if err := model.ValidateRecordID(recordID); err != nil {
		return model.Record{}, err
	}

	if session == nil || session.repository == nil {
		return model.Record{}, newUserError("local cache session is closed", errors.New("cache session is closed"))
	}

	record, err := session.repository.Get(ctx, recordID)
	if err != nil {
		if errors.Is(err, ErrCachedRecordNotFound) {
			return model.Record{}, newUserError("record not found in local cache", err)
		}
		return model.Record{}, newUserError("failed to read encrypted local cache", err)
	}

	return record, nil
}

// Close закрывает репозиторий локального кеша и очищает ссылку на него.
func (session *CacheSession) Close() error {
	if session == nil || session.repository == nil {
		return nil
	}

	repository := session.repository
	session.repository = nil

	if err := repository.Close(); err != nil {
		return newUserError("failed to close encrypted local cache", err)
	}

	return nil
}
