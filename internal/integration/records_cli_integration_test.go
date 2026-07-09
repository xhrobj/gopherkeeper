//go:build integration

package integration_test

import (
	"bytes"
	"context"
	"fmt"
	"os"
	"path/filepath"
	"regexp"
	"strings"
	"testing"

	"github.com/jackc/pgx/v5/pgxpool"
	"github.com/xhrobj/gopherkeeper/internal/buildinfo"
	clientcli "github.com/xhrobj/gopherkeeper/internal/client/cli"
	"github.com/xhrobj/gopherkeeper/internal/server/middleware"
	"github.com/xhrobj/gopherkeeper/internal/server/migration"
)

var createdRecordPattern = regexp.MustCompile(`^Created text record ([0-9a-f-]+) with revision ([0-9]+)\.$`)

func TestIntegration_CLITextRecordUpdateDeleteFlow(t *testing.T) {
	dsn := os.Getenv("DATABASE_DSN")
	if dsn == "" {
		t.Fatal("DATABASE_DSN is not set")
	}

	isolateClientConfig(t)
	t.Setenv("ADDRESS", "")
	t.Setenv("CA_CERT_FILE", "")
	t.Setenv("SESSION_FILE", "")

	ctx, cancel := context.WithTimeout(context.Background(), integrationTestTimeout)
	defer cancel()

	adminPool := openPostgres(t, ctx, dsn)
	t.Cleanup(adminPool.Close)

	databaseName := createTestDatabase(t, ctx, adminPool)
	t.Cleanup(func() {
		dropTestDatabase(t, adminPool, databaseName)
	})

	testPool := openTestPostgres(t, ctx, dsn, databaseName)
	t.Cleanup(testPool.Close)

	if err := migration.Run(testPool); err != nil {
		t.Fatalf("migration.Run() error = %v", err)
	}

	var httpLogs bytes.Buffer
	logger := newIntegrationLogger(&httpLogs)
	defer func() {
		_ = logger.Sync()
	}()

	caCertFile, serverCertFile, serverKeyFile := generateTLSFiles(t)
	serverAddress, stopServer := startHTTPSServer(
		t,
		middleware.WithLogging(newAuthenticatedServerHandler(t, testPool), logger),
		serverCertFile,
		serverKeyFile,
	)
	defer stopServer()

	if _, _, err := runRegisterCommand(ctx, serverAddress, caCertFile, " Alice ", testRegistrationPassword); err != nil {
		t.Fatalf("register Alice: %v", err)
	}
	aliceSessionFile := filepath.Join(t.TempDir(), "alice-session.json")
	if _, _, err := runLoginCommand(
		ctx,
		serverAddress,
		caCertFile,
		aliceSessionFile,
		" Alice ",
		testRegistrationPassword,
	); err != nil {
		t.Fatalf("login Alice: %v", err)
	}

	initialTextFile := writeIntegrationFile(t, "initial-note.txt", "initial secret")
	initialMetadataFile := writeIntegrationFile(t, "initial-metadata.txt", "initial private metadata")
	stdout, stderr, err := runCreateTextRecordCommand(
		ctx,
		serverAddress,
		caCertFile,
		aliceSessionFile,
		"Alice note",
		initialTextFile,
		initialMetadataFile,
	)
	if err != nil {
		t.Fatalf("create text record: %v", err)
	}
	if stderr != "" {
		t.Errorf("create stderr = %q, want empty output", stderr)
	}
	recordID, revision := parseCreatedTextRecordOutput(t, stdout)
	if revision != 1 {
		t.Fatalf("created revision = %d, want 1", revision)
	}
	aliceSecondSessionFile := filepath.Join(t.TempDir(), "alice-second-session.json")
	if _, _, err := runLoginCommand(
		ctx,
		serverAddress,
		caCertFile,
		aliceSecondSessionFile,
		" Alice ",
		testRegistrationPassword,
	); err != nil {
		t.Fatalf("login Alice second client: %v", err)
	}

	updatedTextFile := writeIntegrationFile(t, "updated-note.txt", "updated secret")
	updatedMetadataFile := writeIntegrationFile(t, "updated-metadata.txt", "updated private metadata")
	stdout, stderr, err = runUpdateTextRecordCommand(
		ctx,
		serverAddress,
		caCertFile,
		aliceSessionFile,
		recordID,
		1,
		"Updated Alice note",
		updatedTextFile,
		updatedMetadataFile,
	)
	if err != nil {
		t.Fatalf("update text record: %v", err)
	}
	wantUpdate := fmt.Sprintf("Updated text record %s to revision 2.\n", recordID)
	if stdout != wantUpdate {
		t.Errorf("update stdout = %q, want %q", stdout, wantUpdate)
	}
	if stderr != "" {
		t.Errorf("update stderr = %q, want empty output", stderr)
	}
	assertPlaintextAbsentFromPostgres(t, ctx, testPool, recordID, "updated secret", "updated private metadata")

	stdout, stderr, err = runGetRecordCommand(ctx, serverAddress, caCertFile, aliceSessionFile, recordID)
	if err != nil {
		t.Fatalf("get updated text record: %v", err)
	}
	if !strings.Contains(stdout, "Revision: 2") || !strings.Contains(stdout, "updated secret") {
		t.Errorf("get stdout = %q, want updated revision and text", stdout)
	}
	if stderr != "" {
		t.Errorf("get stderr = %q, want empty output", stderr)
	}

	_, _, err = runUpdateTextRecordCommand(
		ctx,
		serverAddress,
		caCertFile,
		aliceSecondSessionFile,
		recordID,
		1,
		"Stale Alice note",
		updatedTextFile,
		updatedMetadataFile,
	)
	assertCLIErrorContains(t, "stale update", err, "record revision conflict")

	_, _, err = runDeleteRecordCommand(ctx, serverAddress, caCertFile, aliceSessionFile, recordID, 1)
	assertCLIErrorContains(t, "stale delete", err, "record revision conflict")

	if _, _, err := runRegisterCommand(ctx, serverAddress, caCertFile, " Eve ", testRegistrationPassword); err != nil {
		t.Fatalf("register Eve: %v", err)
	}
	eveSessionFile := filepath.Join(t.TempDir(), "eve-session.json")
	if _, _, err := runLoginCommand(ctx, serverAddress, caCertFile, eveSessionFile, " Eve ", testRegistrationPassword); err != nil {
		t.Fatalf("login Eve: %v", err)
	}

	_, _, err = runUpdateTextRecordCommand(
		ctx,
		serverAddress,
		caCertFile,
		eveSessionFile,
		recordID,
		2,
		"Eve note",
		updatedTextFile,
		updatedMetadataFile,
	)
	assertCLIErrorContains(t, "foreign update", err, "record not found")

	_, _, err = runDeleteRecordCommand(ctx, serverAddress, caCertFile, eveSessionFile, recordID, 2)
	assertCLIErrorContains(t, "foreign delete", err, "record not found")

	stdout, stderr, err = runDeleteRecordCommand(ctx, serverAddress, caCertFile, aliceSessionFile, recordID, 2)
	if err != nil {
		t.Fatalf("delete text record: %v", err)
	}
	wantDelete := fmt.Sprintf("Deleted record %s.\n", recordID)
	if stdout != wantDelete {
		t.Errorf("delete stdout = %q, want %q", stdout, wantDelete)
	}
	if stderr != "" {
		t.Errorf("delete stderr = %q, want empty output", stderr)
	}

	_, _, err = runGetRecordCommand(ctx, serverAddress, caCertFile, aliceSessionFile, recordID)
	assertCLIErrorContains(t, "get deleted record", err, "record not found")

	for _, secret := range []string{
		testRegistrationPassword,
		"initial secret",
		"initial private metadata",
		"updated secret",
		"updated private metadata",
	} {
		if strings.Contains(httpLogs.String(), secret) {
			t.Errorf("HTTP logs contain secret %q", secret)
		}
	}
}

func runCreateTextRecordCommand(
	ctx context.Context,
	address string,
	caCertFile string,
	sessionFile string,
	title string,
	textFile string,
	metadataFile string,
) (string, string, error) {
	args := []string{
		"gkeep",
		"--address", address,
		"--ca-cert", caCertFile,
		"--session-file", sessionFile,
		"records", "create-text",
		"--title", title,
		"--text-file", textFile,
	}
	if metadataFile != "" {
		args = append(args, "--metadata-file", metadataFile)
	}

	return runClientCommand(ctx, args)
}

func runUpdateTextRecordCommand(
	ctx context.Context,
	address string,
	caCertFile string,
	sessionFile string,
	recordID string,
	revision int64,
	title string,
	textFile string,
	metadataFile string,
) (string, string, error) {
	args := []string{
		"gkeep",
		"--address", address,
		"--ca-cert", caCertFile,
		"--session-file", sessionFile,
		"records", "update-text", recordID,
		"--revision", fmt.Sprintf("%d", revision),
		"--title", title,
		"--text-file", textFile,
	}
	if metadataFile != "" {
		args = append(args, "--metadata-file", metadataFile)
	}

	return runClientCommand(ctx, args)
}

func runGetRecordCommand(
	ctx context.Context,
	address string,
	caCertFile string,
	sessionFile string,
	recordID string,
) (string, string, error) {
	return runClientCommand(ctx, []string{
		"gkeep",
		"--address", address,
		"--ca-cert", caCertFile,
		"--session-file", sessionFile,
		"records", "get", recordID,
	})
}

func runDeleteRecordCommand(
	ctx context.Context,
	address string,
	caCertFile string,
	sessionFile string,
	recordID string,
	revision int64,
) (string, string, error) {
	return runClientCommand(ctx, []string{
		"gkeep",
		"--address", address,
		"--ca-cert", caCertFile,
		"--session-file", sessionFile,
		"records", "delete", recordID,
		"--revision", fmt.Sprintf("%d", revision),
	})
}

func runClientCommand(ctx context.Context, args []string) (string, string, error) {
	var stdout bytes.Buffer
	var stderr bytes.Buffer

	err := clientcli.RunWithInput(
		ctx,
		args,
		strings.NewReader(""),
		&stdout,
		&stderr,
		buildinfo.Info{},
	)

	return stdout.String(), stderr.String(), err
}

func parseCreatedTextRecordOutput(t *testing.T, output string) (string, int64) {
	t.Helper()

	matches := createdRecordPattern.FindStringSubmatch(strings.TrimSpace(output))
	if matches == nil {
		t.Fatalf("created record output = %q, want created record message", output)
	}

	var revision int64
	if _, err := fmt.Sscanf(matches[2], "%d", &revision); err != nil {
		t.Fatalf("parse revision from %q: %v", matches[2], err)
	}

	return matches[1], revision
}

func assertCLIErrorContains(t *testing.T, operation string, err error, want string) {
	t.Helper()

	if err == nil {
		t.Fatalf("%s error = nil, want %q", operation, want)
	}
	if !strings.Contains(err.Error(), want) {
		t.Fatalf("%s error = %q, want %q", operation, err, want)
	}
}

func assertPlaintextAbsentFromPostgres(
	t *testing.T,
	ctx context.Context,
	pool *pgxpool.Pool,
	recordID string,
	secrets ...string,
) {
	t.Helper()

	var ciphertext []byte
	if err := pool.QueryRow(
		ctx,
		`SELECT ciphertext FROM gopherkeeper.records WHERE id = $1`,
		recordID,
	).Scan(&ciphertext); err != nil {
		t.Fatalf("read record ciphertext: %v", err)
	}

	for _, secret := range secrets {
		if bytes.Contains(ciphertext, []byte(secret)) {
			t.Errorf("PostgreSQL ciphertext contains plaintext secret %q", secret)
		}
	}
}

func writeIntegrationFile(t *testing.T, filename string, content string) string {
	t.Helper()

	path := filepath.Join(t.TempDir(), filename)
	if err := os.WriteFile(path, []byte(content), 0o600); err != nil {
		t.Fatalf("write integration file: %v", err)
	}

	return path
}
