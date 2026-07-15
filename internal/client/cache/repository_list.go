package cache

import (
	"context"
	"errors"
	"fmt"

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
