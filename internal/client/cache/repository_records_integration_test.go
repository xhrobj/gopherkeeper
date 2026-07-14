//go:build integration

package cache

import (
	"bytes"
	"context"
	"errors"
	"reflect"
	"testing"
	"time"

	"github.com/xhrobj/gopherkeeper/internal/model"
)

func TestIntegration_RepositoryUpsertGetDelete(t *testing.T) {
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

	record := model.Record{
		Metadata: model.RecordMetadata{
			ID:        "11111111-1111-4111-8111-111111111111",
			Type:      model.RecordTypeText,
			Title:     "private note",
			Revision:  1,
			CreatedAt: time.Date(2026, 7, 14, 10, 0, 0, 0, time.UTC),
			UpdatedAt: time.Date(2026, 7, 14, 10, 0, 0, 0, time.UTC),
		},
		Payload: &model.TextPayload{
			Text:     "first secret",
			Metadata: "first metadata",
		},
	}

	if err := repository.Upsert(ctx, record); err != nil {
		t.Fatalf("Upsert() create error = %v", err)
	}

	firstNonce, firstCiphertext := readEncryptedRow(t, ctx, repository, record.Metadata.ID)

	got, err := repository.Get(ctx, record.Metadata.ID)
	if err != nil {
		t.Fatalf("Get() created error = %v", err)
	}
	if !reflect.DeepEqual(got, record) {
		t.Fatalf("Get() created = %#v, want %#v", got, record)
	}

	record.Metadata.Title = "updated private note"
	record.Metadata.Revision = 2
	record.Metadata.UpdatedAt = time.Date(2026, 7, 14, 11, 0, 0, 0, time.UTC)
	record.Payload = &model.TextPayload{
		Text:     "updated secret",
		Metadata: "updated metadata",
	}

	if err := repository.Upsert(ctx, record); err != nil {
		t.Fatalf("Upsert() update error = %v", err)
	}

	secondNonce, secondCiphertext := readEncryptedRow(t, ctx, repository, record.Metadata.ID)
	if bytes.Equal(secondNonce, firstNonce) {
		t.Fatal("Upsert() reused AES-GCM nonce")
	}
	if bytes.Equal(secondCiphertext, firstCiphertext) {
		t.Fatal("Upsert() did not replace ciphertext")
	}

	got, err = repository.Get(ctx, record.Metadata.ID)
	if err != nil {
		t.Fatalf("Get() updated error = %v", err)
	}
	if !reflect.DeepEqual(got, record) {
		t.Fatalf("Get() updated = %#v, want %#v", got, record)
	}

	if err := repository.Delete(ctx, record.Metadata.ID); err != nil {
		t.Fatalf("Delete() error = %v", err)
	}
	if _, err := repository.Get(ctx, record.Metadata.ID); !errors.Is(err, ErrLocalRecordNotFound) {
		t.Fatalf("Get() deleted error = %v, want ErrLocalRecordNotFound", err)
	}
	if err := repository.Delete(ctx, record.Metadata.ID); !errors.Is(err, ErrLocalRecordNotFound) {
		t.Fatalf("Delete() missing error = %v, want ErrLocalRecordNotFound", err)
	}
}

func readEncryptedRow(
	t *testing.T,
	ctx context.Context,
	repository *Repository,
	recordID string,
) ([]byte, []byte) {
	t.Helper()

	var nonce []byte
	var ciphertext []byte
	if err := repository.database.db.QueryRowContext(
		ctx,
		"SELECT nonce, ciphertext FROM cached_records WHERE id = ?",
		recordID,
	).Scan(&nonce, &ciphertext); err != nil {
		t.Fatalf("read encrypted row: %v", err)
	}

	return nonce, ciphertext
}
