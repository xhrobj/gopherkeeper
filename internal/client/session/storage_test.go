package session

import (
	"encoding/json"
	"errors"
	"os"
	"path/filepath"
	"runtime"
	"strings"
	"testing"
	"time"
)

func TestFileStorage_SaveAndLoad(t *testing.T) {
	storage := newTestStorage(t, "session.json")
	want := testSession()

	if err := storage.Save(want); err != nil {
		t.Fatalf("Save() error = %v", err)
	}

	got, err := storage.Load(want.ServerAddress)
	if err != nil {
		t.Fatalf("Load() error = %v", err)
	}

	if got.ServerAddress != want.ServerAddress {
		t.Errorf("server address = %q, want %q", got.ServerAddress, want.ServerAddress)
	}
	if got.AccessToken != want.AccessToken {
		t.Errorf("access token = %q, want %q", got.AccessToken, want.AccessToken)
	}
	if !got.ExpiresAt.Equal(want.ExpiresAt) {
		t.Errorf("expires at = %s, want %s", got.ExpiresAt, want.ExpiresAt)
	}
}

func TestFileStorage_SaveCreatesPrivateFile(t *testing.T) {
	storage := newTestStorage(t, filepath.Join("gopherkeeper", "session.json"))

	if err := storage.Save(testSession()); err != nil {
		t.Fatalf("Save() error = %v", err)
	}

	assertFileMode(t, filepath.Dir(storage.path), 0o700)
	assertFileMode(t, storage.path, 0o600)
}

func TestFileStorage_DeleteRemovesSessionFile(t *testing.T) {
	storage := newTestStorage(t, "session.json")

	if err := storage.Save(testSession()); err != nil {
		t.Fatalf("Save() error = %v", err)
	}
	if err := storage.Delete(); err != nil {
		t.Fatalf("Delete() error = %v", err)
	}

	_, err := os.Stat(storage.path)
	if !errors.Is(err, os.ErrNotExist) {
		t.Fatalf("stat deleted session = %v, want not exist", err)
	}
}

func TestFileStorage_DeleteIgnoresMissingSessionFile(t *testing.T) {
	storage := newTestStorage(t, "missing-session.json")

	if err := storage.Delete(); err != nil {
		t.Fatalf("Delete() error = %v", err)
	}
}

func TestFileStorage_LoadReturnsNotFound(t *testing.T) {
	storage := newTestStorage(t, "missing-session.json")

	_, err := storage.Load("localhost:8080")
	if !errors.Is(err, ErrNotFound) {
		t.Fatalf("Load() error = %v, want ErrNotFound", err)
	}
}

func TestFileStorage_LoadRejectsExpiredSession(t *testing.T) {
	storage := newTestStorage(t, "session.json")
	session := testSession()
	session.ExpiresAt = testNow()

	if err := writeRawSession(storage.path, session); err != nil {
		t.Fatalf("write session: %v", err)
	}

	_, err := storage.Load(session.ServerAddress)
	if !errors.Is(err, ErrExpired) {
		t.Fatalf("Load() error = %v, want ErrExpired", err)
	}
}

func TestFileStorage_LoadRejectsServerMismatch(t *testing.T) {
	storage := newTestStorage(t, "session.json")
	session := testSession()

	if err := storage.Save(session); err != nil {
		t.Fatalf("Save() error = %v", err)
	}

	_, err := storage.Load("localhost:8081")
	if !errors.Is(err, ErrServerMismatch) {
		t.Fatalf("Load() error = %v, want ErrServerMismatch", err)
	}
}

func TestFileStorage_LoadRejectsInvalidJSON(t *testing.T) {
	tests := []struct {
		name string
		body string
	}{
		{
			name: "malformed JSON",
			body: `{"server_address":`,
		},
		{
			name: "unknown field",
			body: `{"server_address":"localhost:8080","access_token":"token",` +
				`"expires_at":"2026-07-04T12:15:00Z","extra":"value"}`,
		},
		{
			name: "multiple JSON values",
			body: `{"server_address":"localhost:8080","access_token":"token",` +
				`"expires_at":"2026-07-04T12:15:00Z"} {}`,
		},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			storage := newTestStorage(t, "session.json")
			if err := os.WriteFile(storage.path, []byte(tt.body), 0o600); err != nil {
				t.Fatalf("write session file: %v", err)
			}

			_, err := storage.Load("localhost:8080")
			if !errors.Is(err, ErrInvalid) {
				t.Fatalf("Load() error = %v, want ErrInvalid", err)
			}
		})
	}
}

func TestFileStorage_SaveRejectsInvalidSession(t *testing.T) {
	tests := []struct {
		name    string
		mutate  func(*Session)
		wantErr error
	}{
		{
			name: "missing server address",
			mutate: func(s *Session) {
				s.ServerAddress = ""
			},
			wantErr: ErrInvalid,
		},
		{
			name: "missing access token",
			mutate: func(s *Session) {
				s.AccessToken = ""
			},
			wantErr: ErrInvalid,
		},
		{
			name: "expired session",
			mutate: func(s *Session) {
				s.ExpiresAt = testNow()
			},
			wantErr: ErrExpired,
		},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			storage := newTestStorage(t, "session.json")
			session := testSession()
			tt.mutate(&session)

			err := storage.Save(session)
			if !errors.Is(err, tt.wantErr) {
				t.Fatalf("Save() error = %v, want %v", err, tt.wantErr)
			}
			if err != nil && strings.Contains(err.Error(), testSession().AccessToken) {
				t.Error("Save() error contains access token")
			}
		})
	}
}

func TestNewFileStorageUsesDefaultPath(t *testing.T) {
	storage, err := NewFileStorage("")
	if err != nil {
		t.Fatalf("NewFileStorage() error = %v", err)
	}

	cacheDir, err := os.UserCacheDir()
	if err != nil {
		t.Fatalf("UserCacheDir() error = %v", err)
	}
	want := filepath.Join(cacheDir, "gopherkeeper", "session.json")
	if storage.path != want {
		t.Errorf("storage path = %q, want %q", storage.path, want)
	}
}

func newTestStorage(t *testing.T, name string) *FileStorage {
	t.Helper()

	storage, err := newFileStorage(
		filepath.Join(t.TempDir(), name),
		testNow,
	)
	if err != nil {
		t.Fatalf("newFileStorage() error = %v", err)
	}

	return storage
}

func testSession() Session {
	return Session{
		ServerAddress: "localhost:8080",
		AccessToken:   "test.jwt.token",
		ExpiresAt:     time.Date(2026, time.July, 4, 12, 15, 0, 0, time.UTC),
	}
}

func testNow() time.Time {
	return time.Date(2026, time.July, 4, 12, 0, 0, 0, time.UTC)
}

func writeRawSession(path string, session Session) error {
	data, err := jsonMarshal(session)
	if err != nil {
		return err
	}

	return os.WriteFile(path, data, 0o600)
}

func jsonMarshal(session Session) ([]byte, error) {
	data, err := json.Marshal(session)
	if err != nil {
		return nil, err
	}

	return data, nil
}

func assertFileMode(t *testing.T, path string, want os.FileMode) {
	t.Helper()
	if runtime.GOOS == "windows" {
		t.Skip("POSIX file mode check is not reliable on Windows")
	}

	info, err := os.Stat(path)
	if err != nil {
		t.Fatalf("stat %s: %v", path, err)
	}
	if got := info.Mode().Perm(); got != want {
		t.Errorf("mode %s = %o, want %o", path, got, want)
	}
}
