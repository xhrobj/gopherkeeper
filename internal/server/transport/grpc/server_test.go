package grpcserver

import (
	"context"
	"crypto/rand"
	"crypto/rsa"
	"crypto/x509"
	"crypto/x509/pkix"
	"encoding/pem"
	"math/big"
	"net"
	"os"
	"path/filepath"
	"strings"
	"testing"
	"time"

	"google.golang.org/grpc"
)

func TestNewServer_ReturnsInvalidDependenciesError(t *testing.T) {
	tests := []struct {
		name      string
		change    func(*Dependencies)
		wantError string
	}{
		{name: "database", change: func(deps *Dependencies) { deps.Database = nil }, wantError: "database is required"},
		{name: "registerer", change: func(deps *Dependencies) { deps.Registerer = nil }, wantError: "registerer is required"},
		{name: "authenticator", change: func(deps *Dependencies) { deps.Authenticator = nil }, wantError: "authenticator is required"},
		{name: "token validator", change: func(deps *Dependencies) { deps.TokenValidator = nil }, wantError: "token validator is required"},
		{name: "current user reader", change: func(deps *Dependencies) { deps.CurrentUserReader = nil }, wantError: "current user reader is required"},
		{name: "record manager", change: func(deps *Dependencies) { deps.Records = nil }, wantError: "record manager is required"},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			deps := completeDependencies()
			tt.change(&deps)

			_, err := NewServer("missing-cert.pem", "missing-key.pem", deps, nil)
			if err == nil {
				t.Fatal("NewServer() error = nil, want dependencies error")
			}
			if !strings.Contains(err.Error(), tt.wantError) {
				t.Fatalf("NewServer() error = %q, want substring %q", err, tt.wantError)
			}
		})
	}
}

func TestNewServer_ReturnsTLSCredentialsError(t *testing.T) {
	_, err := NewServer("missing-cert.pem", "missing-key.pem", completeDependencies(), nil)
	if err == nil {
		t.Fatal("NewServer() error = nil, want TLS credentials error")
	}
	if !strings.Contains(err.Error(), "load gRPC TLS credentials") {
		t.Fatalf("NewServer() error = %q, want TLS credentials context", err)
	}
}

func TestNewServer_RegistersServices(t *testing.T) {
	certFile, keyFile := grpcTestCertificateFiles(t)

	server, err := NewServer(certFile, keyFile, completeDependencies(), nil)
	if err != nil {
		t.Fatalf("NewServer() error = %v", err)
	}
	defer server.Stop()
	services := server.GetServiceInfo()
	for _, serviceName := range []string{
		"gopherkeeper.v1.AuthService",
		"gopherkeeper.v1.RecordService",
		"grpc.health.v1.Health",
	} {
		if _, ok := services[serviceName]; !ok {
			t.Fatalf("registered services = %v, missing %s", services, serviceName)
		}
	}
}

func TestServe_ReturnsListenError(t *testing.T) {
	listener, err := net.Listen("tcp", "127.0.0.1:0")
	if err != nil {
		t.Fatalf("listen test address: %v", err)
	}
	t.Cleanup(func() {
		if err := listener.Close(); err != nil {
			t.Errorf("close test listener: %v", err)
		}
	})

	err = Serve(context.Background(), listener.Addr().String(), grpc.NewServer())
	if err == nil {
		t.Fatal("Serve() error = nil, want listen error")
	}
	if !strings.Contains(err.Error(), "listen gRPC") {
		t.Fatalf("Serve() error = %q, want listen context", err)
	}
}

func TestServe_StopsAfterContextCancellation(t *testing.T) {
	address := freeTCPAddress(t)
	ctx, cancel := context.WithCancel(context.Background())
	server := grpc.NewServer()

	serveErr := make(chan error, 1)
	go func() {
		serveErr <- Serve(ctx, address, server)
	}()

	waitForTCPListener(t, address)
	cancel()

	select {
	case err := <-serveErr:
		if err != nil {
			t.Fatalf("Serve() error = %v", err)
		}
	case <-time.After(5 * time.Second):
		t.Fatal("Serve() did not stop after context cancellation")
	}
}

func freeTCPAddress(t *testing.T) string {
	t.Helper()

	listener, err := net.Listen("tcp", "127.0.0.1:0")
	if err != nil {
		t.Fatalf("listen free TCP address: %v", err)
	}
	address := listener.Addr().String()
	if err := listener.Close(); err != nil {
		t.Fatalf("close free TCP listener: %v", err)
	}

	return address
}

func waitForTCPListener(t *testing.T, address string) {
	t.Helper()

	deadline := time.Now().Add(5 * time.Second)
	for time.Now().Before(deadline) {
		connection, err := net.DialTimeout("tcp", address, 50*time.Millisecond)
		if err == nil {
			_ = connection.Close()
			return
		}
		time.Sleep(10 * time.Millisecond)
	}

	t.Fatalf("gRPC listener %s did not start", address)
}

func grpcTestCertificateFiles(t *testing.T) (string, string) {
	t.Helper()

	privateKey, err := rsa.GenerateKey(rand.Reader, 2048)
	if err != nil {
		t.Fatalf("rsa.GenerateKey() error = %v", err)
	}
	now := time.Now().UTC()
	template := &x509.Certificate{
		SerialNumber:          big.NewInt(42),
		Subject:               pkix.Name{CommonName: "localhost"},
		NotBefore:             now.Add(-time.Minute),
		NotAfter:              now.Add(time.Hour),
		KeyUsage:              x509.KeyUsageDigitalSignature | x509.KeyUsageKeyEncipherment,
		ExtKeyUsage:           []x509.ExtKeyUsage{x509.ExtKeyUsageServerAuth},
		BasicConstraintsValid: true,
		DNSNames:              []string{"localhost"},
	}
	certificate, err := x509.CreateCertificate(
		rand.Reader,
		template,
		template,
		&privateKey.PublicKey,
		privateKey,
	)
	if err != nil {
		t.Fatalf("x509.CreateCertificate() error = %v", err)
	}

	directory := t.TempDir()
	certFile := filepath.Join(directory, "server.crt")
	keyFile := filepath.Join(directory, "server.key")
	writeGRPCTestPEM(t, certFile, "CERTIFICATE", certificate, 0o644)
	writeGRPCTestPEM(t, keyFile, "RSA PRIVATE KEY", x509.MarshalPKCS1PrivateKey(privateKey), 0o600)

	return certFile, keyFile
}

func writeGRPCTestPEM(t *testing.T, path, blockType string, data []byte, mode os.FileMode) {
	t.Helper()

	file, err := os.OpenFile(path, os.O_WRONLY|os.O_CREATE|os.O_TRUNC, mode)
	if err != nil {
		t.Fatalf("os.OpenFile() error = %v", err)
	}
	if err := pem.Encode(file, &pem.Block{Type: blockType, Bytes: data}); err != nil {
		_ = file.Close()
		t.Fatalf("pem.Encode() error = %v", err)
	}
	if err := file.Close(); err != nil {
		t.Fatalf("Close() error = %v", err)
	}
}
