//go:build integration

package integration_test

import (
	"bytes"
	"context"
	"crypto/rand"
	"crypto/rsa"
	"crypto/tls"
	"crypto/x509"
	"crypto/x509/pkix"
	"encoding/pem"
	"errors"
	"fmt"
	"io"
	"log"
	"math/big"
	"net"
	"net/http"
	"os"
	"path/filepath"
	"strings"
	"testing"
	"time"

	"github.com/jackc/pgx/v5"
	"github.com/jackc/pgx/v5/pgxpool"
	"github.com/xhrobj/gopherkeeper/internal/buildinfo"
	clientcli "github.com/xhrobj/gopherkeeper/internal/client/cli"
	"github.com/xhrobj/gopherkeeper/internal/server/httpserver"
	"github.com/xhrobj/gopherkeeper/internal/server/migration"
	"github.com/xhrobj/gopherkeeper/internal/server/postgres"
)

const integrationTestTimeout = 30 * time.Second

func TestHTTPSHealthFlow(t *testing.T) {
	dsn := os.Getenv("DATABASE_DSN")
	if dsn == "" {
		t.Skip("DATABASE_DSN is not set")
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

	testDSN := databaseDSN(t, dsn, databaseName)
	testPool := openPostgres(t, ctx, testDSN)
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
		httpserver.NewHandler(testPool),
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

func openPostgres(t *testing.T, ctx context.Context, dsn string) *pgxpool.Pool {
	t.Helper()

	pool, err := postgres.Open(ctx, dsn)
	if err != nil {
		t.Fatalf("open PostgreSQL: %v", err)
	}

	return pool
}

func createTestDatabase(t *testing.T, ctx context.Context, pool *pgxpool.Pool) string {
	t.Helper()

	databaseName := fmt.Sprintf("gopherkeeper_health_%d", time.Now().UnixNano())
	quotedDatabaseName := pgx.Identifier{databaseName}.Sanitize()

	if _, err := pool.Exec(ctx, "CREATE DATABASE "+quotedDatabaseName); err != nil {
		t.Fatalf("create test database: %v", err)
	}

	return databaseName
}

func dropTestDatabase(t *testing.T, pool *pgxpool.Pool, databaseName string) {
	t.Helper()

	ctx, cancel := context.WithTimeout(context.Background(), integrationTestTimeout)
	defer cancel()

	quotedDatabaseName := pgx.Identifier{databaseName}.Sanitize()
	if _, err := pool.Exec(ctx, "DROP DATABASE "+quotedDatabaseName+" WITH (FORCE)"); err != nil {
		t.Errorf("drop test database: %v", err)
	}
}

func databaseDSN(t *testing.T, dsn, databaseName string) string {
	t.Helper()

	cfg, err := pgxpool.ParseConfig(dsn)
	if err != nil {
		t.Fatalf("parse DATABASE_DSN: %v", err)
	}
	cfg.ConnConfig.Database = databaseName

	return cfg.ConnString()
}

func startHTTPSServer(
	t *testing.T,
	handler http.Handler,
	certFile string,
	keyFile string,
) (string, func()) {
	t.Helper()

	listener, err := net.Listen("tcp", "127.0.0.1:0")
	if err != nil {
		t.Fatalf("listen on free port: %v", err)
	}

	server := httpserver.NewServer(listener.Addr().String(), handler)
	server.ErrorLog = log.New(io.Discard, "", 0)

	serveErr := make(chan error, 1)
	go func() {
		serveErr <- server.ServeTLS(listener, certFile, keyFile)
	}()

	stop := func() {
		shutdownCtx, cancel := context.WithTimeout(context.Background(), integrationTestTimeout)
		defer cancel()

		if err := server.Shutdown(shutdownCtx); err != nil {
			t.Errorf("shutdown HTTPS server: %v", err)
		}

		if err := <-serveErr; err != nil && !errors.Is(err, http.ErrServerClosed) {
			t.Errorf("serve HTTPS: %v", err)
		}
	}

	return listener.Addr().String(), stop
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

func generateTLSFiles(t *testing.T) (string, string, string) {
	t.Helper()

	now := time.Now()
	caKey := generateRSAKey(t)
	caTemplate := &x509.Certificate{
		SerialNumber:          big.NewInt(1),
		Subject:               pkix.Name{CommonName: "GophKeeper Integration CA"},
		NotBefore:             now.Add(-time.Minute),
		NotAfter:              now.Add(time.Hour),
		KeyUsage:              x509.KeyUsageCertSign | x509.KeyUsageCRLSign,
		BasicConstraintsValid: true,
		IsCA:                  true,
	}
	caDER := createCertificate(t, caTemplate, caTemplate, &caKey.PublicKey, caKey)

	serverKey := generateRSAKey(t)
	serverTemplate := &x509.Certificate{
		SerialNumber:          big.NewInt(2),
		Subject:               pkix.Name{CommonName: "server"},
		NotBefore:             now.Add(-time.Minute),
		NotAfter:              now.Add(time.Hour),
		KeyUsage:              x509.KeyUsageDigitalSignature | x509.KeyUsageKeyEncipherment,
		ExtKeyUsage:           []x509.ExtKeyUsage{x509.ExtKeyUsageServerAuth},
		BasicConstraintsValid: true,
		IPAddresses:           []net.IP{net.ParseIP("127.0.0.1")},
	}
	serverDER := createCertificate(t, serverTemplate, caTemplate, &serverKey.PublicKey, caKey)

	dir := t.TempDir()
	caCertFile := filepath.Join(dir, "ca.pem")
	serverCertFile := filepath.Join(dir, "server.pem")
	serverKeyFile := filepath.Join(dir, "server-key.pem")

	writePEM(t, caCertFile, "CERTIFICATE", caDER, 0o644)
	writePEM(t, serverCertFile, "CERTIFICATE", serverDER, 0o644)
	writePEM(t, serverKeyFile, "RSA PRIVATE KEY", x509.MarshalPKCS1PrivateKey(serverKey), 0o600)

	return caCertFile, serverCertFile, serverKeyFile
}

func generateRSAKey(t *testing.T) *rsa.PrivateKey {
	t.Helper()

	key, err := rsa.GenerateKey(rand.Reader, 2048)
	if err != nil {
		t.Fatalf("generate RSA key: %v", err)
	}

	return key
}

func createCertificate(
	t *testing.T,
	template *x509.Certificate,
	parent *x509.Certificate,
	publicKey *rsa.PublicKey,
	parentKey *rsa.PrivateKey,
) []byte {
	t.Helper()

	certificate, err := x509.CreateCertificate(
		rand.Reader,
		template,
		parent,
		publicKey,
		parentKey,
	)
	if err != nil {
		t.Fatalf("create certificate: %v", err)
	}

	return certificate
}

func writePEM(t *testing.T, path, blockType string, data []byte, mode os.FileMode) {
	t.Helper()

	file, err := os.OpenFile(path, os.O_WRONLY|os.O_CREATE|os.O_TRUNC, mode)
	if err != nil {
		t.Fatalf("create %s: %v", path, err)
	}

	if err := pem.Encode(file, &pem.Block{Type: blockType, Bytes: data}); err != nil {
		_ = file.Close()
		t.Fatalf("write %s: %v", path, err)
	}

	if err := file.Close(); err != nil {
		t.Fatalf("close %s: %v", path, err)
	}
}
