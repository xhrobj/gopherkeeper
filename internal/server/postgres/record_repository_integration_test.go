//go:build integration

package postgres_test

import (
	"bytes"
	"context"
	"errors"
	"os"
	"testing"
	"time"

	"github.com/xhrobj/gopherkeeper/internal/model"
	"github.com/xhrobj/gopherkeeper/internal/server/postgres"
	"github.com/xhrobj/gopherkeeper/internal/server/recordcrypto"
)

func TestIntegration_RecordRepositoryCreateListGet(t *testing.T) {
	dsn := os.Getenv("DATABASE_DSN")
	if dsn == "" {
		t.Fatal("DATABASE_DSN is not set")
	}

	ctx, cancel := context.WithTimeout(context.Background(), repositoryIntegrationTestTimeout)
	defer cancel()

	pool := openMigratedTestDatabase(t, ctx, dsn)
	users := postgres.NewUserRepository(pool)
	records := postgres.NewRecordRepository(pool)

	alice, err := users.Create(ctx, "alice", []byte("alice-password-hash"))
	if err != nil {
		t.Fatalf("create Alice: %v", err)
	}
	bob, err := users.Create(ctx, "bob", []byte("bob-password-hash"))
	if err != nil {
		t.Fatalf("create Bob: %v", err)
	}

	first := newTestRecord("550e8400-e29b-41d4-a716-446655440000", alice.ID, "first")
	createdFirst, err := records.Create(ctx, first)
	if err != nil {
		t.Fatalf("Create() first error = %v", err)
	}

	// Make ordering deterministic: second record is newer.
	time.Sleep(time.Millisecond)

	second := newTestRecord("550e8400-e29b-41d4-a716-446655440001", alice.ID, "second")
	createdSecond, err := records.Create(ctx, second)
	if err != nil {
		t.Fatalf("Create() second error = %v", err)
	}
	_, err = records.Create(ctx, newTestRecord("550e8400-e29b-41d4-a716-446655440002", bob.ID, "bob record"))
	if err != nil {
		t.Fatalf("Create() Bob record error = %v", err)
	}

	metadata, err := records.ListMetadata(ctx, alice.ID)
	if err != nil {
		t.Fatalf("ListMetadata() error = %v", err)
	}
	if len(metadata) != 2 {
		t.Fatalf("ListMetadata() len = %d, want 2", len(metadata))
	}
	if metadata[0].ID != createdSecond.ID || metadata[1].ID != createdFirst.ID {
		t.Fatalf("ListMetadata() order IDs = %q, %q", metadata[0].ID, metadata[1].ID)
	}
	if metadata[0].Title != "second" || metadata[1].Title != "first" {
		t.Fatalf("ListMetadata() titles = %q, %q", metadata[0].Title, metadata[1].Title)
	}

	found, err := records.Get(ctx, alice.ID, createdFirst.ID)
	if err != nil {
		t.Fatalf("Get() error = %v", err)
	}
	if found.ID != createdFirst.ID || found.UserID != alice.ID || found.Title != "first" {
		t.Fatalf("Get() record = %+v", found)
	}
	if !bytes.Equal(found.Ciphertext, first.Ciphertext) {
		t.Fatal("Get() ciphertext differs from input")
	}

	_, err = records.Get(ctx, bob.ID, createdFirst.ID)
	if !errors.Is(err, model.ErrRecordNotFound) {
		t.Fatalf("Get() foreign record error = %v, want ErrRecordNotFound", err)
	}
	_, err = records.Get(ctx, alice.ID, "550e8400-e29b-41d4-a716-446655440069")
	if !errors.Is(err, model.ErrRecordNotFound) {
		t.Fatalf("Get() missing record error = %v, want ErrRecordNotFound", err)
	}
}

func newTestRecord(id string, userID int64, title string) model.Record {
	return model.Record{
		ID:            id,
		UserID:        userID,
		Type:          model.RecordTypeText,
		Title:         title,
		CryptoVersion: recordcrypto.CryptoVersion,
		KeyID:         recordcrypto.DefaultKeyID,
		Nonce:         []byte("nonce"),
		Ciphertext:    []byte("encrypted payload " + title),
	}
}
