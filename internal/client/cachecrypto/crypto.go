package cachecrypto

import (
	"bytes"
	"crypto/aes"
	"crypto/cipher"
	"crypto/rand"
	"errors"
	"fmt"

	"github.com/xhrobj/gopherkeeper/internal/model"
)

const (
	CryptoVersion   uint8 = 1
	aes256KeySize         = 32
	keyCheckMessage       = "gopherkeeper-local-cache-key-check-v1"
)

var (
	ErrInvalidLocalKey = errors.New("invalid local cache key")
	ErrInvalidAAD      = errors.New("invalid local cache AAD")
	ErrOpenLocalCache  = errors.New("invalid password or corrupted local cache")
)

type EncryptedData struct {
	CryptoVersion uint8
	Nonce         []byte
	Ciphertext    []byte
}

type Service struct {
	aead cipher.AEAD
}

func NewService(key []byte) (*Service, error) {
	if len(key) != aes256KeySize {
		return nil, ErrInvalidLocalKey
	}

	block, err := aes.NewCipher(key)
	if err != nil {
		return nil, fmt.Errorf("create local cache cipher: %w", err)
	}

	aead, err := cipher.NewGCM(block)
	if err != nil {
		return nil, fmt.Errorf("create local cache AEAD: %w", err)
	}

	return &Service{aead: aead}, nil
}

func (service *Service) Encrypt(plaintext, aad []byte) (EncryptedData, error) {
	if len(aad) == 0 {
		return EncryptedData{}, ErrInvalidAAD
	}

	nonce := make([]byte, service.aead.NonceSize())
	if _, err := rand.Read(nonce); err != nil {
		return EncryptedData{}, fmt.Errorf("generate local cache nonce: %w", err)
	}

	return EncryptedData{
		CryptoVersion: CryptoVersion,
		Nonce:         nonce,
		Ciphertext:    service.aead.Seal(nil, nonce, plaintext, aad),
	}, nil
}

func (service *Service) Decrypt(encrypted EncryptedData, aad []byte) ([]byte, error) {
	if encrypted.CryptoVersion != CryptoVersion ||
		len(encrypted.Nonce) != service.aead.NonceSize() ||
		len(encrypted.Ciphertext) == 0 ||
		len(aad) == 0 {
		return nil, ErrOpenLocalCache
	}

	plaintext, err := service.aead.Open(nil, encrypted.Nonce, encrypted.Ciphertext, aad)
	if err != nil {
		return nil, ErrOpenLocalCache
	}

	return plaintext, nil
}

func BuildKeyCheckAAD(accountID string) ([]byte, error) {
	if accountID == "" {
		return nil, ErrInvalidAAD
	}

	return []byte(fmt.Sprintf(
		"gopherkeeper:local-cache:v%d:account:%s:key-check",
		CryptoVersion,
		accountID,
	)), nil
}

func BuildRecordAAD(accountID, recordID string, revision int64) ([]byte, error) {
	if accountID == "" || model.ValidateRecordID(recordID) != nil || revision <= 0 {
		return nil, ErrInvalidAAD
	}

	return []byte(fmt.Sprintf(
		"gopherkeeper:local-cache:v%d:account:%s:record:%s:revision:%d",
		CryptoVersion,
		accountID,
		recordID,
		revision,
	)), nil
}

func (service *Service) CreateKeyCheck(accountID string) (EncryptedData, error) {
	aad, err := BuildKeyCheckAAD(accountID)
	if err != nil {
		return EncryptedData{}, err
	}

	return service.Encrypt([]byte(keyCheckMessage), aad)
}

func (service *Service) VerifyKeyCheck(accountID string, encrypted EncryptedData) error {
	aad, err := BuildKeyCheckAAD(accountID)
	if err != nil {
		return err
	}

	plaintext, err := service.Decrypt(encrypted, aad)
	if err != nil || !bytes.Equal(plaintext, []byte(keyCheckMessage)) {
		return ErrOpenLocalCache
	}

	return nil
}
