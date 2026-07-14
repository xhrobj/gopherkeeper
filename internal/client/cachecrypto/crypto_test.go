package cachecrypto

import (
	"bytes"
	"strings"
	"testing"
)

const testAccountID = "0123456789abcdef0123456789abcdef0123456789abcdef0123456789abcdef"
const testRecordID = "550e8400-e29b-41d4-a716-446655440000"

func TestService_EncryptDecrypt(t *testing.T) {
	service := newTestService(t, "k")
	aad, err := BuildRecordAAD(testAccountID, testRecordID, 1)
	if err != nil {
		t.Fatalf("BuildRecordAAD() error = %v", err)
	}
	plaintext := []byte(`{"text":"secret note"}`)

	encrypted, err := service.Encrypt(plaintext, aad)
	if err != nil {
		t.Fatalf("Encrypt() error = %v", err)
	}
	if bytes.Contains(encrypted.Ciphertext, plaintext) {
		t.Fatal("Encrypt() ciphertext contains plaintext")
	}

	decrypted, err := service.Decrypt(encrypted, aad)
	if err != nil {
		t.Fatalf("Decrypt() error = %v", err)
	}
	if !bytes.Equal(decrypted, plaintext) {
		t.Fatalf("Decrypt() = %q, want %q", decrypted, plaintext)
	}
}

func newTestService(t *testing.T, symbol string) *Service {
	t.Helper()

	service, err := NewService([]byte(strings.Repeat(symbol, aes256KeySize)))
	if err != nil {
		t.Fatalf("NewService() error = %v", err)
	}

	return service
}
