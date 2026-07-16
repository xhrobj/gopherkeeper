package cache

import (
	"context"
	"database/sql"
	"errors"
	"fmt"
	"net/url"
	"os"
	"path/filepath"
	"strings"

	// Регистрирует SQLite-драйвер для database/sql.
	_ "modernc.org/sqlite"
)

const busyTimeoutPragma = "PRAGMA busy_timeout = 5000"

var (
	// ErrInvalidLocation означает, что расположение SQLite-кеша не соответствует
	// ожидаемому layout локального кеша GophKeeper.
	ErrInvalidLocation = errors.New("invalid local cache location")

	// ErrUnsafeCachePath означает, что путь SQLite-файла является symlink
	// либо имеет неожиданный тип файла.
	ErrUnsafeCachePath = errors.New("unsafe local cache path")

	// ErrLocalCacheNotFound означает, что локальный кеш аккаунта ещё не создан.
	ErrLocalCacheNotFound = errors.New("local cache not found")
)

// Database представляет открытый SQLite-файл локального кеша.
type Database struct {
	db       *sql.DB
	location Location
	created  bool
}

// Open создаёт account directory, открывает SQLite-файл и настраивает одно
// logical connection с ограниченным busy timeout.
func Open(ctx context.Context, location Location) (*Database, error) {
	return openDatabase(ctx, location, true)
}

// OpenExisting открывает только уже существующий SQLite-кеш и ничего не создаёт.
func OpenExisting(ctx context.Context, location Location) (*Database, error) {
	return openDatabase(ctx, location, false)
}

func openDatabase(ctx context.Context, location Location, create bool) (*Database, error) {
	resolved, err := validateAndResolveLocation(location)
	if err != nil {
		return nil, err
	}

	created := false
	if create {
		if err := ensurePrivateDirectory(resolved.Directory); err != nil {
			return nil, err
		}

		created, err = ensurePrivateRegularFile(resolved.DatabaseFile)
		if err != nil {
			return nil, err
		}
	} else {
		if err := verifyExistingPrivateDirectory(resolved.Directory); err != nil {
			return nil, err
		}
		if err := verifyExistingPrivateRegularFile(resolved.DatabaseFile); err != nil {
			return nil, err
		}
	}

	db, err := sql.Open("sqlite", sqliteDataSourceName(resolved.DatabaseFile, create))
	if err != nil {
		cleanupNewDatabaseFile(created, resolved.DatabaseFile)
		return nil, fmt.Errorf("open local cache database: %w", err)
	}

	db.SetMaxOpenConns(1)
	db.SetMaxIdleConns(1)

	closeOnError := func(openErr error) (*Database, error) {
		_ = db.Close()
		cleanupNewDatabaseFile(created, resolved.DatabaseFile)
		return nil, openErr
	}

	if err := db.PingContext(ctx); err != nil {
		if !create && databaseFileMissing(resolved.DatabaseFile) {
			return closeOnError(ErrLocalCacheNotFound)
		}

		return closeOnError(fmt.Errorf("ping local cache database: %w", err))
	}

	if _, err := db.ExecContext(ctx, busyTimeoutPragma); err != nil {
		return closeOnError(fmt.Errorf("configure local cache busy timeout: %w", err))
	}

	if create {
		if err := initializeSchema(ctx, db); err != nil {
			return closeOnError(err)
		}
	} else if err := verifyExistingSchema(ctx, db); err != nil {
		return closeOnError(err)
	}

	if err := verifyPrivateRegularFile(resolved.DatabaseFile); err != nil {
		return closeOnError(err)
	}

	return &Database{db: db, location: resolved, created: created}, nil
}

func sqliteDataSourceName(path string, create bool) string {
	if create {
		return path
	}

	uriPath := filepath.ToSlash(path)
	if filepath.VolumeName(path) != "" && !strings.HasPrefix(uriPath, "/") {
		uriPath = "/" + uriPath
	}

	databaseURL := &url.URL{
		Scheme:   "file",
		Path:     uriPath,
		RawQuery: "mode=rw",
	}

	return databaseURL.String()
}

func databaseFileMissing(path string) bool {
	_, err := os.Lstat(path)
	return errors.Is(err, os.ErrNotExist)
}

// Close закрывает SQLite-кеш.
func (database *Database) Close() error {
	if database == nil || database.db == nil {
		return nil
	}

	if err := database.db.Close(); err != nil {
		return fmt.Errorf("close local cache database: %w", err)
	}

	return nil
}

// Location возвращает фактическое абсолютное расположение открытого кеша.
func (database *Database) Location() Location {
	if database == nil {
		return Location{}
	}

	return database.location
}

func validateAndResolveLocation(location Location) (Location, error) {
	if location.AccountID == "" || location.Directory == "" || location.DatabaseFile == "" {
		return Location{}, fmt.Errorf("%w: required path value is empty", ErrInvalidLocation)
	}

	cleanDirectory := filepath.Clean(location.Directory)
	cleanDatabaseFile := filepath.Clean(location.DatabaseFile)
	if cleanDatabaseFile != filepath.Join(cleanDirectory, databaseFileName) {
		return Location{}, fmt.Errorf("%w: database file is outside account directory", ErrInvalidLocation)
	}

	absoluteDirectory, err := filepath.Abs(cleanDirectory)
	if err != nil {
		return Location{}, fmt.Errorf("resolve local cache directory: %w", err)
	}
	absoluteDatabaseFile, err := filepath.Abs(cleanDatabaseFile)
	if err != nil {
		return Location{}, fmt.Errorf("resolve local cache database file: %w", err)
	}
	if absoluteDatabaseFile != filepath.Join(absoluteDirectory, databaseFileName) {
		return Location{}, fmt.Errorf("%w: resolved database file is outside account directory", ErrInvalidLocation)
	}

	return Location{
		AccountID:    location.AccountID,
		Directory:    absoluteDirectory,
		DatabaseFile: absoluteDatabaseFile,
	}, nil
}

func ensurePrivateDirectory(path string) error {
	if err := os.MkdirAll(path, 0o700); err != nil {
		return fmt.Errorf("create local cache account directory: %w", err)
	}
	if err := os.Chmod(path, 0o700); err != nil {
		return fmt.Errorf("set local cache directory permissions: %w", err)
	}

	return nil
}

func verifyExistingPrivateDirectory(path string) error {
	info, err := os.Lstat(path)
	if errors.Is(err, os.ErrNotExist) {
		return ErrLocalCacheNotFound
	}
	if err != nil {
		return fmt.Errorf("inspect local cache account directory: %w", err)
	}
	if info.Mode()&os.ModeSymlink != 0 || !info.IsDir() {
		return fmt.Errorf("%w: account path %q is not a regular directory", ErrUnsafeCachePath, path)
	}
	if err := os.Chmod(path, 0o700); err != nil {
		return fmt.Errorf("set local cache directory permissions: %w", err)
	}

	return nil
}

func ensurePrivateRegularFile(path string) (bool, error) {
	info, err := os.Lstat(path)
	if err == nil {
		if err := validateRegularFile(path, info); err != nil {
			return false, err
		}
		if err := os.Chmod(path, 0o600); err != nil {
			return false, fmt.Errorf("set local cache database permissions: %w", err)
		}

		return false, nil
	}
	if !errors.Is(err, os.ErrNotExist) {
		return false, fmt.Errorf("inspect local cache database: %w", err)
	}

	file, err := os.OpenFile(path, os.O_RDWR|os.O_CREATE|os.O_EXCL, 0o600)
	if err != nil {
		return false, fmt.Errorf("create local cache database: %w", err)
	}
	if err := file.Close(); err != nil {
		_ = os.Remove(path)
		return false, fmt.Errorf("close new local cache database: %w", err)
	}
	if err := verifyPrivateRegularFile(path); err != nil {
		_ = os.Remove(path)
		return false, err
	}

	return true, nil
}

func verifyExistingPrivateRegularFile(path string) error {
	info, err := os.Lstat(path)
	if errors.Is(err, os.ErrNotExist) {
		return ErrLocalCacheNotFound
	}
	if err != nil {
		return fmt.Errorf("inspect local cache database: %w", err)
	}
	if err := validateRegularFile(path, info); err != nil {
		return err
	}
	if err := os.Chmod(path, 0o600); err != nil {
		return fmt.Errorf("set local cache database permissions: %w", err)
	}

	return nil
}

func verifyPrivateRegularFile(path string) error {
	info, err := os.Lstat(path)
	if err != nil {
		return fmt.Errorf("inspect local cache database: %w", err)
	}
	if err := validateRegularFile(path, info); err != nil {
		return err
	}
	if err := os.Chmod(path, 0o600); err != nil {
		return fmt.Errorf("set local cache database permissions: %w", err)
	}

	return nil
}

func validateRegularFile(path string, info os.FileInfo) error {
	if info.Mode()&os.ModeSymlink != 0 {
		return fmt.Errorf("%w: database file %q is a symlink", ErrUnsafeCachePath, path)
	}
	if !info.Mode().IsRegular() {
		return fmt.Errorf("%w: database path %q is not a regular file", ErrUnsafeCachePath, path)
	}

	return nil
}

func cleanupNewDatabaseFile(created bool, path string) {
	if created {
		_ = os.Remove(path)
	}
}
