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

var createdCardRecordPattern = regexp.MustCompile(
	`^Created card record ([0-9a-f-]+) with revision ([0-9]+)\.$`,
)

type cardCLIConfig struct {
	ctx         context.Context
	address     string
	caCertFile  string
	sessionFile string
}

func TestIntegration_CLICardRecordFlow(t *testing.T) {
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

	sessionFile := filepath.Join(t.TempDir(), "alice-session.json")
	loginTestUser(t, ctx, serverAddress, caCertFile, sessionFile, " Alice ")
	config := cardCLIConfig{
		ctx:         ctx,
		address:     serverAddress,
		caCertFile:  caCertFile,
		sessionFile: sessionFile,
	}

	expiryMonth := 3
	expiryYear := 2038
	initial := model.CardPayload{
		Number:      "2013 0614 2020 0619",
		Cardholder:  "Joel Miller",
		ExpiryMonth: &expiryMonth,
		ExpiryYear:  &expiryYear,
		CVV:         "014",
		Metadata:    "test card",
	}
	recordID := createCardRecord(t, config, "Joel's card", initial)
	assertPlaintextAbsentFromPostgres(
		t,
		ctx,
		pool,
		recordID,
		initial.Number,
		initial.Cardholder,
		initial.CVV,
		initial.Metadata,
	)
	assertCardList(t, config, recordID, "Joel's card", initial)
	assertCardRecord(t, config, recordID, 1, initial)

	updatedMonth := 3
	updatedYear := 2038
	updated := model.CardPayload{
		Number:      "2013 0614 2020 0619",
		Cardholder:  "Joel Miller",
		ExpiryMonth: &updatedMonth,
		ExpiryYear:  &updatedYear,
		CVV:         "014",
		Metadata:    "test card updated",
	}
	updateCardRecord(t, config, recordID, 1, "Joel's card updated", updated)
	assertPlaintextAbsentFromPostgres(
		t,
		ctx,
		pool,
		recordID,
		updated.Number,
		updated.Cardholder,
		updated.CVV,
		updated.Metadata,
	)
	assertCardRecord(t, config, recordID, 2, updated)

	stdout, stderr, err := runDeleteRecordCommand(
		ctx,
		serverAddress,
		caCertFile,
		sessionFile,
		recordID,
		2,
	)
	if err != nil {
		t.Fatalf("delete card record: %v", err)
	}
	wantDelete := fmt.Sprintf("Deleted record %s.\n", recordID)
	if stdout != wantDelete {
		t.Errorf("delete stdout = %q, want %q", stdout, wantDelete)
	}
	if stderr != "" {
		t.Errorf("delete stderr = %q, want empty output", stderr)
	}

	_, _, err = runGetRecordCommand(ctx, serverAddress, caCertFile, sessionFile, recordID)
	assertCLIErrorContains(t, "get deleted card record", err, "record not found")
	assertCardHTTPLogsDoNotContainSecrets(t, httpLogs.String(), initial, updated)
}

func createCardRecord(
	t *testing.T,
	config cardCLIConfig,
	title string,
	payload model.CardPayload,
) string {
	t.Helper()

	stdout, stderr, err := runCreateCardRecordCommand(t, config, title, payload)
	if err != nil {
		t.Fatalf("create card record: %v", err)
	}
	assertOutputDoesNotContainCard(t, stdout+stderr, payload)
	if stderr != "" {
		t.Errorf("create stderr = %q, want empty output", stderr)
	}

	matches := createdCardRecordPattern.FindStringSubmatch(strings.TrimSpace(stdout))
	if matches == nil {
		t.Fatalf("created card output = %q, want created record message", stdout)
	}
	if matches[2] != "1" {
		t.Fatalf("created revision = %s, want 1", matches[2])
	}

	return matches[1]
}

func updateCardRecord(
	t *testing.T,
	config cardCLIConfig,
	recordID string,
	revision int64,
	title string,
	payload model.CardPayload,
) {
	t.Helper()

	stdout, stderr, err := runUpdateCardRecordCommand(
		t,
		config,
		recordID,
		revision,
		title,
		payload,
	)
	if err != nil {
		t.Fatalf("update card record: %v", err)
	}
	assertOutputDoesNotContainCard(t, stdout+stderr, payload)

	wantUpdate := fmt.Sprintf("Updated card record %s to revision 2.\n", recordID)
	if stdout != wantUpdate {
		t.Errorf("update stdout = %q, want %q", stdout, wantUpdate)
	}
	if stderr != "" {
		t.Errorf("update stderr = %q, want empty output", stderr)
	}
}

func assertCardList(
	t *testing.T,
	config cardCLIConfig,
	recordID string,
	title string,
	payload model.CardPayload,
) {
	t.Helper()

	stdout, stderr, err := runListRecordsCommand(
		config.ctx,
		config.address,
		config.caCertFile,
		config.sessionFile,
	)
	if err != nil {
		t.Fatalf("list card records: %v", err)
	}
	if stderr != "" {
		t.Errorf("list stderr = %q, want empty output", stderr)
	}
	for _, want := range []string{recordID, "card", title, "1"} {
		if !strings.Contains(stdout, want) {
			t.Errorf("list stdout = %q, want %q", stdout, want)
		}
	}
	assertOutputDoesNotContainCard(t, stdout, payload)
}

func assertCardRecord(
	t *testing.T,
	config cardCLIConfig,
	recordID string,
	revision int64,
	payload model.CardPayload,
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
		t.Fatalf("get card record: %v", err)
	}
	if stderr != "" {
		t.Errorf("get stderr = %q, want empty output", stderr)
	}

	wants := []string{
		"Type: card",
		fmt.Sprintf("Revision: %d", revision),
		"Number: " + payload.Number,
		"Cardholder: " + payload.Cardholder,
		fmt.Sprintf("Expiry: %02d/%d", *payload.ExpiryMonth, *payload.ExpiryYear),
		"CVV: " + payload.CVV,
		payload.Metadata,
	}
	for _, want := range wants {
		if !strings.Contains(stdout, want) {
			t.Errorf("get stdout does not contain %q", want)
		}
	}
}

func runCreateCardRecordCommand(
	t *testing.T,
	config cardCLIConfig,
	title string,
	payload model.CardPayload,
) (string, string, error) {
	t.Helper()

	return runClientCommandWithInput(config.ctx, []string{
		"gkeep",
		"--address", config.address,
		"--ca-cert", config.caCertFile,
		"--session-file", config.sessionFile,
		"records", "create-card",
		"--title", title,
		"--card-stdin",
	}, encodeCardInput(t, payload))
}

func runUpdateCardRecordCommand(
	t *testing.T,
	config cardCLIConfig,
	recordID string,
	revision int64,
	title string,
	payload model.CardPayload,
) (string, string, error) {
	t.Helper()

	return runClientCommandWithInput(config.ctx, []string{
		"gkeep",
		"--address", config.address,
		"--ca-cert", config.caCertFile,
		"--session-file", config.sessionFile,
		"records", "update-card", recordID,
		"--revision", fmt.Sprintf("%d", revision),
		"--title", title,
		"--card-stdin",
	}, encodeCardInput(t, payload))
}

func encodeCardInput(t *testing.T, payload model.CardPayload) string {
	t.Helper()

	data, err := json.Marshal(payload)
	if err != nil {
		t.Fatalf("encode card payload: %v", err)
	}

	return string(data)
}

func assertOutputDoesNotContainCard(t *testing.T, output string, payload model.CardPayload) {
	t.Helper()

	for _, secret := range []string{
		payload.Number,
		payload.Cardholder,
		payload.Metadata,
	} {
		if secret != "" && strings.Contains(output, secret) {
			t.Error("output contains a card secret")
		}
	}

	if strings.Contains(output, "CVV:") ||
		containsQuotedValue(output, "cvv") ||
		containsQuotedValue(output, payload.CVV) {
		t.Error("output contains CVV")
	}
}

func assertCardHTTPLogsDoNotContainSecrets(
	t *testing.T,
	logs string,
	payloads ...model.CardPayload,
) {
	t.Helper()

	type secretValue struct {
		name  string
		value string
	}

	secrets := []secretValue{
		{name: "registration password", value: testRegistrationPassword},
	}

	for _, payload := range payloads {
		secrets = append(
			secrets,
			secretValue{name: "card number", value: payload.Number},
			secretValue{name: "cardholder", value: payload.Cardholder},
			secretValue{name: "metadata", value: payload.Metadata},
		)

		if containsQuotedValue(logs, "cvv") || containsQuotedValue(logs, payload.CVV) {
			t.Error("HTTP logs contain CVV")
		}
	}

	for _, secret := range secrets {
		if secret.value != "" && strings.Contains(logs, secret.value) {
			t.Errorf("HTTP logs contain %s", secret.name)
		}
	}
}

func containsQuotedValue(text, value string) bool {
	if value == "" {
		return false
	}

	return strings.Contains(text, `"`+value+`"`) ||
		strings.Contains(text, `\"`+value+`\"`)
}
