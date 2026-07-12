//go:build integration

package postgres_test

import (
	"bytes"
	"context"
	"errors"
	"os"
	"testing"
	"time"

	"github.com/jackc/pgx/v5/pgxpool"
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

	second := newTestRecord("550e8400-e29b-41d4-a716-446655440001", alice.ID, "second")
	createdSecond, err := records.Create(ctx, second)
	if err != nil {
		t.Fatalf("Create() second error = %v", err)
	}
	_, err = records.Create(ctx, newTestRecord("550e8400-e29b-41d4-a716-446655440002", bob.ID, "bob record"))
	if err != nil {
		t.Fatalf("Create() Bob record error = %v", err)
	}
	setRecordUpdatedAt(t, ctx, pool, alice.ID, createdFirst.ID, time.Date(2026, time.July, 9, 12, 0, 0, 0, time.UTC))
	setRecordUpdatedAt(t, ctx, pool, alice.ID, createdSecond.ID, time.Date(2026, time.July, 9, 12, 1, 0, 0, time.UTC))

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
	fixture := newRecordRepositoryIntegrationFixture(t)
	created := fixture.createRecord("550e8400-e29b-41d4-a716-446655440010", fixture.alice.ID, "original")
	created = setRecordTimestamps(
		fixture.t,
		fixture.ctx,
		fixture.pool,
		fixture.records,
		fixture.alice.ID,
		created.ID,
		time.Date(2026, time.July, 9, 12, 0, 0, 0, time.UTC),
	)

	patch := newTestRecord(created.ID, fixture.alice.ID, "updated")
	patch.Nonce = []byte("updated nonce")
	patch.Ciphertext = []byte("updated ciphertext")

	updated, err := fixture.records.Update(fixture.ctx, patch, created.Revision)
	if err != nil {
		t.Fatalf("Update() error = %v", err)
	}
	assertUpdatedRecord(t, updated, created, patch)
	assertStoredCiphertextEncrypted(t, fixture.ctx, fixture.pool, fixture.alice.ID, created.ID)
	assertStaleUpdateDoesNotChangeRecord(t, fixture, created, updated)
	assertForeignAndMissingUpdateReturnNotFound(t, fixture, created, updated)
	assertInvalidUpdateRevision(t, fixture, patch)
}

type recordRepositoryIntegrationFixture struct {
	t       *testing.T
	ctx     context.Context
	pool    *pgxpool.Pool
	records *postgres.RecordRepository
	alice   model.User
	bob     model.User
}

func newRecordRepositoryIntegrationFixture(t *testing.T) recordRepositoryIntegrationFixture {
	t.Helper()

	dsn := os.Getenv("DATABASE_DSN")
	if dsn == "" {
		t.Fatal("DATABASE_DSN is not set")
	}

	ctx, cancel := context.WithTimeout(context.Background(), repositoryIntegrationTestTimeout)
	t.Cleanup(cancel)

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

	return recordRepositoryIntegrationFixture{
		t:       t,
		ctx:     ctx,
		pool:    pool,
		records: records,
		alice:   alice,
		bob:     bob,
	}
}

func (f recordRepositoryIntegrationFixture) createRecord(id string, userID int64, title string) model.EncryptedRecord {
	f.t.Helper()

	created, err := f.records.Create(f.ctx, newTestRecord(id, userID, title))
	if err != nil {
		f.t.Fatalf("Create() error = %v", err)
	}

	return created
}

func setRecordUpdatedAt(
	t *testing.T,
	ctx context.Context,
	pool *pgxpool.Pool,
	userID int64,
	recordID string,
	updatedAt time.Time,
) {
	t.Helper()

	if _, err := pool.Exec(
		ctx,
		"UPDATE gopherkeeper.records SET updated_at = $3 WHERE user_id = $1 AND id = $2",
		userID,
		recordID,
		updatedAt,
	); err != nil {
		t.Fatalf("set record updated_at: %v", err)
	}
}

func setRecordTimestamps(
	t *testing.T,
	ctx context.Context,
	pool *pgxpool.Pool,
	records *postgres.RecordRepository,
	userID int64,
	recordID string,
	value time.Time,
) model.EncryptedRecord {
	t.Helper()

	if _, err := pool.Exec(
		ctx,
		"UPDATE gopherkeeper.records SET created_at = $3, updated_at = $3 WHERE user_id = $1 AND id = $2",
		userID,
		recordID,
		value,
	); err != nil {
		t.Fatalf("set record timestamps: %v", err)
	}

	record, err := records.Get(ctx, userID, recordID)
	if err != nil {
		t.Fatalf("Get() after timestamp normalization error = %v", err)
	}

	return record
}

func assertUpdatedRecord(t *testing.T, updated model.EncryptedRecord, created model.EncryptedRecord, patch model.EncryptedRecord) {
	t.Helper()

	if updated.ID != created.ID || updated.UserID != created.UserID || updated.Type != model.RecordTypeText {
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
}

func assertStoredCiphertextEncrypted(
	t *testing.T,
	ctx context.Context,
	pool *pgxpool.Pool,
	userID int64,
	recordID string,
) {
	t.Helper()

	var storedCiphertext []byte
	if err := pool.QueryRow(
		ctx,
		"SELECT ciphertext FROM gopherkeeper.records WHERE user_id = $1 AND id = $2",
		userID,
		recordID,
	).Scan(&storedCiphertext); err != nil {
		t.Fatalf("read updated ciphertext: %v", err)
	}
	if bytes.Contains(storedCiphertext, []byte("updated secret note")) {
		t.Fatal("stored ciphertext contains plaintext payload")
	}
}

func assertStaleUpdateDoesNotChangeRecord(
	t *testing.T,
	fixture recordRepositoryIntegrationFixture,
	created model.EncryptedRecord,
	updated model.EncryptedRecord,
) {
	t.Helper()

	stalePatch := newTestRecord(created.ID, fixture.alice.ID, "stale update")
	_, err := fixture.records.Update(fixture.ctx, stalePatch, created.Revision)
	if !errors.Is(err, model.ErrRecordRevisionConflict) {
		t.Fatalf("stale Update() error = %v, want ErrRecordRevisionConflict", err)
	}

	afterStale, err := fixture.records.Get(fixture.ctx, fixture.alice.ID, created.ID)
	if err != nil {
		t.Fatalf("Get() after stale update error = %v", err)
	}
	if afterStale.Title != updated.Title || afterStale.Revision != updated.Revision ||
		!bytes.Equal(afterStale.Ciphertext, updated.Ciphertext) {
		t.Fatalf("record changed after stale update: %+v, want %+v", afterStale, updated)
	}
}

func assertForeignAndMissingUpdateReturnNotFound(
	t *testing.T,
	fixture recordRepositoryIntegrationFixture,
	created model.EncryptedRecord,
	updated model.EncryptedRecord,
) {
	t.Helper()

	_, err := fixture.records.Update(
		fixture.ctx,
		newTestRecord(created.ID, fixture.bob.ID, "foreign update"),
		updated.Revision,
	)
	if !errors.Is(err, model.ErrRecordNotFound) {
		t.Fatalf("foreign Update() error = %v, want ErrRecordNotFound", err)
	}

	_, err = fixture.records.Update(
		fixture.ctx,
		newTestRecord("550e8400-e29b-41d4-a716-446655440069", fixture.alice.ID, "missing update"),
		model.RecordInitialRevision,
	)
	if !errors.Is(err, model.ErrRecordNotFound) {
		t.Fatalf("missing Update() error = %v, want ErrRecordNotFound", err)
	}
}

func assertInvalidUpdateRevision(
	t *testing.T,
	fixture recordRepositoryIntegrationFixture,
	patch model.EncryptedRecord,
) {
	t.Helper()

	_, err := fixture.records.Update(fixture.ctx, patch, 0)
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

func newTestRecord(id string, userID int64, title string) model.EncryptedRecord {
	return model.EncryptedRecord{
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
