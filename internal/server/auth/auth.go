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
