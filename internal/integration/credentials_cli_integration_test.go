//go:build integration

package integration_test

import (
	"context"
	"fmt"
	"regexp"
	"strings"
	"testing"

	"github.com/xhrobj/gopherkeeper/internal/model"
)

var createdCredentialsRecordPattern = regexp.MustCompile(
	`^Created credentials record ([0-9a-f-]+) with revision ([0-9]+)\.$`,
)

func TestIntegration_CLICredentialsRecordRoundTrip(t *testing.T) {
	config, pool, httpLogs := newRecordCLIEnvironment(t)

	initial := model.CredentialsPayload{
		Login:    "alice@example.com",
		Password: "initial-github-secret",
		URL:      "https://github.com/login",
		Metadata: "personal account recovery codes",
	}
	recordID := createCredentialsRecord(
		t,
		config.ctx,
		config.address,
		config.caCertFile,
		config.sessionFile,
		"Alice GitHub",
		initial,
	)
	assertPlaintextAbsentFromPostgres(
		t,
		config.ctx,
		pool,
		recordID,
		initial.Login,
		initial.Password,
		initial.URL,
		initial.Metadata,
	)
	assertCredentialsList(
		t,
		config.ctx,
		config.address,
		config.caCertFile,
		config.sessionFile,
		recordID,
		initial,
	)
	assertCredentialsRecord(t, config, recordID, 1, initial)

	updated := model.CredentialsPayload{
		Login:    "alice.updated@example.com",
		Password: "updated-github-secret",
		URL:      "https://github.com/settings/security",
		Metadata: "updated recovery codes",
	}
	updateCredentialsRecord(t, config, recordID, 1, "Updated Alice GitHub", updated)
	assertPlaintextAbsentFromPostgres(
		t,
		config.ctx,
		pool,
		recordID,
		updated.Login,
		updated.Password,
		updated.URL,
		updated.Metadata,
	)
	assertCredentialsRecord(t, config, recordID, 2, updated)
	assertCredentialsHTTPLogsDoNotContainSecrets(t, httpLogs.String(), initial, updated)
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
	config recordCLIConfig,
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
	config recordCLIConfig,
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

	args := []string{
		"gkeep",
		"--address", address,
		"--ca-cert", caCertFile,
		"--session-file", sessionFile,
		"records", "create-credentials",
		"--title", title,
	}
	if payload.Metadata != "" {
		metadataFile := writeIntegrationFile(t, "credentials-metadata.txt", payload.Metadata)
		args = append(args, "--metadata-file", metadataFile)
	}

	return runClientCommandWithInput(ctx, args, credentialsInput(payload))
}

func runUpdateCredentialsRecordCommand(
	t *testing.T,
	config recordCLIConfig,
	recordID string,
	revision int64,
	title string,
	payload model.CredentialsPayload,
) (string, string, error) {
	t.Helper()

	args := []string{
		"gkeep",
		"--address", config.address,
		"--ca-cert", config.caCertFile,
		"--session-file", config.sessionFile,
		"records", "update-credentials", recordID,
		"--revision", fmt.Sprintf("%d", revision),
		"--title", title,
	}
	if payload.Metadata != "" {
		metadataFile := writeIntegrationFile(t, "credentials-metadata.txt", payload.Metadata)
		args = append(args, "--metadata-file", metadataFile)
	}

	return runClientCommandWithInput(config.ctx, args, credentialsInput(payload))
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

func credentialsInput(payload model.CredentialsPayload) string {
	return strings.Join([]string{
		payload.Login,
		payload.Password,
		payload.URL,
	}, "\n") + "\n"
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
