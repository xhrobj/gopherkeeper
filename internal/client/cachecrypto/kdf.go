package cachecrypto

import (
	"crypto/rand"
	"errors"
	"fmt"

	"golang.org/x/crypto/argon2"
)

const (
	KDFVersion        uint8  = 1
	argon2Time        uint32 = 3
	argon2MemoryKiB   uint32 = 64 * 1024
	argon2Parallelism uint8  = 4
	kdfSaltSize              = 16
	localKeySize             = 32
)

var (
	ErrUnsupportedKDFVersion = errors.New("unsupported local cache KDF version")
	ErrInvalidKDFSalt        = errors.New("invalid local cache KDF salt")
)

func GenerateSalt() ([]byte, error) {
	salt := make([]byte, kdfSaltSize)
	if _, err := rand.Read(salt); err != nil {
		return nil, fmt.Errorf("generate local cache KDF salt: %w", err)
	}

	return salt, nil
}

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
