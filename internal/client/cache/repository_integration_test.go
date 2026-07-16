//go:build integration

package cache

import (
	"context"
	"errors"
	"testing"

	"github.com/xhrobj/gopherkeeper/internal/client/cachecrypto"
)

func TestIntegration_OpenRepository(t *testing.T) {
	ctx := context.Background()
	location := testLocation(t)
	password := []byte("correct-horse-battery-staple")

	repository, err := OpenRepository(ctx, location, password)
	if err != nil {
		t.Fatalf("OpenRepository() first error = %v", err)
	}

	var kdfVersion int
	var salt []byte
	if err := repository.database.db.QueryRowContext(
		ctx,
		"SELECT kdf_version, kdf_salt FROM cache_metadata WHERE singleton = 1",
	).Scan(&kdfVersion, &salt); err != nil {
		t.Fatalf("read cache metadata: %v", err)
	}
	if kdfVersion != int(cachecrypto.KDFVersion) {
		t.Fatalf("KDF version = %d, want %d", kdfVersion, cachecrypto.KDFVersion)
	}
	if len(salt) != 16 {
		t.Fatalf("salt length = %d, want 16", len(salt))
	}
	if err := repository.Close(); err != nil {
		t.Fatalf("Close() first error = %v", err)
	}

	repository, err = OpenRepository(ctx, location, password)
	if err != nil {
		t.Fatalf("OpenRepository() repeated error = %v", err)
	}
	if err := repository.Close(); err != nil {
		t.Fatalf("Close() repeated error = %v", err)
	}

	if _, err := OpenRepository(ctx, location, []byte("wrong-password")); !errors.Is(err, ErrOpenEncryptedCache) {
		t.Fatalf("OpenRepository() wrong password error = %v, want ErrOpenEncryptedCache", err)
	}
}

func TestIntegration_OpenRepository_RejectsMissingMetadataInExistingCache(t *testing.T) {
	ctx := context.Background()
	location := testLocation(t)

	repository, err := OpenRepository(ctx, location, []byte("initial-password"))
	if err != nil {
		t.Fatalf("OpenRepository() error = %v", err)
	}

	if _, err := repository.database.db.ExecContext(
		ctx,
		"DELETE FROM cache_metadata WHERE singleton = 1",
	); err != nil {
		t.Fatalf("delete cache metadata: %v", err)
	}
	if err := repository.Close(); err != nil {
		t.Fatalf("Close() error = %v", err)
	}

	if _, err := OpenRepository(ctx, location, []byte("another-password")); !errors.Is(err, ErrOpenEncryptedCache) {
		t.Fatalf("OpenRepository() error = %v, want ErrOpenEncryptedCache", err)
	}

	database, err := Open(ctx, location)
	if err != nil {
		t.Fatalf("Open() error = %v", err)
	}
	t.Cleanup(func() {
		if err := database.Close(); err != nil {
			t.Errorf("database.Close() error = %v", err)
		}
	})

	var metadataRows int
	if err := database.db.QueryRowContext(
		ctx,
		"SELECT COUNT(*) FROM cache_metadata",
	).Scan(&metadataRows); err != nil {
		t.Fatalf("count cache metadata: %v", err)
	}
	if metadataRows != 0 {
		t.Fatalf("cache metadata rows = %d, want 0", metadataRows)
	}
}

func TestIntegration_OpenExistingRepository(t *testing.T) {
	ctx := context.Background()
	location := testLocation(t)
	password := []byte("correct-horse-battery-staple")

	if _, err := OpenExistingRepository(ctx, location, password); !errors.Is(err, ErrLocalCacheNotFound) {
		t.Fatalf("OpenExistingRepository() missing cache error = %v, want ErrLocalCacheNotFound", err)
	}

	repository, err := OpenRepository(ctx, location, password)
	if err != nil {
		t.Fatalf("OpenRepository() error = %v", err)
	}
	if err := repository.Close(); err != nil {
		t.Fatalf("Close() error = %v", err)
	}

	repository, err = OpenExistingRepository(ctx, location, password)
	if err != nil {
		t.Fatalf("OpenExistingRepository() error = %v", err)
	}
	if err := repository.Close(); err != nil {
		t.Fatalf("Close() existing error = %v", err)
	}

	if _, err := OpenExistingRepository(ctx, location, []byte("wrong-password")); !errors.Is(err, ErrOpenEncryptedCache) {
		t.Fatalf("OpenExistingRepository() wrong password error = %v, want ErrOpenEncryptedCache", err)
	}
}
