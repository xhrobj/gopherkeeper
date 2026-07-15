package cache

import (
	"errors"
	"testing"
	"time"

	"github.com/xhrobj/gopherkeeper/internal/model"
)

const (
	batchRecordIDOne   = "11111111-1111-4111-8111-111111111111"
	batchRecordIDTwo   = "22222222-2222-4222-8222-222222222222"
	invalidBatchID     = "not-a-uuid"
	batchRecordTitle   = "Private note"
	batchRecordContent = "private text"
)

func TestValidateCacheChanges(t *testing.T) {
	tests := []struct {
		name      string
		upserts   []model.Record
		deleteIDs []string
		wantErr   error
	}{
		{
			name:    "valid changes",
			upserts: []model.Record{batchTextRecord(batchRecordIDOne, 1, batchRecordContent)},
			deleteIDs: []string{
				batchRecordIDTwo,
			},
		},
		{
			name:      "invalid delete ID",
			deleteIDs: []string{invalidBatchID},
			wantErr:   model.ErrInvalidRecordID,
		},
		{
			name: "duplicate upsert",
			upserts: []model.Record{
				batchTextRecord(batchRecordIDOne, 1, batchRecordContent),
				batchTextRecord(batchRecordIDOne, 2, "updated text"),
			},
			wantErr: errDuplicateCacheChange,
		},
		{
			name: "duplicate delete",
			deleteIDs: []string{
				batchRecordIDOne,
				batchRecordIDOne,
			},
			wantErr: errDuplicateCacheChange,
		},
		{
			name:    "conflicting change",
			upserts: []model.Record{batchTextRecord(batchRecordIDOne, 1, batchRecordContent)},
			deleteIDs: []string{
				batchRecordIDOne,
			},
			wantErr: errConflictingCacheChange,
		},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			err := validateCacheChanges(tt.upserts, tt.deleteIDs)
			if !errors.Is(err, tt.wantErr) {
				t.Fatalf("validateCacheChanges() error = %v, want %v", err, tt.wantErr)
			}
		})
	}
}

func batchTextRecord(id string, revision int64, text string) model.Record {
	createdAt := time.Date(2026, time.July, 14, 12, 0, 0, 0, time.UTC)

	return model.Record{
		Metadata: model.RecordMetadata{
			ID:        id,
			Type:      model.RecordTypeText,
			Title:     batchRecordTitle,
			Revision:  revision,
			CreatedAt: createdAt,
			UpdatedAt: createdAt.Add(time.Duration(revision) * time.Minute),
		},
		Payload: &model.TextPayload{Text: text},
	}
}
