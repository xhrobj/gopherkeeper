package recordcrypto

import (
	"crypto/aes"
	"crypto/cipher"
	"crypto/rand"
	"errors"
	"fmt"
	"io"
	"strings"

	"github.com/xhrobj/gopherkeeper/internal/model"
)

const (
	// CryptoVersion содержит версию текущего формата серверного шифрования записей.
	CryptoVersion = 1

	// MasterKeySize содержит размер AES-256 master key в байтах.
	MasterKeySize = 32

	// DefaultKeyID содержит идентификатор ключа для локального MVP-окружения.
	DefaultKeyID = "primary"
)

const aadPrefix = "gopherkeeper"

var (
	// ErrInvalidMasterKey сообщает, что record master key не подходит для AES-256-GCM.
	ErrInvalidMasterKey = errors.New("invalid record master key")

	// ErrInvalidKeyID сообщает, что идентификатор record master key некорректен.
	ErrInvalidKeyID = errors.New("invalid record key id")

	// ErrInvalidAAD сообщает, что authenticated data для записи некорректны.
	ErrInvalidAAD = errors.New("invalid record AAD")

	// ErrDecryptPayload сообщает, что encrypted payload записи не удалось расшифровать.
	ErrDecryptPayload = errors.New("decrypt record payload")
)

// EncryptedPayload содержит результат AES-GCM шифрования приватного payload'а.
type EncryptedPayload struct {
	CryptoVersion int
	KeyID         string
	Nonce         []byte
	Ciphertext    []byte
}

// Service выполняет серверное шифрование и расшифрование payload'ов записей.
type Service struct {
	aead   cipher.AEAD
	keyID  string
	random io.Reader
}

// NewService создаёт Service для AES-256-GCM шифрования payload'ов записей.
func NewService(masterKey []byte, keyID string) (*Service, error) {
	return newService(masterKey, keyID, rand.Reader)
}

func newService(masterKey []byte, keyID string, random io.Reader) (*Service, error) {
	if len(masterKey) != MasterKeySize {
		return nil, ErrInvalidMasterKey
	}

	keyID = strings.TrimSpace(keyID)
	if keyID == "" {
		return nil, ErrInvalidKeyID
	}

	if random == nil {
		random = rand.Reader
	}

	block, err := aes.NewCipher(masterKey)
	if err != nil {
		return nil, fmt.Errorf("create record cipher: %w", err)
	}

	aead, err := cipher.NewGCM(block)
	if err != nil {
		return nil, fmt.Errorf("create record AEAD: %w", err)
	}

	return &Service{
		aead:   aead,
		keyID:  keyID,
		random: random,
	}, nil
}

// BuildAAD создаёт authenticated data для привязки ciphertext к владельцу, записи и типу.
func BuildAAD(userID int64, recordID string, recordType model.RecordType) ([]byte, error) {
	if userID <= 0 || model.ValidateRecordID(recordID) != nil || recordType.Validate() != nil {
		return nil, ErrInvalidAAD
	}

	return []byte(fmt.Sprintf(
		"%s:v%d:user:%d:record:%s:type:%s",
		aadPrefix,
		CryptoVersion,
		userID,
		recordID,
		recordType,
	)), nil
}

// Encrypt шифрует plaintext payload и возвращает криптографический контейнер записи.
func (s *Service) Encrypt(plaintext []byte, aad []byte) (EncryptedPayload, error) {
	nonce := make([]byte, s.aead.NonceSize())
	if _, err := io.ReadFull(s.random, nonce); err != nil {
		return EncryptedPayload{}, fmt.Errorf("generate record nonce: %w", err)
	}

	ciphertext := s.aead.Seal(nil, nonce, plaintext, aad)

	return EncryptedPayload{
		CryptoVersion: CryptoVersion,
		KeyID:         s.keyID,
		Nonce:         nonce,
		Ciphertext:    ciphertext,
	}, nil
}

// Decrypt расшифровывает encrypted payload записи и проверяет authentication tag.
func (s *Service) Decrypt(encrypted EncryptedPayload, aad []byte) ([]byte, error) {
	if encrypted.CryptoVersion != CryptoVersion ||
		encrypted.KeyID != s.keyID ||
		len(encrypted.Nonce) != s.aead.NonceSize() ||
		len(encrypted.Ciphertext) == 0 {
		return nil, ErrDecryptPayload
	}

	plaintext, err := s.aead.Open(nil, encrypted.Nonce, encrypted.Ciphertext, aad)
	if err != nil {
		return nil, ErrDecryptPayload
	}

	return plaintext, nil
}
