package usecase

import (
	"context"
	"errors"
	"reflect"
	"strings"
	"testing"
	"time"

	"github.com/xhrobj/gopherkeeper/internal/client/session"
	"github.com/xhrobj/gopherkeeper/internal/model"
)

const (
	syncNewRecordID       = "550e8400-e29b-41d4-a716-446655440001"
	syncStaleRecordID     = "550e8400-e29b-41d4-a716-446655440002"
	syncUnchangedRecordID = "550e8400-e29b-41d4-a716-446655440003"
	syncRemovedRecordID   = "550e8400-e29b-41d4-a716-446655440004"
)

type cacheRepositoryStub struct {
	listState    func(context.Context) ([]RecordState, error)
	applyChanges func(context.Context, []model.Record, []string) error
	close        func() error
}

func (s cacheRepositoryStub) ListState(ctx context.Context) ([]RecordState, error) {
	if s.listState == nil {
		return nil, nil
	}
	return s.listState(ctx)
}

func (s cacheRepositoryStub) ApplyChanges(
	ctx context.Context,
	upserts []model.Record,
	deleteIDs []string,
) error {
	if s.applyChanges == nil {
		return nil
	}
	return s.applyChanges(ctx, upserts, deleteIDs)
}

func (s cacheRepositoryStub) Close() error {
	if s.close == nil {
		return nil
	}
	return s.close()
}

func TestApplication_Sync(t *testing.T) {
	newRecord := syncTextRecord(syncNewRecordID, "New note", 1)
	staleRecord := syncTextRecord(syncStaleRecordID, "Changed note", 2)
	unchangedRecord := syncTextRecord(syncUnchangedRecordID, "Stable note", 3)
	serverRecords := []model.RecordMetadata{
		unchangedRecord.Metadata,
		staleRecord.Metadata,
		newRecord.Metadata,
	}
	localRecords := []RecordState{
		{ID: syncRemovedRecordID, Revision: 4},
		{ID: syncUnchangedRecordID, Revision: 3},
		{ID: syncStaleRecordID, Revision: 1},
	}

	var savedSession session.Session
	cacheClosed := false
	application := newSyncTestApplication(syncTestDependencies{
		users:              successfulSyncUsers(t),
		records:            successfulSyncRecords(t, serverRecords, newRecord),
		sessions:           successfulSyncSessionsSaving(t, &savedSession),
		cache:              successfulSyncCache(t, localRecords, newRecord, &cacheClosed),
		cacheProviderCheck: successfulSyncCacheProviderCheck(t),
	})

	result, err := application.Sync(context.Background(), SyncRequest{Password: testPassword})
	if err != nil {
		t.Fatalf("Sync() error = %v", err)
	}

	assertSuccessfulSyncSession(t, savedSession)
	if !cacheClosed {
		t.Error("cache repository was not closed")
	}
	assertSuccessfulSyncResult(t, result, newRecord.Metadata, staleRecord.Metadata)
}

func TestApplication_SyncRefreshesStaleRecords(t *testing.T) {
	staleRecord := syncTextRecord(syncStaleRecordID, "Changed note", 2)
	getCalls := 0
	application := newSyncTestApplication(syncTestDependencies{
		users: successfulSyncUsers(t),
		records: recordGatewayStub{
			list: func(context.Context, string) ([]model.RecordMetadata, error) {
				return []model.RecordMetadata{staleRecord.Metadata}, nil
			},
			get: func(_ context.Context, _ string, recordID string) (model.Record, error) {
				getCalls++
				if recordID != syncStaleRecordID {
					t.Errorf("GetRecord() ID = %q, want %q", recordID, syncStaleRecordID)
				}
				return staleRecord, nil
			},
		},
		sessions: successfulSyncSessions(),
		cache: cacheRepositoryStub{
			listState: func(context.Context) ([]RecordState, error) {
				return []RecordState{{ID: syncStaleRecordID, Revision: 1}}, nil
			},
			applyChanges: func(_ context.Context, upserts []model.Record, deleteIDs []string) error {
				if !reflect.DeepEqual(upserts, []model.Record{staleRecord}) {
					t.Errorf("cache upserts = %#v, want stale server record", upserts)
				}
				if len(deleteIDs) != 0 {
					t.Errorf("cache deletes = %#v, want empty", deleteIDs)
				}
				return nil
			},
		},
	})

	result, err := application.Sync(context.Background(), SyncRequest{
		Password:     testPassword,
		RefreshStale: true,
	})
	if err != nil {
		t.Fatalf("Sync() error = %v", err)
	}
	if getCalls != 1 {
		t.Errorf("GetRecord() calls = %d, want 1", getCalls)
	}

	wantUpdated := []RevisionChange{{Metadata: staleRecord.Metadata, LocalRevision: 1}}
	if !reflect.DeepEqual(result.Updated, wantUpdated) {
		t.Errorf("updated = %#v, want %#v", result.Updated, wantUpdated)
	}
	if len(result.Stale) != 0 {
		t.Errorf("stale = %#v, want empty after refresh", result.Stale)
	}
}

func TestApplication_SyncRejectsWrongPasswordBeforeOpeningCache(t *testing.T) {
	remoteError := errors.New("remote authentication rejected")
	cacheOpened := false
	sessionSaved := false
	application := newSyncTestApplication(syncTestDependencies{
		users: userGatewayStub{
			whoami: func(context.Context, string) (model.User, error) {
				return testUser(), nil
			},
			login: func(context.Context, string, string) (model.Authentication, error) {
				return model.Authentication{}, errors.Join(remoteError, model.ErrInvalidCredentials)
			},
		},
		sessions: sessionStorageStub{
			load: func(string) (session.Session, error) { return syncStoredSession(), nil },
			save: func(session.Session) error {
				sessionSaved = true
				return nil
			},
		},
		cache: cacheRepositoryStub{},
		cacheProviderCheck: func(string, string, []byte) {
			cacheOpened = true
		},
	})

	_, err := application.Sync(context.Background(), SyncRequest{Password: testPassword})
	if err == nil {
		t.Fatal("Sync() error = nil, want invalid credentials")
	}
	if err.Error() != "invalid login or password" {
		t.Errorf("error = %q, want readable invalid credentials", err)
	}
	if !errors.Is(err, remoteError) {
		t.Error("sync error does not preserve authentication error")
	}
	if strings.Contains(err.Error(), testPassword) {
		t.Error("sync error contains password")
	}
	if cacheOpened {
		t.Error("cache was opened after invalid credentials")
	}
	if sessionSaved {
		t.Error("session was saved after invalid credentials")
	}
}

func TestApplication_SyncRejectsDifferentAuthenticatedUser(t *testing.T) {
	cacheOpened := false
	sessionSaved := false
	authentication := syncAuthentication()
	authentication.User.ID = 69
	application := newSyncTestApplication(syncTestDependencies{
		users: userGatewayStub{
			whoami: func(context.Context, string) (model.User, error) { return testUser(), nil },
			login: func(context.Context, string, string) (model.Authentication, error) {
				return authentication, nil
			},
		},
		sessions: sessionStorageStub{
			load: func(string) (session.Session, error) { return syncStoredSession(), nil },
			save: func(session.Session) error {
				sessionSaved = true
				return nil
			},
		},
		cache:              cacheRepositoryStub{},
		cacheProviderCheck: func(string, string, []byte) { cacheOpened = true },
	})

	_, err := application.Sync(context.Background(), SyncRequest{Password: testPassword})
	if !errors.Is(err, errSyncUserMismatch) {
		t.Fatalf("Sync() error = %v, want user mismatch", err)
	}
	if cacheOpened || sessionSaved {
		t.Error("sync persisted state after authenticated user mismatch")
	}
}

func TestApplication_SyncDoesNotApplyPartialChangesAfterGetError(t *testing.T) {
	firstRecord := syncTextRecord(syncNewRecordID, "First note", 1)
	secondRecord := syncTextRecord(syncStaleRecordID, "Second note", 1)
	getError := errors.New("connection reset")
	getCalls := 0
	applyCalled := false
	cacheClosed := false
	application := newSyncTestApplication(syncTestDependencies{
		users:    successfulSyncUsers(t),
		sessions: successfulSyncSessions(),
		records: recordGatewayStub{
			list: func(context.Context, string) ([]model.RecordMetadata, error) {
				return []model.RecordMetadata{firstRecord.Metadata, secondRecord.Metadata}, nil
			},
			get: func(_ context.Context, _ string, recordID string) (model.Record, error) {
				getCalls++
				if recordID == syncNewRecordID {
					return firstRecord, nil
				}
				return model.Record{}, getError
			},
		},
		cache: cacheRepositoryStub{
			applyChanges: func(context.Context, []model.Record, []string) error {
				applyCalled = true
				return nil
			},
			close: func() error {
				cacheClosed = true
				return nil
			},
		},
	})

	_, err := application.Sync(context.Background(), SyncRequest{Password: testPassword})
	if !errors.Is(err, getError) {
		t.Fatalf("Sync() error = %v, want get error", err)
	}
	if getCalls != 2 {
		t.Errorf("GetRecord() calls = %d, want 2", getCalls)
	}
	if applyCalled {
		t.Error("cache batch was applied after mid-sync get error")
	}
	if !cacheClosed {
		t.Error("cache repository was not closed after get error")
	}
}

func TestApplication_SyncDetectsServerRace(t *testing.T) {
	expected := syncTextRecord(syncNewRecordID, "New note", 1)
	tests := []struct {
		name   string
		get    func(context.Context, string, string) (model.Record, error)
		marker error
	}{
		{
			name: "record disappeared",
			get: func(context.Context, string, string) (model.Record, error) {
				return model.Record{}, model.ErrRecordNotFound
			},
			marker: model.ErrRecordNotFound,
		},
		{
			name: "revision changed",
			get: func(context.Context, string, string) (model.Record, error) {
				changed := expected
				changed.Metadata.Revision = 2
				return changed, nil
			},
			marker: errSyncStateChanged,
		},
		{
			name: "type changed",
			get: func(context.Context, string, string) (model.Record, error) {
				changed := syncCredentialsRecord(syncNewRecordID, "New note", 1)
				return changed, nil
			},
			marker: errSyncStateChanged,
		},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			assertSyncDetectsServerRace(t, expected, tt.get, tt.marker)
		})
	}
}

func assertSyncDetectsServerRace(
	t *testing.T,
	expected model.Record,
	get func(context.Context, string, string) (model.Record, error),
	marker error,
) {
	t.Helper()

	applyCalled := false
	application := newSyncTestApplication(syncTestDependencies{
		users:    successfulSyncUsers(t),
		sessions: successfulSyncSessions(),
		records: recordGatewayStub{
			list: func(context.Context, string) ([]model.RecordMetadata, error) {
				return []model.RecordMetadata{expected.Metadata}, nil
			},
			get: get,
		},
		cache: cacheRepositoryStub{
			applyChanges: func(context.Context, []model.Record, []string) error {
				applyCalled = true
				return nil
			},
		},
	})

	_, err := application.Sync(context.Background(), SyncRequest{Password: testPassword})
	if err == nil {
		t.Fatal("Sync() error = nil, want server race")
	}
	if err.Error() != "server records changed during synchronization, please run sync again" {
		t.Errorf("error = %q, want readable server race message", err)
	}
	if !errors.Is(err, errSyncStateChanged) {
		t.Error("sync error does not preserve state changed marker")
	}
	if !errors.Is(err, marker) {
		t.Errorf("sync error does not preserve %v", marker)
	}
	if applyCalled {
		t.Error("cache batch was applied after server race")
	}
}

func TestApplication_SyncReturnsInfrastructureErrorsAndClosesCache(t *testing.T) {
	listServerError := errors.New("list server unavailable")
	listCacheError := errors.New("cache state unavailable")
	applyError := errors.New("cache write failed")
	tests := []struct {
		name          string
		records       RecordGateway
		cache         cacheRepositoryStub
		wantErr       error
		wantMessage   string
		wantCloseCall bool
	}{
		{
			name: "server list",
			records: recordGatewayStub{
				list: func(context.Context, string) ([]model.RecordMetadata, error) {
					return nil, listServerError
				},
			},
			cache:         cacheRepositoryStub{},
			wantErr:       listServerError,
			wantCloseCall: true,
		},
		{
			name: "cache list",
			records: recordGatewayStub{
				list: func(context.Context, string) ([]model.RecordMetadata, error) { return nil, nil },
			},
			cache: cacheRepositoryStub{
				listState: func(context.Context) ([]RecordState, error) { return nil, listCacheError },
			},
			wantErr:       listCacheError,
			wantMessage:   "failed to read encrypted local cache",
			wantCloseCall: true,
		},
		{
			name: "cache apply",
			records: recordGatewayStub{
				list: func(context.Context, string) ([]model.RecordMetadata, error) { return nil, nil },
			},
			cache: cacheRepositoryStub{
				applyChanges: func(context.Context, []model.Record, []string) error { return applyError },
			},
			wantErr:       applyError,
			wantMessage:   "failed to update encrypted local cache",
			wantCloseCall: true,
		},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			closeCalls := 0
			cache := tt.cache
			cache.close = func() error {
				closeCalls++
				return nil
			}
			application := newSyncTestApplication(syncTestDependencies{
				users:    successfulSyncUsers(t),
				sessions: successfulSyncSessions(),
				records:  tt.records,
				cache:    cache,
			})

			_, err := application.Sync(context.Background(), SyncRequest{Password: testPassword})
			if !errors.Is(err, tt.wantErr) {
				t.Fatalf("Sync() error = %v, want %v", err, tt.wantErr)
			}
			if tt.wantMessage != "" && err.Error() != tt.wantMessage {
				t.Errorf("Sync() error = %q, want %q", err, tt.wantMessage)
			}
			if tt.wantCloseCall && closeCalls != 1 {
				t.Errorf("Close() calls = %d, want 1", closeCalls)
			}
		})
	}
}

func TestApplication_SyncReturnsCloseError(t *testing.T) {
	closeError := errors.New("close failed")
	application := newSyncTestApplication(syncTestDependencies{
		users:    successfulSyncUsers(t),
		sessions: successfulSyncSessions(),
		records: recordGatewayStub{
			list: func(context.Context, string) ([]model.RecordMetadata, error) { return nil, nil },
		},
		cache: cacheRepositoryStub{
			close: func() error { return closeError },
		},
	})

	result, err := application.Sync(context.Background(), SyncRequest{Password: testPassword})
	if !errors.Is(err, closeError) {
		t.Fatalf("Sync() error = %v, want close error", err)
	}
	if err.Error() != "failed to close encrypted local cache" {
		t.Errorf("Sync() error = %q, want safe close error", err)
	}
	if !reflect.DeepEqual(result, SyncResult{}) {
		t.Errorf("Sync() result = %#v, want zero result after close error", result)
	}
}

func TestApplication_SyncStopsBeforeOpeningCacheOnSessionOrUserError(t *testing.T) {
	providerError := errors.New("session storage unavailable")
	loadError := errors.New("session file unavailable")
	currentUserError := errors.New("current user unavailable")
	saveError := errors.New("session save unavailable")
	tests := []struct {
		name         string
		dependencies syncTestDependencies
		wantErr      error
	}{
		{
			name: "session provider",
			dependencies: syncTestDependencies{
				sessionProviderError: providerError,
			},
			wantErr: providerError,
		},
		{
			name: "session load",
			dependencies: syncTestDependencies{
				sessions: sessionStorageStub{
					load: func(string) (session.Session, error) { return session.Session{}, loadError },
				},
			},
			wantErr: loadError,
		},
		{
			name: "current user",
			dependencies: syncTestDependencies{
				users: userGatewayStub{
					whoami: func(context.Context, string) (model.User, error) {
						return model.User{}, currentUserError
					},
				},
				sessions: sessionStorageStub{
					load: func(string) (session.Session, error) { return syncStoredSession(), nil },
				},
			},
			wantErr: currentUserError,
		},
		{
			name: "session save",
			dependencies: syncTestDependencies{
				users: successfulSyncUsers(t),
				sessions: sessionStorageStub{
					load: func(string) (session.Session, error) { return syncStoredSession(), nil },
					save: func(session.Session) error { return saveError },
				},
			},
			wantErr: saveError,
		},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			cacheOpened := false
			dependencies := tt.dependencies
			dependencies.cacheProviderCheck = func(string, string, []byte) { cacheOpened = true }
			application := newSyncTestApplication(dependencies)

			_, err := application.Sync(context.Background(), SyncRequest{Password: testPassword})
			if !errors.Is(err, tt.wantErr) {
				t.Fatalf("Sync() error = %v, want %v", err, tt.wantErr)
			}
			if cacheOpened {
				t.Error("cache was opened after session or current user error")
			}
		})
	}
}

func TestApplication_SyncPreservesOperationAndCloseErrors(t *testing.T) {
	operationError := errors.New("server unavailable")
	closeError := errors.New("close failed")
	application := newSyncTestApplication(syncTestDependencies{
		users:    successfulSyncUsers(t),
		sessions: successfulSyncSessions(),
		records: recordGatewayStub{
			list: func(context.Context, string) ([]model.RecordMetadata, error) {
				return nil, operationError
			},
		},
		cache: cacheRepositoryStub{
			close: func() error { return closeError },
		},
	})

	_, err := application.Sync(context.Background(), SyncRequest{Password: testPassword})
	if !errors.Is(err, operationError) {
		t.Error("sync error does not preserve operation error")
	}
	if !errors.Is(err, closeError) {
		t.Error("sync error does not preserve close error")
	}
}

func TestApplication_SyncSavesSessionBeforeOpeningCache(t *testing.T) {
	openError := errors.New("cache unavailable")
	sessionSaved := false
	application := newSyncTestApplication(syncTestDependencies{
		users: successfulSyncUsers(t),
		sessions: sessionStorageStub{
			load: func(string) (session.Session, error) { return syncStoredSession(), nil },
			save: func(session.Session) error {
				sessionSaved = true
				return nil
			},
		},
		cacheProviderError: openError,
		cacheProviderCheck: func(string, string, []byte) {
			if !sessionSaved {
				t.Error("cache opened before updated session was saved")
			}
		},
	})

	_, err := application.Sync(context.Background(), SyncRequest{Password: testPassword})
	if !errors.Is(err, openError) {
		t.Fatalf("Sync() error = %v, want cache open error", err)
	}
	if err.Error() != "failed to open encrypted local cache" {
		t.Errorf("Sync() error = %q, want safe open error", err)
	}
	if !sessionSaved {
		t.Error("updated session was not saved before cache open error")
	}
}

type syncTestDependencies struct {
	users                UserGateway
	records              RecordGateway
	sessions             SessionStorage
	cache                SyncCacheRepository
	cacheProviderCheck   func(string, string, []byte)
	cacheProviderError   error
	sessionProviderError error
}

func newSyncTestApplication(dependencies syncTestDependencies) *Application {
	return &Application{
		users:   dependencies.users,
		records: dependencies.records,
		sessions: func() (SessionStorage, error) {
			if dependencies.sessionProviderError != nil {
				return nil, dependencies.sessionProviderError
			}
			return dependencies.sessions, nil
		},
		syncCaches: func(
			_ context.Context,
			serverAddress, canonicalLogin string,
			password []byte,
		) (SyncCacheRepository, error) {
			if dependencies.cacheProviderCheck != nil {
				dependencies.cacheProviderCheck(serverAddress, canonicalLogin, password)
			}
			if dependencies.cacheProviderError != nil {
				return nil, dependencies.cacheProviderError
			}
			return dependencies.cache, nil
		},
		serverAddress: "localhost:8080",
	}
}

func successfulSyncRecords(
	t *testing.T,
	serverRecords []model.RecordMetadata,
	newRecord model.Record,
) RecordGateway {
	t.Helper()
	return recordGatewayStub{
		list: func(_ context.Context, accessToken string) ([]model.RecordMetadata, error) {
			if accessToken != "new.jwt.token" {
				t.Errorf("list access token = %q, want new.jwt.token", accessToken)
			}
			return serverRecords, nil
		},
		get: func(_ context.Context, accessToken, recordID string) (model.Record, error) {
			if accessToken != "new.jwt.token" {
				t.Errorf("get access token = %q, want new.jwt.token", accessToken)
			}
			if recordID != syncNewRecordID {
				t.Fatalf("GetRecord() ID = %q, want only new record %q", recordID, syncNewRecordID)
			}
			return newRecord, nil
		},
	}
}

func successfulSyncSessionsSaving(t *testing.T, savedSession *session.Session) SessionStorage {
	t.Helper()
	return sessionStorageStub{
		load: func(expectedServerAddress string) (session.Session, error) {
			if expectedServerAddress != "localhost:8080" {
				t.Errorf("session server address = %q, want localhost:8080", expectedServerAddress)
			}
			return syncStoredSession(), nil
		},
		save: func(stored session.Session) error {
			*savedSession = stored
			return nil
		},
	}
}

func successfulSyncCache(
	t *testing.T,
	localRecords []RecordState,
	newRecord model.Record,
	closed *bool,
) SyncCacheRepository {
	t.Helper()
	return cacheRepositoryStub{
		listState: func(context.Context) ([]RecordState, error) {
			return localRecords, nil
		},
		applyChanges: func(_ context.Context, upserts []model.Record, deleteIDs []string) error {
			if !reflect.DeepEqual(upserts, []model.Record{newRecord}) {
				t.Errorf("cache upserts = %#v, want new record", upserts)
			}
			if !reflect.DeepEqual(deleteIDs, []string{syncRemovedRecordID}) {
				t.Errorf("cache deletes = %#v, want removed record", deleteIDs)
			}
			return nil
		},
		close: func() error {
			*closed = true
			return nil
		},
	}
}

func successfulSyncCacheProviderCheck(t *testing.T) func(string, string, []byte) {
	t.Helper()
	return func(serverAddress, canonicalLogin string, password []byte) {
		if serverAddress != "localhost:8080" || canonicalLogin != "alice" {
			t.Errorf("cache identity = %q/%q, want localhost:8080/alice", serverAddress, canonicalLogin)
		}
		if string(password) != testPassword {
			t.Error("cache provider received unexpected password")
		}
	}
}

func assertSuccessfulSyncSession(t *testing.T, got session.Session) {
	t.Helper()
	want := session.Session{
		ServerAddress: "localhost:8080",
		AccessToken:   "new.jwt.token",
		ExpiresAt:     syncAuthentication().ExpiresAt,
	}
	if got != want {
		t.Errorf("saved session = %+v, want %+v", got, want)
	}
}

func assertSuccessfulSyncResult(
	t *testing.T,
	got SyncResult,
	newRecord, staleRecord model.RecordMetadata,
) {
	t.Helper()
	want := SyncResult{
		Added:   []model.RecordMetadata{newRecord},
		Removed: []RecordState{{ID: syncRemovedRecordID, Revision: 4}},
		Stale: []RevisionChange{{
			Metadata:      staleRecord,
			LocalRevision: 1,
		}},
		Unchanged: 1,
	}
	if !reflect.DeepEqual(got, want) {
		t.Errorf("Sync() result = %#v, want %#v", got, want)
	}
}

func successfulSyncUsers(t *testing.T) UserGateway {
	t.Helper()
	return userGatewayStub{
		whoami: func(_ context.Context, accessToken string) (model.User, error) {
			if accessToken != "old.jwt.token" {
				t.Errorf("current user access token = %q, want old.jwt.token", accessToken)
			}
			return testUser(), nil
		},
		login: successfulSyncLogin(t),
	}
}

func successfulSyncLogin(t *testing.T) func(context.Context, string, string) (model.Authentication, error) {
	t.Helper()
	return func(_ context.Context, login, password string) (model.Authentication, error) {
		if login != "alice" {
			t.Errorf("login = %q, want alice", login)
		}
		if password != testPassword {
			t.Error("login gateway received unexpected password")
		}
		return syncAuthentication(), nil
	}
}

func successfulSyncSessions() SessionStorage {
	return sessionStorageStub{
		load: func(string) (session.Session, error) { return syncStoredSession(), nil },
		save: func(session.Session) error { return nil },
	}
}

func syncStoredSession() session.Session {
	return session.Session{
		ServerAddress: "localhost:8080",
		AccessToken:   "old.jwt.token",
		ExpiresAt:     time.Date(2026, time.July, 15, 12, 15, 0, 0, time.UTC),
	}
}

func syncAuthentication() model.Authentication {
	return model.Authentication{
		AccessToken: "new.jwt.token",
		ExpiresAt:   time.Date(2026, time.July, 15, 12, 30, 0, 0, time.UTC),
		User:        testUser(),
	}
}

func syncTextRecord(id, title string, revision int64) model.Record {
	return model.Record{
		Metadata: model.RecordMetadata{
			ID:        id,
			Type:      model.RecordTypeText,
			Title:     title,
			Revision:  revision,
			CreatedAt: time.Date(2026, time.July, 15, 12, 0, 0, 0, time.UTC),
			UpdatedAt: time.Date(2026, time.July, 15, 12, 0, 0, 0, time.UTC),
		},
		Payload: &model.TextPayload{Text: "private text"},
	}
}

func syncCredentialsRecord(id, title string, revision int64) model.Record {
	return model.Record{
		Metadata: model.RecordMetadata{
			ID:       id,
			Type:     model.RecordTypeCredentials,
			Title:    title,
			Revision: revision,
		},
		Payload: &model.CredentialsPayload{
			Login:    "alice",
			Password: "stored-password",
		},
	}
}
