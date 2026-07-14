package cachecrypto

import (
	"bytes"
	"errors"
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

func TestService_DecryptRejectsChangedData(t *testing.T) {
	service := newTestService(t, "k")
	aad, err := BuildRecordAAD(testAccountID, testRecordID, 1)
	if err != nil {
		t.Fatalf("BuildRecordAAD() error = %v", err)
	}
	otherAccountAAD, err := BuildRecordAAD(strings.Repeat("f", 64), testRecordID, 1)
	if err != nil {
		t.Fatalf("BuildRecordAAD() other account error = %v", err)
	}
	otherRevisionAAD, err := BuildRecordAAD(testAccountID, testRecordID, 2)
	if err != nil {
		t.Fatalf("BuildRecordAAD() other revision error = %v", err)
	}
	encrypted, err := service.Encrypt([]byte("secret"), aad)
	if err != nil {
		t.Fatalf("Encrypt() error = %v", err)
	}

	changedCiphertext := encrypted
	changedCiphertext.Ciphertext = append([]byte(nil), encrypted.Ciphertext...)
	changedCiphertext.Ciphertext[0] ^= 0xff

	changedNonce := encrypted
	changedNonce.Nonce = append([]byte(nil), encrypted.Nonce...)
	changedNonce.Nonce[0] ^= 0xff

	tests := []struct {
		name      string
		encrypted EncryptedData
		aad       []byte
	}{
		{name: "wrong AAD", encrypted: encrypted, aad: []byte("wrong")},
		{name: "other account", encrypted: encrypted, aad: otherAccountAAD},
		{name: "other revision", encrypted: encrypted, aad: otherRevisionAAD},
		{name: "changed ciphertext", encrypted: changedCiphertext, aad: aad},
		{name: "changed nonce", encrypted: changedNonce, aad: aad},
		{name: "unsupported version", encrypted: EncryptedData{CryptoVersion: 69, Nonce: encrypted.Nonce, Ciphertext: encrypted.Ciphertext}, aad: aad},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			if _, err := service.Decrypt(tt.encrypted, tt.aad); !errors.Is(err, ErrOpenLocalCache) {
				t.Fatalf("Decrypt() error = %v, want ErrOpenLocalCache", err)
			}
		})
	}
}

func TestService_KeyCheck(t *testing.T) {
	service := newTestService(t, "k")
	keyCheck, err := service.CreateKeyCheck(testAccountID)
	if err != nil {
		t.Fatalf("CreateKeyCheck() error = %v", err)
	}
	if err := service.VerifyKeyCheck(testAccountID, keyCheck); err != nil {
		t.Fatalf("VerifyKeyCheck() error = %v", err)
	}

	wrongService := newTestService(t, "x")
	if err := wrongService.VerifyKeyCheck(testAccountID, keyCheck); !errors.Is(err, ErrOpenLocalCache) {
		t.Fatalf("VerifyKeyCheck() wrong key error = %v, want ErrOpenLocalCache", err)
	}
	if err := service.VerifyKeyCheck(strings.Repeat("f", 64), keyCheck); !errors.Is(err, ErrOpenLocalCache) {
		t.Fatalf("VerifyKeyCheck() other account error = %v, want ErrOpenLocalCache", err)
	}

	keyCheck.Ciphertext[0] ^= 0xff
	if err := service.VerifyKeyCheck(testAccountID, keyCheck); !errors.Is(err, ErrOpenLocalCache) {
		t.Fatalf("VerifyKeyCheck() changed verifier error = %v, want ErrOpenLocalCache", err)
	}
}

func TestNewService_RejectsInvalidKey(t *testing.T) {
	if _, err := NewService([]byte("short")); !errors.Is(err, ErrInvalidLocalKey) {
		t.Fatalf("NewService() error = %v, want ErrInvalidLocalKey", err)
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
