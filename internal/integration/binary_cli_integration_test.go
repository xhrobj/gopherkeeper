//go:build integration

package integration_test

import (
	"bytes"
	"context"
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

type binaryCLIConfig struct {
	ctx         context.Context
	address     string
	caCertFile  string
	sessionFile string
}

func TestIntegration_CLIBinaryRecordFlow(t *testing.T) {
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
	config := binaryCLIConfig{
		ctx:         ctx,
		address:     serverAddress,
		caCertFile:  caCertFile,
		sessionFile: sessionFile,
	}

	initialData := []byte("gopherkeeper-binary-secret-42\x00\xff\x10")
	initialFile := writeIntegrationBinaryFile(t, "backup.bin", initialData)
	initialMetadata := "private binary metadata"
	initialMetadataFile := writeIntegrationFile(t, "backup-metadata.txt", initialMetadata)
	initialContentType := "application/octet-stream"

	recordID := createBinaryRecord(
		t,
		config,
		"Alice backup",
		initialFile,
		initialContentType,
		initialMetadataFile,
		initialData,
	)
	assertPlaintextAbsentFromPostgres(
		t,
		ctx,
		pool,
		recordID,
		"backup.bin",
		initialContentType,
		initialMetadata,
		"gopherkeeper-binary-secret-42",
	)
	assertBinaryList(
		t,
		config,
		recordID,
		"Alice backup",
		"backup.bin",
		initialContentType,
		initialMetadata,
		initialData,
	)

	initialOutput := filepath.Join(t.TempDir(), "restored-backup.bin")
	assertBinaryRecord(
		t,
		config,
		recordID,
		1,
		initialOutput,
		initialData,
		"backup.bin",
		initialContentType,
		initialMetadata,
	)
	assertBinaryOutputNotOverwritten(t, config, recordID, initialOutput, initialData)

	updatedData := []byte("gopherkeeper-updated-binary-secret-69\x00\xfe\x11")
	updatedFile := writeIntegrationBinaryFile(t, "backup-v2.bin", updatedData)
	updatedMetadata := "updated private binary metadata"
	updatedMetadataFile := writeIntegrationFile(t, "backup-v2-metadata.txt", updatedMetadata)
	updateBinaryRecord(
		t,
		config,
		recordID,
		1,
		"Alice backup updated",
		updatedFile,
		initialContentType,
		updatedMetadataFile,
		updatedData,
	)
	assertPlaintextAbsentFromPostgres(
		t,
		ctx,
		pool,
		recordID,
		"backup-v2.bin",
		initialContentType,
		updatedMetadata,
		"gopherkeeper-updated-binary-secret-69",
	)

	_, _, err := runUpdateBinaryRecordCommand(
		config,
		recordID,
		1,
		"Stale Alice backup",
		updatedFile,
		initialContentType,
		updatedMetadataFile,
	)
	assertCLIErrorContains(t, "stale binary update", err, "record revision conflict")

	updatedOutput := filepath.Join(t.TempDir(), "restored-backup-v2.bin")
	assertBinaryRecord(
		t,
		config,
		recordID,
		2,
		updatedOutput,
		updatedData,
		"backup-v2.bin",
		initialContentType,
		updatedMetadata,
	)

	assertEmptyBinaryRoundTrip(t, config)
	assertOversizedBinaryRejectedLocally(t, config)
	assertInvalidBinaryRequestsRejectedByServer(t, config)

	stdout, stderr, err := runDeleteRecordCommand(
		ctx,
		serverAddress,
		caCertFile,
		sessionFile,
		recordID,
		2,
	)
	if err != nil {
		t.Fatalf("delete binary record: %v", err)
	}
	wantDelete := fmt.Sprintf("Deleted record %s.\n", recordID)
	if stdout != wantDelete {
		t.Errorf("delete stdout = %q, want %q", stdout, wantDelete)
	}
	if stderr != "" {
		t.Errorf("delete stderr = %q, want empty output", stderr)
	}

	deletedOutput := filepath.Join(t.TempDir(), "deleted-backup.bin")
	_, _, err = runGetBinaryRecordCommand(config, recordID, deletedOutput)
	assertCLIErrorContains(t, "get deleted binary record", err, "record not found")
	if _, statErr := os.Stat(deletedOutput); !os.IsNotExist(statErr) {
		t.Fatalf("deleted output stat error = %v, want not exist", statErr)
	}

	assertBinaryHTTPLogsDoNotContainSecrets(
		t,
		httpLogs.String(),
		sessionFile,
		[][]byte{initialData, updatedData},
		"backup.bin",
		"backup-v2.bin",
		initialMetadata,
		updatedMetadata,
	)
}

func createBinaryRecord(
	t *testing.T,
	config binaryCLIConfig,
	title string,
	binaryFile string,
	contentType string,
	metadataFile string,
	data []byte,
) string {
	t.Helper()

	stdout, stderr, err := runCreateBinaryRecordCommand(
		config,
		title,
		binaryFile,
		contentType,
		metadataFile,
	)
	if err != nil {
		t.Fatalf("create binary record: %v", err)
	}
	assertOutputDoesNotContainBinaryData(t, stdout+stderr, data)
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
	config binaryCLIConfig,
	recordID string,
	revision int64,
	title string,
	binaryFile string,
	contentType string,
	metadataFile string,
	data []byte,
) {
	t.Helper()

	stdout, stderr, err := runUpdateBinaryRecordCommand(
		config,
		recordID,
		revision,
		title,
		binaryFile,
		contentType,
		metadataFile,
	)
	if err != nil {
		t.Fatalf("update binary record: %v", err)
	}
	assertOutputDoesNotContainBinaryData(t, stdout+stderr, data)

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
	config binaryCLIConfig,
	recordID string,
	title string,
	filename string,
	contentType string,
	metadata string,
	data []byte,
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
	for _, want := range []string{recordID, "binary", title, "1"} {
		if !strings.Contains(stdout, want) {
			t.Errorf("list stdout = %q, want %q", stdout, want)
		}
	}
	privateValues := []string{
		filename,
		contentType,
		metadata,
		string(data),
		base64.StdEncoding.EncodeToString(data),
	}
	for _, secret := range privateValues {
		if secret != "" && strings.Contains(stdout, secret) {
			t.Errorf("list stdout contains private binary value %q", secret)
		}
	}
}

func assertBinaryRecord(
	t *testing.T,
	config binaryCLIConfig,
	recordID string,
	revision int64,
	outputPath string,
	wantData []byte,
	filename string,
	contentType string,
	metadata string,
) {
	t.Helper()

	stdout, stderr, err := runGetBinaryRecordCommand(config, recordID, outputPath)
	if err != nil {
		t.Fatalf("get binary record: %v", err)
	}
	if stderr != "" {
		t.Errorf("get stderr = %q, want empty output", stderr)
	}

	gotData, err := os.ReadFile(outputPath)
	if err != nil {
		t.Fatalf("read restored binary file: %v", err)
	}
	if !bytes.Equal(gotData, wantData) {
		t.Errorf("restored binary data = %v, want %v", gotData, wantData)
	}

	for _, want := range []string{
		"Type: binary",
		fmt.Sprintf("Revision: %d", revision),
		"Filename: " + filename,
		fmt.Sprintf("Size: %d bytes", len(wantData)),
		"Saved to: " + outputPath,
	} {
		if !strings.Contains(stdout, want) {
			t.Errorf("get stdout = %q, want %q", stdout, want)
		}
	}
	if contentType != "" && !strings.Contains(stdout, "Content type: "+contentType) {
		t.Errorf("get stdout = %q, want content type %q", stdout, contentType)
	}
	if metadata != "" && !strings.Contains(stdout, "Metadata:\n"+metadata) {
		t.Errorf("get stdout = %q, want metadata %q", stdout, metadata)
	}
	assertOutputDoesNotContainBinaryData(t, stdout+stderr, wantData)
}

func assertBinaryOutputNotOverwritten(
	t *testing.T,
	config binaryCLIConfig,
	recordID string,
	outputPath string,
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

func assertEmptyBinaryRoundTrip(t *testing.T, config binaryCLIConfig) {
	t.Helper()

	emptyFile := writeIntegrationBinaryFile(t, "empty.bin", []byte{})
	recordID := createBinaryRecord(t, config, "Empty backup", emptyFile, "", "", []byte{})
	outputPath := filepath.Join(t.TempDir(), "restored-empty.bin")
	assertBinaryRecord(t, config, recordID, 1, outputPath, []byte{}, "empty.bin", "", "")

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

func assertOversizedBinaryRejectedLocally(t *testing.T, config binaryCLIConfig) {
	t.Helper()

	oversized := bytes.Repeat([]byte{0x2a}, model.BinaryPayloadMaxSize+1)
	oversizedFile := writeIntegrationBinaryFile(t, "oversized.bin", oversized)
	stdout, stderr, err := runCreateBinaryRecordCommand(
		config,
		"Oversized backup",
		oversizedFile,
		"application/octet-stream",
		"",
	)
	assertCLIErrorContains(t, "create oversized binary record", err, "payload too large")
	if stdout != "" || stderr != "" {
		t.Errorf("oversized output = %q, stderr = %q, want empty", stdout, stderr)
	}
}

func assertInvalidBinaryRequestsRejectedByServer(t *testing.T, config binaryCLIConfig) {
	t.Helper()

	accessToken := readIntegrationAccessToken(t, config.sessionFile)
	client := newTrustedHTTPSClient(t, config.caCertFile)

	assertBinaryCreateRequestError(
		t,
		config.ctx,
		client,
		config.address,
		accessToken,
		`{"type":"binary","title":"Damaged","payload":{"filename":"damaged.bin","data":"not-base64***"}}`,
		http.StatusBadRequest,
		"invalid_request",
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
		config.ctx,
		client,
		config.address,
		accessToken,
		string(oversizedBody),
		http.StatusRequestEntityTooLarge,
		"payload_too_large",
	)
}

func assertBinaryCreateRequestError(
	t *testing.T,
	ctx context.Context,
	client *http.Client,
	address string,
	accessToken string,
	body string,
	wantStatus int,
	wantCode string,
) {
	t.Helper()

	request, err := http.NewRequestWithContext(
		ctx,
		http.MethodPost,
		"https://"+address+"/api/v1/records",
		strings.NewReader(body),
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
	if response.StatusCode != wantStatus {
		t.Fatalf("binary response status = %d, want %d: %s", response.StatusCode, wantStatus, responseBody)
	}

	var apiError apiErrorResponse
	if err := json.Unmarshal(responseBody, &apiError); err != nil {
		t.Fatalf("decode binary error response: %v", err)
	}
	if apiError.Code != wantCode {
		t.Errorf("binary error code = %q, want %q", apiError.Code, wantCode)
	}
	for _, secret := range []string{accessToken, "not-base64***"} {
		if strings.Contains(string(responseBody), secret) {
			t.Errorf("binary error response contains secret %q", secret)
		}
	}
}

func runCreateBinaryRecordCommand(
	config binaryCLIConfig,
	title string,
	binaryFile string,
	contentType string,
	metadataFile string,
) (string, string, error) {
	args := []string{
		"gkeep",
		"--address", config.address,
		"--ca-cert", config.caCertFile,
		"--session-file", config.sessionFile,
		"records", "create-binary",
		"--title", title,
		"--binary-file", binaryFile,
	}
	if contentType != "" {
		args = append(args, "--content-type", contentType)
	}
	if metadataFile != "" {
		args = append(args, "--metadata-file", metadataFile)
	}

	return runClientCommand(config.ctx, args)
}

func runUpdateBinaryRecordCommand(
	config binaryCLIConfig,
	recordID string,
	revision int64,
	title string,
	binaryFile string,
	contentType string,
	metadataFile string,
) (string, string, error) {
	args := []string{
		"gkeep",
		"--address", config.address,
		"--ca-cert", config.caCertFile,
		"--session-file", config.sessionFile,
		"records", "update-binary", recordID,
		"--revision", fmt.Sprintf("%d", revision),
		"--title", title,
		"--binary-file", binaryFile,
	}
	if contentType != "" {
		args = append(args, "--content-type", contentType)
	}
	if metadataFile != "" {
		args = append(args, "--metadata-file", metadataFile)
	}

	return runClientCommand(config.ctx, args)
}

func runGetBinaryRecordCommand(
	config binaryCLIConfig,
	recordID string,
	outputPath string,
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
	logs string,
	sessionFile string,
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
