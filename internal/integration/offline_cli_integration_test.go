//go:build integration

package integration_test

import (
	"bytes"
	"context"
	"errors"
	"fmt"
	"os"
	"path/filepath"
	"strings"
	"testing"
	"time"

	"github.com/xhrobj/gopherkeeper/internal/client/cache"
	"github.com/xhrobj/gopherkeeper/internal/client/config"
	"github.com/xhrobj/gopherkeeper/internal/model"
)

const offlineCLIWarning = "Source: encrypted local cache (data may be stale)."

func TestIntegration_CLIOfflineReadFlow(t *testing.T) {
	isolateClientConfig(t)

	ctx := context.Background()
	cfg := offlineCLIConfig(t)
	password := testRegistrationPassword
	records := offlineCLIRecords()
	prepareCLIOfflineCache(t, ctx, cfg, "alice", password, records)

	stdout, stderr, err := runOfflineListCommand(ctx, cfg, " Alice ", password)
	if err != nil {
		t.Fatalf("offline list: %v", err)
	}
	if stderr != "" {
		t.Errorf("offline list stderr = %q, want empty output", stderr)
	}
	if !strings.HasPrefix(stdout, offlineCLIWarning+"\n") {
		t.Fatalf("offline list stdout = %q, want source warning first", stdout)
	}
	for _, record := range records {
		for _, want := range []string{
			record.Metadata.ID,
			string(record.Metadata.Type),
			record.Metadata.Title,
			fmt.Sprintf("%d", record.Metadata.Revision),
		} {
			if !strings.Contains(stdout, want) {
				t.Errorf("offline list stdout = %q, want %q", stdout, want)
			}
		}
	}
	for _, privateValue := range []string{
		password,
		"cached text secret",
		"offline-binary-secret",
	} {
		if strings.Contains(stdout, privateValue) {
			t.Errorf("offline list stdout contains private value %q", privateValue)
		}
	}

	assertOfflineTextRecord(t, ctx, cfg, records[0], password)
	assertOfflineBinaryRecord(t, ctx, cfg, records[1], password)

	if _, err := os.Stat(cfg.CACertFile); !errors.Is(err, os.ErrNotExist) {
		t.Fatalf("CA file stat error = %v, want os.ErrNotExist", err)
	}
	if _, err := os.Stat(cfg.SessionFile); !errors.Is(err, os.ErrNotExist) {
		t.Fatalf("session file stat error = %v, want os.ErrNotExist", err)
	}
}

func TestIntegration_CLIOfflineListReturnsEmptyExistingCache(t *testing.T) {
	isolateClientConfig(t)

	ctx := context.Background()
	cfg := offlineCLIConfig(t)
	password := testRegistrationPassword
	prepareCLIOfflineCache(t, ctx, cfg, "alice", password, nil)

	stdout, stderr, err := runOfflineListCommand(ctx, cfg, "alice", password)
	if err != nil {
		t.Fatalf("offline list empty cache: %v", err)
	}
	if stderr != "" {
		t.Errorf("offline list empty cache stderr = %q, want empty output", stderr)
	}
	want := offlineCLIWarning + "\nNo cached records found.\n"
	if stdout != want {
		t.Errorf("offline list empty cache stdout = %q, want %q", stdout, want)
	}
}

func TestIntegration_CLIOfflineReadErrors(t *testing.T) {
	isolateClientConfig(t)

	ctx := context.Background()
	cfg := offlineCLIConfig(t)
	password := testRegistrationPassword
	prepareCLIOfflineCache(t, ctx, cfg, "alice", password, offlineCLIRecords()[:1])

	t.Run("another login", func(t *testing.T) {
		location, err := cache.ResolveLocation(cfg.CacheDir, cfg.Address, "bob")
		if err != nil {
			t.Fatalf("resolve Bob cache location: %v", err)
		}

		stdout, stderr, err := runOfflineListCommand(ctx, cfg, "bob", password)
		assertOfflineCLIError(
			t,
			"offline list for another login",
			stdout,
			stderr,
			err,
			"local cache not found, run sync while online first",
			password,
		)
		if _, statErr := os.Stat(location.Directory); !errors.Is(statErr, os.ErrNotExist) {
			t.Fatalf("Bob cache directory stat error = %v, want os.ErrNotExist", statErr)
		}
	})

	t.Run("wrong password", func(t *testing.T) {
		wrongPassword := "wrong-offline-password-marker"
		stdout, stderr, err := runOfflineListCommand(ctx, cfg, "alice", wrongPassword)
		assertOfflineCLIError(
			t,
			"offline list with wrong password",
			stdout,
			stderr,
			err,
			"failed to open encrypted local cache",
			wrongPassword,
		)
	})

	t.Run("missing record", func(t *testing.T) {
		stdout, stderr, err := runOfflineGetCommand(
			ctx,
			cfg,
			"alice",
			password,
			"99999999-9999-4999-8999-999999999999",
			"",
		)
		assertOfflineCLIError(
			t,
			"offline get missing record",
			stdout,
			stderr,
			err,
			"record not found in local cache",
			password,
		)
	})
}

func TestIntegration_CLIMutationsRejectOfflineFlag(t *testing.T) {
	isolateClientConfig(t)

	tests := []struct {
		name string
		args []string
	}{
		{
			name: "create",
			args: []string{"gkeep", "records", "create-text", "--offline"},
		},
		{
			name: "update",
			args: []string{
				"gkeep", "records", "update-text",
				"11111111-1111-4111-8111-111111111111",
				"--revision", "1", "--offline",
			},
		},
		{
			name: "delete",
			args: []string{
				"gkeep", "records", "delete",
				"11111111-1111-4111-8111-111111111111",
				"--revision", "1", "--offline",
			},
		},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			stdout, stderr, err := runClientCommand(context.Background(), tt.args)
			if err == nil {
				t.Fatal("offline mutation error = nil, want unknown flag error")
			}
			if !strings.Contains(err.Error(), "offline") {
				t.Fatalf("offline mutation error = %q, want rejected offline flag", err)
			}
			if strings.Contains(stdout+stderr, testRegistrationPassword) {
				t.Error("offline mutation output contains password")
			}
		})
	}
}

func offlineCLIConfig(t *testing.T) config.Config {
	t.Helper()

	directory := t.TempDir()
	return config.Config{
		Address:     "127.0.0.1:1",
		CACertFile:  filepath.Join(directory, "missing-ca.pem"),
		SessionFile: filepath.Join(directory, "missing-session.json"),
		CacheDir:    filepath.Join(directory, "cache"),
	}
}

func prepareCLIOfflineCache(
	t *testing.T,
	ctx context.Context,
	cfg config.Config,
	login string,
	password string,
	records []model.Record,
) {
	t.Helper()

	canonicalLogin, err := model.CanonicalizeLogin(login)
	if err != nil {
		t.Fatalf("canonicalize cache login: %v", err)
	}
	location, err := cache.ResolveLocation(cfg.CacheDir, cfg.Address, canonicalLogin)
	if err != nil {
		t.Fatalf("resolve offline cache location: %v", err)
	}
	repository, err := cache.OpenRepository(ctx, location, []byte(password))
	if err != nil {
		t.Fatalf("open offline cache: %v", err)
	}
	defer func() {
		if err := repository.Close(); err != nil {
			t.Errorf("close offline cache: %v", err)
		}
	}()

	if err := repository.ApplyChanges(ctx, records, nil); err != nil {
		t.Fatalf("seed offline cache: %v", err)
	}
}

func offlineCLIRecords() []model.Record {
	createdAt := time.Date(2026, time.July, 15, 12, 0, 0, 0, time.UTC)
	updatedAt := createdAt.Add(time.Minute)

	return []model.Record{
		{
			Metadata: model.RecordMetadata{
				ID:        "11111111-1111-4111-8111-111111111111",
				Type:      model.RecordTypeText,
				Title:     "Cached note",
				Revision:  2,
				CreatedAt: createdAt,
				UpdatedAt: updatedAt,
			},
			Payload: &model.TextPayload{
				Text:     "cached text secret",
				Metadata: "cached text metadata",
			},
		},
		{
			Metadata: model.RecordMetadata{
				ID:        "44444444-4444-4444-8444-444444444444",
				Type:      model.RecordTypeBinary,
				Title:     "Cached binary",
				Revision:  4,
				CreatedAt: createdAt,
				UpdatedAt: updatedAt,
			},
			Payload: &model.BinaryPayload{
				Filename:    "cached.bin",
				Data:        []byte("offline-binary-secret\x00\xff"),
				ContentType: "application/octet-stream",
				Metadata:    "cached binary metadata",
			},
		},
	}
}

func runOfflineListCommand(
	ctx context.Context,
	cfg config.Config,
	login string,
	password string,
) (string, string, error) {
	return runClientCommandWithInput(ctx, []string{
		"gkeep",
		"--address", cfg.Address,
		"--ca-cert", cfg.CACertFile,
		"--session-file", cfg.SessionFile,
		"--cache-dir", cfg.CacheDir,
		"records", "list", "--offline", "--login", login,
	}, password+"\n")
}

func runOfflineGetCommand(
	ctx context.Context,
	cfg config.Config,
	login string,
	password string,
	recordID string,
	outputPath string,
) (string, string, error) {
	args := []string{
		"gkeep",
		"--address", cfg.Address,
		"--ca-cert", cfg.CACertFile,
		"--session-file", cfg.SessionFile,
		"--cache-dir", cfg.CacheDir,
		"records", "get", recordID,
		"--offline", "--login", login,
	}
	if outputPath != "" {
		args = append(args, "--output", outputPath)
	}

	return runClientCommandWithInput(ctx, args, password+"\n")
}

func assertOfflineTextRecord(
	t *testing.T,
	ctx context.Context,
	cfg config.Config,
	record model.Record,
	password string,
) {
	t.Helper()

	stdout, stderr, err := runOfflineGetCommand(
		ctx,
		cfg,
		"ALICE",
		password,
		record.Metadata.ID,
		"",
	)
	if err != nil {
		t.Fatalf("offline get text: %v", err)
	}
	assertOfflineCLIOutput(t, stdout, stderr, password,
		"Type: text",
		"Title: Cached note",
		"Text:\ncached text secret",
		"Metadata:\ncached text metadata",
	)
}

func assertOfflineBinaryRecord(
	t *testing.T,
	ctx context.Context,
	cfg config.Config,
	record model.Record,
	password string,
) {
	t.Helper()

	outputPath := filepath.Join(t.TempDir(), "restored-cached.bin")
	stdout, stderr, err := runOfflineGetCommand(
		ctx,
		cfg,
		"alice",
		password,
		record.Metadata.ID,
		outputPath,
	)
	if err != nil {
		t.Fatalf("offline get binary: %v", err)
	}
	assertOfflineCLIOutput(t, stdout, stderr, password,
		"Type: binary",
		"Filename: cached.bin",
		"Saved to: "+outputPath,
		"Content type: application/octet-stream",
		"Metadata:\ncached binary metadata",
	)

	payload, ok := record.Payload.(*model.BinaryPayload)
	if !ok || payload == nil {
		t.Fatal("offline binary fixture has unexpected payload")
	}
	got, err := os.ReadFile(outputPath)
	if err != nil {
		t.Fatalf("read restored offline binary: %v", err)
	}
	if !bytes.Equal(got, payload.Data) {
		t.Errorf("restored offline binary = %v, want %v", got, payload.Data)
	}
	info, err := os.Stat(outputPath)
	if err != nil {
		t.Fatalf("stat restored offline binary: %v", err)
	}
	if info.Mode().Perm() != 0o600 {
		t.Errorf("restored offline binary mode = %o, want 600", info.Mode().Perm())
	}
}

func assertOfflineCLIOutput(
	t *testing.T,
	stdout string,
	stderr string,
	cachePassword string,
	want ...string,
) {
	t.Helper()

	if stderr != "" {
		t.Errorf("offline get stderr = %q, want empty output", stderr)
	}
	if !strings.HasPrefix(stdout, offlineCLIWarning+"\n") {
		t.Errorf("offline get stdout = %q, want source warning first", stdout)
	}
	for _, value := range want {
		if !strings.Contains(stdout, value) {
			t.Errorf("offline get stdout = %q, want %q", stdout, value)
		}
	}
	if strings.Contains(stdout, cachePassword) {
		t.Errorf("offline get stdout contains cache password %q", cachePassword)
	}
}

func assertOfflineCLIError(
	t *testing.T,
	operation string,
	stdout string,
	stderr string,
	err error,
	want string,
	secret string,
) {
	t.Helper()

	if err == nil {
		t.Fatalf("%s error = nil, want %q", operation, want)
	}
	if !strings.Contains(err.Error(), want) {
		t.Fatalf("%s error = %q, want %q", operation, err, want)
	}
	if strings.Contains(err.Error(), secret) || strings.Contains(stdout+stderr, secret) {
		t.Fatalf("%s exposes secret %q", operation, secret)
	}
	if stdout != "" || stderr != "" {
		t.Errorf("%s output = %q, stderr = %q, want empty", operation, stdout, stderr)
	}
}
