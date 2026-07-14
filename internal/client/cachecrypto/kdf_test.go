package cachecrypto

import (
	"bytes"
	"errors"
	"testing"
)

func TestDeriveKey(t *testing.T) {
	password := []byte("correct-horse-battery-staple")
	salt := []byte("0123456789abcdef")

	key, err := DeriveKey(password, salt, KDFVersion)
	if err != nil {
		t.Fatalf("DeriveKey() error = %v", err)
	}
	secondKey, err := DeriveKey(password, salt, KDFVersion)
	if err != nil {
		t.Fatalf("DeriveKey() repeated error = %v", err)
	}
	if !bytes.Equal(key, secondKey) {
		t.Fatal("DeriveKey() returned different keys for the same input")
	}

	otherPasswordKey, err := DeriveKey([]byte("another-password"), salt, KDFVersion)
	if err != nil {
		t.Fatalf("DeriveKey() other password error = %v", err)
	}
	if bytes.Equal(key, otherPasswordKey) {
		t.Fatal("DeriveKey() returned the same key for a different password")
	}

	otherSaltKey, err := DeriveKey(password, []byte("fedcba9876543210"), KDFVersion)
	if err != nil {
		t.Fatalf("DeriveKey() other salt error = %v", err)
	}
	if bytes.Equal(key, otherSaltKey) {
		t.Fatal("DeriveKey() returned the same key for a different salt")
	}
}

func TestDeriveKey_RejectsInvalidMetadata(t *testing.T) {
	if _, err := DeriveKey([]byte("password"), []byte("short"), KDFVersion); !errors.Is(err, ErrInvalidKDFSalt) {
		t.Fatalf("DeriveKey() salt error = %v, want ErrInvalidKDFSalt", err)
	}
	if _, err := DeriveKey([]byte("password"), []byte("0123456789abcdef"), 69); !errors.Is(err, ErrUnsupportedKDFVersion) {
		t.Fatalf("DeriveKey() version error = %v, want ErrUnsupportedKDFVersion", err)
	}
}

func TestGenerateSalt(t *testing.T) {
	salt, err := GenerateSalt()
	if err != nil {
		t.Fatalf("GenerateSalt() error = %v", err)
	}
	if len(salt) != kdfSaltSize {
		t.Fatalf("GenerateSalt() length = %d, want %d", len(salt), kdfSaltSize)
	}
}
