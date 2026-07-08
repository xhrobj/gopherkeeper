package recordcrypto

import (
	"bytes"
	"errors"
	"strings"
	"testing"

	"github.com/xhrobj/gopherkeeper/internal/model"
)

const testRecordID = "550e8400-e29b-41d4-a716-446655440000"

func TestNewService(t *testing.T) {
	masterKey := []byte(strings.Repeat("k", MasterKeySize))

	service, err := NewService(masterKey, " primary ")
	if err != nil {
		t.Fatalf("NewService() error = %v", err)
	}
	if service.keyID != DefaultKeyID {
		t.Fatalf("NewService() keyID = %q, want %q", service.keyID, DefaultKeyID)
	}

	if _, err := NewService([]byte("short"), DefaultKeyID); !errors.Is(err, ErrInvalidMasterKey) {
		t.Fatalf("NewService() short key error = %v, want ErrInvalidMasterKey", err)
	}
	if _, err := NewService(masterKey, " \t"); !errors.Is(err, ErrInvalidKeyID) {
		t.Fatalf("NewService() blank key id error = %v, want ErrInvalidKeyID", err)
	}
}

func TestBuildAAD(t *testing.T) {
	aad, err := BuildAAD(42, testRecordID, model.RecordTypeText)
	if err != nil {
		t.Fatalf("BuildAAD() error = %v", err)
	}

	want := "gopherkeeper:v1:user:42:record:550e8400-e29b-41d4-a716-446655440000:type:text"
	if string(aad) != want {
		t.Fatalf("BuildAAD() = %q, want %q", aad, want)
	}
}

func TestBuildAAD_RejectsInvalidInput(t *testing.T) {
	tests := []struct {
		name       string
		userID     int64
		recordID   string
		recordType model.RecordType
	}{
		{name: "invalid user", userID: 0, recordID: testRecordID, recordType: model.RecordTypeText},
		{name: "invalid record ID", userID: 42, recordID: "not-a-uuid", recordType: model.RecordTypeText},
		{name: "invalid record type", userID: 42, recordID: testRecordID, recordType: model.RecordType("otp")},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			_, err := BuildAAD(tt.userID, tt.recordID, tt.recordType)
			if !errors.Is(err, ErrInvalidAAD) {
				t.Fatalf("BuildAAD() error = %v, want ErrInvalidAAD", err)
			}
		})
	}
}

func TestService_EncryptDecrypt(t *testing.T) {
	service := newTestService(t)
	plaintext := []byte(`{"text":"secret note"}`)
	aad := []byte("test aad")

	encrypted, err := service.Encrypt(plaintext, aad)
	if err != nil {
		t.Fatalf("Encrypt() error = %v", err)
	}
	if encrypted.CryptoVersion != CryptoVersion {
		t.Fatalf("Encrypt() crypto version = %d, want %d", encrypted.CryptoVersion, CryptoVersion)
	}
	if encrypted.KeyID != DefaultKeyID {
		t.Fatalf("Encrypt() keyID = %q, want %q", encrypted.KeyID, DefaultKeyID)
	}
	if len(encrypted.Nonce) != service.aead.NonceSize() {
		t.Fatalf("Encrypt() nonce length = %d, want %d", len(encrypted.Nonce), service.aead.NonceSize())
	}
	if bytes.Contains(encrypted.Ciphertext, plaintext) {
		t.Fatal("Encrypt() ciphertext contains plaintext")
	}

	decrypted, err := service.Decrypt(encrypted, aad)
	if err != nil {
		t.Fatalf("Decrypt() error = %v", err)
	}
	if !bytes.Equal(decrypted, plaintext) {
		t.Fatalf("Decrypt() plaintext = %q, want %q", decrypted, plaintext)
	}
}

func TestService_DecryptRejectsTampering(t *testing.T) {
	service := newTestService(t)
	encrypted, err := service.Encrypt([]byte("secret"), []byte("test aad"))
	if err != nil {
		t.Fatalf("Encrypt() error = %v", err)
	}

	tests := []struct {
		name      string
		encrypted EncryptedPayload
		aad       []byte
	}{
		{name: "wrong version", encrypted: withVersion(encrypted, 69), aad: []byte("test aad")},
		{name: "wrong key ID", encrypted: withKeyID(encrypted, "secondary"), aad: []byte("test aad")},
		{name: "wrong nonce", encrypted: withNonce(encrypted, []byte("short")), aad: []byte("test aad")},
		{name: "empty ciphertext", encrypted: withCiphertext(encrypted, nil), aad: []byte("test aad")},
		{name: "wrong AAD", encrypted: encrypted, aad: []byte("another aad")},
		{name: "changed ciphertext", encrypted: withChangedCiphertext(encrypted), aad: []byte("test aad")},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			_, err := service.Decrypt(tt.encrypted, tt.aad)
			if !errors.Is(err, ErrDecryptPayload) {
				t.Fatalf("Decrypt() error = %v, want ErrDecryptPayload", err)
			}
		})
	}
}

func newTestService(t *testing.T) *Service {
	t.Helper()

	service, err := newService(
		[]byte(strings.Repeat("k", MasterKeySize)),
		DefaultKeyID,
		bytes.NewReader([]byte("123456789012123456789012")),
	)
	if err != nil {
		t.Fatalf("newService() error = %v", err)
	}

	return service
}

func withVersion(encrypted EncryptedPayload, version int) EncryptedPayload {
	encrypted.CryptoVersion = version
	return encrypted
}

func withKeyID(encrypted EncryptedPayload, keyID string) EncryptedPayload {
	encrypted.KeyID = keyID
	return encrypted
}

func withNonce(encrypted EncryptedPayload, nonce []byte) EncryptedPayload {
	encrypted.Nonce = nonce
	return encrypted
}

func withCiphertext(encrypted EncryptedPayload, ciphertext []byte) EncryptedPayload {
	encrypted.Ciphertext = ciphertext
	return encrypted
}

func withChangedCiphertext(encrypted EncryptedPayload) EncryptedPayload {
	encrypted.Ciphertext = append([]byte(nil), encrypted.Ciphertext...)
	encrypted.Ciphertext[0] ^= 0xff
	return encrypted
}
