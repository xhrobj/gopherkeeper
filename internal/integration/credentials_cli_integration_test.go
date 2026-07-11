//go:build integration

package integration_test

import (
	"context"
	"encoding/json"
	"fmt"
	"path/filepath"
	"regexp"
	"strings"
	"testing"

	"github.com/xhrobj/gopherkeeper/internal/model"
)

var createdCredentialsRecordPattern = regexp.MustCompile(
	`^Created credentials record ([0-9a-f-]+) with revision ([0-9]+)\.$`,
)

type credentialsCLIConfig struct {
	ctx         context.Context
	address     string
	caCertFile  string
	sessionFile string
}

func TestIntegration_CLICredentialsRecordFlow(t *testing.T) {
	ctx, pool, httpLogs, serverAddress, caCertFile := startCLIRecordTestServer(t)
	if _, _, err := runRegisterCommand(
		ctx,
		serverAddress,
		caCertFile,
		" Alice ",
		testRegistrationPassword,
	); err != nil {
		t.Fatalf("register Alice: %v", err)
	}

	aliceSessionFile := filepath.Join(t.TempDir(), "alice-session.json")
	loginTestUser(t, ctx, serverAddress, caCertFile, aliceSessionFile, " Alice ")
	aliceCLI := credentialsCLIConfig{
		ctx:         ctx,
		address:     serverAddress,
		caCertFile:  caCertFile,
		sessionFile: aliceSessionFile,
	}

	initial := model.CredentialsPayload{
		Login:    "alice@example.com",
		Password: "initial-github-secret",
		URL:      "https://github.com/login",
		Metadata: "personal account recovery codes",
	}
	recordID := createCredentialsRecord(
		t,
		ctx,
		serverAddress,
		caCertFile,
		aliceSessionFile,
		"Alice GitHub",
		initial,
	)
	assertPlaintextAbsentFromPostgres(
		t,
		ctx,
		pool,
		recordID,
		initial.Login,
		initial.Password,
		initial.URL,
		initial.Metadata,
	)
	assertCredentialsList(t, ctx, serverAddress, caCertFile, aliceSessionFile, recordID, initial)
	assertCredentialsRecord(t, aliceCLI, recordID, 1, initial)

	aliceSecondSessionFile := filepath.Join(t.TempDir(), "alice-second-session.json")
	loginTestUser(t, ctx, serverAddress, caCertFile, aliceSecondSessionFile, " Alice ")
	aliceSecondCLI := credentialsCLIConfig{
		ctx:         ctx,
		address:     serverAddress,
		caCertFile:  caCertFile,
		sessionFile: aliceSecondSessionFile,
	}

	updated := model.CredentialsPayload{
		Login:    "alice.updated@example.com",
		Password: "updated-github-secret",
		URL:      "https://github.com/settings/security",
		Metadata: "updated recovery codes",
	}
	updateCredentialsRecord(t, aliceCLI, recordID, 1, "Updated Alice GitHub", updated)
	assertPlaintextAbsentFromPostgres(
		t,
		ctx,
		pool,
		recordID,
		updated.Login,
		updated.Password,
		updated.URL,
		updated.Metadata,
	)
	assertCredentialsRecord(t, aliceCLI, recordID, 2, updated)
	assertStaleCredentialsUpdate(
		t,
		aliceSecondCLI,
		recordID,
		updated,
	)

	eveSessionFile := registerAndLoginEve(t, ctx, serverAddress, caCertFile)
	assertForeignCredentialsAccessHidden(
		t,
		ctx,
		serverAddress,
		caCertFile,
		eveSessionFile,
		recordID,
		updated,
	)

	stdout, stderr, err := runDeleteRecordCommand(
		ctx,
		serverAddress,
		caCertFile,
		aliceSessionFile,
		recordID,
		2,
	)
	if err != nil {
		t.Fatalf("delete credentials record: %v", err)
	}
	wantDelete := fmt.Sprintf("Deleted record %s.\n", recordID)
	if stdout != wantDelete {
		t.Errorf("delete stdout = %q, want %q", stdout, wantDelete)
	}
	if stderr != "" {
		t.Errorf("delete stderr = %q, want empty output", stderr)
	}

	_, _, err = runGetRecordCommand(ctx, serverAddress, caCertFile, aliceSessionFile, recordID)
	assertCLIErrorContains(t, "get deleted credentials record", err, "record not found")
	assertCredentialsHTTPLogsDoNotContainSecrets(
		t,
		httpLogs.String(),
		initial,
		updated,
	)
}

func createCredentialsRecord(
	t *testing.T,
	ctx context.Context,
	address string,
	caCertFile string,
	sessionFile string,
	title string,
	payload model.CredentialsPayload,
) string {
	t.Helper()

	stdout, stderr, err := runCreateCredentialsRecordCommand(
		t,
		ctx,
		address,
		caCertFile,
		sessionFile,
		title,
		payload,
	)
	if err != nil {
		t.Fatalf("create credentials record: %v", err)
	}
	assertOutputDoesNotContainCredentials(t, stdout+stderr, payload)
	if stderr != "" {
		t.Errorf("create stderr = %q, want empty output", stderr)
	}

	matches := createdCredentialsRecordPattern.FindStringSubmatch(strings.TrimSpace(stdout))
	if matches == nil {
		t.Fatalf("created credentials output = %q, want created record message", stdout)
	}
	if matches[2] != "1" {
		t.Fatalf("created revision = %s, want 1", matches[2])
	}

	return matches[1]
}

func updateCredentialsRecord(
	t *testing.T,
	config credentialsCLIConfig,
	recordID string,
	revision int64,
	title string,
	payload model.CredentialsPayload,
) {
	t.Helper()

	stdout, stderr, err := runUpdateCredentialsRecordCommand(
		t,
		config,
		recordID,
		revision,
		title,
		payload,
	)
	if err != nil {
		t.Fatalf("update credentials record: %v", err)
	}
	assertOutputDoesNotContainCredentials(t, stdout+stderr, payload)

	wantUpdate := fmt.Sprintf("Updated credentials record %s to revision 2.\n", recordID)
	if stdout != wantUpdate {
		t.Errorf("update stdout = %q, want %q", stdout, wantUpdate)
	}
	if stderr != "" {
		t.Errorf("update stderr = %q, want empty output", stderr)
	}
}

func assertCredentialsList(
	t *testing.T,
	ctx context.Context,
	address string,
	caCertFile string,
	sessionFile string,
	recordID string,
	payload model.CredentialsPayload,
) {
	t.Helper()

	stdout, stderr, err := runListRecordsCommand(ctx, address, caCertFile, sessionFile)
	if err != nil {
		t.Fatalf("list credentials records: %v", err)
	}
	if stderr != "" {
		t.Errorf("list stderr = %q, want empty output", stderr)
	}
	for _, want := range []string{recordID, "credentials", "Alice GitHub", "1"} {
		if !strings.Contains(stdout, want) {
			t.Errorf("list stdout = %q, want %q", stdout, want)
		}
	}
	assertOutputDoesNotContainCredentials(t, stdout, payload)
}

func assertCredentialsRecord(
	t *testing.T,
	config credentialsCLIConfig,
	recordID string,
	revision int64,
	payload model.CredentialsPayload,
) {
	t.Helper()

	stdout, stderr, err := runGetRecordCommand(
		config.ctx,
		config.address,
		config.caCertFile,
		config.sessionFile,
		recordID,
	)
	if err != nil {
		t.Fatalf("get credentials record: %v", err)
	}
	if stderr != "" {
		t.Errorf("get stderr = %q, want empty output", stderr)
	}

	wants := []struct {
		description string
		value       string
	}{
		{description: "record type", value: "Type: credentials"},
		{description: "record revision", value: fmt.Sprintf("Revision: %d", revision)},
		{description: "credentials login", value: "Login: " + payload.Login},
		{description: "credentials password", value: "Password: " + payload.Password},
		{description: "credentials URL", value: "URL: " + payload.URL},
		{description: "credentials metadata", value: payload.Metadata},
	}
	for _, want := range wants {
		if !strings.Contains(stdout, want.value) {
			t.Errorf("get output does not contain %s", want.description)
		}
	}
}

func assertStaleCredentialsUpdate(
	t *testing.T,
	config credentialsCLIConfig,
	recordID string,
	payload model.CredentialsPayload,
) {
	t.Helper()

	stdout, stderr, err := runUpdateCredentialsRecordCommand(
		t,
		config,
		recordID,
		1,
		"Stale credentials",
		payload,
	)
	assertCLIErrorContains(t, "stale credentials update", err, "record revision conflict")
	assertOutputDoesNotContainCredentials(t, stdout+stderr+err.Error(), payload)
}

func registerAndLoginEve(
	t *testing.T,
	ctx context.Context,
	address string,
	caCertFile string,
) string {
	t.Helper()

	if _, _, err := runRegisterCommand(
		ctx,
		address,
		caCertFile,
		" Eve ",
		testRegistrationPassword,
	); err != nil {
		t.Fatalf("register Eve: %v", err)
	}

	sessionFile := filepath.Join(t.TempDir(), "eve-session.json")
	loginTestUser(t, ctx, address, caCertFile, sessionFile, " Eve ")

	return sessionFile
}

func assertForeignCredentialsAccessHidden(
	t *testing.T,
	ctx context.Context,
	address string,
	caCertFile string,
	sessionFile string,
	recordID string,
	payload model.CredentialsPayload,
) {
	t.Helper()

	_, _, err := runGetRecordCommand(ctx, address, caCertFile, sessionFile, recordID)
	assertCLIErrorContains(t, "foreign credentials get", err, "record not found")

	config := credentialsCLIConfig{
		ctx:         ctx,
		address:     address,
		caCertFile:  caCertFile,
		sessionFile: sessionFile,
	}
	stdout, stderr, err := runUpdateCredentialsRecordCommand(
		t,
		config,
		recordID,
		2,
		"Eve credentials",
		payload,
	)
	assertCLIErrorContains(t, "foreign credentials update", err, "record not found")
	assertOutputDoesNotContainCredentials(t, stdout+stderr+err.Error(), payload)

	_, _, err = runDeleteRecordCommand(ctx, address, caCertFile, sessionFile, recordID, 2)
	assertCLIErrorContains(t, "foreign credentials delete", err, "record not found")
}

func runCreateCredentialsRecordCommand(
	t *testing.T,
	ctx context.Context,
	address string,
	caCertFile string,
	sessionFile string,
	title string,
	payload model.CredentialsPayload,
) (string, string, error) {
	t.Helper()

	return runClientCommandWithInput(ctx, []string{
		"gkeep",
		"--address", address,
		"--ca-cert", caCertFile,
		"--session-file", sessionFile,
		"records", "create-credentials",
		"--title", title,
		"--credentials-stdin",
	}, encodeCredentialsInput(t, payload))
}

func runUpdateCredentialsRecordCommand(
	t *testing.T,
	config credentialsCLIConfig,
	recordID string,
	revision int64,
	title string,
	payload model.CredentialsPayload,
) (string, string, error) {
	t.Helper()

	return runClientCommandWithInput(config.ctx, []string{
		"gkeep",
		"--address", config.address,
		"--ca-cert", config.caCertFile,
		"--session-file", config.sessionFile,
		"records", "update-credentials", recordID,
		"--revision", fmt.Sprintf("%d", revision),
		"--title", title,
		"--credentials-stdin",
	}, encodeCredentialsInput(t, payload))
}

func runListRecordsCommand(
	ctx context.Context,
	address string,
	caCertFile string,
	sessionFile string,
) (string, string, error) {
	return runClientCommand(ctx, []string{
		"gkeep",
		"--address", address,
		"--ca-cert", caCertFile,
		"--session-file", sessionFile,
		"records", "list",
	})
}

func encodeCredentialsInput(t *testing.T, payload model.CredentialsPayload) string {
	t.Helper()

	data, err := json.Marshal(payload)
	if err != nil {
		t.Fatalf("encode credentials payload: %v", err)
	}

	return string(data)
}

func assertOutputDoesNotContainCredentials(
	t *testing.T,
	output string,
	payload model.CredentialsPayload,
) {
	t.Helper()

	fields := []struct {
		name  string
		value string
	}{
		{name: "login", value: payload.Login},
		{name: "password", value: payload.Password},
		{name: "URL", value: payload.URL},
		{name: "metadata", value: payload.Metadata},
	}
	for _, field := range fields {
		if field.value != "" && strings.Contains(output, field.value) {
			t.Errorf("output contains credentials %s", field.name)
		}
	}
}

func assertCredentialsHTTPLogsDoNotContainSecrets(
	t *testing.T,
	logs string,
	payloads ...model.CredentialsPayload,
) {
	t.Helper()

	secrets := []string{testRegistrationPassword}
	for _, payload := range payloads {
		secrets = append(secrets, payload.Login, payload.Password, payload.URL, payload.Metadata)
	}

	for _, secret := range secrets {
		if secret != "" && strings.Contains(logs, secret) {
			t.Error("HTTP logs contain a credentials secret")
		}
	}
}
