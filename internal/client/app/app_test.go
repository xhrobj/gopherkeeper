package app

import (
	"testing"

	"github.com/xhrobj/gopherkeeper/internal/client/config"
)

func TestNew(t *testing.T) {
	application, err := New(config.Config{
		Address:     "localhost:8080",
		SessionFile: t.TempDir() + "/session.json",
	})
	if err != nil {
		t.Fatalf("New() error = %v", err)
	}
	if application == nil {
		t.Fatal("New() application = nil")
	}
}

func TestNewLogout(t *testing.T) {
	application, err := NewLogout(config.Config{
		SessionFile: t.TempDir() + "/session.json",
	})
	if err != nil {
		t.Fatalf("NewLogout() error = %v", err)
	}
	if application == nil {
		t.Fatal("NewLogout() application = nil")
	}
}
