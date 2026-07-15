package app

import (
	"context"
	"fmt"

	"github.com/xhrobj/gopherkeeper/internal/client/cache"
	"github.com/xhrobj/gopherkeeper/internal/client/usecase"
)

var (
	_ usecase.SyncCacheRepository    = (*cache.Repository)(nil)
	_ usecase.OfflineCacheRepository = (*cache.Repository)(nil)
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
