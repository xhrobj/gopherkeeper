//go:build integration

package integration_test

import (
	"context"
	"errors"
	"fmt"
	"net/http"
	"os"
	"path/filepath"
	"strings"
	"testing"

	"github.com/xhrobj/gopherkeeper/internal/client/cache"
)

const (
	multiDeviceInitialText     = "initial multi-device secret"
	multiDeviceInitialMetadata = "initial multi-device metadata"
	multiDeviceUpdatedText     = "updated multi-device secret"
	multiDeviceUpdatedMetadata = "updated multi-device metadata"
	multiDeviceOfflineWarning  = "Source: encrypted local cache (data may be stale)."
)

func TestIntegration_CLITwoDeviceOfflineAndConflictFlow(t *testing.T) {
	flow := newMultiDeviceOfflineFlow(t)
	recordID := flow.createAndSynchronizeRecord()

	flow.stopServerAndReadOffline(recordID)
	flow.restartServerAndCreateConflict(recordID)
	flow.assertStaleCacheIsNotReplaced(recordID)
	flow.refreshSecondClient(recordID)
	flow.assertIndependentCaches(recordID)
}

type multiDeviceCLIClient struct {
	sessionFile string
	cacheDir    string
}

type multiDeviceOfflineFlow struct {
	t          *testing.T
	ctx        context.Context
	server     *restartableHTTPSServer
	address    string
	caCertFile string
	first      multiDeviceCLIClient
	second     multiDeviceCLIClient
}

func newMultiDeviceOfflineFlow(t *testing.T) *multiDeviceOfflineFlow {
	t.Helper()

	dsn := os.Getenv("DATABASE_DSN")
	if dsn == "" {
		t.Fatal("DATABASE_DSN is not set")
	}

	isolateClientConfig(t)
	t.Setenv("ADDRESS", "")
	t.Setenv("CA_CERT_FILE", "")
	t.Setenv("SESSION_FILE", "")
	t.Setenv("CACHE_DIR", "")

	ctx, cancel := context.WithTimeout(context.Background(), 2*integrationTestTimeout)
	t.Cleanup(cancel)

	pool := openIsolatedMigratedDatabase(t, ctx, dsn)
	caCertFile, serverCertFile, serverKeyFile := generateTLSFiles(t)
	server := newRestartableHTTPSServer(
		t,
		func() http.Handler { return newAuthenticatedServerHandler(t, pool) },
		serverCertFile,
		serverKeyFile,
	)

	if stdout, stderr, err := runRegisterCommand(
		ctx,
		server.Address(),
		caCertFile,
		" Alice ",
		testRegistrationPassword,
	); err != nil {
		t.Fatalf("register Alice: %v", err)
	} else if stdout != "User alice registered successfully.\n" || stderr != "" {
		t.Fatalf("register Alice output = %q, stderr = %q", stdout, stderr)
	}

	first := multiDeviceCLIClient{
		sessionFile: filepath.Join(t.TempDir(), "client-a-session.json"),
		cacheDir:    filepath.Join(t.TempDir(), "client-a-cache"),
	}
	second := multiDeviceCLIClient{
		sessionFile: filepath.Join(t.TempDir(), "client-b-session.json"),
		cacheDir:    filepath.Join(t.TempDir(), "client-b-cache"),
	}
	loginTestUser(t, ctx, server.Address(), caCertFile, first.sessionFile, " Alice ")
	loginTestUser(t, ctx, server.Address(), caCertFile, second.sessionFile, "ALICE")

	return &multiDeviceOfflineFlow{
		t:          t,
		ctx:        ctx,
		server:     server,
		address:    server.Address(),
		caCertFile: caCertFile,
		first:      first,
		second:     second,
	}
}

func (flow *multiDeviceOfflineFlow) createAndSynchronizeRecord() string {
	flow.t.Helper()

	textFile := writeIntegrationFile(flow.t, "multi-device-initial.txt", multiDeviceInitialText)
	metadataFile := writeIntegrationFile(flow.t, "multi-device-initial-metadata.txt", multiDeviceInitialMetadata)
	stdout, stderr, err := runCreateTextRecordCommand(
		flow.ctx,
		flow.address,
		flow.caCertFile,
		flow.first.sessionFile,
		"Shared note",
		textFile,
		metadataFile,
	)
	if err != nil {
		flow.t.Fatalf("create shared record: %v", err)
	}
	if stderr != "" {
		flow.t.Errorf("create shared record stderr = %q, want empty output", stderr)
	}
	recordID, revision := parseCreatedTextRecordOutput(flow.t, stdout)
	if revision != 1 {
		flow.t.Fatalf("created shared record revision = %d, want 1", revision)
	}

	flow.assertSync(flow.first, false, "Added: 1", "Updated: 0", "Stale: 0")
	flow.assertSync(flow.second, false, "Added: 1", "Updated: 0", "Stale: 0")

	firstLocation := flow.cacheLocation(flow.first)
	secondLocation := flow.cacheLocation(flow.second)
	if firstLocation.AccountID != secondLocation.AccountID {
		flow.t.Fatalf(
			"account IDs differ for the same Server and login: %q != %q",
			firstLocation.AccountID,
			secondLocation.AccountID,
		)
	}
	if firstLocation.Directory == secondLocation.Directory {
		flow.t.Fatal("two clients share one physical cache directory")
	}

	return recordID
}

func (flow *multiDeviceOfflineFlow) stopServerAndReadOffline(recordID string) {
	flow.t.Helper()

	flow.server.Stop()

	stdout, stderr, err := runMultiDeviceOnlineGet(
		flow.ctx,
		flow.address,
		flow.caCertFile,
		flow.second,
		recordID,
	)
	assertNetworkError(flow.t, err)
	if stdout != "" || stderr != "" {
		flow.t.Errorf("online get while stopped output = %q, stderr = %q, want empty", stdout, stderr)
	}
	if strings.Contains(err.Error(), multiDeviceInitialText) ||
		strings.Contains(stdout+stderr, multiDeviceOfflineWarning) {
		flow.t.Fatalf("online get used or exposed offline data: error = %q, output = %q", err, stdout+stderr)
	}

	listOutput := flow.runOfflineList(flow.second)
	for _, want := range []string{multiDeviceOfflineWarning, recordID, "Shared note", "1"} {
		if !strings.Contains(listOutput, want) {
			flow.t.Errorf("offline list output = %q, want %q", listOutput, want)
		}
	}
	if strings.Contains(listOutput, multiDeviceInitialText) {
		flow.t.Errorf("offline list exposes private payload %q", multiDeviceInitialText)
	}

	flow.assertOfflineRecord(flow.second, recordID, 1, multiDeviceInitialText, multiDeviceUpdatedText)
}

func (flow *multiDeviceOfflineFlow) restartServerAndCreateConflict(recordID string) {
	flow.t.Helper()

	flow.server.Start()
	if flow.server.Address() != flow.address {
		flow.t.Fatalf("restarted Server address = %q, want %q", flow.server.Address(), flow.address)
	}
	if _, _, err := runWhoamiCommand(
		flow.ctx,
		flow.address,
		flow.caCertFile,
		flow.second.sessionFile,
	); err != nil {
		flow.t.Fatalf("second Client session after restart: %v", err)
	}

	updatedTextFile := writeIntegrationFile(flow.t, "multi-device-updated.txt", multiDeviceUpdatedText)
	updatedMetadataFile := writeIntegrationFile(
		flow.t,
		"multi-device-updated-metadata.txt",
		multiDeviceUpdatedMetadata,
	)
	stdout, stderr, err := runUpdateTextRecordCommand(
		flow.ctx,
		flow.address,
		flow.caCertFile,
		flow.first.sessionFile,
		textRecordUpdateCLIRequest{
			recordID:     recordID,
			revision:     1,
			title:        "Updated shared note",
			textFile:     updatedTextFile,
			metadataFile: updatedMetadataFile,
		},
	)
	if err != nil {
		flow.t.Fatalf("first Client update after restart: %v", err)
	}
	wantUpdate := fmt.Sprintf("Updated text record %s to revision 2.\n", recordID)
	if stdout != wantUpdate || stderr != "" {
		flow.t.Errorf("first Client update output = %q, stderr = %q, want %q", stdout, stderr, wantUpdate)
	}

	staleTextFile := writeIntegrationFile(flow.t, "multi-device-stale.txt", "stale client B secret")
	stdout, stderr, err = runUpdateTextRecordCommand(
		flow.ctx,
		flow.address,
		flow.caCertFile,
		flow.second.sessionFile,
		textRecordUpdateCLIRequest{
			recordID: recordID,
			revision: 1,
			title:    "Stale shared note",
			textFile: staleTextFile,
		},
	)
	assertCLIErrorContains(flow.t, "second Client stale update", err, "record revision conflict")
	if stdout != "" || stderr != "" {
		flow.t.Errorf("stale update output = %q, stderr = %q, want empty", stdout, stderr)
	}
}

func (flow *multiDeviceOfflineFlow) assertStaleCacheIsNotReplaced(recordID string) {
	flow.t.Helper()

	output := flow.assertSync(
		flow.second,
		false,
		"Added: 0",
		"Updated: 0",
		"Stale: 1",
		"LOCAL REVISION",
		"SERVER REVISION",
		"Run `gkeep sync --refresh` to update stale records.",
	)
	for _, want := range []string{recordID, "Updated shared note", "1", "2"} {
		if !strings.Contains(output, want) {
			flow.t.Errorf("stale synchronization output = %q, want %q", output, want)
		}
	}

	flow.assertOfflineRecord(flow.second, recordID, 1, multiDeviceInitialText, multiDeviceUpdatedText)
}

func (flow *multiDeviceOfflineFlow) refreshSecondClient(recordID string) {
	flow.t.Helper()

	flow.assertSync(flow.second, true, "Added: 0", "Updated: 1", "Stale: 0")
	flow.assertOfflineRecord(flow.second, recordID, 2, multiDeviceUpdatedText, multiDeviceInitialText)
}

func (flow *multiDeviceOfflineFlow) assertIndependentCaches(recordID string) {
	flow.t.Helper()

	// Refreshing Client B must not mutate Client A's independent local cache.
	flow.assertOfflineRecord(flow.first, recordID, 1, multiDeviceInitialText, multiDeviceUpdatedText)
}

func (flow *multiDeviceOfflineFlow) assertSync(
	client multiDeviceCLIClient,
	refresh bool,
	want ...string,
) string {
	flow.t.Helper()

	stdout, stderr, err := runMultiDeviceSync(
		flow.ctx,
		flow.address,
		flow.caCertFile,
		client,
		testRegistrationPassword,
		refresh,
	)
	if err != nil {
		flow.t.Fatalf("synchronize Client refresh=%t: %v", refresh, err)
	}
	if stderr != "" {
		flow.t.Errorf("synchronize Client stderr = %q, want empty output", stderr)
	}
	for _, value := range append([]string{"Cache synchronization completed."}, want...) {
		if !strings.Contains(stdout, value) {
			flow.t.Errorf("synchronization output = %q, want %q", stdout, value)
		}
	}
	if strings.Contains(stdout+stderr, testRegistrationPassword) {
		flow.t.Error("synchronization output exposes cache password")
	}

	return stdout
}

func (flow *multiDeviceOfflineFlow) assertOfflineRecord(
	client multiDeviceCLIClient,
	recordID string,
	revision int64,
	wantText string,
	absentText string,
) {
	flow.t.Helper()

	stdout, stderr, err := runMultiDeviceOfflineGet(
		flow.ctx,
		flow.address,
		flow.caCertFile,
		client,
		" Alice ",
		testRegistrationPassword,
		recordID,
	)
	if err != nil {
		flow.t.Fatalf("offline get revision %d: %v", revision, err)
	}
	if stderr != "" {
		flow.t.Errorf("offline get stderr = %q, want empty output", stderr)
	}
	for _, want := range []string{
		multiDeviceOfflineWarning,
		fmt.Sprintf("Revision: %d", revision),
		wantText,
	} {
		if !strings.Contains(stdout, want) {
			flow.t.Errorf("offline get output = %q, want %q", stdout, want)
		}
	}
	if strings.Contains(stdout, absentText) {
		flow.t.Errorf("offline get output = %q, must not contain %q", stdout, absentText)
	}
	if strings.Contains(stdout+stderr, testRegistrationPassword) {
		flow.t.Error("offline get output exposes cache password")
	}
}

func (flow *multiDeviceOfflineFlow) runOfflineList(client multiDeviceCLIClient) string {
	flow.t.Helper()

	stdout, stderr, err := runClientCommandWithInput(flow.ctx, []string{
		"gkeep",
		"--address", flow.address,
		"--ca-cert", flow.caCertFile,
		"--session-file", client.sessionFile,
		"--cache-dir", client.cacheDir,
		"records", "list", "--offline", "--login", " Alice ",
	}, testRegistrationPassword+"\n")
	if err != nil {
		flow.t.Fatalf("offline list: %v", err)
	}
	if stderr != "" {
		flow.t.Errorf("offline list stderr = %q, want empty output", stderr)
	}

	return stdout
}

func (flow *multiDeviceOfflineFlow) cacheLocation(client multiDeviceCLIClient) cache.Location {
	flow.t.Helper()

	location, err := cache.ResolveLocation(client.cacheDir, flow.address, "alice")
	if err != nil {
		flow.t.Fatalf("resolve local cache location: %v", err)
	}
	return location
}

func runMultiDeviceSync(
	ctx context.Context,
	address string,
	caCertFile string,
	client multiDeviceCLIClient,
	password string,
	refresh bool,
) (string, string, error) {
	args := []string{
		"gkeep",
		"--address", address,
		"--ca-cert", caCertFile,
		"--session-file", client.sessionFile,
		"--cache-dir", client.cacheDir,
		"sync",
	}
	if refresh {
		args = append(args, "--refresh")
	}

	return runClientCommandWithInput(ctx, args, password+"\n")
}

func runMultiDeviceOnlineGet(
	ctx context.Context,
	address string,
	caCertFile string,
	client multiDeviceCLIClient,
	recordID string,
) (string, string, error) {
	return runClientCommand(ctx, []string{
		"gkeep",
		"--address", address,
		"--ca-cert", caCertFile,
		"--session-file", client.sessionFile,
		"--cache-dir", client.cacheDir,
		"records", "get", recordID,
	})
}

func runMultiDeviceOfflineGet(
	ctx context.Context,
	address string,
	caCertFile string,
	client multiDeviceCLIClient,
	login string,
	password string,
	recordID string,
) (string, string, error) {
	return runClientCommandWithInput(ctx, []string{
		"gkeep",
		"--address", address,
		"--ca-cert", caCertFile,
		"--session-file", client.sessionFile,
		"--cache-dir", client.cacheDir,
		"records", "get", recordID,
		"--offline", "--login", login,
	}, password+"\n")
}

func assertNetworkError(t *testing.T, err error) {
	t.Helper()

	if err == nil {
		t.Fatal("network error = nil")
	}
	if errors.Is(err, context.Canceled) || errors.Is(err, context.DeadlineExceeded) {
		t.Fatalf("network operation ended with context error: %v", err)
	}
}
