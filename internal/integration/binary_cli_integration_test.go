//go:build integration

package integration_test

import (
	"bytes"
	"encoding/base64"
	"encoding/json"
	"fmt"
	"io"
	"net/http"
	"os"
	"path/filepath"
	"regexp"
	"strings"
	"testing"

	"github.com/xhrobj/gopherkeeper/internal/model"
)

var createdBinaryRecordPattern = regexp.MustCompile(
	`^Created binary record ([0-9a-f-]+) with revision ([0-9]+)\.$`,
)

type binaryRecordFixture struct {
	title        string
	binaryFile   string
	filename     string
	contentType  string
	metadataFile string
	metadata     string
	data         []byte
}

type binaryRecordExpectation struct {
	recordID   string
	revision   int64
	outputPath string
	fixture    binaryRecordFixture
}

type binaryRequestErrorExpectation struct {
	body   string
	status int
	code   string
}

func TestIntegration_CLIBinaryRecordRoundTrip(t *testing.T) {
	config, pool, httpLogs := newRecordCLIEnvironment(t)

	initial := binaryRecordFixture{
		title:       "Alice backup",
		filename:    "backup.bin",
		contentType: "application/octet-stream",
		metadata:    "private binary metadata",
		data:        []byte("gopherkeeper-binary-secret-42\x00\xff\x10"),
	}
	initial.binaryFile = writeIntegrationBinaryFile(t, initial.filename, initial.data)
	initial.metadataFile = writeIntegrationFile(t, "backup-metadata.txt", initial.metadata)

	recordID := createBinaryRecord(t, config, initial)
	assertPlaintextAbsentFromPostgres(
		t,
		config.ctx,
		pool,
		recordID,
		initial.filename,
		initial.contentType,
		initial.metadata,
		"gopherkeeper-binary-secret-42",
	)
	assertBinaryList(t, config, recordID, initial)

	initialOutput := filepath.Join(t.TempDir(), "restored-backup.bin")
	assertBinaryRecord(t, config, binaryRecordExpectation{
		recordID:   recordID,
		revision:   1,
		outputPath: initialOutput,
		fixture:    initial,
	})
	assertBinaryOutputNotOverwritten(t, config, recordID, initialOutput, initial.data)

	updated := binaryRecordFixture{
		title:       "Alice backup updated",
		filename:    "backup-v2.bin",
		contentType: initial.contentType,
		metadata:    "updated private binary metadata",
		data:        []byte("gopherkeeper-updated-binary-secret-69\x00\xfe\x11"),
	}
	updated.binaryFile = writeIntegrationBinaryFile(t, updated.filename, updated.data)
	updated.metadataFile = writeIntegrationFile(t, "backup-v2-metadata.txt", updated.metadata)
	updateBinaryRecord(t, config, recordID, 1, updated)
	assertPlaintextAbsentFromPostgres(
		t,
		config.ctx,
		pool,
		recordID,
		updated.filename,
		updated.contentType,
		updated.metadata,
		"gopherkeeper-updated-binary-secret-69",
	)

	updatedOutput := filepath.Join(t.TempDir(), "restored-backup-v2.bin")
	assertBinaryRecord(t, config, binaryRecordExpectation{
		recordID:   recordID,
		revision:   2,
		outputPath: updatedOutput,
		fixture:    updated,
	})

	assertEmptyBinaryRoundTrip(t, config)
	assertOversizedBinaryRejectedLocally(t, config)
	assertInvalidBinaryRequestsRejectedByServer(t, config)
	assertBinaryHTTPLogsDoNotContainSecrets(
		t,
		httpLogs.String(),
		config.sessionFile,
		[][]byte{initial.data, updated.data},
		initial.filename,
		updated.filename,
		initial.metadata,
		updated.metadata,
	)
}

func createBinaryRecord(
	t *testing.T,
	config recordCLIConfig,
	fixture binaryRecordFixture,
) string {
	t.Helper()

	stdout, stderr, err := runCreateBinaryRecordCommand(config, fixture)
	if err != nil {
		t.Fatalf("create binary record: %v", err)
	}
	assertOutputDoesNotContainBinaryData(t, stdout+stderr, fixture.data)
	if stderr != "" {
		t.Errorf("create stderr = %q, want empty output", stderr)
	}

	matches := createdBinaryRecordPattern.FindStringSubmatch(strings.TrimSpace(stdout))
	if matches == nil {
		t.Fatalf("created binary output = %q, want created record message", stdout)
	}
	if matches[2] != "1" {
		t.Fatalf("created revision = %s, want 1", matches[2])
	}

	return matches[1]
}

func updateBinaryRecord(
	t *testing.T,
	config recordCLIConfig,
	recordID string,
	revision int64,
	fixture binaryRecordFixture,
) {
	t.Helper()

	stdout, stderr, err := runUpdateBinaryRecordCommand(config, recordID, revision, fixture)
	if err != nil {
		t.Fatalf("update binary record: %v", err)
	}
	assertOutputDoesNotContainBinaryData(t, stdout+stderr, fixture.data)

	wantUpdate := fmt.Sprintf("Updated binary record %s to revision 2.\n", recordID)
	if stdout != wantUpdate {
		t.Errorf("update stdout = %q, want %q", stdout, wantUpdate)
	}
	if stderr != "" {
		t.Errorf("update stderr = %q, want empty output", stderr)
	}
}

func assertBinaryList(
	t *testing.T,
	config recordCLIConfig,
	recordID string,
	fixture binaryRecordFixture,
) {
	t.Helper()

	stdout, stderr, err := runListRecordsCommand(
		config.ctx,
		config.address,
		config.caCertFile,
		config.sessionFile,
	)
	if err != nil {
		t.Fatalf("list binary records: %v", err)
	}
	if stderr != "" {
		t.Errorf("list stderr = %q, want empty output", stderr)
	}
	for _, want := range []string{recordID, "binary", fixture.title, "1"} {
		if !strings.Contains(stdout, want) {
			t.Errorf("list stdout = %q, want %q", stdout, want)
		}
	}
	privateValues := []string{
		fixture.filename,
		fixture.contentType,
		fixture.metadata,
		string(fixture.data),
		base64.StdEncoding.EncodeToString(fixture.data),
	}
	for _, secret := range privateValues {
		if secret != "" && strings.Contains(stdout, secret) {
			t.Errorf("list stdout contains private binary value %q", secret)
		}
	}
}

func assertBinaryRecord(
	t *testing.T,
	config recordCLIConfig,
	expected binaryRecordExpectation,
) {
	t.Helper()

	stdout, stderr, err := runGetBinaryRecordCommand(config, expected.recordID, expected.outputPath)
	if err != nil {
		t.Fatalf("get binary record: %v", err)
	}
	if stderr != "" {
		t.Errorf("get stderr = %q, want empty output", stderr)
	}

	gotData, err := os.ReadFile(expected.outputPath)
	if err != nil {
		t.Fatalf("read restored binary file: %v", err)
	}
	if !bytes.Equal(gotData, expected.fixture.data) {
		t.Errorf("restored binary data = %v, want %v", gotData, expected.fixture.data)
	}

	for _, want := range []string{
		"Type: binary",
		fmt.Sprintf("Revision: %d", expected.revision),
		"Filename: " + expected.fixture.filename,
		fmt.Sprintf("Size: %d bytes", len(expected.fixture.data)),
		"Saved to: " + expected.outputPath,
	} {
		if !strings.Contains(stdout, want) {
			t.Errorf("get stdout = %q, want %q", stdout, want)
		}
	}
	if expected.fixture.contentType != "" &&
		!strings.Contains(stdout, "Content type: "+expected.fixture.contentType) {
		t.Errorf("get stdout = %q, want content type %q", stdout, expected.fixture.contentType)
	}
	if expected.fixture.metadata != "" && !strings.Contains(stdout, "Metadata:\n"+expected.fixture.metadata) {
		t.Errorf("get stdout = %q, want metadata %q", stdout, expected.fixture.metadata)
	}
	assertOutputDoesNotContainBinaryData(t, stdout+stderr, expected.fixture.data)
}

func assertBinaryOutputNotOverwritten(
	t *testing.T,
	config recordCLIConfig,
	recordID, outputPath string,
	wantData []byte,
) {
	t.Helper()

	stdout, stderr, err := runGetBinaryRecordCommand(config, recordID, outputPath)
	assertCLIErrorContains(t, "overwrite binary output", err, "create output file")
	if stdout != "" || stderr != "" {
		t.Errorf("overwrite output = %q, stderr = %q, want empty", stdout, stderr)
	}

	gotData, readErr := os.ReadFile(outputPath)
	if readErr != nil {
		t.Fatalf("read existing binary output: %v", readErr)
	}
	if !bytes.Equal(gotData, wantData) {
		t.Errorf("existing binary output = %v, want unchanged %v", gotData, wantData)
	}
}

func assertEmptyBinaryRoundTrip(t *testing.T, config recordCLIConfig) {
	t.Helper()

	fixture := binaryRecordFixture{
		title:    "Empty backup",
		filename: "empty.bin",
		data:     []byte{},
	}
	fixture.binaryFile = writeIntegrationBinaryFile(t, fixture.filename, fixture.data)
	recordID := createBinaryRecord(t, config, fixture)
	outputPath := filepath.Join(t.TempDir(), "restored-empty.bin")
	assertBinaryRecord(t, config, binaryRecordExpectation{
		recordID:   recordID,
		revision:   1,
		outputPath: outputPath,
		fixture:    fixture,
	})

	stdout, stderr, err := runDeleteRecordCommand(
		config.ctx,
		config.address,
		config.caCertFile,
		config.sessionFile,
		recordID,
		1,
	)
	if err != nil {
		t.Fatalf("delete empty binary record: %v", err)
	}
	if stdout != fmt.Sprintf("Deleted record %s.\n", recordID) || stderr != "" {
		t.Errorf("delete empty output = %q, stderr = %q", stdout, stderr)
	}
}

func assertOversizedBinaryRejectedLocally(t *testing.T, config recordCLIConfig) {
	t.Helper()

	fixture := binaryRecordFixture{
		title:       "Oversized backup",
		filename:    "oversized.bin",
		contentType: "application/octet-stream",
		data:        bytes.Repeat([]byte{0x2a}, model.BinaryPayloadMaxSize+1),
	}
	fixture.binaryFile = writeIntegrationBinaryFile(t, fixture.filename, fixture.data)
	stdout, stderr, err := runCreateBinaryRecordCommand(config, fixture)
	assertCLIErrorContains(t, "create oversized binary record", err, "payload too large")
	if stdout != "" || stderr != "" {
		t.Errorf("oversized output = %q, stderr = %q, want empty", stdout, stderr)
	}
}

func assertInvalidBinaryRequestsRejectedByServer(t *testing.T, config recordCLIConfig) {
	t.Helper()

	accessToken := readIntegrationAccessToken(t, config.sessionFile)
	client := newTrustedHTTPSClient(t, config.caCertFile)

	assertBinaryCreateRequestError(
		t,
		config,
		client,
		accessToken,
		binaryRequestErrorExpectation{
			body:   `{"type":"binary","title":"Damaged","payload":{"filename":"damaged.bin","data":"not-base64***"}}`,
			status: http.StatusBadRequest,
			code:   "invalid_request",
		},
	)

	oversizedBody, err := json.Marshal(map[string]any{
		"type":  "binary",
		"title": "Oversized",
		"payload": model.BinaryPayload{
			Filename: "oversized.bin",
			Data:     bytes.Repeat([]byte{0x2a}, model.BinaryPayloadMaxSize+1),
		},
	})
	if err != nil {
		t.Fatalf("encode oversized binary request: %v", err)
	}
	assertBinaryCreateRequestError(
		t,
		config,
		client,
		accessToken,
		binaryRequestErrorExpectation{
			body:   string(oversizedBody),
			status: http.StatusRequestEntityTooLarge,
			code:   "payload_too_large",
		},
	)
}

func assertBinaryCreateRequestError(
	t *testing.T,
	config recordCLIConfig,
	client *http.Client,
	accessToken string,
	expected binaryRequestErrorExpectation,
) {
	t.Helper()

	request, err := http.NewRequestWithContext(
		config.ctx,
		http.MethodPost,
		"https://"+config.address+"/api/v1/records",
		strings.NewReader(expected.body),
	)
	if err != nil {
		t.Fatalf("create binary request: %v", err)
	}
	request.Header.Set("Authorization", "Bearer "+accessToken)
	request.Header.Set("Content-Type", "application/json")

	response, err := client.Do(request)
	if err != nil {
		t.Fatalf("send binary request: %v", err)
	}
	defer func() { _ = response.Body.Close() }()

	responseBody, err := io.ReadAll(response.Body)
	if err != nil {
		t.Fatalf("read binary error response: %v", err)
	}
	if response.StatusCode != expected.status {
		t.Fatalf(
			"binary response status = %d, want %d: %s",
			response.StatusCode,
			expected.status,
			responseBody,
		)
	}

	var apiError apiErrorResponse
	if err := json.Unmarshal(responseBody, &apiError); err != nil {
		t.Fatalf("decode binary error response: %v", err)
	}
	if apiError.Code != expected.code {
		t.Errorf("binary error code = %q, want %q", apiError.Code, expected.code)
	}
	for _, secret := range []string{accessToken, "not-base64***"} {
		if strings.Contains(string(responseBody), secret) {
			t.Errorf("binary error response contains secret %q", secret)
		}
	}
}

func runCreateBinaryRecordCommand(
	config recordCLIConfig,
	fixture binaryRecordFixture,
) (string, string, error) {
	args := []string{
		"gkeep",
		"--address", config.address,
		"--ca-cert", config.caCertFile,
		"--session-file", config.sessionFile,
		"records", "create-binary",
		"--title", fixture.title,
		"--binary-file", fixture.binaryFile,
	}
	if fixture.contentType != "" {
		args = append(args, "--content-type", fixture.contentType)
	}
	if fixture.metadataFile != "" {
		args = append(args, "--metadata-file", fixture.metadataFile)
	}

	return runClientCommand(config.ctx, args)
}

func runUpdateBinaryRecordCommand(
	config recordCLIConfig,
	recordID string,
	revision int64,
	fixture binaryRecordFixture,
) (string, string, error) {
	args := []string{
		"gkeep",
		"--address", config.address,
		"--ca-cert", config.caCertFile,
		"--session-file", config.sessionFile,
		"records", "update-binary", recordID,
		"--revision", fmt.Sprintf("%d", revision),
		"--title", fixture.title,
		"--binary-file", fixture.binaryFile,
	}
	if fixture.contentType != "" {
		args = append(args, "--content-type", fixture.contentType)
	}
	if fixture.metadataFile != "" {
		args = append(args, "--metadata-file", fixture.metadataFile)
	}

	return runClientCommand(config.ctx, args)
}

func runGetBinaryRecordCommand(
	config recordCLIConfig,
	recordID, outputPath string,
) (string, string, error) {
	return runClientCommand(config.ctx, []string{
		"gkeep",
		"--address", config.address,
		"--ca-cert", config.caCertFile,
		"--session-file", config.sessionFile,
		"records", "get", recordID,
		"--output", outputPath,
	})
}

func writeIntegrationBinaryFile(t *testing.T, filename string, data []byte) string {
	t.Helper()

	path := filepath.Join(t.TempDir(), filename)
	if err := os.WriteFile(path, data, 0o600); err != nil {
		t.Fatalf("write integration binary file: %v", err)
	}

	return path
}

func readIntegrationAccessToken(t *testing.T, sessionFile string) string {
	t.Helper()

	data, err := os.ReadFile(sessionFile)
	if err != nil {
		t.Fatalf("read integration session: %v", err)
	}
	var stored struct {
		AccessToken string `json:"access_token"`
	}
	if err := json.Unmarshal(data, &stored); err != nil {
		t.Fatalf("decode integration session: %v", err)
	}
	if stored.AccessToken == "" {
		t.Fatal("integration access token is empty")
	}

	return stored.AccessToken
}

func assertOutputDoesNotContainBinaryData(t *testing.T, output string, data []byte) {
	t.Helper()

	for _, secret := range []string{string(data), base64.StdEncoding.EncodeToString(data)} {
		if secret != "" && strings.Contains(output, secret) {
			t.Error("output contains binary data")
		}
	}
}

func assertBinaryHTTPLogsDoNotContainSecrets(
	t *testing.T,
	logs, sessionFile string,
	payloads [][]byte,
	secrets ...string,
) {
	t.Helper()

	secrets = append(secrets, testRegistrationPassword, readIntegrationAccessToken(t, sessionFile))
	for _, payload := range payloads {
		secrets = append(secrets, string(payload), base64.StdEncoding.EncodeToString(payload))
	}
	for _, secret := range secrets {
		if secret != "" && strings.Contains(logs, secret) {
			t.Errorf("HTTP logs contain binary secret %q", secret)
		}
	}
}
