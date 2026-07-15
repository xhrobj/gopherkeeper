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
	// ErrLocalRecordNotFound сообщает, что запись отсутствует в локальном кеше.
	ErrLocalRecordNotFound = errors.New("local cache record not found")

	// ErrCorruptedCacheRecord сообщает, что зашифрованная строка локального кеша повреждена.
	ErrCorruptedCacheRecord = errors.New("corrupted local cache record")
)

type encryptedRecordRow struct {
	id            string
	revision      int64
	cryptoVersion int
	nonce         []byte
	ciphertext    []byte
}

type rowScanner interface {
	Scan(...any) error
}

// Upsert сохраняет полную запись в зашифрованном виде, заменяя локальную строку с тем же ID.
func (repository *Repository) Upsert(ctx context.Context, record model.Record) error {
	row, err := repository.prepareEncryptedRow(record)
	if err != nil {
		return err
	}

	transaction, err := repository.database.db.BeginTx(ctx, nil)
	if err != nil {
		return fmt.Errorf("begin local cache record transaction: %w", err)
	}
	defer func() {
		_ = transaction.Rollback()
	}()

	if err := applyEncryptedRows(ctx, transaction, []encryptedRecordRow{row}); err != nil {
		return err
	}
	if err := transaction.Commit(); err != nil {
		return fmt.Errorf("commit local cache record: %w", err)
	}

	return nil
}

// Get возвращает расшифрованную запись локального кеша по ID.
func (repository *Repository) Get(ctx context.Context, recordID string) (model.Record, error) {
	if err := model.ValidateRecordID(recordID); err != nil {
		return model.Record{}, err
	}

	const query = `
SELECT id, revision, crypto_version, nonce, ciphertext
FROM cached_records
WHERE id = ?`

	row, err := scanEncryptedRecord(repository.database.db.QueryRowContext(ctx, query, recordID))
	if errors.Is(err, sql.ErrNoRows) {
		return model.Record{}, ErrLocalRecordNotFound
	}
	if err != nil {
		return model.Record{}, fmt.Errorf("read local cache record: %w", err)
	}

	return repository.decodeRecordRow(row)
}

// Delete физически удаляет запись из локального кеша.
func (repository *Repository) Delete(ctx context.Context, recordID string) error {
	if err := model.ValidateRecordID(recordID); err != nil {
		return err
	}

	result, err := repository.database.db.ExecContext(
		ctx,
		"DELETE FROM cached_records WHERE id = ?",
		recordID,
	)
	if err != nil {
		return fmt.Errorf("delete local cache record: %w", err)
	}

	deleted, err := result.RowsAffected()
	if err != nil {
		return fmt.Errorf("inspect local cache record delete: %w", err)
	}
	if deleted == 0 {
		return ErrLocalRecordNotFound
	}

	return nil
}

func scanEncryptedRecord(scanner rowScanner) (encryptedRecordRow, error) {
	var row encryptedRecordRow
	err := scanner.Scan(
		&row.id,
		&row.revision,
		&row.cryptoVersion,
		&row.nonce,
		&row.ciphertext,
	)
	if err != nil {
		return encryptedRecordRow{}, err
	}

	return row, nil
}

func (repository *Repository) decodeRecordRow(row encryptedRecordRow) (model.Record, error) {
	if row.cryptoVersion != int(cachecrypto.CryptoVersion) {
		return model.Record{}, fmt.Errorf(
			"%w: record %s uses version %d",
			ErrUnsupportedCacheCryptoVersion,
			row.id,
			row.cryptoVersion,
		)
	}

	aad, err := cachecrypto.BuildRecordAAD(repository.account, row.id, row.revision)
	if err != nil {
		return model.Record{}, ErrCorruptedCacheRecord
	}

	encoded, err := repository.crypto.Decrypt(cachecrypto.EncryptedData{
		CryptoVersion: uint8(row.cryptoVersion),
		Nonce:         row.nonce,
		Ciphertext:    row.ciphertext,
	}, aad)
	if err != nil {
		return model.Record{}, ErrCorruptedCacheRecord
	}

	record, err := cachecrypto.DecodeRecord(encoded)
	if errors.Is(err, cachecrypto.ErrUnsupportedRecordFormatVersion) {
		return model.Record{}, err
	}
	if err != nil {
		return model.Record{}, ErrCorruptedCacheRecord
	}
	if record.Metadata.ID != row.id || record.Metadata.Revision != row.revision {
		return model.Record{}, ErrCorruptedCacheRecord
	}

	return record, nil
}
