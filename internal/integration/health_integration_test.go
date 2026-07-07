//go:build integration

package integration_test

import (
	"bytes"
	"context"
	"crypto/tls"
	"errors"
	"io"
	"os"
	"strings"
	"testing"

	"github.com/xhrobj/gopherkeeper/internal/buildinfo"
	clientcli "github.com/xhrobj/gopherkeeper/internal/client/cli"
	"github.com/xhrobj/gopherkeeper/internal/server/migration"
)

func TestIntegration_HTTPSHealthFlow(t *testing.T) {
	dsn := os.Getenv("DATABASE_DSN")
	if dsn == "" {
		t.Fatal("DATABASE_DSN is not set")
	}

	isolateClientConfig(t)
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
	t.Cleanup(func() {
		if testPool != nil {
			testPool.Close()
		}
	})

	if err := migration.Run(testPool); err != nil {
		t.Fatalf("first migration.Run() error = %v", err)
	}
	if err := migration.Run(testPool); err != nil {
		t.Fatalf("second migration.Run() error = %v", err)
	}

	caCertFile, serverCertFile, serverKeyFile := generateTLSFiles(t)
	serverAddress, stopServer := startHTTPSServer(
		t,
		newServerHandler(testPool),
		serverCertFile,
		serverKeyFile,
	)
	defer stopServer()

	output, err := runHealthCommand(ctx, serverAddress, caCertFile)
	if err != nil {
		t.Fatalf("trusted health command error = %v", err)
	}
	if output != "Server status: ok\n" {
		t.Errorf("trusted health output = %q, want %q", output, "Server status: ok\n")
	}

	output, err = runHealthCommand(ctx, serverAddress, "")
	if err == nil {
		t.Fatal("untrusted health command error = nil, want TLS verification error")
	}
	if output != "" {
		t.Errorf("untrusted health output = %q, want empty output", output)
	}

	var verificationError *tls.CertificateVerificationError
	if !errors.As(err, &verificationError) {
		t.Errorf("untrusted health command error = %v, want *tls.CertificateVerificationError", err)
	}

	testPool.Close()
	testPool = nil

	output, err = runHealthCommand(ctx, serverAddress, caCertFile)
	if err == nil {
		t.Fatal("unavailable database health command error = nil, want status error")
	}
	if output != "" {
		t.Errorf("unavailable database health output = %q, want empty output", output)
	}
	if !strings.Contains(err.Error(), "503 Service Unavailable") {
		t.Errorf("unavailable database health command error = %q, want status 503", err)
	}
}

func runHealthCommand(ctx context.Context, address, caCertFile string) (string, error) {
	var output bytes.Buffer

	args := []string{"gkeep", "--address", address}
	if caCertFile != "" {
		args = append(args, "--ca-cert", caCertFile)
	}
	args = append(args, "health")

	err := clientcli.Run(
		ctx,
		args,
		&output,
		io.Discard,
		buildinfo.Info{},
	)

	return output.String(), err
}
