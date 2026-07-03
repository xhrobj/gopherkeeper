package auth

import "errors"

var (
	// ErrPasswordTooLong означает, что пароль превышает допустимую длину
	// для алгоритма хеширования.
	ErrPasswordTooLong = errors.New("password too long")

	// ErrPasswordMismatch означает, что пароль не соответствует
	// сохранённому хэшу.
	ErrPasswordMismatch = errors.New("password mismatch")
)

// PasswordManager описывает операции хеширования и проверки пароля.
type PasswordManager interface {
	// Hash возвращает хэш переданного пароля.
	Hash(password string) ([]byte, error)

	// Check проверяет соответствие пароля переданному хэшу.
	Check(password string, hash []byte) error
}
