//go:build integration

package integration_test

import (
	"bytes"
	"context"
	"encoding/base64"
	"os"
	"strings"
	"testing"

	"github.com/jackc/pgx/v5/pgxpool"
	"github.com/xhrobj/gopherkeeper/internal/model"
	"github.com/xhrobj/gopherkeeper/internal/server/postgres"
	"github.com/xhrobj/gopherkeeper/internal/server/recordcrypto"
	"github.com/xhrobj/gopherkeeper/internal/server/service"
)

func TestIntegration_BinaryRecordServiceFlow(t *testing.T) {
	dsn := os.Getenv("DATABASE_DSN")
	if dsn == "" {
		t.Fatal("DATABASE_DSN is not set")
	}

	ctx, cancel := context.WithTimeout(context.Background(), integrationTestTimeout)
	t.Cleanup(cancel)

	pool := openIsolatedMigratedDatabase(t, ctx, dsn)
	userID := insertBinaryRecordTestUser(t, ctx, pool)
	records := postgres.NewRecordRepository(pool)
	crypto, err := recordcrypto.NewService(
		[]byte(strings.Repeat("k", recordcrypto.MasterKeySize)),
		recordcrypto.DefaultKeyID,
	)
	if err != nil {
		t.Fatalf("create record crypto service: %v", err)
	}
	recordsService := service.NewRecordService(records, crypto)

	initial := &model.BinaryPayload{
		Filename:    "alice-backup.bin",
		Data:        []byte{0x00, 0x42, 0x7f, 0x80, 0xfe, 0xff},
		ContentType: "application/octet-stream",
		Metadata:    "Alice private binary backup",
	}
	created, err := recordsService.Create(ctx, service.CreateRecordRequest{
		UserID:  userID,
		Title:   "Alice backup",
		Payload: initial,
	})
	if err != nil {
		t.Fatalf("Create() error = %v", err)
	}
	if created.Metadata.Type != model.RecordTypeBinary || created.Metadata.Revision != model.RecordInitialRevision {
		t.Fatalf("Create() metadata = %+v", created.Metadata)
	}

	initialNonce, initialCiphertext := readStoredBinaryRecord(
		t,
		ctx,
		pool,
		created.Metadata.ID,
		model.RecordInitialRevision,
	)
	assertBinaryCiphertextDoesNotContain(t, initialCiphertext, initial)
	assertBinaryRecordPayload(t, recordsService, ctx, userID, created.Metadata.ID, initial)

	updatedPayload := &model.BinaryPayload{
		Filename:    "alice-backup-v2.bin",
		Data:        []byte{0xff, 0xfe, 0x80, 0x7f, 0x42, 0x00},
		ContentType: "application/octet-stream",
		Metadata:    "Updated Alice private binary backup",
	}
	updated, err := recordsService.Update(ctx, service.UpdateRecordRequest{
		UserID:           userID,
		RecordID:         created.Metadata.ID,
		ExpectedRevision: model.RecordInitialRevision,
		Title:            "Updated Alice backup",
		Payload:          updatedPayload,
	})
	if err != nil {
		t.Fatalf("Update() error = %v", err)
	}
	if updated.Metadata.Type != model.RecordTypeBinary || updated.Metadata.Revision != 2 {
		t.Fatalf("Update() metadata = %+v", updated.Metadata)
	}

	updatedNonce, updatedCiphertext := readStoredBinaryRecord(t, ctx, pool, created.Metadata.ID, 2)
	if bytes.Equal(updatedNonce, initialNonce) {
		t.Fatal("Update() reused record nonce")
	}
	if bytes.Equal(updatedCiphertext, initialCiphertext) {
		t.Fatal("Update() reused record ciphertext")
	}
	assertBinaryCiphertextDoesNotContain(t, updatedCiphertext, updatedPayload)
	assertBinaryRecordPayload(t, recordsService, ctx, userID, created.Metadata.ID, updatedPayload)
}

func insertBinaryRecordTestUser(t *testing.T, ctx context.Context, pool *pgxpool.Pool) int64 {
	t.Helper()

	var userID int64
	if err := pool.QueryRow(
		ctx,
		`INSERT INTO gopherkeeper.users (login, password_hash)
		 VALUES ($1, $2)
		 RETURNING id`,
		"alice",
		[]byte("integration password hash"),
	).Scan(&userID); err != nil {
		t.Fatalf("insert test user: %v", err)
	}

	return userID
}

func readStoredBinaryRecord(
	t *testing.T,
	ctx context.Context,
	pool *pgxpool.Pool,
	recordID string,
	wantRevision int64,
) ([]byte, []byte) {
	t.Helper()

	var recordType string
	var revision int64
	var nonce []byte
	var ciphertext []byte
	if err := pool.QueryRow(
		ctx,
		`SELECT type, revision, nonce, ciphertext
		 FROM gopherkeeper.records
		 WHERE id = $1`,
		recordID,
	).Scan(&recordType, &revision, &nonce, &ciphertext); err != nil {
		t.Fatalf("read stored binary record: %v", err)
	}
	if recordType != string(model.RecordTypeBinary) {
		t.Fatalf("stored record type = %q, want binary", recordType)
	}
	if revision != wantRevision {
		t.Fatalf("stored revision = %d, want %d", revision, wantRevision)
	}

	return nonce, ciphertext
}

func assertBinaryCiphertextDoesNotContain(
	t *testing.T,
	ciphertext []byte,
	payload *model.BinaryPayload,
) {
	t.Helper()

	secrets := [][]byte{
		[]byte(payload.Filename),
		[]byte(payload.ContentType),
		[]byte(payload.Metadata),
		[]byte(base64.StdEncoding.EncodeToString(payload.Data)),
	}
	for _, secret := range secrets {
		if bytes.Contains(ciphertext, secret) {
			t.Fatalf("PostgreSQL ciphertext contains binary payload plaintext %q", secret)
		}
	}
}

func assertBinaryRecordPayload(
	t *testing.T,
	recordsService *service.RecordService,
	ctx context.Context,
	userID int64,
	recordID string,
	want *model.BinaryPayload,
) {
	t.Helper()

	got, err := recordsService.Get(ctx, userID, recordID)
	if err != nil {
		t.Fatalf("Get() error = %v", err)
	}
	payload, ok := got.Payload.(*model.BinaryPayload)
	if !ok || payload == nil {
		t.Fatalf("Get() payload = %#v, want BinaryPayload", got.Payload)
	}
	if payload.Filename != want.Filename ||
		payload.ContentType != want.ContentType ||
		payload.Metadata != want.Metadata ||
		!bytes.Equal(payload.Data, want.Data) {
		t.Fatalf("Get() payload = %#v, want %#v", payload, want)
	}
}
