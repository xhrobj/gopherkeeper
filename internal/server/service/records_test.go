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

func TestRecordService_CreateText(t *testing.T) {
	createdAt := time.Date(2026, time.July, 8, 12, 0, 0, 0, time.UTC)
	updatedAt := time.Date(2026, time.July, 8, 12, 1, 0, 0, time.UTC)
	encrypted := recordcrypto.EncryptedPayload{
		CryptoVersion: recordcrypto.CryptoVersion,
		KeyID:         recordcrypto.DefaultKeyID,
		Nonce:         []byte("nonce"),
		Ciphertext:    []byte("ciphertext"),
	}

	crypto := &recordPayloadCryptoStub{
		encryptFunc: func(plaintext []byte, aad []byte) (recordcrypto.EncryptedPayload, error) {
			var payload model.TextPayload
			if err := json.Unmarshal(plaintext, &payload); err != nil {
				t.Fatalf("Encrypt() plaintext is not TextPayload JSON: %v", err)
			}
			if payload.Text != "secret note" || payload.Metadata != "private metadata" {
				t.Fatalf("Encrypt() payload = %+v", payload)
			}
			if !strings.Contains(string(aad), "gopherkeeper:v1:user:42:record:") ||
				!strings.Contains(string(aad), ":type:text") {
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
			if record.UserID != 42 || record.Type != model.RecordTypeText || record.Title != "Alice note" {
				t.Fatalf("Create() record = %+v", record)
			}
			if record.Revision != 0 {
				t.Fatalf("Create() revision = %d, want DB default", record.Revision)
			}
			if record.CryptoVersion != encrypted.CryptoVersion || record.KeyID != encrypted.KeyID ||
				!bytes.Equal(record.Nonce, encrypted.Nonce) || !bytes.Equal(record.Ciphertext, encrypted.Ciphertext) {
				t.Fatalf("Create() encrypted record = %+v", record)
			}

			record.Revision = model.RecordInitialRevision
			record.CreatedAt = createdAt
			record.UpdatedAt = updatedAt
			return record, nil
		},
	}
	service := NewRecordService(records, crypto)

	created, err := service.CreateText(context.Background(), CreateTextRecordRequest{
		UserID: 42,
		Title:  "Alice note",
		Payload: model.TextPayload{
			Text:     "secret note",
			Metadata: "private metadata",
		},
	})
	if err != nil {
		t.Fatalf("CreateText() error = %v", err)
	}
	if created.Metadata.Title != "Alice note" || created.Metadata.CreatedAt != createdAt || created.Metadata.UpdatedAt != updatedAt {
		t.Fatalf("CreateText() metadata = %+v", created.Metadata)
	}
	if created.Payload.Text != "secret note" || created.Payload.Metadata != "private metadata" {
		t.Fatalf("CreateText() payload = %+v", created.Payload)
	}
	if crypto.encryptCalls != 1 || records.createCalls != 1 {
		t.Fatalf("calls: Encrypt=%d Create=%d", crypto.encryptCalls, records.createCalls)
	}
}

func TestRecordService_CreateTextValidationError(t *testing.T) {
	crypto := &recordPayloadCryptoStub{}
	records := &recordRepositoryStub{}
	service := NewRecordService(records, crypto)

	_, err := service.CreateText(context.Background(), CreateTextRecordRequest{
		UserID:  42,
		Title:   "Alice note",
		Payload: model.TextPayload{},
	})
	if !errors.Is(err, model.ErrInvalidTextPayload) {
		t.Fatalf("CreateText() error = %v, want ErrInvalidTextPayload", err)
	}
	if crypto.encryptCalls != 0 || records.createCalls != 0 {
		t.Fatalf("calls: Encrypt=%d Create=%d, want 0", crypto.encryptCalls, records.createCalls)
	}
}

func TestRecordService_List(t *testing.T) {
	updatedAt := time.Date(2026, time.July, 8, 12, 0, 0, 0, time.UTC)
	want := []model.RecordMetadata{
		{
			ID:        "550e8400-e29b-41d4-a716-446655440000",
			Type:      model.RecordTypeText,
			Title:     "Alice note",
			Revision:  1,
			UpdatedAt: updatedAt,
		},
	}
	records := &recordRepositoryStub{
		listFunc: func(_ context.Context, userID int64) ([]model.RecordMetadata, error) {
			if userID != 42 {
				t.Fatalf("ListMetadata() userID = %d, want 42", userID)
			}

			return want, nil
		},
	}
	crypto := &recordPayloadCryptoStub{}
	service := NewRecordService(records, crypto)

	got, err := service.List(context.Background(), 42)
	if err != nil {
		t.Fatalf("List() error = %v", err)
	}
	if len(got) != 1 || got[0] != want[0] {
		t.Fatalf("List() = %+v, want %+v", got, want)
	}
	if crypto.decryptCalls != 0 {
		t.Fatalf("Decrypt() calls = %d, want 0", crypto.decryptCalls)
	}
}

func TestRecordService_GetText(t *testing.T) {
	recordID := "550e8400-e29b-41d4-a716-446655440000"
	stored := model.Record{
		ID:            recordID,
		UserID:        42,
		Type:          model.RecordTypeText,
		Title:         "Alice note",
		Revision:      1,
		CryptoVersion: recordcrypto.CryptoVersion,
		KeyID:         recordcrypto.DefaultKeyID,
		Nonce:         []byte("nonce"),
		Ciphertext:    []byte("ciphertext"),
	}
	payload := model.TextPayload{Text: "secret note", Metadata: "private metadata"}
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
			if encrypted.CryptoVersion != stored.CryptoVersion || encrypted.KeyID != stored.KeyID ||
				!bytes.Equal(encrypted.Nonce, stored.Nonce) || !bytes.Equal(encrypted.Ciphertext, stored.Ciphertext) {
				t.Fatalf("Decrypt() encrypted payload = %+v", encrypted)
			}
			wantAAD := "gopherkeeper:v1:user:42:record:550e8400-e29b-41d4-a716-446655440000:type:text"
			if string(aad) != wantAAD {
				t.Fatalf("Decrypt() AAD = %q, want %q", aad, wantAAD)
			}

			return plaintext, nil
		},
	}
	service := NewRecordService(records, crypto)

	got, err := service.GetText(context.Background(), 42, recordID)
	if err != nil {
		t.Fatalf("GetText() error = %v", err)
	}
	if got.Metadata.ID != recordID || got.Metadata.Title != "Alice note" {
		t.Fatalf("GetText() metadata = %+v", got.Metadata)
	}
	if got.Payload != payload {
		t.Fatalf("GetText() payload = %+v, want %+v", got.Payload, payload)
	}
}

func TestRecordService_GetTextRecordNotFound(t *testing.T) {
	records := &recordRepositoryStub{
		getFunc: func(context.Context, int64, string) (model.Record, error) {
			return model.Record{}, model.ErrRecordNotFound
		},
	}
	crypto := &recordPayloadCryptoStub{}
	service := NewRecordService(records, crypto)

	_, err := service.GetText(context.Background(), 42, "550e8400-e29b-41d4-a716-446655440000")
	if !errors.Is(err, model.ErrRecordNotFound) {
		t.Fatalf("GetText() error = %v, want ErrRecordNotFound", err)
	}
	if crypto.decryptCalls != 0 {
		t.Fatalf("Decrypt() calls = %d, want 0", crypto.decryptCalls)
	}
}

func TestRecordService_GetTextUnsupportedType(t *testing.T) {
	records := &recordRepositoryStub{
		getFunc: func(context.Context, int64, string) (model.Record, error) {
			return model.Record{
				ID:     "550e8400-e29b-41d4-a716-446655440000",
				UserID: 42,
				Type:   model.RecordTypeCredentials,
			}, nil
		},
	}
	crypto := &recordPayloadCryptoStub{}
	service := NewRecordService(records, crypto)

	_, err := service.GetText(context.Background(), 42, "550e8400-e29b-41d4-a716-446655440000")
	if !errors.Is(err, model.ErrRecordTypeUnsupported) {
		t.Fatalf("GetText() error = %v, want ErrRecordTypeUnsupported", err)
	}
	if crypto.decryptCalls != 0 {
		t.Fatalf("Decrypt() calls = %d, want 0", crypto.decryptCalls)
	}
}

type recordRepositoryStub struct {
	createFunc func(context.Context, model.Record) (model.Record, error)
	listFunc   func(context.Context, int64) ([]model.RecordMetadata, error)
	getFunc    func(context.Context, int64, string) (model.Record, error)

	createCalls int
	listCalls   int
	getCalls    int
}

func (s *recordRepositoryStub) Create(ctx context.Context, record model.Record) (model.Record, error) {
	s.createCalls++
	if s.createFunc == nil {
		return model.Record{}, errors.New("unexpected Create call")
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

func (s *recordRepositoryStub) Get(ctx context.Context, userID int64, recordID string) (model.Record, error) {
	s.getCalls++
	if s.getFunc == nil {
		return model.Record{}, errors.New("unexpected Get call")
	}

	return s.getFunc(ctx, userID, recordID)
}

type recordPayloadCryptoStub struct {
	encryptFunc func([]byte, []byte) (recordcrypto.EncryptedPayload, error)
	decryptFunc func(recordcrypto.EncryptedPayload, []byte) ([]byte, error)

	encryptCalls int
	decryptCalls int
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
