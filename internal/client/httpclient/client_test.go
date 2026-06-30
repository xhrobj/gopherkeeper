package httpclient

import (
	"context"
	"crypto/x509"
	"encoding/pem"
	"net/http"
	"net/http/httptest"
	"os"
	"strings"
	"testing"
)

func TestHealthWithAdditionalCA(t *testing.T) {
	server := newHealthTLSServer(t, http.StatusOK, `{"status":"ok"}`)
	defer server.Close()

	client, err := New(serverAddress(server), writeServerCertificate(t, server))
	if err != nil {
		t.Fatalf("New() error = %v", err)
	}

	status, err := client.Health(context.Background())
	if err != nil {
		t.Fatalf("Health() error = %v", err)
	}

	if status != "ok" {
		t.Errorf("Health() status = %q, want %q", status, "ok")
	}
}

func TestHealthRejectsUntrustedCertificate(t *testing.T) {
	server := newHealthTLSServer(t, http.StatusOK, `{"status":"ok"}`)
	defer server.Close()

	client, err := New(serverAddress(server), "")
	if err != nil {
		t.Fatalf("New() error = %v", err)
	}

	_, err = client.Health(context.Background())
	if err == nil {
		t.Fatal("Health() error = nil, want TLS verification error")
	}
}

func TestHealthReturnsStatusError(t *testing.T) {
	server := newHealthTLSServer(
		t,
		http.StatusServiceUnavailable,
		`{"status":"unavailable"}`,
	)
	defer server.Close()

	client, err := New(serverAddress(server), writeServerCertificate(t, server))
	if err != nil {
		t.Fatalf("New() error = %v", err)
	}

	_, err = client.Health(context.Background())
	if err == nil {
		t.Fatal("Health() error = nil, want status error")
	}

	if !strings.Contains(err.Error(), "503 Service Unavailable") {
		t.Errorf("Health() error = %q, want status 503", err)
	}
}

func TestNewReturnsCertificateErrors(t *testing.T) {
	t.Run("missing file", func(t *testing.T) {
		_, err := New("localhost:8080", "missing-ca.pem")
		if err == nil {
			t.Fatal("New() error = nil, want file error")
		}
	})

	t.Run("invalid PEM", func(t *testing.T) {
		path := t.TempDir() + "/ca.pem"
		if err := os.WriteFile(path, []byte("not a certificate"), 0o600); err != nil {
			t.Fatalf("write CA certificate: %v", err)
		}

		_, err := New("localhost:8080", path)
		if err == nil {
			t.Fatal("New() error = nil, want PEM parsing error")
		}
	})
}

func newHealthTLSServer(t *testing.T, status int, body string) *httptest.Server {
	t.Helper()

	return httptest.NewTLSServer(http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		if r.Method != http.MethodGet {
			t.Errorf("method = %s, want %s", r.Method, http.MethodGet)
		}

		if r.URL.Path != "/health" {
			t.Errorf("path = %s, want /health", r.URL.Path)
		}

		w.Header().Set("Content-Type", "application/json")
		w.WriteHeader(status)

		if _, err := w.Write([]byte(body)); err != nil {
			t.Errorf("write response: %v", err)
		}
	}))
}

func writeServerCertificate(t *testing.T, server *httptest.Server) string {
	t.Helper()

	certificate, err := x509.ParseCertificate(server.Certificate().Raw)
	if err != nil {
		t.Fatalf("parse server certificate: %v", err)
	}

	data := pem.EncodeToMemory(&pem.Block{
		Type:  "CERTIFICATE",
		Bytes: certificate.Raw,
	})

	path := t.TempDir() + "/ca.pem"
	if err := os.WriteFile(path, data, 0o600); err != nil {
		t.Fatalf("write CA certificate: %v", err)
	}

	return path
}

func serverAddress(server *httptest.Server) string {
	return strings.TrimPrefix(server.URL, "https://")
}
