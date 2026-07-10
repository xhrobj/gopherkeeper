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
	flow := newCLITextRecordFlow(t)
	recordID := flow.createInitialRecord()

	flow.loginSecondAliceClient()
	flow.updateRecord(recordID)
	flow.assertUpdatedRecord(recordID)
	flow.assertStaleConflicts(recordID)
	flow.loginEve()
	flow.assertForeignAccessHidden(recordID)
	flow.deleteRecord(recordID)
	flow.assertDeleted(recordID)
	flow.assertHTTPLogsDoNotContainSecrets()
}

type cliTextRecordFlow struct {
	t                      *testing.T
	ctx                    context.Context
	serverAddress          string
	caCertFile             string
	pool                   *pgxpool.Pool
	httpLogs               *bytes.Buffer
	aliceSessionFile       string
	aliceSecondSessionFile string
	eveSessionFile         string
	updatedTextFile        string
	updatedMetadataFile    string
}

func newCLITextRecordFlow(t *testing.T) *cliTextRecordFlow {
	t.Helper()

	ctx, pool, httpLogs, serverAddress, caCertFile := startCLITextRecordTestServer(t)
	if _, _, err := runRegisterCommand(ctx, serverAddress, caCertFile, " Alice ", testRegistrationPassword); err != nil {
		t.Fatalf("register Alice: %v", err)
	}

	aliceSessionFile := filepath.Join(t.TempDir(), "alice-session.json")
	loginTestUser(t, ctx, serverAddress, caCertFile, aliceSessionFile, " Alice ")

	return &cliTextRecordFlow{
		t:                   t,
		ctx:                 ctx,
		serverAddress:       serverAddress,
		caCertFile:          caCertFile,
		pool:                pool,
		httpLogs:            httpLogs,
		aliceSessionFile:    aliceSessionFile,
		updatedTextFile:     writeIntegrationFile(t, "updated-note.txt", "updated secret"),
		updatedMetadataFile: writeIntegrationFile(t, "updated-metadata.txt", "updated private metadata"),
	}
}

func startCLITextRecordTestServer(
	t *testing.T,
) (context.Context, *pgxpool.Pool, *bytes.Buffer, string, string) {
	t.Helper()

	dsn := os.Getenv("DATABASE_DSN")
	if dsn == "" {
		t.Fatal("DATABASE_DSN is not set")
	}
	isolateClientConfig(t)
	t.Setenv("ADDRESS", "")
	t.Setenv("CA_CERT_FILE", "")
	t.Setenv("SESSION_FILE", "")

	ctx, cancel := context.WithTimeout(context.Background(), integrationTestTimeout)
	t.Cleanup(cancel)

	pool := openIsolatedMigratedDatabase(t, ctx, dsn)
	httpLogs := &bytes.Buffer{}
	logger := newIntegrationLogger(httpLogs)
	t.Cleanup(func() { _ = logger.Sync() })

	caCertFile, serverCertFile, serverKeyFile := generateTLSFiles(t)
	serverAddress, stopServer := startHTTPSServer(
		t,
		middleware.WithLogging(newAuthenticatedServerHandler(t, pool), logger),
		serverCertFile,
		serverKeyFile,
	)
	t.Cleanup(stopServer)

	return ctx, pool, httpLogs, serverAddress, caCertFile
}

func openIsolatedMigratedDatabase(t *testing.T, ctx context.Context, dsn string) *pgxpool.Pool {
	t.Helper()

	adminPool := openPostgres(t, ctx, dsn)
	t.Cleanup(adminPool.Close)

	databaseName := createTestDatabase(t, ctx, adminPool)
	t.Cleanup(func() { dropTestDatabase(t, adminPool, databaseName) })

	testPool := openTestPostgres(t, ctx, dsn, databaseName)
	t.Cleanup(testPool.Close)

	if err := migration.Run(testPool); err != nil {
		t.Fatalf("migration.Run() error = %v", err)
	}

	return testPool
}

func loginTestUser(
	t *testing.T,
	ctx context.Context,
	serverAddress, caCertFile, sessionFile, login string,
) {
	t.Helper()

	if _, _, err := runLoginCommand(ctx, serverAddress, caCertFile, sessionFile, login, testRegistrationPassword); err != nil {
		t.Fatalf("login %s: %v", strings.TrimSpace(login), err)
	}
}

func (f *cliTextRecordFlow) createInitialRecord() string {
	f.t.Helper()

	initialTextFile := writeIntegrationFile(f.t, "initial-note.txt", "initial secret")
	initialMetadataFile := writeIntegrationFile(f.t, "initial-metadata.txt", "initial private metadata")
	stdout, stderr, err := runCreateTextRecordCommand(
		f.ctx,
		f.serverAddress,
		f.caCertFile,
		f.aliceSessionFile,
		"Alice note",
		initialTextFile,
		initialMetadataFile,
	)
	if err != nil {
		f.t.Fatalf("create text record: %v", err)
	}
	if stderr != "" {
		f.t.Errorf("create stderr = %q, want empty output", stderr)
	}

	recordID, revision := parseCreatedTextRecordOutput(f.t, stdout)
	if revision != 1 {
		f.t.Fatalf("created revision = %d, want 1", revision)
	}

	return recordID
}

func (f *cliTextRecordFlow) loginSecondAliceClient() {
	f.t.Helper()

	f.aliceSecondSessionFile = filepath.Join(f.t.TempDir(), "alice-second-session.json")
	loginTestUser(f.t, f.ctx, f.serverAddress, f.caCertFile, f.aliceSecondSessionFile, " Alice ")
}

func (f *cliTextRecordFlow) updateRecord(recordID string) {
	f.t.Helper()

	stdout, stderr, err := runUpdateTextRecordCommand(
		f.ctx,
		f.serverAddress,
		f.caCertFile,
		f.aliceSessionFile,
		textRecordUpdateCLIRequest{
			recordID:     recordID,
			revision:     1,
			title:        "Updated Alice note",
			textFile:     f.updatedTextFile,
			metadataFile: f.updatedMetadataFile,
		},
	)
	if err != nil {
		f.t.Fatalf("update text record: %v", err)
	}
	wantUpdate := fmt.Sprintf("Updated text record %s to revision 2.\n", recordID)
	if stdout != wantUpdate {
		f.t.Errorf("update stdout = %q, want %q", stdout, wantUpdate)
	}
	if stderr != "" {
		f.t.Errorf("update stderr = %q, want empty output", stderr)
	}
	assertPlaintextAbsentFromPostgres(f.t, f.ctx, f.pool, recordID, "updated secret", "updated private metadata")
}

func (f *cliTextRecordFlow) assertUpdatedRecord(recordID string) {
	f.t.Helper()

	stdout, stderr, err := runGetRecordCommand(f.ctx, f.serverAddress, f.caCertFile, f.aliceSessionFile, recordID)
	if err != nil {
		f.t.Fatalf("get updated text record: %v", err)
	}
	if !strings.Contains(stdout, "Revision: 2") || !strings.Contains(stdout, "updated secret") {
		f.t.Errorf("get stdout = %q, want updated revision and text", stdout)
	}
	if stderr != "" {
		f.t.Errorf("get stderr = %q, want empty output", stderr)
	}
}

func (f *cliTextRecordFlow) assertStaleConflicts(recordID string) {
	f.t.Helper()

	_, _, err := runUpdateTextRecordCommand(
		f.ctx,
		f.serverAddress,
		f.caCertFile,
		f.aliceSecondSessionFile,
		textRecordUpdateCLIRequest{
			recordID:     recordID,
			revision:     1,
			title:        "Stale Alice note",
			textFile:     f.updatedTextFile,
			metadataFile: f.updatedMetadataFile,
		},
	)
	assertCLIErrorContains(f.t, "stale update", err, "record revision conflict")

	_, _, err = runDeleteRecordCommand(f.ctx, f.serverAddress, f.caCertFile, f.aliceSessionFile, recordID, 1)
	assertCLIErrorContains(f.t, "stale delete", err, "record revision conflict")
}

func (f *cliTextRecordFlow) loginEve() {
	f.t.Helper()

	if _, _, err := runRegisterCommand(f.ctx, f.serverAddress, f.caCertFile, " Eve ", testRegistrationPassword); err != nil {
		f.t.Fatalf("register Eve: %v", err)
	}
	f.eveSessionFile = filepath.Join(f.t.TempDir(), "eve-session.json")
	loginTestUser(f.t, f.ctx, f.serverAddress, f.caCertFile, f.eveSessionFile, " Eve ")
}

func (f *cliTextRecordFlow) assertForeignAccessHidden(recordID string) {
	f.t.Helper()

	_, _, err := runUpdateTextRecordCommand(
		f.ctx,
		f.serverAddress,
		f.caCertFile,
		f.eveSessionFile,
		textRecordUpdateCLIRequest{
			recordID:     recordID,
			revision:     2,
			title:        "Eve note",
			textFile:     f.updatedTextFile,
			metadataFile: f.updatedMetadataFile,
		},
	)
	assertCLIErrorContains(f.t, "foreign update", err, "record not found")

	_, _, err = runDeleteRecordCommand(f.ctx, f.serverAddress, f.caCertFile, f.eveSessionFile, recordID, 2)
	assertCLIErrorContains(f.t, "foreign delete", err, "record not found")
}

func (f *cliTextRecordFlow) deleteRecord(recordID string) {
	f.t.Helper()

	stdout, stderr, err := runDeleteRecordCommand(f.ctx, f.serverAddress, f.caCertFile, f.aliceSessionFile, recordID, 2)
	if err != nil {
		f.t.Fatalf("delete text record: %v", err)
	}
	wantDelete := fmt.Sprintf("Deleted record %s.\n", recordID)
	if stdout != wantDelete {
		f.t.Errorf("delete stdout = %q, want %q", stdout, wantDelete)
	}
	if stderr != "" {
		f.t.Errorf("delete stderr = %q, want empty output", stderr)
	}
}

func (f *cliTextRecordFlow) assertDeleted(recordID string) {
	f.t.Helper()

	_, _, err := runGetRecordCommand(f.ctx, f.serverAddress, f.caCertFile, f.aliceSessionFile, recordID)
	assertCLIErrorContains(f.t, "get deleted record", err, "record not found")
}

func (f *cliTextRecordFlow) assertHTTPLogsDoNotContainSecrets() {
	f.t.Helper()

	for _, secret := range []string{
		testRegistrationPassword,
		"initial secret",
		"initial private metadata",
		"updated secret",
		"updated private metadata",
	} {
		if strings.Contains(f.httpLogs.String(), secret) {
			f.t.Errorf("HTTP logs contain secret %q", secret)
		}
	}
}

func runCreateTextRecordCommand(
	ctx context.Context,
	address, caCertFile, sessionFile string,
	title, textFile, metadataFile string,
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

type textRecordUpdateCLIRequest struct {
	recordID     string
	revision     int64
	title        string
	textFile     string
	metadataFile string
}

func runUpdateTextRecordCommand(
	ctx context.Context,
	address, caCertFile, sessionFile string,
	request textRecordUpdateCLIRequest,
) (string, string, error) {
	args := []string{
		"gkeep",
		"--address", address,
		"--ca-cert", caCertFile,
		"--session-file", sessionFile,
		"records", "update-text", request.recordID,
		"--revision", fmt.Sprintf("%d", request.revision),
		"--title", request.title,
		"--text-file", request.textFile,
	}
	if request.metadataFile != "" {
		args = append(args, "--metadata-file", request.metadataFile)
	}

	return runClientCommand(ctx, args)
}

func runGetRecordCommand(
	ctx context.Context,
	address, caCertFile, sessionFile, recordID string,
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
	address, caCertFile, sessionFile, recordID string,
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

func writeIntegrationFile(t *testing.T, filename, content string) string {
	t.Helper()

	path := filepath.Join(t.TempDir(), filename)
	if err := os.WriteFile(path, []byte(content), 0o600); err != nil {
		t.Fatalf("write integration file: %v", err)
	}

	return path
}
