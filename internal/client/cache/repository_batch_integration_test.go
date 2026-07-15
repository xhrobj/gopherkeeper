//go:build integration

package cache

import (
	"context"
	"errors"
	"reflect"
	"strings"
	"testing"

	"github.com/xhrobj/gopherkeeper/internal/client/usecase"
	"github.com/xhrobj/gopherkeeper/internal/model"
)

const (
	batchKeepID       = "11111111-1111-4111-8111-111111111111"
	batchUpdateID     = "22222222-2222-4222-8222-222222222222"
	batchDeleteID     = "33333333-3333-4333-8333-333333333333"
	batchNewID        = "44444444-4444-4444-8444-444444444444"
	batchMissingID    = "55555555-5555-4555-8555-555555555555"
	batchFailureID    = "66666666-6666-4666-8666-666666666666"
	batchSuccessfulID = "77777777-7777-4777-8777-777777777777"
)

func TestIntegration_RepositoryApplyChanges(t *testing.T) {
	ctx := context.Background()
	repository := openBatchTestRepository(t, ctx)

	keep := batchTextRecord(batchKeepID, 1, "keep secret")
	update := batchTextRecord(batchUpdateID, 1, "old secret")
	deleted := batchTextRecord(batchDeleteID, 1, "delete secret")
	for _, record := range []model.Record{keep, update, deleted} {
		if err := repository.Upsert(ctx, record); err != nil {
			t.Fatalf("Upsert(%s) error = %v", record.Metadata.ID, err)
		}
	}

	update = batchTextRecord(batchUpdateID, 2, "updated secret")
	created := batchTextRecord(batchNewID, 1, "new secret")
	if err := repository.ApplyChanges(
		ctx,
		[]model.Record{created, update},
		[]string{batchDeleteID, batchMissingID},
	); err != nil {
		t.Fatalf("ApplyChanges() error = %v", err)
	}

	wantState := []usecase.RecordState{
		{ID: batchKeepID, Revision: 1},
		{ID: batchUpdateID, Revision: 2},
		{ID: batchNewID, Revision: 1},
	}
	assertBatchState(t, ctx, repository, wantState)
	assertBatchRecord(t, ctx, repository, update)
	assertBatchRecord(t, ctx, repository, created)
	if _, err := repository.Get(ctx, batchDeleteID); !errors.Is(err, ErrLocalRecordNotFound) {
		t.Fatalf("Get() deleted error = %v, want ErrLocalRecordNotFound", err)
	}

	if err := repository.ApplyChanges(ctx, nil, []string{batchDeleteID, batchMissingID}); err != nil {
		t.Fatalf("ApplyChanges() repeated delete error = %v", err)
	}
	assertBatchState(t, ctx, repository, wantState)
}

func TestIntegration_RepositoryApplyChangesRollsBack(t *testing.T) {
	ctx := context.Background()
	repository := openBatchTestRepository(t, ctx)

	existing := batchTextRecord(batchDeleteID, 1, "existing secret")
	if err := repository.Upsert(ctx, existing); err != nil {
		t.Fatalf("Upsert() error = %v", err)
	}
	before, err := repository.ListState(ctx)
	if err != nil {
		t.Fatalf("ListState() before error = %v", err)
	}

	const trigger = `
CREATE TRIGGER reject_batch_record
BEFORE INSERT ON cached_records
WHEN NEW.id = '66666666-6666-4666-8666-666666666666'
BEGIN
    SELECT RAISE(ABORT, 'forced batch failure');
END`
	if _, err := repository.database.db.ExecContext(ctx, trigger); err != nil {
		t.Fatalf("create failure trigger: %v", err)
	}

	err = repository.ApplyChanges(
		ctx,
		[]model.Record{
			batchTextRecord(batchSuccessfulID, 1, "first batch secret"),
			batchTextRecord(batchFailureID, 1, "failing batch secret"),
		},
		[]string{batchDeleteID},
	)
	if err == nil || !strings.Contains(err.Error(), "forced batch failure") {
		t.Fatalf("ApplyChanges() error = %v, want forced batch failure", err)
	}

	after, err := repository.ListState(ctx)
	if err != nil {
		t.Fatalf("ListState() after error = %v", err)
	}
	if !reflect.DeepEqual(after, before) {
		t.Fatalf("ListState() after rollback = %#v, want %#v", after, before)
	}
	assertBatchRecord(t, ctx, repository, existing)
	if _, err := repository.Get(ctx, batchSuccessfulID); !errors.Is(err, ErrLocalRecordNotFound) {
		t.Fatalf("Get() rolled back record error = %v, want ErrLocalRecordNotFound", err)
	}
}

func TestIntegration_RepositoryApplyChangesRollsBackDeletes(t *testing.T) {
	ctx := context.Background()
	repository := openBatchTestRepository(t, ctx)

	first := batchTextRecord(batchDeleteID, 1, "first delete secret")
	second := batchTextRecord(batchFailureID, 1, "second delete secret")
	for _, record := range []model.Record{first, second} {
		if err := repository.Upsert(ctx, record); err != nil {
			t.Fatalf("Upsert(%s) error = %v", record.Metadata.ID, err)
		}
	}
	before, err := repository.ListState(ctx)
	if err != nil {
		t.Fatalf("ListState() before error = %v", err)
	}

	const trigger = `
CREATE TRIGGER reject_batch_delete
BEFORE DELETE ON cached_records
WHEN OLD.id = '66666666-6666-4666-8666-666666666666'
BEGIN
    SELECT RAISE(ABORT, 'forced delete failure');
END`
	if _, err := repository.database.db.ExecContext(ctx, trigger); err != nil {
		t.Fatalf("create delete failure trigger: %v", err)
	}

	err = repository.ApplyChanges(ctx, nil, []string{batchDeleteID, batchFailureID})
	if err == nil || !strings.Contains(err.Error(), "forced delete failure") {
		t.Fatalf("ApplyChanges() error = %v, want forced delete failure", err)
	}

	after, err := repository.ListState(ctx)
	if err != nil {
		t.Fatalf("ListState() after error = %v", err)
	}
	if !reflect.DeepEqual(after, before) {
		t.Fatalf("ListState() after rollback = %#v, want %#v", after, before)
	}
	assertBatchRecord(t, ctx, repository, first)
	assertBatchRecord(t, ctx, repository, second)
}

func openBatchTestRepository(t *testing.T, ctx context.Context) *Repository {
	t.Helper()

	repository, err := OpenRepository(ctx, testLocation(t), []byte("cache-password"))
	if err != nil {
		t.Fatalf("OpenRepository() error = %v", err)
	}
	t.Cleanup(func() {
		if err := repository.Close(); err != nil {
			t.Errorf("Repository.Close() error = %v", err)
		}
	})

	return repository
}

func assertBatchState(
	t *testing.T,
	ctx context.Context,
	repository *Repository,
	want []usecase.RecordState,
) {
	t.Helper()

	got, err := repository.ListState(ctx)
	if err != nil {
		t.Fatalf("ListState() error = %v", err)
	}
	if !reflect.DeepEqual(got, want) {
		t.Fatalf("ListState() = %#v, want %#v", got, want)
	}
}

func assertBatchRecord(
	t *testing.T,
	ctx context.Context,
	repository *Repository,
	want model.Record,
) {
	t.Helper()

	got, err := repository.Get(ctx, want.Metadata.ID)
	if err != nil {
		t.Fatalf("Get(%s) error = %v", want.Metadata.ID, err)
	}
	if !reflect.DeepEqual(got, want) {
		t.Fatalf("Get(%s) = %#v, want %#v", want.Metadata.ID, got, want)
	}
}
