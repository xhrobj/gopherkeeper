package service

import (
	"errors"

	"github.com/xhrobj/gopherkeeper/internal/model"
)

const (
	// NOTE: с большей минимальной длиной будет неудобно тестировать,
	// а тестирование пока самый вероятный сценарий использования этого продукта (^_-)
	minPasswordLength = 3

	// NOTE: пароль ограничен печатными ASCII-символами без пробела.
	// Это делает длину в символах равной длине в байтах и исключает необходимость
	// Unicode-нормализации
	maxPasswordLength = 64
)

var (
	// ErrInvalidLogin означает, что логин не соответствует правилам GophKeeper.
	ErrInvalidLogin = model.ErrInvalidLogin

	// ErrInvalidPassword означает, что пароль содержит недопустимые символы.
	ErrInvalidPassword = errors.New("invalid password")

	// ErrPasswordTooShort означает, что пароль короче минимально допустимой длины.
	ErrPasswordTooShort = errors.New("password too short")

	// ErrPasswordTooLong означает, что пароль превышает максимально допустимую длину.
	ErrPasswordTooLong = errors.New("password too long")
)

func validateCredentials(login, password string) (string, error) {
	canonicalLogin, err := model.CanonicalizeLogin(login)
	if err != nil {
		return "", ErrInvalidLogin
	}

	if err := validatePassword(password); err != nil {
		return "", err
	}

	return canonicalLogin, nil
}

func validatePassword(password string) error {
	if len(password) < minPasswordLength {
		return ErrPasswordTooShort
	}

	if len(password) > maxPasswordLength {
		return ErrPasswordTooLong
	}

	for i := 0; i < len(password); i++ {
		if !isPasswordCharacter(password[i]) {
			return ErrInvalidPassword
		}
	}

	return nil
}

func isPasswordCharacter(character byte) bool {
	// Печатный ASCII без пробела: от '!' до '~'.
	return character >= '!' && character <= '~'
}
