package app

import (
	"context"
	"fmt"

	"github.com/xhrobj/gopherkeeper/internal/client/cache"
	"github.com/xhrobj/gopherkeeper/internal/client/usecase"
)

var _ usecase.CacheRepository = (*cache.Repository)(nil)

func encryptedCacheRepositoryProvider(baseDirectory string) usecase.CacheRepositoryProvider {
	return func(
		ctx context.Context,
		serverAddress string,
		canonicalLogin string,
		password []byte,
	) (usecase.CacheRepository, error) {
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
