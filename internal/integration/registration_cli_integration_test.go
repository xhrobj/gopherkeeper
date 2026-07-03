//go:build integration

package integration_test

import (
	"bytes"
	"context"
	"io"
	"os"
	"strings"
	"testing"

	"go.uber.org/zap"
	"go.uber.org/zap/zapcore"

	"github.com/xhrobj/gopherkeeper/internal/buildinfo"
	clientcli "github.com/xhrobj/gopherkeeper/internal/client/cli"
	"github.com/xhrobj/gopherkeeper/internal/server/httpserver"
	"github.com/xhrobj/gopherkeeper/internal/server/migration"
)

func TestIntegration_CLIRegistrationFlow(t *testing.T) {
	dsn := os.Getenv("DATABASE_DSN")
	if dsn == "" {
		t.Fatal("DATABASE_DSN is not set")
	}

	t.Setenv("ADDRESS", "")
	t.Setenv("CA_CERT_FILE", "")

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
		httpserver.WithLogging(newServerHandler(testPool), logger),
		serverCertFile,
		serverKeyFile,
	)
	defer stopServer()

	stdout, stderr, err := runRegisterCommand(
		ctx,
		serverAddress,
		caCertFile,
		" Alice ",
		testRegistrationPassword,
	)
	if err != nil {
		t.Fatalf("registration command error = %v", err)
	}
	if stdout != "User alice registered successfully.\n" {
		t.Errorf("registration stdout = %q, want success message", stdout)
	}
	if stderr != "" {
		t.Errorf("registration stderr = %q, want empty output", stderr)
	}

	assertSecretAbsent(t, "registration stdout", stdout)
	assertSecretAbsent(t, "registration stderr", stderr)
	assertSecretAbsent(t, "HTTP logs", httpLogs.String())
	assertStoredRegistration(t, ctx, testPool)

	stdout, stderr, err = runRegisterCommand(
		ctx,
		serverAddress,
		caCertFile,
		"ALICE",
		testRegistrationPassword,
	)
	if err == nil {
		t.Fatal("duplicate registration command error = nil")
	}
	if !strings.Contains(err.Error(), `login "ALICE" is already registered`) {
		t.Errorf("duplicate registration error = %q, want readable message", err)
	}
	if stdout != "" {
		t.Errorf("duplicate registration stdout = %q, want empty output", stdout)
	}
	if stderr != "" {
		t.Errorf("duplicate registration stderr = %q, want empty output", stderr)
	}

	assertSecretAbsent(t, "duplicate error", err.Error())
	assertSecretAbsent(t, "duplicate stdout", stdout)
	assertSecretAbsent(t, "duplicate stderr", stderr)
	assertSecretAbsent(t, "HTTP logs", httpLogs.String())

	var userCount int
	if err := testPool.QueryRow(ctx, "SELECT count(*) FROM gopherkeeper.users").Scan(&userCount); err != nil {
		t.Fatalf("count users: %v", err)
	}
	if userCount != 1 {
		t.Errorf("users count = %d, want 1", userCount)
	}
}

func runRegisterCommand(
	ctx context.Context,
	address string,
	caCertFile string,
	login string,
	password string,
) (string, string, error) {
	var stdout bytes.Buffer
	var stderr bytes.Buffer

	err := clientcli.RunWithInput(
		ctx,
		[]string{
			"gkeep",
			"--address", address,
			"--ca-cert", caCertFile,
			"register",
			"--login", login,
			"--password-stdin",
		},
		strings.NewReader(password+"\n"),
		&stdout,
		&stderr,
		buildinfo.Info{},
	)

	return stdout.String(), stderr.String(), err
}

func newIntegrationLogger(output io.Writer) *zap.Logger {
	encoderConfig := zap.NewProductionEncoderConfig()
	encoderConfig.TimeKey = ""

	return zap.New(zapcore.NewCore(
		zapcore.NewJSONEncoder(encoderConfig),
		zapcore.AddSync(output),
		zap.DebugLevel,
	))
}

func assertSecretAbsent(t *testing.T, source, value string) {
	t.Helper()

	if strings.Contains(value, testRegistrationPassword) {
		t.Errorf("%s contains password", source)
	}
}
