package cache

import (
	"path/filepath"
	"strings"
	"testing"
)

func TestResolveLocation(t *testing.T) {
	baseDirectory := t.TempDir()

	first, err := ResolveLocation(baseDirectory, "LOCALHOST:08080", "alice")
	if err != nil {
		t.Fatalf("ResolveLocation() error = %v", err)
	}
	second, err := ResolveLocation(baseDirectory, "localhost:8080", "alice")
	if err != nil {
		t.Fatalf("ResolveLocation() repeated error = %v", err)
	}
	otherLogin, err := ResolveLocation(baseDirectory, "localhost:8080", "bob")
	if err != nil {
		t.Fatalf("ResolveLocation() other login error = %v", err)
	}
	otherServer, err := ResolveLocation(baseDirectory, "example.com:8080", "alice")
	if err != nil {
		t.Fatalf("ResolveLocation() other server error = %v", err)
	}

	if first != second {
		t.Fatalf("ResolveLocation() = %+v, repeated = %+v", first, second)
	}
	if first.AccountID == otherLogin.AccountID || first.AccountID == otherServer.AccountID {
		t.Fatal("ResolveLocation() did not separate accounts")
	}
	if first.DatabaseFile != filepath.Join(first.Directory, databaseFileName) {
		t.Fatalf("DatabaseFile = %q, want file inside account directory", first.DatabaseFile)
	}
	if strings.Contains(first.DatabaseFile, "alice") || strings.Contains(first.DatabaseFile, "localhost") {
		t.Fatalf("DatabaseFile %q exposes account identity", first.DatabaseFile)
	}
}
