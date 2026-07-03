package auth

import (
	"bytes"
	"errors"
	"strings"
	"testing"
)

const testPassword = "correct-horse-battery-staple"

func TestBcryptPasswordManager_HashAndCheck(t *testing.T) {
	manager := NewBcryptPasswordManager()

	hash, err := manager.Hash(testPassword)
	if err != nil {
		t.Fatalf("Hash() error = %v", err)
	}

	if len(hash) == 0 {
		t.Fatal("Hash() returned an empty hash")
	}

	if bytes.Equal(hash, []byte(testPassword)) {
		t.Fatal("Hash() returned plaintext password")
	}

	if err := manager.Check(testPassword, hash); err != nil {
		t.Errorf("Check() error = %v", err)
	}
}

func TestBcryptPasswordManager_CheckMismatch(t *testing.T) {
	manager := NewBcryptPasswordManager()

	hash, err := manager.Hash(testPassword)
	if err != nil {
		t.Fatalf("Hash() error = %v", err)
	}

	err = manager.Check("wrong-password", hash)
	if !errors.Is(err, ErrPasswordMismatch) {
		t.Errorf("Check() error = %v, want %v", err, ErrPasswordMismatch)
	}
}

func TestBcryptPasswordManager_HashPasswordTooLong(t *testing.T) {
	manager := NewBcryptPasswordManager()
	password := strings.Repeat("a", maxBcryptPasswordLength+1)

	_, err := manager.Hash(password)
	if !errors.Is(err, ErrPasswordTooLong) {
		t.Errorf("Hash() error = %v, want %v", err, ErrPasswordTooLong)
	}
}

func TestBcryptPasswordManager_CheckPasswordTooLong(t *testing.T) {
	manager := NewBcryptPasswordManager()

	hash, err := manager.Hash(testPassword)
	if err != nil {
		t.Fatalf("Hash() error = %v", err)
	}

	password := strings.Repeat("a", maxBcryptPasswordLength+1)

	err = manager.Check(password, hash)
	if !errors.Is(err, ErrPasswordTooLong) {
		t.Errorf("Check() error = %v, want %v", err, ErrPasswordTooLong)
	}
}

func TestBcryptPasswordManager_HashMaximumLength(t *testing.T) {
	manager := NewBcryptPasswordManager()
	password := strings.Repeat("a", maxBcryptPasswordLength)

	hash, err := manager.Hash(password)
	if err != nil {
		t.Fatalf("Hash() error = %v", err)
	}

	if err := manager.Check(password, hash); err != nil {
		t.Errorf("Check() error = %v", err)
	}
}
