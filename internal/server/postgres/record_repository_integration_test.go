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

func TestIntegration_RecordRepositoryUpdate(t *testing.T) {
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

	created, err := records.Create(ctx, newTestRecord("550e8400-e29b-41d4-a716-446655440010", alice.ID, "original"))
	if err != nil {
		t.Fatalf("Create() error = %v", err)
	}

	// Make updated_at comparison deterministic on PostgreSQL timestamp precision.
	time.Sleep(time.Millisecond)

	patch := newTestRecord(created.ID, alice.ID, "updated")
	patch.Nonce = []byte("updated nonce")
	patch.Ciphertext = []byte("updated ciphertext")

	updated, err := records.Update(ctx, patch, created.Revision)
	if err != nil {
		t.Fatalf("Update() error = %v", err)
	}
	if updated.ID != created.ID || updated.UserID != alice.ID || updated.Type != model.RecordTypeText {
		t.Fatalf("Update() record identity = %+v", updated)
	}
	if updated.Title != "updated" {
		t.Fatalf("Update() title = %q, want updated", updated.Title)
	}
	if updated.Revision != created.Revision+1 {
		t.Fatalf("Update() revision = %d, want %d", updated.Revision, created.Revision+1)
	}
	if !updated.CreatedAt.Equal(created.CreatedAt) {
		t.Fatalf("Update() created_at = %v, want %v", updated.CreatedAt, created.CreatedAt)
	}
	if !updated.UpdatedAt.After(created.UpdatedAt) {
		t.Fatalf("Update() updated_at = %v, want after %v", updated.UpdatedAt, created.UpdatedAt)
	}
	if !bytes.Equal(updated.Nonce, patch.Nonce) || !bytes.Equal(updated.Ciphertext, patch.Ciphertext) {
		t.Fatalf("Update() encrypted fields = nonce %q ciphertext %q", updated.Nonce, updated.Ciphertext)
	}

	var storedCiphertext []byte
	if err := pool.QueryRow(
		ctx,
		"SELECT ciphertext FROM gopherkeeper.records WHERE user_id = $1 AND id = $2",
		alice.ID,
		created.ID,
	).Scan(&storedCiphertext); err != nil {
		t.Fatalf("read updated ciphertext: %v", err)
	}
	if bytes.Contains(storedCiphertext, []byte("updated secret note")) {
		t.Fatal("stored ciphertext contains plaintext payload")
	}

	stalePatch := newTestRecord(created.ID, alice.ID, "stale update")
	_, err = records.Update(ctx, stalePatch, created.Revision)
	if !errors.Is(err, model.ErrRecordRevisionConflict) {
		t.Fatalf("stale Update() error = %v, want ErrRecordRevisionConflict", err)
	}

	afterStale, err := records.Get(ctx, alice.ID, created.ID)
	if err != nil {
		t.Fatalf("Get() after stale update error = %v", err)
	}
	if afterStale.Title != updated.Title || afterStale.Revision != updated.Revision ||
		!bytes.Equal(afterStale.Ciphertext, updated.Ciphertext) {
		t.Fatalf("record changed after stale update: %+v, want %+v", afterStale, updated)
	}

	_, err = records.Update(ctx, newTestRecord(created.ID, bob.ID, "foreign update"), updated.Revision)
	if !errors.Is(err, model.ErrRecordNotFound) {
		t.Fatalf("foreign Update() error = %v, want ErrRecordNotFound", err)
	}

	_, err = records.Update(
		ctx,
		newTestRecord("550e8400-e29b-41d4-a716-446655440069", alice.ID, "missing update"),
		model.RecordInitialRevision,
	)
	if !errors.Is(err, model.ErrRecordNotFound) {
		t.Fatalf("missing Update() error = %v, want ErrRecordNotFound", err)
	}

	_, err = records.Update(ctx, patch, 0)
	if !errors.Is(err, model.ErrInvalidRecordRevision) {
		t.Fatalf("invalid revision Update() error = %v, want ErrInvalidRecordRevision", err)
	}
}

func TestIntegration_RecordRepositoryDelete(t *testing.T) {
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

	created, err := records.Create(ctx, newTestRecord("550e8400-e29b-41d4-a716-446655440020", alice.ID, "to delete"))
	if err != nil {
		t.Fatalf("Create() error = %v", err)
	}

	err = records.Delete(ctx, alice.ID, created.ID, created.Revision+69)
	if !errors.Is(err, model.ErrRecordRevisionConflict) {
		t.Fatalf("stale Delete() error = %v, want ErrRecordRevisionConflict", err)
	}
	if _, err := records.Get(ctx, alice.ID, created.ID); err != nil {
		t.Fatalf("Get() after stale delete error = %v", err)
	}

	err = records.Delete(ctx, bob.ID, created.ID, created.Revision)
	if !errors.Is(err, model.ErrRecordNotFound) {
		t.Fatalf("foreign Delete() error = %v, want ErrRecordNotFound", err)
	}
	if _, err := records.Get(ctx, alice.ID, created.ID); err != nil {
		t.Fatalf("Get() after foreign delete error = %v", err)
	}

	err = records.Delete(ctx, alice.ID, created.ID, 0)
	if !errors.Is(err, model.ErrInvalidRecordRevision) {
		t.Fatalf("invalid revision Delete() error = %v, want ErrInvalidRecordRevision", err)
	}

	if err := records.Delete(ctx, alice.ID, created.ID, created.Revision); err != nil {
		t.Fatalf("Delete() error = %v", err)
	}

	_, err = records.Get(ctx, alice.ID, created.ID)
	if !errors.Is(err, model.ErrRecordNotFound) {
		t.Fatalf("Get() deleted record error = %v, want ErrRecordNotFound", err)
	}

	var storedCount int
	if err := pool.QueryRow(
		ctx,
		"SELECT count(*) FROM gopherkeeper.records WHERE user_id = $1 AND id = $2",
		alice.ID,
		created.ID,
	).Scan(&storedCount); err != nil {
		t.Fatalf("count deleted record: %v", err)
	}
	if storedCount != 0 {
		t.Fatalf("stored record count = %d, want 0", storedCount)
	}

	err = records.Delete(ctx, alice.ID, created.ID, created.Revision)
	if !errors.Is(err, model.ErrRecordNotFound) {
		t.Fatalf("repeated Delete() error = %v, want ErrRecordNotFound", err)
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
