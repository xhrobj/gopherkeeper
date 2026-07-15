package usecase

import (
	"context"

	"github.com/xhrobj/gopherkeeper/internal/model"
)

type CacheRepository interface {
	ListState(ctx context.Context) ([]RecordState, error)
	ApplyChanges(ctx context.Context, upserts []model.Record, deleteIDs []string) error
	Close() error
}

type CacheRepositoryProvider func(
	ctx context.Context,
	serverAddress string,
	canonicalLogin string,
	password []byte,
) (CacheRepository, error)

type SyncRequest struct {
	Password     string
	RefreshStale bool
}

type SyncResult struct {
	Added     []model.RecordMetadata
	Updated   []RevisionChange
	Removed   []RecordState
	Stale     []RevisionChange
	Unchanged int
}
