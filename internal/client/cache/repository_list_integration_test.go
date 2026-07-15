//go:build integration

package cache

import (
	"context"
	"errors"
	"reflect"
	"testing"
	"time"

	"github.com/xhrobj/gopherkeeper/internal/client/usecase"
	"github.com/xhrobj/gopherkeeper/internal/model"
)

func TestIntegration_RepositoryListAndReopen(t *testing.T) {
	ctx := context.Background()
	location := testLocation(t)
	password := []byte("cache-password")
	records := repositoryTestRecords()

	repository, err := OpenRepository(ctx, location, password)
	if err != nil {
		t.Fatalf("OpenRepository() first error = %v", err)
	}
	for _, record := range records {
		if err := repository.Upsert(ctx, record); err != nil {
			t.Fatalf("Upsert(%s) error = %v", record.Metadata.Type, err)
		}
	}
	if err := repository.Close(); err != nil {
		t.Fatalf("Repository.Close() first error = %v", err)
	}

	reopened, err := OpenRepository(ctx, location, password)
	if err != nil {
		t.Fatalf("OpenRepository() repeated error = %v", err)
	}
	t.Cleanup(func() {
		if err := reopened.Close(); err != nil {
			t.Errorf("Repository.Close() repeated error = %v", err)
		}
	})

	states, err := reopened.ListState(ctx)
	if err != nil {
		t.Fatalf("ListState() error = %v", err)
	}
	wantStates := make([]usecase.RecordState, 0, len(records))
	for _, record := range records {
		wantStates = append(wantStates, usecase.RecordState{
			ID:       record.Metadata.ID,
			Revision: record.Metadata.Revision,
		})
	}
	if !reflect.DeepEqual(states, wantStates) {
		t.Fatalf("ListState() = %#v, want %#v", states, wantStates)
	}

	got, err := reopened.List(ctx)
	if err != nil {
		t.Fatalf("List() error = %v", err)
	}
	if !reflect.DeepEqual(got, records) {
		t.Fatalf("List() = %#v, want %#v", got, records)
	}
}

func TestIntegration_RepositoryRejectsRevisionMismatch(t *testing.T) {
	ctx := context.Background()
	repository, err := OpenRepository(ctx, testLocation(t), []byte("cache-password"))
	if err != nil {
		t.Fatalf("OpenRepository() error = %v", err)
	}
	t.Cleanup(func() {
		if err := repository.Close(); err != nil {
			t.Errorf("Repository.Close() error = %v", err)
		}
	})

	record := repositoryTestRecords()[0]
	if err := repository.Upsert(ctx, record); err != nil {
		t.Fatalf("Upsert() error = %v", err)
	}
	if _, err := repository.database.db.ExecContext(
		ctx,
		"UPDATE cached_records SET revision = revision + 1 WHERE id = ?",
		record.Metadata.ID,
	); err != nil {
		t.Fatalf("tamper record revision: %v", err)
	}

	states, err := repository.ListState(ctx)
	if err != nil {
		t.Fatalf("ListState() error = %v", err)
	}
	wantStates := []usecase.RecordState{{
		ID:       record.Metadata.ID,
		Revision: record.Metadata.Revision + 1,
	}}
	if !reflect.DeepEqual(states, wantStates) {
		t.Fatalf("ListState() = %#v, want %#v", states, wantStates)
	}

	got, err := repository.List(ctx)
	if !errors.Is(err, ErrCorruptedCacheRecord) {
		t.Fatalf("List() error = %v, want ErrCorruptedCacheRecord", err)
	}
	if got != nil {
		t.Fatalf("List() returned partial records = %#v", got)
	}

	if _, err := repository.Get(ctx, record.Metadata.ID); !errors.Is(err, ErrCorruptedCacheRecord) {
		t.Fatalf("Get() error = %v, want ErrCorruptedCacheRecord", err)
	}
}

func repositoryTestRecords() []model.Record {
	createdAt := time.Date(2026, 7, 14, 12, 0, 0, 0, time.UTC)

	return []model.Record{
		{
			Metadata: model.RecordMetadata{
				ID:        "11111111-1111-4111-8111-111111111111",
				Type:      model.RecordTypeText,
				Title:     "text title",
				Revision:  1,
				CreatedAt: createdAt,
				UpdatedAt: createdAt,
			},
			Payload: &model.TextPayload{
				Text:     "private text",
				Metadata: "text metadata",
			},
		},
		{
			Metadata: model.RecordMetadata{
				ID:        "22222222-2222-4222-8222-222222222222",
				Type:      model.RecordTypeBinary,
				Title:     "binary title",
				Revision:  2,
				CreatedAt: createdAt,
				UpdatedAt: createdAt.Add(time.Minute),
			},
			Payload: &model.BinaryPayload{
				Filename:    "private.bin",
				Data:        []byte{0x00, 0x01, 0x02, 0xff},
				ContentType: "application/octet-stream",
				Metadata:    "binary metadata",
			},
		},
	}
}
