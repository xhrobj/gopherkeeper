//go:build integration

package integration_test

import (
	"context"
	"errors"
	"net"
	"os"
	"strings"
	"testing"
	"time"

	"github.com/jackc/pgx/v5/pgxpool"
	clientapp "github.com/xhrobj/gopherkeeper/internal/client/app"
	"github.com/xhrobj/gopherkeeper/internal/client/config"
	"github.com/xhrobj/gopherkeeper/internal/server/auth"
	"github.com/xhrobj/gopherkeeper/internal/server/postgres"
	"github.com/xhrobj/gopherkeeper/internal/server/recordcrypto"
	"github.com/xhrobj/gopherkeeper/internal/server/service"
	grpcserver "github.com/xhrobj/gopherkeeper/internal/server/transport/grpc"
	httpserver "github.com/xhrobj/gopherkeeper/internal/server/transport/http"
	"go.uber.org/zap"
	"google.golang.org/grpc"
)

type dualTransportFixture struct {
	ctx          context.Context
	httpsAddress string
	grpcAddress  string
	caCertFile   string
	sessionDir   string
	cacheDir     string
}

func newDualTransportFixture(t *testing.T) dualTransportFixture {
	t.Helper()

	dsn := strings.TrimSpace(os.Getenv("DATABASE_DSN"))
	if dsn == "" {
		t.Fatal("DATABASE_DSN is not set")
	}

	isolateClientConfig(t)
	ctx, cancel := context.WithTimeout(context.Background(), 2*integrationTestTimeout)
	t.Cleanup(cancel)

	pool := openIsolatedMigratedDatabase(t, ctx, dsn)
	httpDependencies, grpcDependencies := newDualTransportDependencies(t, pool)
	caCertFile, serverCertFile, serverKeyFile := generateTLSFiles(t)

	httpsAddress, stopHTTPS := startHTTPSServer(
		t,
		httpserver.NewHandler(httpDependencies),
		serverCertFile,
		serverKeyFile,
	)
	t.Cleanup(stopHTTPS)

	grpcAddress := startIntegrationGRPCServer(
		t,
		serverCertFile,
		serverKeyFile,
		grpcDependencies,
	)

	return dualTransportFixture{
		ctx:          ctx,
		httpsAddress: httpsAddress,
		grpcAddress:  grpcAddress,
		caCertFile:   caCertFile,
		sessionDir:   t.TempDir(),
		cacheDir:     t.TempDir(),
	}
}

func newDualTransportDependencies(
	t *testing.T,
	pool *pgxpool.Pool,
) (httpserver.Dependencies, grpcserver.Dependencies) {
	t.Helper()

	userRepository := postgres.NewUserRepository(pool)
	recordRepository := postgres.NewRecordRepository(pool)
	passwordManager := auth.NewBcryptPasswordManager()
	registrationService := service.NewRegistrationService(userRepository, passwordManager)
	tokenManager := auth.NewJWTTokenManager(integrationJWTSecret, 15*time.Minute)
	authenticationService := service.NewAuthenticationService(
		userRepository,
		passwordManager,
		tokenManager,
	)
	recordCrypto, err := recordcrypto.NewService(
		[]byte(strings.Repeat("k", recordcrypto.MasterKeySize)),
		recordcrypto.DefaultKeyID,
	)
	if err != nil {
		t.Fatalf("create record crypto service: %v", err)
	}
	recordService := service.NewRecordService(recordRepository, recordCrypto)

	return httpserver.Dependencies{
			Database:          pool,
			Registerer:        registrationService,
			Authenticator:     authenticationService,
			TokenValidator:    tokenManager,
			CurrentUserReader: userRepository,
			Records:           recordService,
		}, grpcserver.Dependencies{
			Database:          pool,
			Registerer:        registrationService,
			Authenticator:     authenticationService,
			TokenValidator:    tokenManager,
			CurrentUserReader: userRepository,
			Records:           recordService,
		}
}

func startIntegrationGRPCServer(
	t *testing.T,
	certFile string,
	keyFile string,
	dependencies grpcserver.Dependencies,
) string {
	t.Helper()

	listener, err := net.Listen("tcp", "127.0.0.1:0")
	if err != nil {
		t.Fatalf("listen on free gRPC port: %v", err)
	}

	server, err := grpcserver.NewServer(certFile, keyFile, dependencies, zap.NewNop())
	if err != nil {
		_ = listener.Close()
		t.Fatalf("create gRPC server: %v", err)
	}

	serveErr := make(chan error, 1)
	go func() {
		serveErr <- server.Serve(listener)
	}()

	t.Cleanup(func() {
		server.Stop()
		select {
		case err := <-serveErr:
			if err != nil && !errors.Is(err, grpc.ErrServerStopped) {
				t.Errorf("serve gRPC: %v", err)
			}
		case <-time.After(integrationTestTimeout):
			t.Error("wait for gRPC server shutdown: timeout")
		}
	})

	return listener.Addr().String()
}

func (fixture dualTransportFixture) clientConfig(transport config.Transport) config.Config {
	return config.Config{
		Transport:   transport,
		Address:     fixture.httpsAddress,
		GRPCAddress: fixture.grpcAddress,
		CACertFile:  fixture.caCertFile,
		SessionDir:  fixture.sessionDir,
		CacheDir:    fixture.cacheDir,
	}
}

func (fixture dualTransportFixture) newRuntime(
	t *testing.T,
	transport config.Transport,
) *clientapp.Runtime {
	t.Helper()

	runtime, err := clientapp.NewRuntime(fixture.clientConfig(transport))
	if err != nil {
		t.Fatalf("create %s runtime: %v", transport, err)
	}
	t.Cleanup(func() {
		if err := runtime.Close(); err != nil {
			t.Errorf("close %s runtime: %v", transport, err)
		}
	})

	return runtime
}

func reserveClosedAddress(t *testing.T) string {
	t.Helper()

	listener, err := net.Listen("tcp", "127.0.0.1:0")
	if err != nil {
		t.Fatalf("reserve closed address: %v", err)
	}
	address := listener.Addr().String()
	if err := listener.Close(); err != nil {
		t.Fatalf("close reserved address: %v", err)
	}

	return address
}
