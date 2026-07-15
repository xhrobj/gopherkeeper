package app

import (
	"context"
	"errors"
	"fmt"

	"github.com/xhrobj/gopherkeeper/internal/client/cache"
	"github.com/xhrobj/gopherkeeper/internal/client/usecase"
	"github.com/xhrobj/gopherkeeper/internal/model"
)

var (
	_ usecase.SyncCacheRepository    = (*cache.Repository)(nil)
	_ usecase.OfflineCacheRepository = (*offlineCacheRepository)(nil)
)

func encryptedSyncCacheRepositoryProvider(baseDirectory string) usecase.SyncCacheRepositoryProvider {
	return func(
		ctx context.Context,
		serverAddress string,
		canonicalLogin string,
		password []byte,
	) (usecase.SyncCacheRepository, error) {
		location, err := cache.ResolveLocation(baseDirectory, serverAddress, canonicalLogin)
		if err != nil {
			return nil, fmt.Errorf("resolve encrypted local cache: %w", err)
		}

		repository, err := cache.OpenRepository(ctx, location, password)
		if err != nil {
			return nil, fmt.Errorf("open encrypted local cache: %w", err)
		}

		return repository, nil
	}
}

func encryptedOfflineCacheRepositoryProvider(baseDirectory string) usecase.OfflineCacheRepositoryProvider {
	return func(
		ctx context.Context,
		serverAddress string,
		canonicalLogin string,
		password []byte,
	) (usecase.OfflineCacheRepository, error) {
		location, err := cache.ResolveLocation(baseDirectory, serverAddress, canonicalLogin)
		if err != nil {
			return nil, fmt.Errorf("resolve encrypted local cache: %w", err)
		}

		repository, err := cache.OpenExistingRepository(ctx, location, password)
		if errors.Is(err, cache.ErrLocalCacheNotFound) {
			return nil, errors.Join(usecase.ErrLocalCacheNotFound, err)
		}
		if err != nil {
			return nil, fmt.Errorf("open existing encrypted local cache: %w", err)
		}

		return &offlineCacheRepository{repository: repository}, nil
	}
}

type offlineCacheRepository struct {
	repository *cache.Repository
}

func (repository *offlineCacheRepository) ListMetadata(
	ctx context.Context,
) ([]model.RecordMetadata, error) {
	return repository.repository.ListMetadata(ctx)
}

func (repository *offlineCacheRepository) Get(
	ctx context.Context,
	recordID string,
) (model.Record, error) {
	record, err := repository.repository.Get(ctx, recordID)
	if errors.Is(err, cache.ErrLocalRecordNotFound) {
		return model.Record{}, errors.Join(usecase.ErrCachedRecordNotFound, err)
	}
	if err != nil {
		return model.Record{}, err
	}

	return record, nil
}

func (repository *offlineCacheRepository) Close() error {
	return repository.repository.Close()
}
