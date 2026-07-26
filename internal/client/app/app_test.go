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
		Address:    "localhost:8080",
		SessionDir: t.TempDir(),
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
		Address:    "localhost:8080",
		CACertFile: filepath.Join(t.TempDir(), "missing-ca.pem"),
		SessionDir: filepath.Join(t.TempDir(), "missing-session"),
		CacheDir:   cacheDirectory,
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
		SessionDir: t.TempDir(),
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
		Address:    "localhost:8080",
		SessionDir: t.TempDir(),
		CacheDir:   cacheDirectory,
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

	_, err := provider(context.Background(), "Alice", []byte("password"))
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

func TestNewRuntime_SelectsConfiguredTransport(t *testing.T) {
	tests := []struct {
		name      string
		transport config.Transport
	}{
		{name: "HTTPS", transport: config.TransportHTTPS},
		{name: "gRPC", transport: config.TransportGRPC},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			cfg := config.Default()
			cfg.Transport = tt.transport
			cfg.SessionDir = t.TempDir()

			runtime, err := NewRuntime(cfg)
			if err != nil {
				t.Fatalf("NewRuntime() error = %v", err)
			}

			if runtime == nil || runtime.Application == nil {
				t.Fatal("NewRuntime() runtime or application = nil")
			}
			if runtime.closer == nil {
				t.Fatal("runtime transport closer = nil")
			}
			if err := runtime.Close(); err != nil {
				t.Fatalf("Close() error = %v", err)
			}
			if err := runtime.Close(); err != nil {
				t.Fatalf("second Close() error = %v", err)
			}
		})
	}
}

func TestNewRuntime_RejectsUnknownTransport(t *testing.T) {
	cfg := config.Default()
	cfg.Transport = config.Transport("quic")

	if _, err := NewRuntime(cfg); err == nil {
		t.Fatal("NewRuntime() error = nil")
	}
}

type closerStub struct {
	calls int
	err   error
}

func (stub *closerStub) Close() error {
	stub.calls++
	return stub.err
}

func TestRuntimeClose(t *testing.T) {
	closer := &closerStub{}
	runtime, err := newRuntime(usecase.New(nil, nil, nil, nil, nil, nil), closer)
	if err != nil {
		t.Fatalf("newRuntime() error = %v", err)
	}

	if err := runtime.Close(); err != nil {
		t.Fatalf("Close() error = %v", err)
	}
	if err := runtime.Close(); err != nil {
		t.Fatalf("second Close() error = %v", err)
	}
	if closer.calls != 1 {
		t.Fatalf("closer calls = %d, want 1", closer.calls)
	}
}

func TestRuntimeClose_ReturnsStoredCloseError(t *testing.T) {
	closeErr := errors.New("close failed")
	closer := &closerStub{err: closeErr}
	runtime, err := newRuntime(usecase.New(nil, nil, nil, nil, nil, nil), closer)
	if err != nil {
		t.Fatalf("newRuntime() error = %v", err)
	}

	for range 2 {
		if err := runtime.Close(); !errors.Is(err, closeErr) {
			t.Fatalf("Close() error = %v, want %v", err, closeErr)
		}
	}
	if closer.calls != 1 {
		t.Fatalf("closer calls = %d, want 1", closer.calls)
	}
}
