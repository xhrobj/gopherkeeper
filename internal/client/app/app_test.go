package app

import (
	"context"
	"errors"
	"os"
	"path/filepath"
	"testing"

	"github.com/xhrobj/gopherkeeper/internal/client/cache"
	"github.com/xhrobj/gopherkeeper/internal/client/config"
	"github.com/xhrobj/gopherkeeper/internal/client/usecase"
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

func TestNewOffline_DoesNotLoadNetworkOrSessionDependencies(t *testing.T) {
	cacheDirectory := filepath.Join(t.TempDir(), "encrypted-cache")

	application := NewOffline(config.Config{
		Address:     "localhost:8080",
		CACertFile:  filepath.Join(t.TempDir(), "missing-ca.pem"),
		SessionFile: filepath.Join(t.TempDir(), "missing-session.json"),
		CacheDir:    cacheDirectory,
	})
	if application == nil {
		t.Fatal("NewOffline() application = nil")
	}

	if _, err := os.Stat(cacheDirectory); !errors.Is(err, os.ErrNotExist) {
		t.Fatalf("cache directory stat error = %v, want os.ErrNotExist", err)
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

func TestNew_DoesNotOpenEncryptedCache(t *testing.T) {
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

func TestEncryptedSyncCacheRepositoryProvider_RejectsInvalidIdentityWithoutCreatingCache(t *testing.T) {
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

func TestEncryptedOfflineCacheRepositoryProvider_DoesNotCreateMissingCache(t *testing.T) {
	cacheDirectory := filepath.Join(t.TempDir(), "encrypted-cache")
	provider := encryptedOfflineCacheRepositoryProvider(cacheDirectory)

	_, err := provider(
		context.Background(),
		"localhost:8080",
		"alice",
		[]byte("correct-horse-battery-staple"),
	)
	if !errors.Is(err, usecase.ErrLocalCacheNotFound) {
		t.Fatalf("offline cache provider error = %v, want ErrLocalCacheNotFound", err)
	}
	if !errors.Is(err, cache.ErrLocalCacheNotFound) {
		t.Fatalf("offline cache provider error = %v, want cache ErrLocalCacheNotFound", err)
	}

	if _, err := os.Stat(cacheDirectory); !errors.Is(err, os.ErrNotExist) {
		t.Fatalf("cache directory stat error = %v, want os.ErrNotExist", err)
	}
}
