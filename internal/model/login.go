package model

import (
	"errors"
	"strings"
)

const (
	// MinLoginLength содержит минимальную допустимую длину login в байтах.
	MinLoginLength = 3

	// MaxLoginLength содержит максимальную допустимую длину login в байтах.
	MaxLoginLength = 32
)

// ErrInvalidLogin означает, что login не соответствует правилам GophKeeper.
var ErrInvalidLogin = errors.New("invalid login")

// CanonicalizeLogin приводит login к каноническому виду и проверяет его.
//
// Окружающие пробельные символы удаляются, а ASCII-буквы A-Z переводятся
// в нижний регистр. Другой Unicode не преобразуется и отклоняется.
func CanonicalizeLogin(login string) (string, error) {
	canonicalLogin := []byte(strings.TrimSpace(login))

	// Намеренно не используем strings.ToLower: она преобразует Unicode,
	// из-за чего недопустимый символ может превратиться в допустимый ASCII.
	for i, character := range canonicalLogin {
		if character >= 'A' && character <= 'Z' {
			canonicalLogin[i] = character + ('a' - 'A')
		}
	}

	result := string(canonicalLogin)
	if err := ValidateCanonicalLogin(result); err != nil {
		return "", err
	}

	return result, nil
}

// ValidateCanonicalLogin проверяет уже канонический lowercase login.
func ValidateCanonicalLogin(login string) error {
	if len(login) < MinLoginLength || len(login) > MaxLoginLength {
		return ErrInvalidLogin
	}

	if !isLowerASCIILetterOrDigit(login[0]) {
		return ErrInvalidLogin
	}

	for i := 1; i < len(login); i++ {
		if !isCanonicalLoginCharacter(login[i]) {
			return ErrInvalidLogin
		}
	}

	return nil
}

func isCanonicalLoginCharacter(character byte) bool {
	return isLowerASCIILetterOrDigit(character) || character == '.' || character == '_' || character == '-'
}

func isLowerASCIILetterOrDigit(character byte) bool {
	return (character >= 'a' && character <= 'z') || (character >= '0' && character <= '9')
}
