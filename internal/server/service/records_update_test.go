package service

import (
	"context"
	"errors"
	"testing"

	"github.com/xhrobj/gopherkeeper/internal/model"
	"github.com/xhrobj/gopherkeeper/internal/server/recordcrypto"
)

func TestRecordService_Update(t *testing.T) {
	fixture := newUpdateRecordFixture()
	crypto := newUpdateCryptoStub(t, fixture)
	records := newUpdateRepositoryStub(t, fixture)
	service := NewRecordService(records, crypto)

	updated, err := service.Update(context.Background(), UpdateRecordRequest{
		UserID:           42,
		RecordID:         fixture.recordID,
		ExpectedRevision: 1,
		Title:            "Updated Alice note",
		Payload:          &fixture.payload,
	})
	if err != nil {
		t.Fatalf("Update() error = %v", err)
	}

	assertUpdatedRecord(t, updated, fixture)
	assertRecordServiceCalls(t, records, crypto, recordServiceCallCounts{
		get:     1,
		update:  1,
		encrypt: 1,
	})
}

func TestRecordService_UpdateValidationError(t *testing.T) {
	recordID := "550e8400-e29b-41d4-a716-446655440000"
	tests := []struct {
		name    string
		request UpdateRecordRequest
		wantErr error
	}{
		{
			name: "invalid owner",
			request: UpdateRecordRequest{
				UserID:           0,
				RecordID:         recordID,
				ExpectedRevision: 1,
				Title:            "Alice note",
				Payload:          &model.TextPayload{Text: "secret note"},
			},
			wantErr: errInvalidRecordOwner,
		},
		{
			name: "invalid record ID",
			request: UpdateRecordRequest{
				UserID:           42,
				RecordID:         "not-a-uuid",
				ExpectedRevision: 1,
				Title:            "Alice note",
				Payload:          &model.TextPayload{Text: "secret note"},
			},
			wantErr: model.ErrInvalidRecordID,
		},
		{
			name: "invalid revision",
			request: UpdateRecordRequest{
				UserID:           42,
				RecordID:         recordID,
				ExpectedRevision: 0,
				Title:            "Alice note",
				Payload:          &model.TextPayload{Text: "secret note"},
			},
			wantErr: model.ErrInvalidRecordRevision,
		},
		{
			name: "invalid title",
			request: UpdateRecordRequest{
				UserID:           42,
				RecordID:         recordID,
				ExpectedRevision: 1,
				Title:            "   ",
				Payload:          &model.TextPayload{Text: "secret note"},
			},
			wantErr: model.ErrInvalidRecordTitle,
		},
		{
			name: "control character in title",
			request: UpdateRecordRequest{
				UserID:           42,
				RecordID:         recordID,
				ExpectedRevision: 1,
				Title:            "Alice\tnote",
				Payload:          &model.TextPayload{Text: "secret note"},
			},
			wantErr: model.ErrInvalidRecordTitle,
		},
		{
			name: "invalid payload",
			request: UpdateRecordRequest{
				UserID:           42,
				RecordID:         recordID,
				ExpectedRevision: 1,
				Title:            "Alice note",
				Payload:          &model.TextPayload{},
			},
			wantErr: model.ErrInvalidTextPayload,
		},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			crypto := &recordPayloadCryptoStub{}
			records := &recordRepositoryStub{}
			service := NewRecordService(records, crypto)

			_, err := service.Update(context.Background(), tt.request)
			if !errors.Is(err, tt.wantErr) {
				t.Fatalf("Update() error = %v, want %v", err, tt.wantErr)
			}
			if records.getCalls != 0 || records.updateCalls != 0 || crypto.encryptCalls != 0 || crypto.decryptCalls != 0 {
				t.Fatalf(
					"calls: Get=%d Update=%d Encrypt=%d Decrypt=%d, want 0",
					records.getCalls,
					records.updateCalls,
					crypto.encryptCalls,
					crypto.decryptCalls,
				)
			}
		})
	}
}

func TestRecordService_UpdateCurrentRecordErrors(t *testing.T) {
	recordID := "550e8400-e29b-41d4-a716-446655440000"
	tests := []struct {
		name    string
		current model.EncryptedRecord
		getErr  error
		wantErr error
	}{
		{
			name:    "record not found",
			getErr:  model.ErrRecordNotFound,
			wantErr: model.ErrRecordNotFound,
		},
		{
			name: "unsupported type",
			current: model.EncryptedRecord{
				ID:       recordID,
				UserID:   42,
				Type:     model.RecordTypeCredentials,
				Revision: 1,
			},
			wantErr: model.ErrRecordTypeUnsupported,
		},
		{
			name: "stale revision",
			current: model.EncryptedRecord{
				ID:       recordID,
				UserID:   42,
				Type:     model.RecordTypeText,
				Revision: 2,
			},
			wantErr: model.ErrRecordRevisionConflict,
		},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			records := &recordRepositoryStub{
				getFunc: func(context.Context, int64, string) (model.EncryptedRecord, error) {
					return tt.current, tt.getErr
				},
			}
			crypto := &recordPayloadCryptoStub{}
			service := NewRecordService(records, crypto)

			_, err := service.Update(context.Background(), UpdateRecordRequest{
				UserID:           42,
				RecordID:         recordID,
				ExpectedRevision: 1,
				Title:            "Alice note",
				Payload:          &model.TextPayload{Text: "secret note"},
			})
			if !errors.Is(err, tt.wantErr) {
				t.Fatalf("Update() error = %v, want %v", err, tt.wantErr)
			}
			if records.updateCalls != 0 || crypto.encryptCalls != 0 || crypto.decryptCalls != 0 {
				t.Fatalf(
					"calls: Update=%d Encrypt=%d Decrypt=%d, want 0",
					records.updateCalls,
					crypto.encryptCalls,
					crypto.decryptCalls,
				)
			}
		})
	}
}

func TestRecordService_UpdateCryptoError(t *testing.T) {
	recordID := "550e8400-e29b-41d4-a716-446655440000"
	errCrypto := errors.New("crypto unavailable")
	records := &recordRepositoryStub{
		getFunc: func(context.Context, int64, string) (model.EncryptedRecord, error) {
			return model.EncryptedRecord{
				ID:       recordID,
				UserID:   42,
				Type:     model.RecordTypeText,
				Revision: 1,
			}, nil
		},
	}
	crypto := &recordPayloadCryptoStub{
		encryptFunc: func([]byte, []byte) (recordcrypto.EncryptedPayload, error) {
			return recordcrypto.EncryptedPayload{}, errCrypto
		},
	}
	service := NewRecordService(records, crypto)

	_, err := service.Update(context.Background(), UpdateRecordRequest{
		UserID:           42,
		RecordID:         recordID,
		ExpectedRevision: 1,
		Title:            "Alice note",
		Payload:          &model.TextPayload{Text: "secret note"},
	})
	if !errors.Is(err, errCrypto) {
		t.Fatalf("Update() error = %v, want crypto error", err)
	}
	if records.updateCalls != 0 || crypto.encryptCalls != 1 || crypto.decryptCalls != 0 {
		t.Fatalf(
			"calls: Update=%d Encrypt=%d Decrypt=%d",
			records.updateCalls,
			crypto.encryptCalls,
			crypto.decryptCalls,
		)
	}
}

func TestRecordService_UpdateRepositoryError(t *testing.T) {
	recordID := "550e8400-e29b-41d4-a716-446655440000"
	errRepository := errors.New("repository unavailable")
	tests := []struct {
		name      string
		updateErr error
		wantErr   error
	}{
		{
			name:      "revision conflict",
			updateErr: model.ErrRecordRevisionConflict,
			wantErr:   model.ErrRecordRevisionConflict,
		},
		{
			name:      "record not found",
			updateErr: model.ErrRecordNotFound,
			wantErr:   model.ErrRecordNotFound,
		},
		{
			name:      "repository error",
			updateErr: errRepository,
			wantErr:   errRepository,
		},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			records := &recordRepositoryStub{
				getFunc: func(context.Context, int64, string) (model.EncryptedRecord, error) {
					return model.EncryptedRecord{
						ID:       recordID,
						UserID:   42,
						Type:     model.RecordTypeText,
						Revision: 1,
					}, nil
				},
				updateFunc: func(context.Context, model.EncryptedRecord, int64) (model.EncryptedRecord, error) {
					return model.EncryptedRecord{}, tt.updateErr
				},
			}
			crypto := &recordPayloadCryptoStub{
				encryptFunc: func([]byte, []byte) (recordcrypto.EncryptedPayload, error) {
					return recordcrypto.EncryptedPayload{
						CryptoVersion: recordcrypto.CryptoVersion,
						KeyID:         recordcrypto.DefaultKeyID,
						Nonce:         []byte("nonce"),
						Ciphertext:    []byte("ciphertext"),
					}, nil
				},
			}
			service := NewRecordService(records, crypto)

			_, err := service.Update(context.Background(), UpdateRecordRequest{
				UserID:           42,
				RecordID:         recordID,
				ExpectedRevision: 1,
				Title:            "Alice note",
				Payload:          &model.TextPayload{Text: "secret note"},
			})
			if !errors.Is(err, tt.wantErr) {
				t.Fatalf("Update() error = %v, want %v", err, tt.wantErr)
			}
			if records.updateCalls != 1 || crypto.encryptCalls != 1 || crypto.decryptCalls != 0 {
				t.Fatalf(
					"calls: Update=%d Encrypt=%d Decrypt=%d",
					records.updateCalls,
					crypto.encryptCalls,
					crypto.decryptCalls,
				)
			}
		})
	}
}
