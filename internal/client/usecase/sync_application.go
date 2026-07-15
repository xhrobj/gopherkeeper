package usecase

import (
	"context"
	"errors"
	"fmt"

	"github.com/xhrobj/gopherkeeper/internal/client/session"
	"github.com/xhrobj/gopherkeeper/internal/model"
)

var (
	errSyncUserMismatch = errors.New("synchronization user mismatch")
	errSyncStateChanged = errors.New("server state changed during synchronization")
)

// Sync явно синхронизирует зашифрованный локальный кеш с актуальным состоянием Сервера.
func (a *Application) Sync(ctx context.Context, request SyncRequest) (result SyncResult, err error) {
	storedSessions, authentication, err := a.authenticateSync(ctx, request.Password)
	if err != nil {
		return SyncResult{}, err
	}

	if err := storedSessions.Save(session.Session{
		ServerAddress: a.serverAddress,
		AccessToken:   authentication.AccessToken,
		ExpiresAt:     authentication.ExpiresAt,
	}); err != nil {
		return SyncResult{}, fmt.Errorf("save online session: %w", err)
	}

	repository, err := a.openSyncCache(
		ctx,
		authentication.User.Login,
		request.Password,
	)
	if err != nil {
		return SyncResult{}, err
	}
	defer func() {
		closeErr := repository.Close()
		if closeErr == nil {
			return
		}

		result = SyncResult{}
		wrapped := newUserError("failed to close encrypted local cache", closeErr)
		if err == nil {
			err = wrapped
			return
		}
		err = errors.Join(err, wrapped)
	}()

	serverRecords, err := a.records.ListRecords(ctx, authentication.AccessToken)
	if err != nil {
		return SyncResult{}, mapRecordGatewayError("list records for synchronization", err)
	}

	localRecords, err := repository.ListState(ctx)
	if err != nil {
		return SyncResult{}, newUserError("failed to read encrypted local cache", err)
	}

	plan, err := buildSyncPlan(serverRecords, localRecords)
	if err != nil {
		return SyncResult{}, fmt.Errorf("build synchronization plan: %w", err)
	}

	upserts, err := a.loadSyncRecords(
		ctx,
		authentication.AccessToken,
		plan,
		request.RefreshStale,
	)
	if err != nil {
		return SyncResult{}, err
	}

	deleteIDs := syncDeleteIDs(plan.removed)
	if err := repository.ApplyChanges(ctx, upserts, deleteIDs); err != nil {
		return SyncResult{}, newUserError("failed to update encrypted local cache", err)
	}

	return syncResult(plan, request.RefreshStale), nil
}

func (a *Application) authenticateSync(
	ctx context.Context,
	password string,
) (SessionStorage, model.Authentication, error) {
	storedSessions, err := a.sessions()
	if err != nil {
		return nil, model.Authentication{}, err
	}

	storedSession, err := storedSessions.Load(a.serverAddress)
	if err != nil {
		return nil, model.Authentication{}, mapSessionLoadError(err)
	}

	currentUser, err := a.users.CurrentUser(ctx, storedSession.AccessToken)
	if err != nil {
		return nil, model.Authentication{}, mapCurrentUserError(err)
	}

	authentication, err := a.users.Login(ctx, currentUser.Login, password)
	if err != nil {
		return nil, model.Authentication{}, mapLoginGatewayError(err)
	}
	if currentUser.ID != authentication.User.ID || currentUser.Login != authentication.User.Login {
		return nil, model.Authentication{}, errSyncUserMismatch
	}

	return storedSessions, authentication, nil
}

func (a *Application) openSyncCache(
	ctx context.Context,
	canonicalLogin string,
	password string,
) (SyncCacheRepository, error) {
	repository, err := a.caches(
		ctx,
		a.serverAddress,
		canonicalLogin,
		[]byte(password),
	)
	if err != nil {
		return nil, newUserError("failed to open encrypted local cache", err)
	}

	return repository, nil
}

func (a *Application) loadSyncRecords(
	ctx context.Context,
	accessToken string,
	plan syncPlan,
	refreshStale bool,
) ([]model.Record, error) {
	metadata := make([]model.RecordMetadata, 0, len(plan.newRecords)+len(plan.stale))
	metadata = append(metadata, plan.newRecords...)
	if refreshStale {
		for _, change := range plan.stale {
			metadata = append(metadata, change.Metadata)
		}
	}

	records := make([]model.Record, 0, len(metadata))
	for _, expected := range metadata {
		record, err := a.records.GetRecord(ctx, accessToken, expected.ID)
		if err != nil {
			return nil, mapSyncGetRecordError(err)
		}
		if err := validateSyncRecord(expected, record); err != nil {
			return nil, err
		}
		records = append(records, record)
	}

	return records, nil
}

func validateSyncRecord(expected model.RecordMetadata, record model.Record) error {
	if err := record.Validate(); err != nil {
		return fmt.Errorf("validate synchronized record: %w", err)
	}
	if record.Metadata.ID != expected.ID ||
		record.Metadata.Type != expected.Type ||
		record.Metadata.Revision != expected.Revision {
		return newUserError(
			"server records changed during synchronization, please run sync again",
			errSyncStateChanged,
		)
	}

	return nil
}

func mapSyncGetRecordError(err error) error {
	if errors.Is(err, model.ErrRecordNotFound) {
		return newUserError(
			"server records changed during synchronization, please run sync again",
			errors.Join(errSyncStateChanged, err),
		)
	}

	return mapRecordGatewayError("get record for synchronization", err)
}

func syncDeleteIDs(records []RecordState) []string {
	ids := make([]string, 0, len(records))
	for _, record := range records {
		ids = append(ids, record.ID)
	}

	return ids
}

func syncResult(plan syncPlan, refreshStale bool) SyncResult {
	result := SyncResult{
		Added:     append([]model.RecordMetadata(nil), plan.newRecords...),
		Removed:   append([]RecordState(nil), plan.removed...),
		Unchanged: plan.unchanged,
	}
	if refreshStale {
		result.Updated = append([]RevisionChange(nil), plan.stale...)
	} else {
		result.Stale = append([]RevisionChange(nil), plan.stale...)
	}

	return result
}
