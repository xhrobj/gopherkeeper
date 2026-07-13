package cache

import (
	"context"
	"database/sql"
	"errors"
	"fmt"
)

const currentSchemaVersion = 1

var (
	// ErrUnsupportedSchemaVersion означает, что SQLite-файл использует версию
	// локальной схемы, которую текущий Клиент не поддерживает.
	ErrUnsupportedSchemaVersion = errors.New("unsupported local cache schema version")

	// ErrInvalidSchema означает, что локальная SQLite-схема повреждена либо не
	// соответствует объявленной версии.
	ErrInvalidSchema = errors.New("invalid local cache schema")
)

const createCacheMetadataTable = `
CREATE TABLE cache_metadata (
    singleton INTEGER NOT NULL PRIMARY KEY CHECK (singleton = 1),
    format_version INTEGER NOT NULL CHECK (format_version > 0),
    account_id TEXT NOT NULL CHECK (length(account_id) = 64),
    kdf_version INTEGER NOT NULL CHECK (kdf_version > 0),
    kdf_salt BLOB NOT NULL CHECK (length(kdf_salt) > 0),
    crypto_version INTEGER NOT NULL CHECK (crypto_version > 0),
    key_check_nonce BLOB NOT NULL CHECK (length(key_check_nonce) > 0),
    key_check_ciphertext BLOB NOT NULL CHECK (length(key_check_ciphertext) > 0),
    created_at TEXT NOT NULL CHECK (length(created_at) > 0)
) STRICT`

const createCachedRecordsTable = `
CREATE TABLE cached_records (
    id TEXT NOT NULL PRIMARY KEY CHECK (length(id) > 0),
    revision INTEGER NOT NULL CHECK (revision > 0),
    crypto_version INTEGER NOT NULL CHECK (crypto_version > 0),
    nonce BLOB NOT NULL CHECK (length(nonce) > 0),
    ciphertext BLOB NOT NULL CHECK (length(ciphertext) > 0)
) STRICT`

func initializeSchema(ctx context.Context, db *sql.DB) error {
	version, err := readSchemaVersion(ctx, db)
	if err != nil {
		return err
	}

	switch version {
	case 0:
		if err := createSchemaV1(ctx, db); err != nil {
			return err
		}
	case currentSchemaVersion:
		// Схема уже инициализирована.
	default:
		return fmt.Errorf(
			"%w: got %d, supported %d",
			ErrUnsupportedSchemaVersion,
			version,
			currentSchemaVersion,
		)
	}

	return verifySchemaV1(ctx, db)
}

func createSchemaV1(ctx context.Context, db *sql.DB) error {
	transaction, err := db.BeginTx(ctx, nil)
	if err != nil {
		return fmt.Errorf("begin local cache schema transaction: %w", err)
	}
	defer func() {
		_ = transaction.Rollback()
	}()

	version, err := readSchemaVersionFrom(ctx, transaction)
	if err != nil {
		return err
	}

	switch version {
	case currentSchemaVersion:
		// Другой процесс успел инициализировать схему.
	case 0:
		if _, err := transaction.ExecContext(ctx, createCacheMetadataTable); err != nil {
			return fmt.Errorf("%w: create cache_metadata: %v", ErrInvalidSchema, err)
		}
		if _, err := transaction.ExecContext(ctx, createCachedRecordsTable); err != nil {
			return fmt.Errorf("%w: create cached_records: %v", ErrInvalidSchema, err)
		}
		if _, err := transaction.ExecContext(ctx, "PRAGMA user_version = 1"); err != nil {
			return fmt.Errorf("set local cache schema version: %w", err)
		}
	default:
		return fmt.Errorf(
			"%w: got %d, supported %d",
			ErrUnsupportedSchemaVersion,
			version,
			currentSchemaVersion,
		)
	}

	if err := transaction.Commit(); err != nil {
		return fmt.Errorf("commit local cache schema: %w", err)
	}

	return nil
}

type queryRower interface {
	QueryRowContext(context.Context, string, ...any) *sql.Row
}

func readSchemaVersion(ctx context.Context, db *sql.DB) (int, error) {
	return readSchemaVersionFrom(ctx, db)
}

func readSchemaVersionFrom(ctx context.Context, queryer queryRower) (int, error) {
	var version int
	if err := queryer.QueryRowContext(ctx, "PRAGMA user_version").Scan(&version); err != nil {
		return 0, fmt.Errorf("read local cache schema version: %w", err)
	}

	return version, nil
}

func verifySchemaV1(ctx context.Context, db *sql.DB) error {
	version, err := readSchemaVersion(ctx, db)
	if err != nil {
		return err
	}
	if version != currentSchemaVersion {
		return fmt.Errorf(
			"%w: got %d, want %d",
			ErrInvalidSchema,
			version,
			currentSchemaVersion,
		)
	}

	const requiredTablesQuery = `
SELECT count(*)
FROM sqlite_schema
WHERE type = 'table'
  AND name IN ('cache_metadata', 'cached_records')`

	var tableCount int
	if err := db.QueryRowContext(ctx, requiredTablesQuery).Scan(&tableCount); err != nil {
		return fmt.Errorf("verify local cache tables: %w", err)
	}
	if tableCount != 2 {
		return fmt.Errorf("%w: required tables are missing", ErrInvalidSchema)
	}

	return nil
}
