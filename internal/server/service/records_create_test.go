package service

import (
	"bytes"
	"context"
	"encoding/json"
	"errors"
	"fmt"
	"reflect"
	"strings"
	"testing"
	"time"

	"github.com/xhrobj/gopherkeeper/internal/model"
	"github.com/xhrobj/gopherkeeper/internal/server/recordcrypto"
)

func TestRecordService_Create(t *testing.T) {
	createdAt := time.Date(2026, time.July, 11, 12, 0, 0, 0, time.UTC)
	updatedAt := time.Date(2026, time.July, 11, 12, 1, 0, 0, time.UTC)
	expiryMonth := 3
	expiryYear := 38
	tests := []struct {
		name    string
		title   string
		payload model.RecordPayload
	}{
		{
			name:  "text record",
			title: "Alice note",
			payload: &model.TextPayload{
				Text:     "secret note",
				Metadata: "private metadata",
			},
		},
		{
			name:  "credentials record",
			title: "Alice GitHub",
			payload: &model.CredentialsPayload{
				Login:    "alice",
				Password: "correct-horse-battery-staple",
				URL:      "https://github.com",
				Metadata: "personal account",
			},
		},
		{
			name:  "card record",
			title: "Joel's card",
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
			name:  "binary record",
			title: "Alice backup",
			payload: &model.BinaryPayload{
				Filename: "backup.bin",
				Data:     []byte{0x00, 0x42, 0xfe, 0xff},
				Metadata: "private backup",
			},
		},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			encrypted := recordcrypto.EncryptedPayload{
				CryptoVersion: recordcrypto.CryptoVersion,
				KeyID:         recordcrypto.DefaultKeyID,
				Nonce:         []byte("nonce"),
				Ciphertext:    []byte("ciphertext"),
			}
			crypto := &recordPayloadCryptoStub{
				encryptFunc: func(plaintext, aad []byte) (recordcrypto.EncryptedPayload, error) {
					decoded, err := model.NewRecordPayload(tt.payload.RecordType())
					if err != nil {
						t.Fatalf("NewRecordPayload() error = %v", err)
					}
					if err := json.Unmarshal(plaintext, decoded); err != nil {
						t.Fatalf("Encrypt() plaintext JSON error = %v", err)
					}
					if !reflect.DeepEqual(decoded, tt.payload) {
						t.Fatalf("Encrypt() payload = %#v, want %#v", decoded, tt.payload)
					}
					wantAADPart := fmt.Sprintf(":type:%s", tt.payload.RecordType())
					if !strings.Contains(string(aad), "gopherkeeper:v1:user:42:record:") ||
						!strings.Contains(string(aad), wantAADPart) {
						t.Fatalf("Encrypt() AAD = %q", aad)
					}

					return encrypted, nil
				},
			}
			records := &recordRepositoryStub{
				createFunc: func(_ context.Context, record model.EncryptedRecord) (model.EncryptedRecord, error) {
					if err := model.ValidateRecordID(record.ID); err != nil {
						t.Fatalf("Create() record ID is invalid: %v", err)
					}
					if record.UserID != 42 || record.Type != tt.payload.RecordType() || record.Title != tt.title {
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

			created, err := service.Create(context.Background(), CreateRecordRequest{
				UserID:  42,
				Title:   tt.title,
				Payload: tt.payload,
			})
			if err != nil {
				t.Fatalf("Create() error = %v", err)
			}
			if created.Metadata.Title != tt.title || created.Metadata.CreatedAt != createdAt ||
				created.Metadata.UpdatedAt != updatedAt {
				t.Fatalf("Create() metadata = %+v", created.Metadata)
			}
			if !reflect.DeepEqual(created.Payload, tt.payload) {
				t.Fatalf("Create() payload = %#v, want %#v", created.Payload, tt.payload)
			}
			if crypto.encryptCalls != 1 || records.createCalls != 1 {
				t.Fatalf("calls: Encrypt=%d Create=%d", crypto.encryptCalls, records.createCalls)
			}
		})
	}
}

func TestRecordService_CreateValidationError(t *testing.T) {
	tests := []struct {
		name    string
		request CreateRecordRequest
		wantErr error
	}{
		{
			name: "invalid owner",
			request: CreateRecordRequest{
				Title:   "Alice note",
				Payload: &model.TextPayload{Text: "secret note"},
			},
			wantErr: errInvalidRecordOwner,
		},
		{
			name: "invalid title",
			request: CreateRecordRequest{
				UserID:  42,
				Title:   "   ",
				Payload: &model.TextPayload{Text: "secret note"},
			},
			wantErr: model.ErrInvalidRecordTitle,
		},
		{
			name: "invalid payload",
			request: CreateRecordRequest{
				UserID:  42,
				Title:   "Alice note",
				Payload: &model.TextPayload{},
			},
			wantErr: model.ErrInvalidTextPayload,
		},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			crypto := &recordPayloadCryptoStub{}
			records := &recordRepositoryStub{}
			service := NewRecordService(records, crypto)

			_, err := service.Create(context.Background(), tt.request)
			if !errors.Is(err, tt.wantErr) {
				t.Fatalf("Create() error = %v, want %v", err, tt.wantErr)
			}
			if crypto.encryptCalls != 0 || records.createCalls != 0 {
				t.Fatalf("calls: Encrypt=%d Create=%d, want 0", crypto.encryptCalls, records.createCalls)
			}
		})
	}
}

func TestRecordService_RejectsNilPayload(t *testing.T) {
	var textPayload *model.TextPayload
	var credentialsPayload *model.CredentialsPayload
	var cardPayload *model.CardPayload
	var binaryPayload *model.BinaryPayload

	tests := []struct {
		name    string
		payload model.RecordPayload
		wantErr error
	}{
		{name: "text", payload: textPayload, wantErr: model.ErrInvalidTextPayload},
		{name: "credentials", payload: credentialsPayload, wantErr: model.ErrInvalidCredentialsPayload},
		{name: "card", payload: cardPayload, wantErr: model.ErrInvalidCardPayload},
		{name: "binary", payload: binaryPayload, wantErr: model.ErrInvalidBinaryPayload},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			records := &recordRepositoryStub{}
			crypto := &recordPayloadCryptoStub{}
			service := NewRecordService(records, crypto)

			_, err := service.Create(context.Background(), CreateRecordRequest{
				UserID:  42,
				Title:   "private record",
				Payload: tt.payload,
			})
			if !errors.Is(err, tt.wantErr) {
				t.Fatalf("Create() error = %v, want %v", err, tt.wantErr)
			}

			_, err = service.Update(context.Background(), UpdateRecordRequest{
				UserID:           42,
				RecordID:         "550e8400-e29b-41d4-a716-446655440000",
				ExpectedRevision: 1,
				Title:            "private record",
				Payload:          tt.payload,
			})
			if !errors.Is(err, tt.wantErr) {
				t.Fatalf("Update() error = %v, want %v", err, tt.wantErr)
			}

			if records.createCalls != 0 || records.getCalls != 0 || records.updateCalls != 0 ||
				crypto.encryptCalls != 0 || crypto.decryptCalls != 0 {
				t.Fatalf(
					"calls: Create=%d Get=%d Update=%d Encrypt=%d Decrypt=%d, want 0",
					records.createCalls,
					records.getCalls,
					records.updateCalls,
					crypto.encryptCalls,
					crypto.decryptCalls,
				)
			}
		})
	}
}
