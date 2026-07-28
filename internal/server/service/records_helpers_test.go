package service

import (
	"bytes"
	"context"
	"encoding/json"
	"errors"
	"testing"
	"time"

	"github.com/xhrobj/gopherkeeper/internal/model"
	"github.com/xhrobj/gopherkeeper/internal/server/recordcrypto"
)

type updateRecordFixture struct {
	recordID  string
	createdAt time.Time
	updatedAt time.Time
	current   model.EncryptedRecord
	encrypted recordcrypto.EncryptedPayload
	payload   model.TextPayload
}

type recordServiceCallCounts struct {
	get     int
	update  int
	encrypt int
	decrypt int
}

type recordRepositoryStub struct {
	createFunc func(context.Context, model.EncryptedRecord) (model.EncryptedRecord, error)
	listFunc   func(context.Context, int64) ([]model.RecordMetadata, error)
	getFunc    func(context.Context, int64, string) (model.EncryptedRecord, error)
	updateFunc func(context.Context, model.EncryptedRecord, int64) (model.EncryptedRecord, error)
	deleteFunc func(context.Context, int64, string, int64) error

	createCalls int
	listCalls   int
	getCalls    int
	updateCalls int
	deleteCalls int
}

type recordPayloadCryptoStub struct {
	encryptFunc func([]byte, []byte) (recordcrypto.EncryptedPayload, error)
	decryptFunc func(recordcrypto.EncryptedPayload, []byte) ([]byte, error)

	encryptCalls int
	decryptCalls int
}

func (s *recordRepositoryStub) Create(ctx context.Context, record model.EncryptedRecord) (model.EncryptedRecord, error) {
	s.createCalls++
	if s.createFunc == nil {
		return model.EncryptedRecord{}, errors.New("unexpected Create call")
	}

	return s.createFunc(ctx, record)
}

func (s *recordRepositoryStub) ListMetadata(ctx context.Context, userID int64) ([]model.RecordMetadata, error) {
	s.listCalls++
	if s.listFunc == nil {
		return nil, errors.New("unexpected ListMetadata call")
	}

	return s.listFunc(ctx, userID)
}

func (s *recordRepositoryStub) Get(ctx context.Context, userID int64, recordID string) (model.EncryptedRecord, error) {
	s.getCalls++
	if s.getFunc == nil {
		return model.EncryptedRecord{}, errors.New("unexpected Get call")
	}

	return s.getFunc(ctx, userID, recordID)
}

func (s *recordRepositoryStub) Update(
	ctx context.Context,
	record model.EncryptedRecord,
	expectedRevision int64,
) (model.EncryptedRecord, error) {
	s.updateCalls++
	if s.updateFunc == nil {
		return model.EncryptedRecord{}, errors.New("unexpected Update call")
	}

	return s.updateFunc(ctx, record, expectedRevision)
}

func (s *recordRepositoryStub) Delete(
	ctx context.Context,
	userID int64,
	recordID string,
	expectedRevision int64,
) error {
	s.deleteCalls++
	if s.deleteFunc == nil {
		return errors.New("unexpected Delete call")
	}

	return s.deleteFunc(ctx, userID, recordID, expectedRevision)
}

func (s *recordPayloadCryptoStub) Encrypt(plaintext []byte, aad []byte) (recordcrypto.EncryptedPayload, error) {
	s.encryptCalls++
	if s.encryptFunc == nil {
		return recordcrypto.EncryptedPayload{}, errors.New("unexpected Encrypt call")
	}

	return s.encryptFunc(plaintext, aad)
}

func (s *recordPayloadCryptoStub) Decrypt(encrypted recordcrypto.EncryptedPayload, aad []byte) ([]byte, error) {
	s.decryptCalls++
	if s.decryptFunc == nil {
		return nil, errors.New("unexpected Decrypt call")
	}

	return s.decryptFunc(encrypted, aad)
}

func newUpdateRecordFixture() updateRecordFixture {
	recordID := "550e8400-e29b-41d4-a716-446655440000"
	createdAt := time.Date(2026, time.July, 9, 12, 0, 0, 0, time.UTC)
	updatedAt := time.Date(2026, time.July, 9, 12, 5, 0, 0, time.UTC)

	return updateRecordFixture{
		recordID:  recordID,
		createdAt: createdAt,
		updatedAt: updatedAt,
		current: model.EncryptedRecord{
			ID:            recordID,
			UserID:        42,
			Type:          model.RecordTypeText,
			Title:         "Alice note",
			Revision:      1,
			CreatedAt:     createdAt,
			UpdatedAt:     createdAt,
			CryptoVersion: recordcrypto.CryptoVersion,
			KeyID:         recordcrypto.DefaultKeyID,
			Nonce:         []byte("old nonce"),
			Ciphertext:    []byte("old ciphertext"),
		},
		encrypted: recordcrypto.EncryptedPayload{
			CryptoVersion: recordcrypto.CryptoVersion,
			KeyID:         recordcrypto.DefaultKeyID,
			Nonce:         []byte("new nonce"),
			Ciphertext:    []byte("new ciphertext"),
		},
		payload: model.TextPayload{Text: "updated secret note", Metadata: "updated private metadata"},
	}
}

func newUpdateCryptoStub(t *testing.T, fixture updateRecordFixture) *recordPayloadCryptoStub {
	t.Helper()

	return &recordPayloadCryptoStub{
		encryptFunc: func(plaintext, aad []byte) (recordcrypto.EncryptedPayload, error) {
			assertEncryptedRecordPayload(t, plaintext, fixture.payload)
			wantAAD := "gopherkeeper:v1:user:42:record:550e8400-e29b-41d4-a716-446655440000:type:text"
			if string(aad) != wantAAD {
				t.Fatalf("Encrypt() AAD = %q, want %q", aad, wantAAD)
			}

			return fixture.encrypted, nil
		},
	}
}

func assertEncryptedRecordPayload(t *testing.T, plaintext []byte, want model.TextPayload) {
	t.Helper()

	var got model.TextPayload
	if err := json.Unmarshal(plaintext, &got); err != nil {
		t.Fatalf("Encrypt() plaintext is not TextPayload JSON: %v", err)
	}
	if got != want {
		t.Fatalf("Encrypt() payload = %+v, want %+v", got, want)
	}
}

func newUpdateRepositoryStub(t *testing.T, fixture updateRecordFixture) *recordRepositoryStub {
	t.Helper()

	return &recordRepositoryStub{
		getFunc: func(_ context.Context, userID int64, recordID string) (model.EncryptedRecord, error) {
			assertUpdateGetArgs(t, userID, recordID, fixture)
			return fixture.current, nil
		},
		updateFunc: func(_ context.Context, record model.EncryptedRecord, expectedRevision int64) (model.EncryptedRecord, error) {
			assertUpdateRecordPatch(t, record, expectedRevision, fixture)
			updated := record
			updated.Revision = 2
			updated.CreatedAt = fixture.createdAt
			updated.UpdatedAt = fixture.updatedAt
			return updated, nil
		},
	}
}

func assertUpdateGetArgs(t *testing.T, userID int64, recordID string, fixture updateRecordFixture) {
	t.Helper()

	if userID != 42 || recordID != fixture.recordID {
		t.Fatalf("Get() args = %d, %q", userID, recordID)
	}
}

func assertUpdateRecordPatch(
	t *testing.T,
	record model.EncryptedRecord,
	expectedRevision int64,
	fixture updateRecordFixture,
) {
	t.Helper()

	if expectedRevision != 1 {
		t.Fatalf("Update() expectedRevision = %d, want 1", expectedRevision)
	}
	if record.ID != fixture.recordID || record.UserID != 42 || record.Type != model.RecordTypeText {
		t.Fatalf("Update() record identity = %+v", record)
	}
	if record.Title != "Updated Alice note" {
		t.Fatalf("Update() title = %q, want Updated Alice note", record.Title)
	}
	if bytes.Equal(record.Nonce, fixture.current.Nonce) || bytes.Equal(record.Ciphertext, fixture.current.Ciphertext) {
		t.Fatalf("Update() reused old encrypted payload: %+v", record)
	}
	if record.CryptoVersion != fixture.encrypted.CryptoVersion || record.KeyID != fixture.encrypted.KeyID ||
		!bytes.Equal(record.Nonce, fixture.encrypted.Nonce) ||
		!bytes.Equal(record.Ciphertext, fixture.encrypted.Ciphertext) {
		t.Fatalf("Update() encrypted record = %+v", record)
	}
}

func assertUpdatedRecord(t *testing.T, got model.Record, fixture updateRecordFixture) {
	t.Helper()

	if got.Metadata.ID != fixture.recordID || got.Metadata.Title != "Updated Alice note" ||
		got.Metadata.Revision != 2 || got.Metadata.CreatedAt != fixture.createdAt ||
		got.Metadata.UpdatedAt != fixture.updatedAt {
		t.Fatalf("Update() metadata = %+v", got.Metadata)
	}
	payload, ok := got.Payload.(*model.TextPayload)
	if !ok || payload == nil || *payload != fixture.payload {
		t.Fatalf("Update() payload = %#v, want %+v", got.Payload, fixture.payload)
	}
}

func assertRecordServiceCalls(
	t *testing.T,
	records *recordRepositoryStub,
	crypto *recordPayloadCryptoStub,
	want recordServiceCallCounts,
) {
	t.Helper()

	if records.getCalls != want.get || records.updateCalls != want.update ||
		crypto.encryptCalls != want.encrypt || crypto.decryptCalls != want.decrypt {
		t.Fatalf(
			"calls: Get=%d Update=%d Encrypt=%d Decrypt=%d",
			records.getCalls,
			records.updateCalls,
			crypto.encryptCalls,
			crypto.decryptCalls,
		)
	}
}
