package app

import (
	"context"
	"errors"
	"os"
	"path/filepath"
	"testing"

	"github.com/xhrobj/gopherkeeper/internal/client/cache"
	"github.com/xhrobj/gopherkeeper/internal/client/config"
)

func TestNew(t *testing.T) {
	application, err := New(config.Config{
		Address:     "localhost:8080",
		SessionFile: t.TempDir() + "/session.json",
	})
	if err != nil {
		t.Fatalf("New() error = %v", err)
	}
	if application == nil {
		t.Fatal("New() application = nil")
	}
}

func TestNewLogout(t *testing.T) {
	application, err := NewLogout(config.Config{
		SessionFile: t.TempDir() + "/session.json",
	})
	if err != nil {
		t.Fatalf("NewLogout() error = %v", err)
	}
	if application == nil {
		t.Fatal("NewLogout() application = nil")
	}
}

func TestNewDoesNotOpenEncryptedCache(t *testing.T) {
	cacheDirectory := filepath.Join(t.TempDir(), "encrypted-cache")

	application, err := New(config.Config{
		Address:     "localhost:8080",
		SessionFile: filepath.Join(t.TempDir(), "session.json"),
		CacheDir:    cacheDirectory,
	})
	if err != nil {
		t.Fatalf("New() error = %v", err)
	}
	if application == nil {
		t.Fatal("New() application = nil")
	}

	if _, err := os.Stat(cacheDirectory); !errors.Is(err, os.ErrNotExist) {
		t.Fatalf("cache directory stat error = %v, want os.ErrNotExist", err)
	}
}

func TestEncryptedSyncCacheRepositoryProviderRejectsInvalidIdentityWithoutCreatingCache(t *testing.T) {
	cacheDirectory := filepath.Join(t.TempDir(), "encrypted-cache")
	provider := encryptedSyncCacheRepositoryProvider(cacheDirectory)

	_, err := provider(context.Background(), "localhost:8080", "Alice", []byte("password"))
	if !errors.Is(err, cache.ErrInvalidAccountIdentity) {
		t.Fatalf("cache provider error = %v, want ErrInvalidAccountIdentity", err)
	}

	if _, err := os.Stat(cacheDirectory); !errors.Is(err, os.ErrNotExist) {
		t.Fatalf("cache directory stat error = %v, want os.ErrNotExist", err)
	}
}
