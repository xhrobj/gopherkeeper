package model

import (
	"strings"
	"unicode"
	"unicode/utf8"
)

func validateRequiredSingleLine(value string, maxRunes int) bool {
	return strings.TrimSpace(value) != "" && validateOptionalSingleLine(value, maxRunes)
}

func validateOptionalSingleLine(value string, maxRunes int) bool {
	if !utf8.ValidString(value) || utf8.RuneCountInString(value) > maxRunes {
		return false
	}
	for _, symbol := range value {
		if unicode.IsControl(symbol) {
			return false
		}
	}
	return true
}

func validateRequiredMultilineBytes(value string, maxBytes int) (bool, bool) {
	if value == "" || !utf8.ValidString(value) {
		return false, false
	}
	if len(value) > maxBytes {
		return false, true
	}
	for _, symbol := range value {
		if unicode.IsControl(symbol) && symbol != '\t' && symbol != '\n' && symbol != '\r' {
			return false, false
		}
	}
	return true, false
}

func validateASCIIDigits(value string, minLength, maxLength int) bool {
	if len(value) < minLength || len(value) > maxLength {
		return false
	}
	for index := 0; index < len(value); index++ {
		if value[index] < '0' || value[index] > '9' {
			return false
		}
	}
	return true
}
