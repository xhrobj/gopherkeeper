package cache

import (
	"context"
	"database/sql"
	"errors"
	"os"
	"path/filepath"
	"runtime"
	"testing"
)

func TestOpen(t *testing.T) {
	location := testLocation(t)

	first, err := Open(context.Background(), location)
	if err != nil {
		t.Fatalf("Open() error = %v", err)
	}
	if err := first.Close(); err != nil {
		t.Fatalf("Close() error = %v", err)
	}

	second, err := Open(context.Background(), location)
	if err != nil {
		t.Fatalf("Open() repeated error = %v", err)
	}
	t.Cleanup(func() {
		if err := second.Close(); err != nil {
			t.Errorf("second.Close() error = %v", err)
		}
	})

	version, err := readSchemaVersion(context.Background(), second.db)
	if err != nil {
		t.Fatalf("readSchemaVersion() error = %v", err)
	}
	if version != currentSchemaVersion {
		t.Fatalf("schema version = %d, want %d", version, currentSchemaVersion)
	}
}

func TestOpen_RejectsUnsupportedSchemaVersion(t *testing.T) {
	location := testLocation(t)
	if err := os.MkdirAll(location.Directory, 0o700); err != nil {
		t.Fatalf("MkdirAll() error = %v", err)
	}

	database, err := sql.Open("sqlite", location.DatabaseFile)
	if err != nil {
		t.Fatalf("sql.Open() error = %v", err)
	}
	if _, err := database.Exec("PRAGMA user_version = 69"); err != nil {
		t.Fatalf("set schema version: %v", err)
	}
	if err := database.Close(); err != nil {
		t.Fatalf("database.Close() error = %v", err)
	}

	if _, err := Open(context.Background(), location); !errors.Is(err, ErrUnsupportedSchemaVersion) {
		t.Fatalf("Open() error = %v, want ErrUnsupportedSchemaVersion", err)
	}
}

func TestOpen_RejectsUnsafeDatabaseFile(t *testing.T) {
	t.Run("directory", func(t *testing.T) {
		location := testLocation(t)
		if err := os.MkdirAll(location.DatabaseFile, 0o700); err != nil {
			t.Fatalf("MkdirAll() error = %v", err)
		}
		if _, err := Open(context.Background(), location); !errors.Is(err, ErrUnsafeCachePath) {
			t.Fatalf("Open() error = %v, want ErrUnsafeCachePath", err)
		}
	})

	t.Run("symlink", func(t *testing.T) {
		if runtime.GOOS == "windows" {
			t.Skip("symlink creation can require additional privileges on Windows")
		}

		location := testLocation(t)
		if err := os.MkdirAll(location.Directory, 0o700); err != nil {
			t.Fatalf("MkdirAll() error = %v", err)
		}
		target := filepath.Join(t.TempDir(), "target.db")
		if err := os.WriteFile(target, nil, 0o600); err != nil {
			t.Fatalf("WriteFile() error = %v", err)
		}
		if err := os.Symlink(target, location.DatabaseFile); err != nil {
			t.Fatalf("Symlink() error = %v", err)
		}
		if _, err := Open(context.Background(), location); !errors.Is(err, ErrUnsafeCachePath) {
			t.Fatalf("Open() error = %v, want ErrUnsafeCachePath", err)
		}
	})
}

func testLocation(t *testing.T) Location {
	t.Helper()

	location, err := ResolveLocation(t.TempDir(), "localhost:8080", "alice")
	if err != nil {
		t.Fatalf("ResolveLocation() error = %v", err)
	}
	return location
}

func TestOpenExisting(t *testing.T) {
	ctx := context.Background()
	location := testLocation(t)

	if _, err := OpenExisting(ctx, location); !errors.Is(err, ErrLocalCacheNotFound) {
		t.Fatalf("OpenExisting() missing cache error = %v, want ErrLocalCacheNotFound", err)
	}
	if _, err := os.Stat(location.Directory); !errors.Is(err, os.ErrNotExist) {
		t.Fatalf("cache directory stat error = %v, want os.ErrNotExist", err)
	}

	database, err := Open(ctx, location)
	if err != nil {
		t.Fatalf("Open() error = %v", err)
	}
	if err := database.Close(); err != nil {
		t.Fatalf("Close() error = %v", err)
	}

	database, err = OpenExisting(ctx, location)
	if err != nil {
		t.Fatalf("OpenExisting() error = %v", err)
	}
	if err := database.Close(); err != nil {
		t.Fatalf("Close() existing error = %v", err)
	}
}

func TestOpenExisting_RejectsMissingDatabaseWithoutCreatingIt(t *testing.T) {
	location := testLocation(t)
	if err := os.MkdirAll(location.Directory, 0o700); err != nil {
		t.Fatalf("MkdirAll() error = %v", err)
	}

	if _, err := OpenExisting(context.Background(), location); !errors.Is(err, ErrLocalCacheNotFound) {
		t.Fatalf("OpenExisting() error = %v, want ErrLocalCacheNotFound", err)
	}
	if _, err := os.Stat(location.DatabaseFile); !errors.Is(err, os.ErrNotExist) {
		t.Fatalf("database file stat error = %v, want os.ErrNotExist", err)
	}
}

func TestSQLiteDataSourceName_ExistingModeRejectsMissingDatabase(t *testing.T) {
	location := testLocation(t)
	if err := os.MkdirAll(location.Directory, 0o700); err != nil {
		t.Fatalf("MkdirAll() error = %v", err)
	}
	if err := os.WriteFile(location.DatabaseFile, nil, 0o600); err != nil {
		t.Fatalf("WriteFile() error = %v", err)
	}

	dataSourceName := sqliteDataSourceName(location.DatabaseFile, false)
	if err := os.Remove(location.DatabaseFile); err != nil {
		t.Fatalf("Remove() error = %v", err)
	}

	database, err := sql.Open("sqlite", dataSourceName)
	if err != nil {
		t.Fatalf("sql.Open() error = %v", err)
	}
	t.Cleanup(func() {
		if err := database.Close(); err != nil {
			t.Errorf("database.Close() error = %v", err)
		}
	})

	if err := database.PingContext(context.Background()); err == nil {
		t.Fatal("PingContext() error = nil, want missing database error")
	}
	if _, err := os.Stat(location.DatabaseFile); !errors.Is(err, os.ErrNotExist) {
		t.Fatalf("database file stat error = %v, want os.ErrNotExist", err)
	}
}
