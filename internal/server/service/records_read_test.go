package service

import (
	"bytes"
	"context"
	"encoding/json"
	"errors"
	"fmt"
	"reflect"
	"testing"
	"time"

	"github.com/xhrobj/gopherkeeper/internal/model"
	"github.com/xhrobj/gopherkeeper/internal/server/recordcrypto"
)

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

func TestRecordService_Get(t *testing.T) {
	recordID := "550e8400-e29b-41d4-a716-446655440000"
	createdAt := time.Date(2026, time.July, 11, 12, 0, 0, 0, time.UTC)
	updatedAt := time.Date(2026, time.July, 11, 12, 1, 0, 0, time.UTC)
	expiryMonth := 3
	expiryYear := 38
	tests := []struct {
		name       string
		recordType model.RecordType
		payload    model.RecordPayload
	}{
		{
			name:       "text record",
			recordType: model.RecordTypeText,
			payload:    &model.TextPayload{Text: "secret note", Metadata: "private metadata"},
		},
		{
			name:       "credentials record",
			recordType: model.RecordTypeCredentials,
			payload: &model.CredentialsPayload{
				Login:    "alice",
				Password: "correct-horse-battery-staple",
				URL:      "https://github.com",
				Metadata: "personal account",
			},
		},
		{
			name:       "card record",
			recordType: model.RecordTypeCard,
			payload: &model.CardPayload{
				Number:      "2013061420200619",
				Cardholder:  "Joel Miller",
				ExpiryMonth: &expiryMonth,
				ExpiryYear:  &expiryYear,
				CVV:         "014",
				Metadata:    "test card",
			},
		},
		{
			name:       "binary record",
			recordType: model.RecordTypeBinary,
			payload: &model.BinaryPayload{
				Filename: "backup.bin",
				Data:     []byte{0x00, 0x42, 0xfe, 0xff},
				Metadata: "private backup",
			},
		},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			plaintext, err := json.Marshal(tt.payload)
			if err != nil {
				t.Fatalf("json.Marshal() error = %v", err)
			}
			stored := model.EncryptedRecord{
				ID:            recordID,
				UserID:        42,
				Type:          tt.recordType,
				Title:         "Alice record",
				Revision:      model.RecordInitialRevision,
				CreatedAt:     createdAt,
				UpdatedAt:     updatedAt,
				CryptoVersion: recordcrypto.CryptoVersion,
				KeyID:         recordcrypto.DefaultKeyID,
				Nonce:         []byte("nonce"),
				Ciphertext:    []byte("ciphertext"),
			}
			records := &recordRepositoryStub{
				getFunc: func(_ context.Context, userID int64, gotRecordID string) (model.EncryptedRecord, error) {
					if userID != 42 || gotRecordID != recordID {
						t.Fatalf("Get() args = %d, %q", userID, gotRecordID)
					}

					return stored, nil
				},
			}
			crypto := &recordPayloadCryptoStub{
				decryptFunc: func(encrypted recordcrypto.EncryptedPayload, aad []byte) ([]byte, error) {
					if encrypted.CryptoVersion != stored.CryptoVersion || encrypted.KeyID != stored.KeyID ||
						!bytes.Equal(encrypted.Nonce, stored.Nonce) ||
						!bytes.Equal(encrypted.Ciphertext, stored.Ciphertext) {
						t.Fatalf("Decrypt() encrypted payload = %+v", encrypted)
					}
					wantAAD := fmt.Sprintf(
						"gopherkeeper:v1:user:42:record:%s:type:%s",
						recordID,
						tt.recordType,
					)
					if string(aad) != wantAAD {
						t.Fatalf("Decrypt() AAD = %q, want %q", aad, wantAAD)
					}

					return plaintext, nil
				},
			}
			service := NewRecordService(records, crypto)

			got, err := service.Get(context.Background(), 42, recordID)
			if err != nil {
				t.Fatalf("Get() error = %v", err)
			}
			if got.Metadata != stored.Metadata() {
				t.Errorf("Get() metadata = %+v, want %+v", got.Metadata, stored.Metadata())
			}
			if !reflect.DeepEqual(got.Payload, tt.payload) {
				t.Errorf("Get() payload = %#v, want %#v", got.Payload, tt.payload)
			}
		})
	}
}

func TestRecordService_GetMapsDecryptFailure(t *testing.T) {
	recordID := "550e8400-e29b-41d4-a716-446655440000"
	records := &recordRepositoryStub{
		getFunc: func(context.Context, int64, string) (model.EncryptedRecord, error) {
			return model.EncryptedRecord{
				ID:            recordID,
				UserID:        42,
				Type:          model.RecordTypeText,
				CryptoVersion: recordcrypto.CryptoVersion,
				KeyID:         recordcrypto.DefaultKeyID,
			}, nil
		},
	}
	crypto := &recordPayloadCryptoStub{
		decryptFunc: func(recordcrypto.EncryptedPayload, []byte) ([]byte, error) {
			return nil, recordcrypto.ErrDecryptPayload
		},
	}
	service := NewRecordService(records, crypto)

	_, err := service.Get(context.Background(), 42, recordID)
	if !errors.Is(err, model.ErrRecordDecryptionFailed) {
		t.Fatalf("Get() error = %v, want ErrRecordDecryptionFailed", err)
	}
	if !errors.Is(err, recordcrypto.ErrDecryptPayload) {
		t.Fatalf("Get() error = %v, want ErrDecryptPayload in chain", err)
	}
}

func TestRecordService_GetRejectsInvalidDecryptedPayload(t *testing.T) {
	recordID := "550e8400-e29b-41d4-a716-446655440000"
	records := &recordRepositoryStub{
		getFunc: func(context.Context, int64, string) (model.EncryptedRecord, error) {
			return model.EncryptedRecord{
				ID:            recordID,
				UserID:        42,
				Type:          model.RecordTypeCredentials,
				CryptoVersion: recordcrypto.CryptoVersion,
				KeyID:         recordcrypto.DefaultKeyID,
			}, nil
		},
	}
	crypto := &recordPayloadCryptoStub{
		decryptFunc: func(recordcrypto.EncryptedPayload, []byte) ([]byte, error) {
			return []byte(`{"login":"alice"}`), nil
		},
	}
	service := NewRecordService(records, crypto)

	_, err := service.Get(context.Background(), 42, recordID)
	if !errors.Is(err, model.ErrInvalidCredentialsPayload) {
		t.Fatalf("Get() error = %v, want ErrInvalidCredentialsPayload", err)
	}
	if records.getCalls != 1 || crypto.decryptCalls != 1 {
		t.Fatalf("calls: Get=%d Decrypt=%d, want 1", records.getCalls, crypto.decryptCalls)
	}
}

func TestRecordService_GetRejectsUnsupportedType(t *testing.T) {
	records := &recordRepositoryStub{
		getFunc: func(context.Context, int64, string) (model.EncryptedRecord, error) {
			return model.EncryptedRecord{
				ID:     "550e8400-e29b-41d4-a716-446655440000",
				UserID: 42,
				Type:   model.RecordType("otp"),
			}, nil
		},
	}
	crypto := &recordPayloadCryptoStub{}
	service := NewRecordService(records, crypto)

	_, err := service.Get(context.Background(), 42, "550e8400-e29b-41d4-a716-446655440000")
	if !errors.Is(err, errInvalidStoredRecord) {
		t.Fatalf("Get() error = %v, want errInvalidStoredRecord", err)
	}
	if crypto.decryptCalls != 0 {
		t.Fatalf("Decrypt() calls = %d, want 0", crypto.decryptCalls)
	}
}

func TestRecordService_GetRepositoryError(t *testing.T) {
	records := &recordRepositoryStub{
		getFunc: func(context.Context, int64, string) (model.EncryptedRecord, error) {
			return model.EncryptedRecord{}, model.ErrRecordNotFound
		},
	}
	crypto := &recordPayloadCryptoStub{}
	service := NewRecordService(records, crypto)

	_, err := service.Get(context.Background(), 42, "550e8400-e29b-41d4-a716-446655440000")
	if !errors.Is(err, model.ErrRecordNotFound) {
		t.Fatalf("Get() error = %v, want ErrRecordNotFound", err)
	}
	if crypto.decryptCalls != 0 {
		t.Fatalf("Decrypt() calls = %d, want 0", crypto.decryptCalls)
	}
}
