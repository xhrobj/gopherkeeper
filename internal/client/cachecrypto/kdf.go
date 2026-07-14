package cachecrypto

import (
	"crypto/rand"
	"errors"
	"fmt"

	"golang.org/x/crypto/argon2"
)

const (
	// KDFVersion содержит текущую версию профиля получения локального ключа.
	KDFVersion uint8 = 1

	argon2Time        uint32 = 3
	argon2MemoryKiB   uint32 = 64 * 1024
	argon2Parallelism uint8  = 4
	kdfSaltSize              = 16
	localKeySize             = 32
)

var (
	// ErrUnsupportedKDFVersion сообщает, что версия профиля KDF не поддерживается.
	ErrUnsupportedKDFVersion = errors.New("unsupported local cache KDF version")

	// ErrInvalidKDFSalt сообщает, что salt локального кеша имеет неверный размер.
	ErrInvalidKDFSalt = errors.New("invalid local cache KDF salt")
)

// GenerateSalt создаёт случайную salt для нового локального кеша.
func GenerateSalt() ([]byte, error) {
	salt := make([]byte, kdfSaltSize)
	if _, err := rand.Read(salt); err != nil {
		return nil, fmt.Errorf("generate local cache KDF salt: %w", err)
	}

	return salt, nil
}

// DeriveKey получает 32-байтовый локальный ключ из password и salt согласно версии профиля KDF.
func DeriveKey(password, salt []byte, version uint8) ([]byte, error) {
	if len(salt) != kdfSaltSize {
		return nil, ErrInvalidKDFSalt
	}

	switch version {
	case KDFVersion:
		return argon2.IDKey(
			password,
			salt,
			argon2Time,
			argon2MemoryKiB,
			argon2Parallelism,
			localKeySize,
		), nil
	default:
		return nil, fmt.Errorf("%w: %d", ErrUnsupportedKDFVersion, version)
	}
}
