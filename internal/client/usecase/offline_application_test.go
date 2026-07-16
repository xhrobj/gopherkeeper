package usecase

import (
	"context"
	"errors"
	"reflect"
	"testing"

	"github.com/xhrobj/gopherkeeper/internal/client/session"
	"github.com/xhrobj/gopherkeeper/internal/model"
)

type offlineCacheRepositoryStub struct {
	listMetadata func(context.Context) ([]model.RecordMetadata, error)
	get          func(context.Context, string) (model.Record, error)
	close        func() error
}

func (stub offlineCacheRepositoryStub) ListMetadata(ctx context.Context) ([]model.RecordMetadata, error) {
	if stub.listMetadata == nil {
		return nil, nil
	}

	return stub.listMetadata(ctx)
}

func (stub offlineCacheRepositoryStub) Get(ctx context.Context, recordID string) (model.Record, error) {
	if stub.get == nil {
		return model.Record{}, nil
	}

	return stub.get(ctx, recordID)
}

func (stub offlineCacheRepositoryStub) Close() error {
	if stub.close == nil {
		return nil
	}

	return stub.close()
}

func TestApplication_ListCachedRecords(t *testing.T) {
	want := []model.RecordMetadata{{
		ID:       testRecordID,
		Type:     model.RecordTypeText,
		Title:    "Private note",
		Revision: 1,
	}}
	closed := false
	application := newOfflineTestApplication(t, func(
		_ context.Context,
		serverAddress string,
		canonicalLogin string,
		password []byte,
	) (OfflineCacheRepository, error) {
		if serverAddress != "localhost:8080" {
			t.Errorf("server address = %q, want localhost:8080", serverAddress)
		}
		if canonicalLogin != "alice" {
			t.Errorf("canonical login = %q, want alice", canonicalLogin)
		}
		if string(password) != testPassword {
			t.Error("offline cache provider received unexpected password")
		}

		return offlineCacheRepositoryStub{
			listMetadata: func(context.Context) ([]model.RecordMetadata, error) {
				return want, nil
			},
			close: func() error {
				closed = true
				return nil
			},
		}, nil
	})

	result, err := application.ListCachedRecords(context.Background(), OfflineReadRequest{
		Login:    " Alice ",
		Password: testPassword,
	})
	if err != nil {
		t.Fatalf("ListCachedRecords() error = %v", err)
	}
	if !reflect.DeepEqual(result.Records, want) {
		t.Errorf("ListCachedRecords() records = %#v, want %#v", result.Records, want)
	}
	if result.Source != OfflineSourceLocalCache {
		t.Errorf("ListCachedRecords() source = %q, want %q", result.Source, OfflineSourceLocalCache)
	}
	if !result.MayBeStale {
		t.Error("ListCachedRecords() MayBeStale = false, want true")
	}
	if !closed {
		t.Error("offline cache repository was not closed")
	}
}

func TestApplication_GetCachedRecord(t *testing.T) {
	want := model.Record{
		Metadata: model.RecordMetadata{
			ID:       testRecordID,
			Type:     model.RecordTypeCredentials,
			Title:    "GitHub",
			Revision: 2,
		},
		Payload: &model.CredentialsPayload{
			Login:    "alice",
			Password: testPassword,
		},
	}
	closed := false
	application := newOfflineTestApplication(t, func(
		context.Context,
		string,
		string,
		[]byte,
	) (OfflineCacheRepository, error) {
		return offlineCacheRepositoryStub{
			get: func(_ context.Context, recordID string) (model.Record, error) {
				if recordID != testRecordID {
					t.Errorf("record ID = %q, want %q", recordID, testRecordID)
				}
				return want, nil
			},
			close: func() error {
				closed = true
				return nil
			},
		}, nil
	})

	result, err := application.GetCachedRecord(
		context.Background(),
		OfflineReadRequest{Login: "alice", Password: testPassword},
		testRecordID,
	)
	if err != nil {
		t.Fatalf("GetCachedRecord() error = %v", err)
	}
	if !reflect.DeepEqual(result.Record, want) {
		t.Errorf("GetCachedRecord() record = %#v, want %#v", result.Record, want)
	}
	if result.Source != OfflineSourceLocalCache {
		t.Errorf("GetCachedRecord() source = %q, want %q", result.Source, OfflineSourceLocalCache)
	}
	if !result.MayBeStale {
		t.Error("GetCachedRecord() MayBeStale = false, want true")
	}
	if !closed {
		t.Error("offline cache repository was not closed")
	}
}

func TestApplication_OfflineReadRejectsInvalidInputBeforeOpeningCache(t *testing.T) {
	tests := []struct {
		name    string
		read    func(*Application) error
		wantErr error
	}{
		{
			name: "invalid login",
			read: func(application *Application) error {
				_, err := application.ListCachedRecords(
					context.Background(),
					OfflineReadRequest{Login: "álîçé", Password: testPassword},
				)
				return err
			},
			wantErr: model.ErrInvalidLogin,
		},
		{
			name: "invalid record ID",
			read: func(application *Application) error {
				_, err := application.GetCachedRecord(
					context.Background(),
					OfflineReadRequest{Login: "alice", Password: testPassword},
					"not-a-uuid",
				)
				return err
			},
			wantErr: model.ErrInvalidRecordID,
		},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			cacheOpened := false
			application := newOfflineTestApplication(t, func(
				context.Context,
				string,
				string,
				[]byte,
			) (OfflineCacheRepository, error) {
				cacheOpened = true
				return offlineCacheRepositoryStub{}, nil
			})

			err := tt.read(application)
			if !errors.Is(err, tt.wantErr) {
				t.Fatalf("offline read error = %v, want %v", err, tt.wantErr)
			}
			if cacheOpened {
				t.Error("offline cache was opened after invalid input")
			}
		})
	}
}

func TestApplication_OfflineReadMapsCacheOpenErrors(t *testing.T) {
	openError := errors.New("cache storage unavailable")
	tests := []struct {
		name        string
		providerErr error
		wantErr     error
		wantMessage string
	}{
		{
			name:        "missing cache",
			providerErr: ErrLocalCacheNotFound,
			wantErr:     ErrLocalCacheNotFound,
			wantMessage: "local cache not found, run sync while online first",
		},
		{
			name:        "open failure",
			providerErr: openError,
			wantErr:     openError,
			wantMessage: "failed to open encrypted local cache",
		},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			application := newOfflineTestApplication(t, func(
				context.Context,
				string,
				string,
				[]byte,
			) (OfflineCacheRepository, error) {
				return nil, tt.providerErr
			})

			_, err := application.ListCachedRecords(
				context.Background(),
				OfflineReadRequest{Login: "alice", Password: testPassword},
			)
			if !errors.Is(err, tt.wantErr) {
				t.Fatalf("ListCachedRecords() error = %v, want %v", err, tt.wantErr)
			}
			if err.Error() != tt.wantMessage {
				t.Errorf("ListCachedRecords() error = %q, want %q", err, tt.wantMessage)
			}
		})
	}
}

func TestApplication_OfflineReadMapsRepositoryErrorsAndClosesCache(t *testing.T) {
	readError := errors.New("cache read failed")
	tests := []struct {
		name        string
		repository  offlineCacheRepositoryStub
		read        func(*Application) error
		wantErr     error
		wantMessage string
	}{
		{
			name: "list failure",
			repository: offlineCacheRepositoryStub{
				listMetadata: func(context.Context) ([]model.RecordMetadata, error) {
					return nil, readError
				},
			},
			read: func(application *Application) error {
				_, err := application.ListCachedRecords(
					context.Background(),
					OfflineReadRequest{Login: "alice", Password: testPassword},
				)
				return err
			},
			wantErr:     readError,
			wantMessage: "failed to read encrypted local cache",
		},
		{
			name: "record not found",
			repository: offlineCacheRepositoryStub{
				get: func(context.Context, string) (model.Record, error) {
					return model.Record{}, ErrCachedRecordNotFound
				},
			},
			read: func(application *Application) error {
				_, err := application.GetCachedRecord(
					context.Background(),
					OfflineReadRequest{Login: "alice", Password: testPassword},
					testRecordID,
				)
				return err
			},
			wantErr:     ErrCachedRecordNotFound,
			wantMessage: "record not found in local cache",
		},
		{
			name: "get failure",
			repository: offlineCacheRepositoryStub{
				get: func(context.Context, string) (model.Record, error) {
					return model.Record{}, readError
				},
			},
			read: func(application *Application) error {
				_, err := application.GetCachedRecord(
					context.Background(),
					OfflineReadRequest{Login: "alice", Password: testPassword},
					testRecordID,
				)
				return err
			},
			wantErr:     readError,
			wantMessage: "failed to read encrypted local cache",
		},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			closeCalls := 0
			repository := tt.repository
			repository.close = func() error {
				closeCalls++
				return nil
			}
			application := newOfflineTestApplication(t, func(
				context.Context,
				string,
				string,
				[]byte,
			) (OfflineCacheRepository, error) {
				return repository, nil
			})

			err := tt.read(application)
			if !errors.Is(err, tt.wantErr) {
				t.Fatalf("offline read error = %v, want %v", err, tt.wantErr)
			}
			if err.Error() != tt.wantMessage {
				t.Errorf("offline read error = %q, want %q", err, tt.wantMessage)
			}
			if closeCalls != 1 {
				t.Errorf("Close() calls = %d, want 1", closeCalls)
			}
		})
	}
}

func TestApplication_OfflineReadReturnsCloseErrorWithoutPartialResult(t *testing.T) {
	closeError := errors.New("close failed")
	application := newOfflineTestApplication(t, func(
		context.Context,
		string,
		string,
		[]byte,
	) (OfflineCacheRepository, error) {
		return offlineCacheRepositoryStub{
			listMetadata: func(context.Context) ([]model.RecordMetadata, error) {
				return []model.RecordMetadata{{ID: testRecordID}}, nil
			},
			close: func() error { return closeError },
		}, nil
	})

	result, err := application.ListCachedRecords(
		context.Background(),
		OfflineReadRequest{Login: "alice", Password: testPassword},
	)
	if !errors.Is(err, closeError) {
		t.Fatalf("ListCachedRecords() error = %v, want close error", err)
	}
	if err.Error() != "failed to close encrypted local cache" {
		t.Errorf("ListCachedRecords() error = %q, want safe close error", err)
	}
	if !reflect.DeepEqual(result, OfflineListResult{}) {
		t.Errorf("ListCachedRecords() result = %#v, want zero result", result)
	}
}

func TestApplication_GetCachedRecordPreservesOperationAndCloseErrors(t *testing.T) {
	readError := errors.New("cache read failed")
	closeError := errors.New("close failed")
	application := newOfflineTestApplication(t, func(
		context.Context,
		string,
		string,
		[]byte,
	) (OfflineCacheRepository, error) {
		return offlineCacheRepositoryStub{
			get: func(context.Context, string) (model.Record, error) {
				return model.Record{}, readError
			},
			close: func() error { return closeError },
		}, nil
	})

	_, err := application.GetCachedRecord(
		context.Background(),
		OfflineReadRequest{Login: "alice", Password: testPassword},
		testRecordID,
	)
	if !errors.Is(err, readError) {
		t.Error("offline read error does not preserve operation error")
	}
	if !errors.Is(err, closeError) {
		t.Error("offline read error does not preserve close error")
	}
}

func newOfflineTestApplication(
	t *testing.T,
	provider OfflineCacheRepositoryProvider,
) *Application {
	t.Helper()

	unexpectedNetworkCall := func() {
		t.Fatal("offline read must not call network or online session dependencies")
	}

	return &Application{
		users: userGatewayStub{
			register: func(context.Context, string, string) (model.User, error) {
				unexpectedNetworkCall()
				return model.User{}, nil
			},
			login: func(context.Context, string, string) (model.Authentication, error) {
				unexpectedNetworkCall()
				return model.Authentication{}, nil
			},
			whoami: func(context.Context, string) (model.User, error) {
				unexpectedNetworkCall()
				return model.User{}, nil
			},
		},
		records: recordGatewayStub{
			list: func(context.Context, string) ([]model.RecordMetadata, error) {
				unexpectedNetworkCall()
				return nil, nil
			},
			get: func(context.Context, string, string) (model.Record, error) {
				unexpectedNetworkCall()
				return model.Record{}, nil
			},
		},
		sessions: func() (SessionStorage, error) {
			unexpectedNetworkCall()
			return sessionStorageStub{
				save: func(session.Session) error { return nil },
				load: func(string) (session.Session, error) { return session.Session{}, nil },
			}, nil
		},
		offlineCaches: provider,
		serverAddress: "localhost:8080",
	}
}
