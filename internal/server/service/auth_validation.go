package service

import (
	"errors"
	"strings"
)

const (
	minLoginLength = 3
	maxLoginLength = 32

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
	ErrInvalidLogin = errors.New("invalid login")

	// ErrInvalidPassword означает, что пароль содержит недопустимые символы.
	ErrInvalidPassword = errors.New("invalid password")

	// ErrPasswordTooShort означает, что пароль короче минимально допустимой длины.
	ErrPasswordTooShort = errors.New("password too short")

	// ErrPasswordTooLong означает, что пароль превышает максимально допустимую длину.
	ErrPasswordTooLong = errors.New("password too long")
)

func validateCredentials(login, password string) (string, error) {
	canonicalLogin := canonicalizeLogin(login)
	if err := validateLogin(canonicalLogin); err != nil {
		return "", err
	}

	if err := validatePassword(password); err != nil {
		return "", err
	}

	return canonicalLogin, nil
}

func canonicalizeLogin(login string) string {
	canonicalLogin := []byte(strings.TrimSpace(login))

	// NOTE: намеренно не используем strings.ToLower: она преобразует Unicode,
	// из-за чего недопустимый символ может превратиться в допустимый ASCII.
	// Переводим в нижний регистр только ASCII A–Z, а остальной Unicode
	// оставляем для последующего отклонения в validateLogin.
	for i, character := range canonicalLogin {
		if character >= 'A' && character <= 'Z' {
			canonicalLogin[i] = character + ('a' - 'A')
		}
	}

	return string(canonicalLogin)
}

func validateLogin(login string) error {
	if len(login) < minLoginLength || len(login) > maxLoginLength {
		return ErrInvalidLogin
	}

	if !isASCIILetterOrDigit(login[0]) {
		return ErrInvalidLogin
	}

	for i := 1; i < len(login); i++ {
		if !isLoginCharacter(login[i]) {
			return ErrInvalidLogin
		}
	}

	return nil
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

func isLoginCharacter(character byte) bool {
	return isASCIILetterOrDigit(character) || character == '.' || character == '_' || character == '-'
}

func isASCIILetterOrDigit(character byte) bool {
	return (character >= 'a' && character <= 'z') ||
		(character >= 'A' && character <= 'Z') ||
		(character >= '0' && character <= '9')
}

func isPasswordCharacter(character byte) bool {
	// Печатный ASCII без пробела: от '!' до '~'.
	return character >= '!' && character <= '~'
}
