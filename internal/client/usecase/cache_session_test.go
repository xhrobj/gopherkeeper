package usecase

import (
	"context"
	"errors"
	"reflect"
	"testing"

	"github.com/xhrobj/gopherkeeper/internal/model"
)

func TestApplication_OpenCacheSessionReusesRepositoryUntilClose(t *testing.T) {
	wantMetadata := []model.RecordMetadata{{
		ID:       testRecordID,
		Type:     model.RecordTypeText,
		Title:    "Private note",
		Revision: 3,
	}}
	wantRecord := model.Record{
		Metadata: wantMetadata[0],
		Payload:  &model.TextPayload{Text: "offline value"},
	}

	openCalls := 0
	closeCalls := 0
	application := newOfflineTestApplication(t, func(
		_ context.Context,
		serverAddress string,
		canonicalLogin string,
		password []byte,
	) (OfflineCacheRepository, error) {
		openCalls++
		if serverAddress != "localhost:8080" || canonicalLogin != "alice" || string(password) != testPassword {
			t.Fatalf("cache open arguments = %q %q %q", serverAddress, canonicalLogin, password)
		}

		return offlineCacheRepositoryStub{
			listMetadata: func(context.Context) ([]model.RecordMetadata, error) {
				return wantMetadata, nil
			},
			get: func(_ context.Context, recordID string) (model.Record, error) {
				if recordID != testRecordID {
					t.Fatalf("record ID = %q, want %q", recordID, testRecordID)
				}
				return wantRecord, nil
			},
			close: func() error {
				closeCalls++
				return nil
			},
		}, nil
	})

	session, err := application.OpenCacheSession(context.Background(), OfflineReadRequest{
		Login:    " Alice ",
		Password: testPassword,
	})
	if err != nil {
		t.Fatalf("OpenCacheSession() error = %v", err)
	}

	metadata, err := session.ListRecords(context.Background())
	if err != nil {
		t.Fatalf("ListRecords() error = %v", err)
	}
	if !reflect.DeepEqual(metadata, wantMetadata) {
		t.Fatalf("ListRecords() = %#v, want %#v", metadata, wantMetadata)
	}

	record, err := session.GetRecord(context.Background(), testRecordID)
	if err != nil {
		t.Fatalf("GetRecord() error = %v", err)
	}
	if !reflect.DeepEqual(record, wantRecord) {
		t.Fatalf("GetRecord() = %#v, want %#v", record, wantRecord)
	}
	if openCalls != 1 {
		t.Fatalf("cache open calls = %d, want 1", openCalls)
	}

	if err := session.Close(); err != nil {
		t.Fatalf("Close() error = %v", err)
	}
	if err := session.Close(); err != nil {
		t.Fatalf("second Close() error = %v", err)
	}
	if closeCalls != 1 {
		t.Fatalf("cache close calls = %d, want 1", closeCalls)
	}
}

func TestCacheSessionRejectsReadsAfterClose(t *testing.T) {
	application := newOfflineTestApplication(t, func(
		context.Context,
		string,
		string,
		[]byte,
	) (OfflineCacheRepository, error) {
		return offlineCacheRepositoryStub{}, nil
	})

	session, err := application.OpenCacheSession(context.Background(), OfflineReadRequest{
		Login:    "alice",
		Password: testPassword,
	})
	if err != nil {
		t.Fatalf("OpenCacheSession() error = %v", err)
	}
	if err := session.Close(); err != nil {
		t.Fatalf("Close() error = %v", err)
	}

	if _, err := session.ListRecords(context.Background()); err == nil || err.Error() != "local cache session is closed" {
		t.Fatalf("ListRecords() error = %v, want closed-session error", err)
	}
	if _, err := session.GetRecord(context.Background(), testRecordID); err == nil || err.Error() != "local cache session is closed" {
		t.Fatalf("GetRecord() error = %v, want closed-session error", err)
	}
}

func TestCacheSessionMapsRepositoryErrors(t *testing.T) {
	readErr := errors.New("cache read failed")
	application := newOfflineTestApplication(t, func(
		context.Context,
		string,
		string,
		[]byte,
	) (OfflineCacheRepository, error) {
		return offlineCacheRepositoryStub{
			listMetadata: func(context.Context) ([]model.RecordMetadata, error) {
				return nil, readErr
			},
			get: func(context.Context, string) (model.Record, error) {
				return model.Record{}, ErrCachedRecordNotFound
			},
		}, nil
	})

	session, err := application.OpenCacheSession(context.Background(), OfflineReadRequest{
		Login:    "alice",
		Password: testPassword,
	})
	if err != nil {
		t.Fatalf("OpenCacheSession() error = %v", err)
	}
	defer func() { _ = session.Close() }()

	if _, err := session.ListRecords(context.Background()); !errors.Is(err, readErr) || err.Error() != "failed to read encrypted local cache" {
		t.Fatalf("ListRecords() error = %v", err)
	}
	if _, err := session.GetRecord(context.Background(), testRecordID); !errors.Is(err, ErrCachedRecordNotFound) || err.Error() != "record not found in local cache" {
		t.Fatalf("GetRecord() error = %v", err)
	}
}

func TestCacheSessionPreservesUnreadableRecordMarker(t *testing.T) {
	application := newOfflineTestApplication(t, func(
		context.Context,
		string,
		string,
		[]byte,
	) (OfflineCacheRepository, error) {
		return offlineCacheRepositoryStub{
			get: func(context.Context, string) (model.Record, error) {
				return model.Record{}, ErrLocalCacheRecordsUnreadable
			},
		}, nil
	})

	session, err := application.OpenCacheSession(context.Background(), OfflineReadRequest{
		Login:    "alice",
		Password: testPassword,
	})
	if err != nil {
		t.Fatalf("OpenCacheSession() error = %v", err)
	}
	defer func() { _ = session.Close() }()

	_, err = session.GetRecord(context.Background(), testRecordID)
	if !errors.Is(err, ErrLocalCacheRecordsUnreadable) {
		t.Fatalf("GetRecord() error = %v, want ErrLocalCacheRecordsUnreadable", err)
	}
	if err.Error() != "failed to read encrypted local cache" {
		t.Fatalf("GetRecord() error = %q, want safe cache read error", err)
	}
}
