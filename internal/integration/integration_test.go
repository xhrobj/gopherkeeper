//go:build integration

package integration_test

import (
	"bytes"
	"context"
	"crypto/rand"
	"crypto/rsa"
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
	"testing"
	"time"

	"github.com/jackc/pgx/v5"
	"github.com/jackc/pgx/v5/pgxpool"
	"github.com/xhrobj/gopherkeeper/internal/model"
	"github.com/xhrobj/gopherkeeper/internal/server/auth"
	"github.com/xhrobj/gopherkeeper/internal/server/httpserver"
	"github.com/xhrobj/gopherkeeper/internal/server/postgres"
	"github.com/xhrobj/gopherkeeper/internal/server/service"
)

const (
	integrationTestTimeout   = 30 * time.Second
	testRegistrationPassword = "correct-horse-battery-staple"
)

var integrationJWTSecret = []byte("0123456789abcdef0123456789abcdef")

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

	databaseName := fmt.Sprintf("gopherkeeper_test_%d", time.Now().UnixNano())
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

func openTestPostgres(
	t *testing.T,
	ctx context.Context,
	dsn string,
	databaseName string,
) *pgxpool.Pool {
	t.Helper()

	cfg, err := pgxpool.ParseConfig(dsn)
	if err != nil {
		t.Fatalf("parse DATABASE_DSN: %v", err)
	}
	cfg.ConnConfig.Database = databaseName

	pool, err := pgxpool.NewWithConfig(ctx, cfg)
	if err != nil {
		t.Fatalf("open test PostgreSQL: %v", err)
	}

	if err := pool.Ping(ctx); err != nil {
		pool.Close()
		t.Fatalf("ping test PostgreSQL: %v", err)
	}

	return pool
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

func newServerHandler(pool *pgxpool.Pool) http.Handler {
	userRepository := postgres.NewUserRepository(pool)
	passwordManager := auth.NewBcryptPasswordManager()
	registrationService := service.NewRegistrationService(userRepository, passwordManager)

	return httpserver.NewHandler(
		pool,
		registrationService,
		unusedIntegrationAuthenticator,
		unusedIntegrationTokenValidator,
		unusedIntegrationCurrentUserReader,
	)
}

func newAuthenticatedServerHandler(pool *pgxpool.Pool) http.Handler {
	userRepository := postgres.NewUserRepository(pool)
	passwordManager := auth.NewBcryptPasswordManager()
	registrationService := service.NewRegistrationService(userRepository, passwordManager)
	tokenManager := auth.NewJWTTokenManager(integrationJWTSecret, 15*time.Minute)
	authenticationService := service.NewAuthenticationService(
		userRepository,
		passwordManager,
		tokenManager,
	)

	return httpserver.NewHandler(
		pool,
		registrationService,
		authenticationService,
		tokenManager,
		userRepository,
	)
}

var unusedIntegrationAuthenticator = integrationAuthenticatorFunc(func(
	context.Context,
	string,
	string,
) (service.AuthenticationResult, error) {
	return service.AuthenticationResult{}, errors.New("unexpected authentication call")
})

type integrationAuthenticatorFunc func(
	ctx context.Context,
	login string,
	password string,
) (service.AuthenticationResult, error)

func (f integrationAuthenticatorFunc) Authenticate(
	ctx context.Context,
	login string,
	password string,
) (service.AuthenticationResult, error) {
	return f(ctx, login, password)
}

var unusedIntegrationTokenValidator = integrationTokenValidatorFunc(func(
	context.Context,
	string,
) (int64, error) {
	return 0, errors.New("unexpected token validation call")
})

type integrationTokenValidatorFunc func(context.Context, string) (int64, error)

func (f integrationTokenValidatorFunc) Validate(ctx context.Context, token string) (int64, error) {
	return f(ctx, token)
}

var unusedIntegrationCurrentUserReader = integrationCurrentUserReaderFunc(func(
	context.Context,
	int64,
) (model.User, error) {
	return model.User{}, errors.New("unexpected current user read call")
})

type integrationCurrentUserReaderFunc func(context.Context, int64) (model.User, error)

func (f integrationCurrentUserReaderFunc) FindByID(ctx context.Context, id int64) (model.User, error) {
	return f(ctx, id)
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
