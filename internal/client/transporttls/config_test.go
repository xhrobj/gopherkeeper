package transporttls

import (
	"crypto/tls"
	"os"
	"path/filepath"
	"testing"

	"github.com/xhrobj/gopherkeeper/internal/client/failure"
)

func TestNewConfig(t *testing.T) {
	config, err := NewConfig("")
	if err != nil {
		t.Fatalf("NewConfig() error = %v", err)
	}
	if config.MinVersion != tls.VersionTLS12 {
		t.Fatalf("NewConfig() MinVersion = %d, want %d", config.MinVersion, tls.VersionTLS12)
	}
	if config.RootCAs == nil {
		t.Fatal("NewConfig() RootCAs = nil")
	}
}

func TestNewConfig_ReturnsCertificateErrors(t *testing.T) {
	t.Run("missing file", func(t *testing.T) {
		_, err := NewConfig(filepath.Join(t.TempDir(), "missing.pem"))
		if err == nil {
			t.Fatal("NewConfig() error = nil, want file error")
		}
		if got := failure.KindOf(err); got != failure.TLSCertificate {
			t.Fatalf("failure.KindOf() = %d, want %d", got, failure.TLSCertificate)
		}
	})

	t.Run("invalid PEM", func(t *testing.T) {
		path := filepath.Join(t.TempDir(), "ca.pem")
		if err := os.WriteFile(path, []byte("not a certificate"), 0o600); err != nil {
			t.Fatalf("write CA certificate: %v", err)
		}

		_, err := NewConfig(path)
		if err == nil {
			t.Fatal("NewConfig() error = nil, want PEM parsing error")
		}
		if got := failure.KindOf(err); got != failure.TLSCertificate {
			t.Fatalf("failure.KindOf() = %d, want %d", got, failure.TLSCertificate)
		}
	})
}
