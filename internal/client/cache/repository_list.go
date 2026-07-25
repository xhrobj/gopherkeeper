package cache

import (
	"context"
	"errors"
	"fmt"

	"github.com/xhrobj/gopherkeeper/internal/client/cachecrypto"
	"github.com/xhrobj/gopherkeeper/internal/client/usecase"
	"github.com/xhrobj/gopherkeeper/internal/model"
)

// ListState возвращает только открытые ID и revision без расшифрования записей.
func (repository *Repository) ListState(ctx context.Context) (states []usecase.RecordState, err error) {
	const query = `
SELECT id, revision
FROM cached_records
ORDER BY id`

	rows, err := repository.database.db.QueryContext(ctx, query)
	if err != nil {
		return nil, fmt.Errorf("list local cache record state: %w", err)
	}
	defer func() {
		err = errors.Join(err, rows.Close())
	}()

	states = make([]usecase.RecordState, 0)
	for rows.Next() {
		var state usecase.RecordState
		if err := rows.Scan(&state.ID, &state.Revision); err != nil {
			return nil, fmt.Errorf("scan local cache record state: %w", err)
		}
		if model.ValidateRecordID(state.ID) != nil || model.ValidateRecordRevision(state.Revision) != nil {
			return nil, ErrCorruptedCacheRecord
		}

		states = append(states, state)
	}
	if err := rows.Err(); err != nil {
		return nil, fmt.Errorf("iterate local cache record state: %w", err)
	}

	return states, nil
}

// ListMetadata возвращает metadata всех локальных записей после расшифрования,
// не разбирая приватные payload остальных записей.
func (repository *Repository) ListMetadata(ctx context.Context) (metadata []model.RecordMetadata, err error) {
	const query = `
SELECT id, revision, crypto_version, nonce, ciphertext
FROM cached_records
ORDER BY id`

	rows, err := repository.database.db.QueryContext(ctx, query)
	if err != nil {
		return nil, fmt.Errorf("list local cache record metadata: %w", err)
	}
	defer func() {
		err = errors.Join(err, rows.Close())
	}()

	metadata = make([]model.RecordMetadata, 0)
	for rows.Next() {
		row, err := scanEncryptedRecord(rows)
		if err != nil {
			return nil, fmt.Errorf("scan local cache record metadata: %w", err)
		}

		item, err := repository.decodeRecordMetadataRow(row)
		if err != nil {
			return nil, err
		}

		metadata = append(metadata, item)
	}
	if err := rows.Err(); err != nil {
		return nil, fmt.Errorf("iterate local cache record metadata: %w", err)
	}

	return metadata, nil
}

// List возвращает все локальные записи после расшифрования и строгой проверки формата.
func (repository *Repository) List(ctx context.Context) (records []model.Record, err error) {
	const query = `
SELECT id, revision, crypto_version, nonce, ciphertext
FROM cached_records
ORDER BY id`

	rows, err := repository.database.db.QueryContext(ctx, query)
	if err != nil {
		return nil, fmt.Errorf("list local cache records: %w", err)
	}
	defer func() {
		err = errors.Join(err, rows.Close())
	}()

	records = make([]model.Record, 0)
	for rows.Next() {
		row, err := scanEncryptedRecord(rows)
		if err != nil {
			return nil, fmt.Errorf("scan local cache record: %w", err)
		}

		record, err := repository.decodeRecordRow(row)
		if err != nil {
			return nil, err
		}

		records = append(records, record)
	}
	if err := rows.Err(); err != nil {
		return nil, fmt.Errorf("iterate local cache records: %w", err)
	}

	return records, nil
}

// ValidateRecords потоково проверяет, что все зашифрованные записи локального
// кеша расшифровываются и соответствуют текущему формату клиента.
func (repository *Repository) ValidateRecords(ctx context.Context) (err error) {
	const query = `
SELECT id, revision, crypto_version, nonce, ciphertext
FROM cached_records
ORDER BY id`

	rows, err := repository.database.db.QueryContext(ctx, query)
	if err != nil {
		return fmt.Errorf("validate local cache records: %w", err)
	}
	defer func() {
		err = errors.Join(err, rows.Close())
	}()

	for rows.Next() {
		row, scanErr := scanEncryptedRecord(rows)
		if scanErr != nil {
			return fmt.Errorf("scan local cache record for validation: %w", scanErr)
		}

		if _, decodeErr := repository.decodeRecordRow(row); decodeErr != nil {
			return markUnreadableRecordError(decodeErr)
		}
	}
	if err := rows.Err(); err != nil {
		return fmt.Errorf("iterate local cache records for validation: %w", err)
	}

	return nil
}

func markUnreadableRecordError(err error) error {
	if errors.Is(err, ErrCorruptedCacheRecord) ||
		errors.Is(err, ErrUnsupportedCacheCryptoVersion) ||
		errors.Is(err, cachecrypto.ErrUnsupportedRecordFormatVersion) {
		return errors.Join(usecase.ErrLocalCacheRecordsUnreadable, err)
	}

	return err
}
