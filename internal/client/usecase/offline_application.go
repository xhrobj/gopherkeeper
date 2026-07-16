package usecase

import (
	"context"
	"errors"

	"github.com/xhrobj/gopherkeeper/internal/model"
)

// ListCachedRecords возвращает metadata записей из существующего
// зашифрованного локального кеша без обращения к online-сессии или Серверу.
func (a *Application) ListCachedRecords(
	ctx context.Context,
	request OfflineReadRequest,
) (result OfflineListResult, err error) {
	repository, err := a.openOfflineCache(ctx, request)
	if err != nil {
		return OfflineListResult{}, err
	}
	defer func() {
		err = closeOfflineCache(repository, err)
		if err != nil {
			result = OfflineListResult{}
		}
	}()

	records, err := repository.ListMetadata(ctx)
	if err != nil {
		return OfflineListResult{}, newUserError("failed to read encrypted local cache", err)
	}

	return OfflineListResult{
		Records:    records,
		Source:     OfflineSourceLocalCache,
		MayBeStale: true,
	}, nil
}

// GetCachedRecord возвращает полную запись из существующего зашифрованного
// локального кеша без обращения к online-сессии или Серверу.
func (a *Application) GetCachedRecord(
	ctx context.Context,
	request OfflineReadRequest,
	recordID string,
) (result OfflineGetResult, err error) {
	if err := model.ValidateRecordID(recordID); err != nil {
		return OfflineGetResult{}, err
	}

	repository, err := a.openOfflineCache(ctx, request)
	if err != nil {
		return OfflineGetResult{}, err
	}
	defer func() {
		err = closeOfflineCache(repository, err)
		if err != nil {
			result = OfflineGetResult{}
		}
	}()

	record, err := repository.Get(ctx, recordID)
	if err != nil {
		if errors.Is(err, ErrCachedRecordNotFound) {
			return OfflineGetResult{}, newUserError("record not found in local cache", err)
		}

		return OfflineGetResult{}, newUserError("failed to read encrypted local cache", err)
	}

	return OfflineGetResult{
		Record:     record,
		Source:     OfflineSourceLocalCache,
		MayBeStale: true,
	}, nil
}

func (a *Application) openOfflineCache(
	ctx context.Context,
	request OfflineReadRequest,
) (OfflineCacheRepository, error) {
	canonicalLogin, err := model.CanonicalizeLogin(request.Login)
	if err != nil {
		return nil, newUserError("invalid login", err)
	}

	repository, err := a.offlineCaches(
		ctx,
		a.serverAddress,
		canonicalLogin,
		[]byte(request.Password),
	)
	if err != nil {
		if errors.Is(err, ErrLocalCacheNotFound) {
			return nil, newUserError("local cache not found, run sync while online first", err)
		}

		return nil, newUserError("failed to open encrypted local cache", err)
	}

	return repository, nil
}

func closeOfflineCache(repository OfflineCacheRepository, operationErr error) error {
	closeErr := repository.Close()
	if closeErr == nil {
		return operationErr
	}

	wrapped := newUserError("failed to close encrypted local cache", closeErr)
	if operationErr == nil {
		return wrapped
	}

	return errors.Join(operationErr, wrapped)
}
