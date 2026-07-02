package auth

import (
	"errors"

	"golang.org/x/crypto/bcrypt"
)

// maxBcryptPasswordLength задаёт максимальную длину пароля в байтах,
// поддерживаемую bcrypt.
const maxBcryptPasswordLength = 72

// BcryptPasswordManager хеширует и проверяет пароли с помощью bcrypt.
type BcryptPasswordManager struct{}

// NewBcryptPasswordManager создаёт менеджер паролей на базе bcrypt.
func NewBcryptPasswordManager() *BcryptPasswordManager {
	return &BcryptPasswordManager{}
}

// Hash возвращает bcrypt-хэш переданного пароля.
func (m *BcryptPasswordManager) Hash(password string) ([]byte, error) {
	if len(password) > maxBcryptPasswordLength {
		return nil, ErrPasswordTooLong
	}

	hash, err := bcrypt.GenerateFromPassword(
		[]byte(password),
		bcrypt.DefaultCost,
	)
	if err != nil {
		return nil, err
	}

	return hash, nil
}

// Check проверяет соответствие пароля переданному bcrypt-хэшу.
func (m *BcryptPasswordManager) Check(password string, hash []byte) error {
	if len(password) > maxBcryptPasswordLength {
		return ErrPasswordTooLong
	}

	err := bcrypt.CompareHashAndPassword(hash, []byte(password))
	if err == nil {
		return nil
	}

	if errors.Is(err, bcrypt.ErrMismatchedHashAndPassword) {
		return ErrPasswordMismatch
	}

	return err
}
