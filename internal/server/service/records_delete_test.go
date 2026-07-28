package service

import (
	"context"
	"errors"
	"testing"

	"github.com/xhrobj/gopherkeeper/internal/model"
)

func TestRecordService_Delete(t *testing.T) {
	recordID := "550e8400-e29b-41d4-a716-446655440000"
	records := &recordRepositoryStub{
		deleteFunc: func(_ context.Context, userID int64, gotRecordID string, expectedRevision int64) error {
			if userID != 42 || gotRecordID != recordID || expectedRevision != 1 {
				t.Fatalf("Delete() args = %d, %q, %d", userID, gotRecordID, expectedRevision)
			}

			return nil
		},
	}
	crypto := &recordPayloadCryptoStub{}
	service := NewRecordService(records, crypto)

	if err := service.Delete(context.Background(), DeleteRecordRequest{
		UserID:           42,
		RecordID:         recordID,
		ExpectedRevision: 1,
	}); err != nil {
		t.Fatalf("Delete() error = %v", err)
	}
	if records.getCalls != 0 || records.deleteCalls != 1 || crypto.encryptCalls != 0 || crypto.decryptCalls != 0 {
		t.Fatalf(
			"calls: Get=%d Delete=%d Encrypt=%d Decrypt=%d",
			records.getCalls,
			records.deleteCalls,
			crypto.encryptCalls,
			crypto.decryptCalls,
		)
	}
}

func TestRecordService_DeleteValidationError(t *testing.T) {
	recordID := "550e8400-e29b-41d4-a716-446655440000"
	tests := []struct {
		name    string
		request DeleteRecordRequest
		wantErr error
	}{
		{
			name: "invalid owner",
			request: DeleteRecordRequest{
				UserID:           0,
				RecordID:         recordID,
				ExpectedRevision: 1,
			},
			wantErr: errInvalidRecordOwner,
		},
		{
			name: "invalid record ID",
			request: DeleteRecordRequest{
				UserID:           42,
				RecordID:         "not-a-uuid",
				ExpectedRevision: 1,
			},
			wantErr: model.ErrInvalidRecordID,
		},
		{
			name: "invalid revision",
			request: DeleteRecordRequest{
				UserID:           42,
				RecordID:         recordID,
				ExpectedRevision: 0,
			},
			wantErr: model.ErrInvalidRecordRevision,
		},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			records := &recordRepositoryStub{}
			crypto := &recordPayloadCryptoStub{}
			service := NewRecordService(records, crypto)

			err := service.Delete(context.Background(), tt.request)
			if !errors.Is(err, tt.wantErr) {
				t.Fatalf("Delete() error = %v, want %v", err, tt.wantErr)
			}
			if records.deleteCalls != 0 || crypto.encryptCalls != 0 || crypto.decryptCalls != 0 {
				t.Fatalf(
					"calls: Delete=%d Encrypt=%d Decrypt=%d, want 0",
					records.deleteCalls,
					crypto.encryptCalls,
					crypto.decryptCalls,
				)
			}
		})
	}
}

func TestRecordService_DeleteRepositoryError(t *testing.T) {
	recordID := "550e8400-e29b-41d4-a716-446655440000"
	errRepository := errors.New("repository unavailable")
	tests := []struct {
		name      string
		deleteErr error
		wantErr   error
	}{
		{
			name:      "revision conflict",
			deleteErr: model.ErrRecordRevisionConflict,
			wantErr:   model.ErrRecordRevisionConflict,
		},
		{
			name:      "record not found",
			deleteErr: model.ErrRecordNotFound,
			wantErr:   model.ErrRecordNotFound,
		},
		{
			name:      "repository error",
			deleteErr: errRepository,
			wantErr:   errRepository,
		},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			records := &recordRepositoryStub{
				deleteFunc: func(context.Context, int64, string, int64) error {
					return tt.deleteErr
				},
			}
			crypto := &recordPayloadCryptoStub{}
			service := NewRecordService(records, crypto)

			err := service.Delete(context.Background(), DeleteRecordRequest{
				UserID:           42,
				RecordID:         recordID,
				ExpectedRevision: 1,
			})
			if !errors.Is(err, tt.wantErr) {
				t.Fatalf("Delete() error = %v, want %v", err, tt.wantErr)
			}
			if records.getCalls != 0 || records.deleteCalls != 1 || crypto.encryptCalls != 0 || crypto.decryptCalls != 0 {
				t.Fatalf(
					"calls: Get=%d Delete=%d Encrypt=%d Decrypt=%d",
					records.getCalls,
					records.deleteCalls,
					crypto.encryptCalls,
					crypto.decryptCalls,
				)
			}
		})
	}
}
