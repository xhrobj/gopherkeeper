//go:build integration

package cache

import (
	"bytes"
	"context"
	"encoding/base64"
	"errors"
	"os"
	"path/filepath"
	"runtime"
	"strings"
	"testing"
	"time"

	"github.com/xhrobj/gopherkeeper/internal/client/cachecrypto"
	"github.com/xhrobj/gopherkeeper/internal/model"
)

func TestIntegration_RepositoryDoesNotPersistPlaintext(t *testing.T) {
	ctx := context.Background()
	location := testLocation(t)
	password := []byte("cache-password-marker-38-correct-horse-battery-staple")

	repository, err := OpenRepository(ctx, location, password)
	if err != nil {
		t.Fatalf("OpenRepository() error = %v", err)
	}
	t.Cleanup(func() {
		if err := repository.Close(); err != nil {
			t.Errorf("Repository.Close() error = %v", err)
		}
	})

	var journalMode string
	if err := repository.database.db.QueryRowContext(ctx, "PRAGMA journal_mode = WAL").Scan(&journalMode); err != nil {
		t.Fatalf("enable WAL journal mode: %v", err)
	}
	if !strings.EqualFold(journalMode, "wal") {
		t.Fatalf("journal mode = %q, want WAL", journalMode)
	}
	if _, err := repository.database.db.ExecContext(ctx, "PRAGMA wal_autocheckpoint = 0"); err != nil {
		t.Fatalf("disable WAL autocheckpoint: %v", err)
	}

	records, markers := securityTestRecords()
	for _, record := range records {
		if err := repository.Upsert(ctx, record); err != nil {
			t.Fatalf("Upsert(%s) error = %v", record.Metadata.Type, err)
		}
	}

	var salt []byte
	if err := repository.database.db.QueryRowContext(
		ctx,
		"SELECT kdf_salt FROM cache_metadata WHERE singleton = 1",
	).Scan(&salt); err != nil {
		t.Fatalf("read KDF salt: %v", err)
	}
	derivedKey, err := cachecrypto.DeriveKey(password, salt, cachecrypto.KDFVersion)
	if err != nil {
		t.Fatalf("DeriveKey() error = %v", err)
	}
	markers = append(markers, password, derivedKey)

	files, err := os.ReadDir(location.Directory)
	if err != nil {
		t.Fatalf("ReadDir() error = %v", err)
	}

	regularFiles := 0
	for _, entry := range files {
		info, err := entry.Info()
		if err != nil {
			t.Fatalf("inspect %s: %v", entry.Name(), err)
		}
		if !info.Mode().IsRegular() {
			continue
		}
		regularFiles++

		path := filepath.Join(location.Directory, entry.Name())
		data, err := os.ReadFile(path)
		if err != nil {
			t.Fatalf("read %s: %v", entry.Name(), err)
		}
		for _, marker := range markers {
			if bytes.Contains(data, marker) {
				t.Fatalf("local cache file %s contains plaintext marker %q", entry.Name(), marker)
			}
		}
	}
	if regularFiles == 0 {
		t.Fatal("account directory contains no regular SQLite files")
	}
}

func TestIntegration_OpenRepositoryDoesNotExposePassword(t *testing.T) {
	ctx := context.Background()
	location := testLocation(t)
	correctPassword := "GK38-CORRECT-PASSWORD-MARKER-5d9a1e7c3f2b8064"
	wrongPassword := "GK38-WRONG-PASSWORD-MARKER-8c2f6a1d9e3b7054"

	repository, err := OpenRepository(ctx, location, []byte(correctPassword))
	if err != nil {
		t.Fatalf("OpenRepository() first error = %v", err)
	}
	if err := repository.Close(); err != nil {
		t.Fatalf("Repository.Close() error = %v", err)
	}

	_, err = OpenRepository(ctx, location, []byte(wrongPassword))
	if !errors.Is(err, ErrOpenEncryptedCache) {
		t.Fatalf("OpenRepository() error = %v, want ErrOpenEncryptedCache", err)
	}
	for _, password := range []string{correctPassword, wrongPassword} {
		if strings.Contains(err.Error(), password) {
			t.Fatalf("OpenRepository() error exposes password marker %q", password)
		}
	}
}

func TestIntegration_RepositoryRejectsTamperedEncryptedData(t *testing.T) {
	tests := []struct {
		name  string
		query string
	}{
		{
			name:  "nonce",
			query: "UPDATE cached_records SET nonce = zeroblob(length(nonce)) WHERE id = ?",
		},
		{
			name:  "ciphertext",
			query: "UPDATE cached_records SET ciphertext = zeroblob(length(ciphertext)) WHERE id = ?",
		},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			ctx := context.Background()
			repository, err := OpenRepository(ctx, testLocation(t), []byte("cache-password"))
			if err != nil {
				t.Fatalf("OpenRepository() error = %v", err)
			}
			t.Cleanup(func() {
				if err := repository.Close(); err != nil {
					t.Errorf("Repository.Close() error = %v", err)
				}
			})

			record := securityTextRecord()
			if err := repository.Upsert(ctx, record); err != nil {
				t.Fatalf("Upsert() error = %v", err)
			}
			if _, err := repository.database.db.ExecContext(ctx, tt.query, record.Metadata.ID); err != nil {
				t.Fatalf("tamper %s: %v", tt.name, err)
			}

			if _, err := repository.Get(ctx, record.Metadata.ID); !errors.Is(err, ErrCorruptedCacheRecord) {
				t.Fatalf("Get() error = %v, want ErrCorruptedCacheRecord", err)
			}
		})
	}
}

func TestIntegration_OpenRepositoryRejectsTamperedMetadata(t *testing.T) {
	tests := []struct {
		name    string
		query   string
		wantErr error
	}{
		{
			name:    "account identity",
			query:   "UPDATE cache_metadata SET account_id = printf('%064d', 0) WHERE singleton = 1",
			wantErr: ErrOpenEncryptedCache,
		},
		{
			name:    "key-check nonce",
			query:   "UPDATE cache_metadata SET key_check_nonce = zeroblob(length(key_check_nonce)) WHERE singleton = 1",
			wantErr: ErrOpenEncryptedCache,
		},
		{
			name:    "key-check ciphertext",
			query:   "UPDATE cache_metadata SET key_check_ciphertext = zeroblob(length(key_check_ciphertext)) WHERE singleton = 1",
			wantErr: ErrOpenEncryptedCache,
		},
		{
			name:    "metadata version",
			query:   "UPDATE cache_metadata SET format_version = 69 WHERE singleton = 1",
			wantErr: ErrUnsupportedCacheMetadataVersion,
		},
		{
			name:    "KDF version",
			query:   "UPDATE cache_metadata SET kdf_version = 69 WHERE singleton = 1",
			wantErr: cachecrypto.ErrUnsupportedKDFVersion,
		},
		{
			name:    "crypto version",
			query:   "UPDATE cache_metadata SET crypto_version = 69 WHERE singleton = 1",
			wantErr: ErrUnsupportedCacheCryptoVersion,
		},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			ctx := context.Background()
			location := testLocation(t)
			password := []byte("cache-password")

			repository, err := OpenRepository(ctx, location, password)
			if err != nil {
				t.Fatalf("OpenRepository() first error = %v", err)
			}
			if _, err := repository.database.db.ExecContext(ctx, tt.query); err != nil {
				t.Fatalf("tamper metadata: %v", err)
			}
			if err := repository.Close(); err != nil {
				t.Fatalf("Repository.Close() error = %v", err)
			}

			if _, err := OpenRepository(ctx, location, password); !errors.Is(err, tt.wantErr) {
				t.Fatalf("OpenRepository() error = %v, want %v", err, tt.wantErr)
			}
		})
	}
}

func TestIntegration_CachePermissions(t *testing.T) {
	if runtime.GOOS == "windows" {
		t.Skip("Windows ACLs are outside the MVP permission boundary")
	}

	repository, err := OpenRepository(
		context.Background(),
		testLocation(t),
		[]byte("cache-password"),
	)
	if err != nil {
		t.Fatalf("OpenRepository() error = %v", err)
	}
	t.Cleanup(func() {
		if err := repository.Close(); err != nil {
			t.Errorf("Repository.Close() error = %v", err)
		}
	})

	location := repository.Location()
	assertFilePermissions(t, location.Directory, 0o700)
	assertFilePermissions(t, location.DatabaseFile, 0o600)
}

func securityTestRecords() ([]model.Record, [][]byte) {
	const (
		titleMarker       = "GK38-TITLE-MARKER-0b9e6d4f2a7c1e35"
		textMarker        = "GK38-TEXT-MARKER-4c7a1d9e2f6b8c30"
		metadataMarker    = "GK38-METADATA-MARKER-8e2c5a1f7d9b3c60"
		credentialsMarker = "GK38-CREDENTIALS-MARKER-6b2e8c4a1d7f9053"
		cardMarker        = "491761339284675109384726"
	)

	createdAt := time.Date(2026, time.July, 14, 12, 0, 0, 0, time.UTC)
	binaryMarker := []byte("GK38-BINARY-MARKER-9f5b7c2e4a6d8f1c")
	records := []model.Record{
		{
			Metadata: securityRecordMetadata(
				"11111111-1111-4111-8111-111111111111",
				model.RecordTypeText,
				titleMarker,
				createdAt,
			),
			Payload: &model.TextPayload{Text: textMarker, Metadata: metadataMarker},
		},
		{
			Metadata: securityRecordMetadata(
				"22222222-2222-4222-8222-222222222222",
				model.RecordTypeCredentials,
				"credentials record",
				createdAt,
			),
			Payload: &model.CredentialsPayload{Login: "alice", Password: credentialsMarker},
		},
		{
			Metadata: securityRecordMetadata(
				"33333333-3333-4333-8333-333333333333",
				model.RecordTypeCard,
				"card record",
				createdAt,
			),
			Payload: &model.CardPayload{Number: cardMarker},
		},
		{
			Metadata: securityRecordMetadata(
				"44444444-4444-4444-8444-444444444444",
				model.RecordTypeBinary,
				"binary record",
				createdAt,
			),
			Payload: &model.BinaryPayload{Filename: "backup.bin", Data: binaryMarker},
		},
	}

	markers := [][]byte{
		[]byte(titleMarker),
		[]byte(textMarker),
		[]byte(metadataMarker),
		[]byte(credentialsMarker),
		[]byte(cardMarker),
		binaryMarker,
		[]byte(base64.StdEncoding.EncodeToString(binaryMarker)),
	}

	return records, markers
}

func securityRecordMetadata(
	id string,
	recordType model.RecordType,
	title string,
	createdAt time.Time,
) model.RecordMetadata {
	return model.RecordMetadata{
		ID:        id,
		Type:      recordType,
		Title:     title,
		Revision:  1,
		CreatedAt: createdAt,
		UpdatedAt: createdAt,
	}
}

func securityTextRecord() model.Record {
	createdAt := time.Date(2026, time.July, 14, 12, 0, 0, 0, time.UTC)

	return model.Record{
		Metadata: model.RecordMetadata{
			ID:        "55555555-5555-4555-8555-555555555555",
			Type:      model.RecordTypeText,
			Title:     "tamper test",
			Revision:  1,
			CreatedAt: createdAt,
			UpdatedAt: createdAt,
		},
		Payload: &model.TextPayload{Text: "tamper marker"},
	}
}

func assertFilePermissions(t *testing.T, path string, want os.FileMode) {
	t.Helper()

	info, err := os.Stat(path)
	if err != nil {
		t.Fatalf("Stat(%s) error = %v", path, err)
	}
	if got := info.Mode().Perm(); got != want {
		t.Fatalf("permissions for %s = %04o, want %04o", path, got, want)
	}
}
