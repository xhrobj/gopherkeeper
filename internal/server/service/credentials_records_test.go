package service

import (
	"bytes"
	"context"
	"encoding/json"
	"errors"
	"strings"
	"testing"
	"time"

	"github.com/xhrobj/gopherkeeper/internal/model"
	"github.com/xhrobj/gopherkeeper/internal/server/recordcrypto"
)

func TestRecordService_CreateCredentials(t *testing.T) {
	createdAt := time.Date(2026, time.July, 10, 12, 0, 0, 0, time.UTC)
	updatedAt := time.Date(2026, time.July, 10, 12, 1, 0, 0, time.UTC)
	payload := testCredentialsPayload()
	encrypted := recordcrypto.EncryptedPayload{
		CryptoVersion: recordcrypto.CryptoVersion,
		KeyID:         recordcrypto.DefaultKeyID,
		Nonce:         []byte("nonce"),
		Ciphertext:    []byte("ciphertext"),
	}

	crypto := &recordPayloadCryptoStub{
		encryptFunc: func(plaintext, aad []byte) (recordcrypto.EncryptedPayload, error) {
			assertEncryptedCredentialsPayload(t, plaintext, payload)
			if !strings.Contains(string(aad), "gopherkeeper:v1:user:42:record:") ||
				!strings.Contains(string(aad), ":type:credentials") {
				t.Fatalf("Encrypt() AAD = %q", aad)
			}

			return encrypted, nil
		},
	}
	records := &recordRepositoryStub{
		createFunc: func(_ context.Context, record model.Record) (model.Record, error) {
			if err := model.ValidateRecordID(record.ID); err != nil {
				t.Fatalf("Create() record ID is invalid: %v", err)
			}
			if record.UserID != 42 || record.Type != model.RecordTypeCredentials || record.Title != "GitHub" {
				t.Fatalf("Create() record = %+v", record)
			}
			if record.Revision != 0 {
				t.Fatalf("Create() revision = %d, want DB default", record.Revision)
			}
			assertEncryptedRecord(t, record, encrypted)

			record.Revision = model.RecordInitialRevision
			record.CreatedAt = createdAt
			record.UpdatedAt = updatedAt
			return record, nil
		},
	}
	service := NewRecordService(records, crypto)

	created, err := service.CreateCredentials(context.Background(), CreateCredentialsRecordRequest{
		UserID:  42,
		Title:   "GitHub",
		Payload: payload,
	})
	if err != nil {
		t.Fatalf("CreateCredentials() error = %v", err)
	}
	if created.Metadata.Type != model.RecordTypeCredentials || created.Metadata.Title != "GitHub" ||
		created.Metadata.Revision != model.RecordInitialRevision || created.Metadata.CreatedAt != createdAt ||
		created.Metadata.UpdatedAt != updatedAt {
		t.Fatalf("CreateCredentials() metadata = %+v", created.Metadata)
	}
	if created.Payload != payload {
		t.Fatalf("CreateCredentials() payload = %+v, want %+v", created.Payload, payload)
	}
	if crypto.encryptCalls != 1 || records.createCalls != 1 {
		t.Fatalf("calls: Encrypt=%d Create=%d", crypto.encryptCalls, records.createCalls)
	}
}

func TestRecordService_CreateCredentialsValidationError(t *testing.T) {
	tests := []struct {
		name    string
		request CreateCredentialsRecordRequest
		wantErr error
	}{
		{
			name: "invalid owner",
			request: CreateCredentialsRecordRequest{
				Title:   "GitHub",
				Payload: testCredentialsPayload(),
			},
			wantErr: errInvalidRecordOwner,
		},
		{
			name: "invalid title",
			request: CreateCredentialsRecordRequest{
				UserID:  42,
				Title:   "   ",
				Payload: testCredentialsPayload(),
			},
			wantErr: model.ErrInvalidRecordTitle,
		},
		{
			name: "invalid payload",
			request: CreateCredentialsRecordRequest{
				UserID: 42,
				Title:  "GitHub",
				Payload: model.CredentialsPayload{
					Login: "alice",
				},
			},
			wantErr: model.ErrInvalidCredentialsPayload,
		},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			records := &recordRepositoryStub{}
			crypto := &recordPayloadCryptoStub{}
			service := NewRecordService(records, crypto)

			_, err := service.CreateCredentials(context.Background(), tt.request)
			if !errors.Is(err, tt.wantErr) {
				t.Fatalf("CreateCredentials() error = %v, want %v", err, tt.wantErr)
			}
			if records.createCalls != 0 || crypto.encryptCalls != 0 {
				t.Fatalf("calls: Create=%d Encrypt=%d, want 0", records.createCalls, crypto.encryptCalls)
			}
		})
	}
}

func TestRecordService_GetCredentials(t *testing.T) {
	recordID := "550e8400-e29b-41d4-a716-446655440000"
	createdAt := time.Date(2026, time.July, 10, 12, 0, 0, 0, time.UTC)
	updatedAt := time.Date(2026, time.July, 10, 12, 1, 0, 0, time.UTC)
	payload := testCredentialsPayload()
	stored := model.Record{
		ID:            recordID,
		UserID:        42,
		Type:          model.RecordTypeCredentials,
		Title:         "GitHub",
		Revision:      1,
		CreatedAt:     createdAt,
		UpdatedAt:     updatedAt,
		CryptoVersion: recordcrypto.CryptoVersion,
		KeyID:         recordcrypto.DefaultKeyID,
		Nonce:         []byte("nonce"),
		Ciphertext:    []byte("ciphertext"),
	}
	plaintext, err := json.Marshal(payload)
	if err != nil {
		t.Fatalf("json.Marshal() error = %v", err)
	}

	records := &recordRepositoryStub{
		getFunc: func(_ context.Context, userID int64, gotRecordID string) (model.Record, error) {
			if userID != 42 || gotRecordID != recordID {
				t.Fatalf("Get() args = %d, %q", userID, gotRecordID)
			}

			return stored, nil
		},
	}
	crypto := &recordPayloadCryptoStub{
		decryptFunc: func(encrypted recordcrypto.EncryptedPayload, aad []byte) ([]byte, error) {
			assertDecryptInput(t, encrypted, stored)
			wantAAD := "gopherkeeper:v1:user:42:record:550e8400-e29b-41d4-a716-446655440000:type:credentials"
			if string(aad) != wantAAD {
				t.Fatalf("Decrypt() AAD = %q, want %q", aad, wantAAD)
			}

			return plaintext, nil
		},
	}
	service := NewRecordService(records, crypto)

	got, err := service.GetCredentials(context.Background(), 42, recordID)
	if err != nil {
		t.Fatalf("GetCredentials() error = %v", err)
	}
	if got.Metadata != stored.Metadata() {
		t.Fatalf("GetCredentials() metadata = %+v, want %+v", got.Metadata, stored.Metadata())
	}
	if got.Payload != payload {
		t.Fatalf("GetCredentials() payload = %+v, want %+v", got.Payload, payload)
	}
	if records.getCalls != 1 || crypto.decryptCalls != 1 {
		t.Fatalf("calls: Get=%d Decrypt=%d", records.getCalls, crypto.decryptCalls)
	}
}

func TestRecordService_GetCredentialsRejectsUnexpectedPayload(t *testing.T) {
	recordID := "550e8400-e29b-41d4-a716-446655440000"
	tests := []struct {
		name        string
		storedType  model.RecordType
		plaintext   []byte
		wantErr     error
		wantDecrypt int
	}{
		{
			name:       "unsupported record type",
			storedType: model.RecordTypeText,
			wantErr:    model.ErrRecordTypeUnsupported,
		},
		{
			name:        "invalid decrypted payload",
			storedType:  model.RecordTypeCredentials,
			plaintext:   []byte(`{"login":"alice"}`),
			wantErr:     model.ErrInvalidCredentialsPayload,
			wantDecrypt: 1,
		},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			records := &recordRepositoryStub{
				getFunc: func(context.Context, int64, string) (model.Record, error) {
					return model.Record{
						ID:            recordID,
						UserID:        42,
						Type:          tt.storedType,
						Revision:      1,
						CryptoVersion: recordcrypto.CryptoVersion,
						KeyID:         recordcrypto.DefaultKeyID,
					}, nil
				},
			}
			crypto := &recordPayloadCryptoStub{
				decryptFunc: func(recordcrypto.EncryptedPayload, []byte) ([]byte, error) {
					return tt.plaintext, nil
				},
			}
			service := NewRecordService(records, crypto)

			_, err := service.GetCredentials(context.Background(), 42, recordID)
			if !errors.Is(err, tt.wantErr) {
				t.Fatalf("GetCredentials() error = %v, want %v", err, tt.wantErr)
			}
			if crypto.decryptCalls != tt.wantDecrypt {
				t.Fatalf("Decrypt() calls = %d, want %d", crypto.decryptCalls, tt.wantDecrypt)
			}
		})
	}
}

func TestRecordService_UpdateCredentials(t *testing.T) {
	recordID := "550e8400-e29b-41d4-a716-446655440000"
	createdAt := time.Date(2026, time.July, 10, 12, 0, 0, 0, time.UTC)
	updatedAt := time.Date(2026, time.July, 10, 12, 5, 0, 0, time.UTC)
	payload := model.CredentialsPayload{
		Login:    "alice@example.com",
		Password: "updated-secret",
		URL:      "https://github.com",
		Metadata: "updated personal account",
	}
	current := model.Record{
		ID:            recordID,
		UserID:        42,
		Type:          model.RecordTypeCredentials,
		Title:         "GitHub",
		Revision:      1,
		CreatedAt:     createdAt,
		UpdatedAt:     createdAt,
		CryptoVersion: recordcrypto.CryptoVersion,
		KeyID:         recordcrypto.DefaultKeyID,
		Nonce:         []byte("old nonce"),
		Ciphertext:    []byte("old ciphertext"),
	}
	encrypted := recordcrypto.EncryptedPayload{
		CryptoVersion: recordcrypto.CryptoVersion,
		KeyID:         recordcrypto.DefaultKeyID,
		Nonce:         []byte("new nonce"),
		Ciphertext:    []byte("new ciphertext"),
	}

	records := &recordRepositoryStub{
		getFunc: func(_ context.Context, userID int64, gotRecordID string) (model.Record, error) {
			if userID != 42 || gotRecordID != recordID {
				t.Fatalf("Get() args = %d, %q", userID, gotRecordID)
			}

			return current, nil
		},
		updateFunc: func(_ context.Context, record model.Record, expectedRevision int64) (model.Record, error) {
			if expectedRevision != 1 || record.ID != recordID || record.UserID != 42 ||
				record.Type != model.RecordTypeCredentials || record.Title != "Updated GitHub" {
				t.Fatalf("Update() record = %+v, revision = %d", record, expectedRevision)
			}
			assertEncryptedRecord(t, record, encrypted)

			record.Revision = 2
			record.CreatedAt = createdAt
			record.UpdatedAt = updatedAt
			return record, nil
		},
	}
	crypto := &recordPayloadCryptoStub{
		encryptFunc: func(plaintext, aad []byte) (recordcrypto.EncryptedPayload, error) {
			assertEncryptedCredentialsPayload(t, plaintext, payload)
			wantAAD := "gopherkeeper:v1:user:42:record:550e8400-e29b-41d4-a716-446655440000:type:credentials"
			if string(aad) != wantAAD {
				t.Fatalf("Encrypt() AAD = %q, want %q", aad, wantAAD)
			}

			return encrypted, nil
		},
	}
	service := NewRecordService(records, crypto)

	updated, err := service.UpdateCredentials(context.Background(), UpdateCredentialsRecordRequest{
		UserID:           42,
		RecordID:         recordID,
		ExpectedRevision: 1,
		Title:            "Updated GitHub",
		Payload:          payload,
	})
	if err != nil {
		t.Fatalf("UpdateCredentials() error = %v", err)
	}
	if updated.Metadata.Type != model.RecordTypeCredentials || updated.Metadata.Title != "Updated GitHub" ||
		updated.Metadata.Revision != 2 || updated.Metadata.CreatedAt != createdAt || updated.Metadata.UpdatedAt != updatedAt {
		t.Fatalf("UpdateCredentials() metadata = %+v", updated.Metadata)
	}
	if updated.Payload != payload {
		t.Fatalf("UpdateCredentials() payload = %+v, want %+v", updated.Payload, payload)
	}
	if records.getCalls != 1 || records.updateCalls != 1 || crypto.encryptCalls != 1 {
		t.Fatalf("calls: Get=%d Update=%d Encrypt=%d", records.getCalls, records.updateCalls, crypto.encryptCalls)
	}
}

func TestRecordService_UpdateCredentialsRejectsInvalidState(t *testing.T) {
	recordID := "550e8400-e29b-41d4-a716-446655440000"
	tests := []struct {
		name    string
		payload model.CredentialsPayload
		current model.Record
		wantErr error
		wantGet int
	}{
		{
			name: "invalid payload",
			payload: model.CredentialsPayload{
				Login: "alice",
			},
			wantErr: model.ErrInvalidCredentialsPayload,
		},
		{
			name:    "unsupported record type",
			payload: testCredentialsPayload(),
			current: model.Record{
				ID:       recordID,
				UserID:   42,
				Type:     model.RecordTypeText,
				Revision: 1,
			},
			wantErr: model.ErrRecordTypeUnsupported,
			wantGet: 1,
		},
		{
			name:    "stale revision",
			payload: testCredentialsPayload(),
			current: model.Record{
				ID:       recordID,
				UserID:   42,
				Type:     model.RecordTypeCredentials,
				Revision: 2,
			},
			wantErr: model.ErrRecordRevisionConflict,
			wantGet: 1,
		},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			records := &recordRepositoryStub{}
			if tt.wantGet > 0 {
				records.getFunc = func(context.Context, int64, string) (model.Record, error) {
					return tt.current, nil
				}
			}
			crypto := &recordPayloadCryptoStub{}
			service := NewRecordService(records, crypto)

			_, err := service.UpdateCredentials(context.Background(), UpdateCredentialsRecordRequest{
				UserID:           42,
				RecordID:         recordID,
				ExpectedRevision: 1,
				Title:            "GitHub",
				Payload:          tt.payload,
			})
			if !errors.Is(err, tt.wantErr) {
				t.Fatalf("UpdateCredentials() error = %v, want %v", err, tt.wantErr)
			}
			if records.getCalls != tt.wantGet || records.updateCalls != 0 || crypto.encryptCalls != 0 {
				t.Fatalf(
					"calls: Get=%d Update=%d Encrypt=%d",
					records.getCalls,
					records.updateCalls,
					crypto.encryptCalls,
				)
			}
		})
	}
}

func testCredentialsPayload() model.CredentialsPayload {
	return model.CredentialsPayload{
		Login:    "alice@example.com",
		Password: "correct-horse-battery-staple",
		URL:      "https://github.com",
		Metadata: "personal account",
	}
}

func assertEncryptedCredentialsPayload(t *testing.T, plaintext []byte, want model.CredentialsPayload) {
	t.Helper()

	var got model.CredentialsPayload
	if err := json.Unmarshal(plaintext, &got); err != nil {
		t.Fatalf("Encrypt() plaintext is not CredentialsPayload JSON: %v", err)
	}
	if got != want {
		t.Fatalf("Encrypt() payload = %+v, want %+v", got, want)
	}
}

func assertEncryptedRecord(t *testing.T, record model.Record, want recordcrypto.EncryptedPayload) {
	t.Helper()

	if record.CryptoVersion != want.CryptoVersion || record.KeyID != want.KeyID ||
		!bytes.Equal(record.Nonce, want.Nonce) || !bytes.Equal(record.Ciphertext, want.Ciphertext) {
		t.Fatalf("encrypted record = %+v", record)
	}
}

func assertDecryptInput(t *testing.T, encrypted recordcrypto.EncryptedPayload, record model.Record) {
	t.Helper()

	if encrypted.CryptoVersion != record.CryptoVersion || encrypted.KeyID != record.KeyID ||
		!bytes.Equal(encrypted.Nonce, record.Nonce) || !bytes.Equal(encrypted.Ciphertext, record.Ciphertext) {
		t.Fatalf("Decrypt() encrypted payload = %+v", encrypted)
	}
}
