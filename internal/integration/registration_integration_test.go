//go:build integration

package integration_test

import (
	"bytes"
	"context"
	"crypto/tls"
	"crypto/x509"
	"encoding/json"
	"io"
	"net/http"
	"os"
	"strings"
	"testing"
	"time"

	"github.com/jackc/pgx/v5/pgxpool"
	"github.com/xhrobj/gopherkeeper/internal/server/auth"
	"github.com/xhrobj/gopherkeeper/internal/server/migration"
)

const testRegistrationPassword = "correct-horse-battery-staple"

type registrationResponse struct {
	ID        int64     `json:"id"`
	Login     string    `json:"login"`
	CreatedAt time.Time `json:"created_at"`
}

type apiErrorResponse struct {
	Code    string `json:"code"`
	Message string `json:"message"`
}

func TestIntegration_HTTPSRegistrationFlow(t *testing.T) {
	dsn := os.Getenv("DATABASE_DSN")
	if dsn == "" {
		t.Fatal("DATABASE_DSN is not set")
	}

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

	caCertFile, serverCertFile, serverKeyFile := generateTLSFiles(t)
	serverAddress, stopServer := startHTTPSServer(
		t,
		newServerHandler(testPool),
		serverCertFile,
		serverKeyFile,
	)
	defer stopServer()

	client := newTrustedHTTPSClient(t, caCertFile)

	statusCode, responseBody := registerUser(
		t,
		ctx,
		client,
		serverAddress,
		" Alice ",
		testRegistrationPassword,
	)
	if statusCode != http.StatusCreated {
		t.Fatalf("registration status = %d, want %d; body = %s", statusCode, http.StatusCreated, responseBody)
	}

	var created registrationResponse
	if err := json.Unmarshal(responseBody, &created); err != nil {
		t.Fatalf("decode registration response: %v", err)
	}
	if created.ID <= 0 {
		t.Errorf("created user id = %d, want positive value", created.ID)
	}
	if created.Login != "alice" {
		t.Errorf("created user login = %q, want alice", created.Login)
	}
	if created.CreatedAt.IsZero() {
		t.Error("created user created_at is zero")
	}
	if bytes.Contains(responseBody, []byte(testRegistrationPassword)) {
		t.Error("registration response contains password")
	}

	assertStoredRegistration(t, ctx, testPool)

	statusCode, responseBody = registerUser(
		t,
		ctx,
		client,
		serverAddress,
		"ALICE",
		testRegistrationPassword,
	)
	if statusCode != http.StatusConflict {
		t.Fatalf("duplicate registration status = %d, want %d; body = %s", statusCode, http.StatusConflict, responseBody)
	}

	var apiError apiErrorResponse
	if err := json.Unmarshal(responseBody, &apiError); err != nil {
		t.Fatalf("decode duplicate registration response: %v", err)
	}
	if apiError.Code != "login_already_exists" {
		t.Errorf("duplicate registration code = %q, want login_already_exists", apiError.Code)
	}
	if strings.Contains(string(responseBody), testRegistrationPassword) {
		t.Error("duplicate registration response contains password")
	}

	var userCount int
	if err := testPool.QueryRow(ctx, "SELECT count(*) FROM gopherkeeper.users").Scan(&userCount); err != nil {
		t.Fatalf("count users: %v", err)
	}
	if userCount != 1 {
		t.Errorf("users count = %d, want 1", userCount)
	}
}

func newTrustedHTTPSClient(t *testing.T, caCertFile string) *http.Client {
	t.Helper()

	caPEM, err := os.ReadFile(caCertFile)
	if err != nil {
		t.Fatalf("read CA certificate: %v", err)
	}

	rootCAs := x509.NewCertPool()
	if !rootCAs.AppendCertsFromPEM(caPEM) {
		t.Fatal("append CA certificate")
	}

	transport := &http.Transport{
		TLSClientConfig: &tls.Config{
			MinVersion: tls.VersionTLS12,
			RootCAs:    rootCAs,
		},
	}
	t.Cleanup(transport.CloseIdleConnections)

	return &http.Client{Transport: transport}
}

func registerUser(
	t *testing.T,
	ctx context.Context,
	client *http.Client,
	serverAddress string,
	login string,
	password string,
) (int, []byte) {
	t.Helper()

	requestBody, err := json.Marshal(map[string]string{
		"login":    login,
		"password": password,
	})
	if err != nil {
		t.Fatalf("encode registration request: %v", err)
	}

	request, err := http.NewRequestWithContext(
		ctx,
		http.MethodPost,
		"https://"+serverAddress+"/api/v1/auth/register",
		bytes.NewReader(requestBody),
	)
	if err != nil {
		t.Fatalf("create registration request: %v", err)
	}
	request.Header.Set("Content-Type", "application/json")

	response, err := client.Do(request)
	if err != nil {
		t.Fatalf("send registration request: %v", err)
	}
	defer func() {
		_ = response.Body.Close()
	}()

	responseBody, err := io.ReadAll(response.Body)
	if err != nil {
		t.Fatalf("read registration response: %v", err)
	}

	return response.StatusCode, responseBody
}

func assertStoredRegistration(
	t *testing.T,
	ctx context.Context,
	pool *pgxpool.Pool,
) {
	t.Helper()

	var storedLogin string
	var passwordHash []byte
	if err := pool.QueryRow(
		ctx,
		`SELECT login, password_hash
		 FROM gopherkeeper.users
		 WHERE login = $1`,
		"alice",
	).Scan(&storedLogin, &passwordHash); err != nil {
		t.Fatalf("read stored user: %v", err)
	}

	if storedLogin != "alice" {
		t.Errorf("stored login = %q, want alice", storedLogin)
	}
	if bytes.Equal(passwordHash, []byte(testRegistrationPassword)) {
		t.Error("PostgreSQL contains plaintext password")
	}

	passwordManager := auth.NewBcryptPasswordManager()
	if err := passwordManager.Check(testRegistrationPassword, passwordHash); err != nil {
		t.Errorf("stored password hash does not match password: %v", err)
	}
}
