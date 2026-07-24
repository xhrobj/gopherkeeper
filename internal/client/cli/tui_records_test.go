package cli

import (
	"path/filepath"
	"strings"
	"testing"

	"github.com/xhrobj/gopherkeeper/internal/client/config"
)

func TestNewTUIBackend_ReturnsApplicationError(t *testing.T) {
	_, err := newTUIBackend(config.Config{
		Address:    "localhost:8443",
		CACertFile: filepath.Join(t.TempDir(), "missing-ca.pem"),
	})
	if err == nil || !strings.Contains(err.Error(), "create client application") {
		t.Fatalf("newTUIBackend() error = %v, want application creation error", err)
	}
}
