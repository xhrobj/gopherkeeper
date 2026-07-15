package cache

import (
	"context"
	"errors"
	"fmt"

	"github.com/xhrobj/gopherkeeper/internal/client/cachecrypto"
)

var (
	// ErrOpenEncryptedCache скрывает различие между неправильным password и повреждением локального кеша.
	ErrOpenEncryptedCache = errors.New("failed to open local cache: invalid password or corrupted cache")

	// ErrUnsupportedCacheMetadataVersion означает, что формат metadata локального кеша не поддерживается.
	ErrUnsupportedCacheMetadataVersion = errors.New("unsupported local cache metadata version")

	// ErrUnsupportedCacheCryptoVersion означает, что формат локального ciphertext не поддерживается.
	ErrUnsupportedCacheCryptoVersion = errors.New("unsupported local cache crypto version")
)

// Repository представляет открытый локальный кеш с проверенным ключом шифрования.
type Repository struct {
	database *Database
	crypto   *cachecrypto.Service
	account  string
}

// OpenRepository открывает кеш аккаунта, создаёт metadata при первом запуске и проверяет password при повторном.
func OpenRepository(ctx context.Context, location Location, password []byte) (*Repository, error) {
	return openRepository(ctx, location, password, Open)
}

// OpenExistingRepository открывает только уже существующий зашифрованный кеш аккаунта.
func OpenExistingRepository(ctx context.Context, location Location, password []byte) (*Repository, error) {
	return openRepository(ctx, location, password, OpenExisting)
}

type databaseOpener func(context.Context, Location) (*Database, error)

func openRepository(
	ctx context.Context,
	location Location,
	password []byte,
	openDatabase databaseOpener,
) (*Repository, error) {
	database, err := openDatabase(ctx, location)
	if err != nil {
		return nil, err
	}

	service, err := openCacheCrypto(ctx, database, password)
	if err != nil {
		closeErr := database.Close()
		cleanupNewDatabaseFile(database.created, database.location.DatabaseFile)
		return nil, errors.Join(err, closeErr)
	}

	return &Repository{
		database: database,
		crypto:   service,
		account:  database.location.AccountID,
	}, nil
}

// Close закрывает локальный SQLite-кеш.
func (repository *Repository) Close() error {
	if repository == nil || repository.database == nil {
		return nil
	}

	if err := repository.database.Close(); err != nil {
		return fmt.Errorf("close encrypted local cache: %w", err)
	}

	return nil
}

// Location возвращает фактическое расположение локального кеша.
func (repository *Repository) Location() Location {
	if repository == nil || repository.database == nil {
		return Location{}
	}

	return repository.database.Location()
}
