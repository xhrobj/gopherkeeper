package cli

import (
	"errors"
	"strings"
)

const (
	minLoginLength = 3
	maxLoginLength = 32
)

var errInvalidLoginArgument = errors.New(
	"invalid login: must be 3-32 characters, start with an ASCII letter or digit, " +
		"and contain only ASCII letters, digits, '.', '_' or '-'",
)

func validateLoginArgument(login string) error {
	trimmedLogin := strings.TrimSpace(login)
	if len(trimmedLogin) < minLoginLength || len(trimmedLogin) > maxLoginLength {
		return errInvalidLoginArgument
	}

	if !isASCIILetterOrDigit(trimmedLogin[0]) {
		return errInvalidLoginArgument
	}

	for i := 1; i < len(trimmedLogin); i++ {
		if !isLoginCharacter(trimmedLogin[i]) {
			return errInvalidLoginArgument
		}
	}

	return nil
}

func isLoginCharacter(character byte) bool {
	return isASCIILetterOrDigit(character) || character == '.' || character == '_' || character == '-'
}

func isASCIILetterOrDigit(character byte) bool {
	return character >= 'a' && character <= 'z' ||
		character >= 'A' && character <= 'Z' ||
		character >= '0' && character <= '9'
}
