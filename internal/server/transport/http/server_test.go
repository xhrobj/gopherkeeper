package httpserver

import (
	"crypto/tls"
	"net/http"
	"testing"
)

func TestNewServer(t *testing.T) {
	handler := http.NewServeMux()

	server := NewServer("localhost:8080", handler)

	if server.Addr != "localhost:8080" {
		t.Errorf("Addr = %q, want localhost:8080", server.Addr)
	}

	if server.Handler != handler {
		t.Error("Handler differs from the provided handler")
	}

	if server.ReadHeaderTimeout != readHeaderTimeout {
		t.Errorf("ReadHeaderTimeout = %s, want %s", server.ReadHeaderTimeout, readHeaderTimeout)
	}

	if server.ReadTimeout != readTimeout {
		t.Errorf("ReadTimeout = %s, want %s", server.ReadTimeout, readTimeout)
	}

	if server.WriteTimeout != writeTimeout {
		t.Errorf("WriteTimeout = %s, want %s", server.WriteTimeout, writeTimeout)
	}

	if server.IdleTimeout != idleTimeout {
		t.Errorf("IdleTimeout = %s, want %s", server.IdleTimeout, idleTimeout)
	}

	if server.TLSConfig == nil {
		t.Fatal("TLSConfig is nil")
	}

	if server.TLSConfig.MinVersion != tls.VersionTLS12 {
		t.Errorf("TLS MinVersion = %d, want %d", server.TLSConfig.MinVersion, tls.VersionTLS12)
	}
}
