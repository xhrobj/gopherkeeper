package postgres

import (
	"context"
	"errors"
	"fmt"

	"github.com/jackc/pgx/v5"
	"github.com/jackc/pgx/v5/pgxpool"
	"github.com/xhrobj/gopherkeeper/internal/model"
)

// RecordRepository является PostgreSQL-адаптером репозитория приватных записей.
type RecordRepository struct {
	pool *pgxpool.Pool
}

// NewRecordRepository создаёт PostgreSQL-адаптер репозитория приватных записей.
func NewRecordRepository(pool *pgxpool.Pool) *RecordRepository {
	return &RecordRepository{pool: pool}
}

// Create сохраняет encrypted record и возвращает состояние, зафиксированное PostgreSQL.
func (r *RecordRepository) Create(ctx context.Context, record model.Record) (model.Record, error) {
	var created model.Record
	var recordType string

	err := r.pool.QueryRow(
		ctx,
		`INSERT INTO gopherkeeper.records (
			id,
			user_id,
			type,
			title,
			crypto_version,
			key_id,
			nonce,
			ciphertext
		 ) VALUES ($1, $2, $3, $4, $5, $6, $7, $8)
		 RETURNING id::text, user_id, type, title, revision, created_at, updated_at,
			crypto_version::int, key_id, nonce, ciphertext`,
		record.ID,
		record.UserID,
		string(record.Type),
		record.Title,
		record.CryptoVersion,
		record.KeyID,
		record.Nonce,
		record.Ciphertext,
	).Scan(
		&created.ID,
		&created.UserID,
		&recordType,
		&created.Title,
		&created.Revision,
		&created.CreatedAt,
		&created.UpdatedAt,
		&created.CryptoVersion,
		&created.KeyID,
		&created.Nonce,
		&created.Ciphertext,
	)
	if err != nil {
		return model.Record{}, fmt.Errorf("create record: %w", err)
	}
	created.Type = model.RecordType(recordType)

	return created, nil
}

// Update изменяет encrypted record при совпадении владельца, идентификатора и ожидаемой ревизии.
func (r *RecordRepository) Update(
	ctx context.Context,
	record model.Record,
	expectedRevision int64,
) (model.Record, error) {
	if err := model.ValidateRecordRevision(expectedRevision); err != nil {
		return model.Record{}, fmt.Errorf("update record: %w", err)
	}

	var updated model.Record
	var recordType string

	err := r.pool.QueryRow(
		ctx,
		`UPDATE gopherkeeper.records
		 SET title = $4,
			 crypto_version = $5,
			 key_id = $6,
			 nonce = $7,
			 ciphertext = $8,
			 revision = revision + 1,
			 updated_at = CURRENT_TIMESTAMP
		 WHERE user_id = $1 AND id = $2 AND revision = $3
		 RETURNING id::text, user_id, type, title, revision, created_at, updated_at,
			crypto_version::int, key_id, nonce, ciphertext`,
		record.UserID,
		record.ID,
		expectedRevision,
		record.Title,
		record.CryptoVersion,
		record.KeyID,
		record.Nonce,
		record.Ciphertext,
	).Scan(
		&updated.ID,
		&updated.UserID,
		&recordType,
		&updated.Title,
		&updated.Revision,
		&updated.CreatedAt,
		&updated.UpdatedAt,
		&updated.CryptoVersion,
		&updated.KeyID,
		&updated.Nonce,
		&updated.Ciphertext,
	)
	if err == nil {
		updated.Type = model.RecordType(recordType)
		return updated, nil
	}

	if errors.Is(err, pgx.ErrNoRows) {
		return model.Record{}, r.updateDeleteMissError(ctx, record.UserID, record.ID, "update")
	}

	return model.Record{}, fmt.Errorf("update record: %w", err)
}

// Delete физически удаляет encrypted record при совпадении владельца, идентификатора и ожидаемой ревизии.
func (r *RecordRepository) Delete(ctx context.Context, userID int64, recordID string, expectedRevision int64) error {
	if err := model.ValidateRecordRevision(expectedRevision); err != nil {
		return fmt.Errorf("delete record: %w", err)
	}

	commandTag, err := r.pool.Exec(
		ctx,
		`DELETE FROM gopherkeeper.records
		 WHERE user_id = $1 AND id = $2 AND revision = $3`,
		userID,
		recordID,
		expectedRevision,
	)
	if err != nil {
		return fmt.Errorf("delete record: %w", err)
	}
	if commandTag.RowsAffected() == 1 {
		return nil
	}

	return r.updateDeleteMissError(ctx, userID, recordID, "delete")
}

func (r *RecordRepository) updateDeleteMissError(
	ctx context.Context,
	userID int64,
	recordID string,
	operation string,
) error {
	exists, err := r.exists(ctx, userID, recordID)
	if err != nil {
		return fmt.Errorf("%s record existence check: %w", operation, err)
	}
	if !exists {
		return fmt.Errorf("%s record: %w", operation, model.ErrRecordNotFound)
	}

	return fmt.Errorf("%s record: %w", operation, model.ErrRecordRevisionConflict)
}

func (r *RecordRepository) exists(ctx context.Context, userID int64, recordID string) (bool, error) {
	var exists bool
	if err := r.pool.QueryRow(
		ctx,
		`SELECT EXISTS(
			SELECT 1
			FROM gopherkeeper.records
			WHERE user_id = $1 AND id = $2
		)`,
		userID,
		recordID,
	).Scan(&exists); err != nil {
		return false, err
	}

	return exists, nil
}

// ListMetadata возвращает открытые поля записей пользователя без encrypted payload.
func (r *RecordRepository) ListMetadata(ctx context.Context, userID int64) ([]model.RecordMetadata, error) {
	rows, err := r.pool.Query(
		ctx,
		`SELECT id::text, type, title, revision, created_at, updated_at
		 FROM gopherkeeper.records
		 WHERE user_id = $1
		 ORDER BY updated_at DESC, id DESC`,
		userID,
	)
	if err != nil {
		return nil, fmt.Errorf("list record metadata: %w", err)
	}
	defer rows.Close()

	metadata := make([]model.RecordMetadata, 0)
	for rows.Next() {
		var item model.RecordMetadata
		var recordType string
		if err := rows.Scan(
			&item.ID,
			&recordType,
			&item.Title,
			&item.Revision,
			&item.CreatedAt,
			&item.UpdatedAt,
		); err != nil {
			return nil, fmt.Errorf("scan record metadata: %w", err)
		}

		item.Type = model.RecordType(recordType)
		metadata = append(metadata, item)
	}

	if err := rows.Err(); err != nil {
		return nil, fmt.Errorf("iterate record metadata: %w", err)
	}

	return metadata, nil
}

// Get возвращает encrypted record пользователя по идентификатору.
func (r *RecordRepository) Get(ctx context.Context, userID int64, recordID string) (model.Record, error) {
	var record model.Record
	var recordType string

	err := r.pool.QueryRow(
		ctx,
		`SELECT id::text, user_id, type, title, revision, created_at, updated_at,
			crypto_version::int, key_id, nonce, ciphertext
		 FROM gopherkeeper.records
		 WHERE user_id = $1 AND id = $2`,
		userID,
		recordID,
	).Scan(
		&record.ID,
		&record.UserID,
		&recordType,
		&record.Title,
		&record.Revision,
		&record.CreatedAt,
		&record.UpdatedAt,
		&record.CryptoVersion,
		&record.KeyID,
		&record.Nonce,
		&record.Ciphertext,
	)
	if err == nil {
		record.Type = model.RecordType(recordType)
		return record, nil
	}

	if errors.Is(err, pgx.ErrNoRows) {
		return model.Record{}, fmt.Errorf("get record: %w", model.ErrRecordNotFound)
	}

	return model.Record{}, fmt.Errorf("get record: %w", err)
}
