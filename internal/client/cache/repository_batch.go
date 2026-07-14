package cache

import (
	"context"
	"database/sql"
	"errors"
	"fmt"

	"github.com/xhrobj/gopherkeeper/internal/client/cachecrypto"
	"github.com/xhrobj/gopherkeeper/internal/model"
)

var (
	errDuplicateCacheChange   = errors.New("duplicate local cache record change")
	errConflictingCacheChange = errors.New("conflicting local cache record change")
)

const upsertCachedRecordQuery = `
INSERT INTO cached_records (
    id,
    revision,
    crypto_version,
    nonce,
    ciphertext
) VALUES (?, ?, ?, ?, ?)
ON CONFLICT(id) DO UPDATE SET
    revision = excluded.revision,
    crypto_version = excluded.crypto_version,
    nonce = excluded.nonce,
    ciphertext = excluded.ciphertext`

func (repository *Repository) ApplyChanges(
	ctx context.Context,
	upserts []model.Record,
	deleteIDs []string,
) error {
	if err := validateCacheChanges(upserts, deleteIDs); err != nil {
		return err
	}

	rows, err := repository.prepareEncryptedRows(upserts)
	if err != nil {
		return err
	}
	if len(rows) == 0 && len(deleteIDs) == 0 {
		return nil
	}

	transaction, err := repository.database.db.BeginTx(ctx, nil)
	if err != nil {
		return fmt.Errorf("begin local cache batch transaction: %w", err)
	}
	defer func() {
		_ = transaction.Rollback()
	}()

	if err := applyEncryptedRows(ctx, transaction, rows); err != nil {
		return err
	}
	if err := applyCacheDeletes(ctx, transaction, deleteIDs); err != nil {
		return err
	}
	if err := transaction.Commit(); err != nil {
		return fmt.Errorf("commit local cache batch: %w", err)
	}

	return nil
}

func validateCacheChanges(upserts []model.Record, deleteIDs []string) error {
	upsertIDs := make(map[string]struct{}, len(upserts))
	for index, record := range upserts {
		if err := record.Validate(); err != nil {
			return fmt.Errorf("validate local cache upsert %d: %w", index, err)
		}
		if _, exists := upsertIDs[record.Metadata.ID]; exists {
			return fmt.Errorf("%w: upsert %s", errDuplicateCacheChange, record.Metadata.ID)
		}
		upsertIDs[record.Metadata.ID] = struct{}{}
	}

	deleteSet := make(map[string]struct{}, len(deleteIDs))
	for index, recordID := range deleteIDs {
		if err := model.ValidateRecordID(recordID); err != nil {
			return fmt.Errorf("validate local cache delete %d: %w", index, err)
		}
		if _, exists := deleteSet[recordID]; exists {
			return fmt.Errorf("%w: delete %s", errDuplicateCacheChange, recordID)
		}
		if _, exists := upsertIDs[recordID]; exists {
			return fmt.Errorf("%w: record %s", errConflictingCacheChange, recordID)
		}
		deleteSet[recordID] = struct{}{}
	}

	return nil
}

func (repository *Repository) prepareEncryptedRows(records []model.Record) ([]encryptedRecordRow, error) {
	rows := make([]encryptedRecordRow, 0, len(records))
	for index, record := range records {
		row, err := repository.prepareEncryptedRow(record)
		if err != nil {
			return nil, fmt.Errorf("prepare local cache upsert %d: %w", index, err)
		}
		rows = append(rows, row)
	}

	return rows, nil
}

func (repository *Repository) prepareEncryptedRow(record model.Record) (encryptedRecordRow, error) {
	encoded, err := cachecrypto.EncodeRecord(record)
	if err != nil {
		return encryptedRecordRow{}, err
	}

	aad, err := cachecrypto.BuildRecordAAD(
		repository.account,
		record.Metadata.ID,
		record.Metadata.Revision,
	)
	if err != nil {
		return encryptedRecordRow{}, err
	}

	encrypted, err := repository.crypto.Encrypt(encoded, aad)
	if err != nil {
		return encryptedRecordRow{}, fmt.Errorf("encrypt local cache record: %w", err)
	}

	return encryptedRecordRow{
		id:            record.Metadata.ID,
		revision:      record.Metadata.Revision,
		cryptoVersion: int(encrypted.CryptoVersion),
		nonce:         encrypted.Nonce,
		ciphertext:    encrypted.Ciphertext,
	}, nil
}

func applyEncryptedRows(
	ctx context.Context,
	transaction *sql.Tx,
	rows []encryptedRecordRow,
) error {
	for _, row := range rows {
		if _, err := transaction.ExecContext(
			ctx,
			upsertCachedRecordQuery,
			row.id,
			row.revision,
			row.cryptoVersion,
			row.nonce,
			row.ciphertext,
		); err != nil {
			return fmt.Errorf("upsert local cache record %s: %w", row.id, err)
		}
	}

	return nil
}

func applyCacheDeletes(ctx context.Context, transaction *sql.Tx, deleteIDs []string) error {
	for _, recordID := range deleteIDs {
		if _, err := transaction.ExecContext(
			ctx,
			"DELETE FROM cached_records WHERE id = ?",
			recordID,
		); err != nil {
			return fmt.Errorf("delete local cache record %s: %w", recordID, err)
		}
	}

	return nil
}
