package tui

import (
	"strings"
	"unicode"
	"unicode/utf8"
)

const maxTextFieldLength = 255

func normalizeTextFieldValue(value string) string {
	if index := strings.IndexAny(value, "\r\n"); index >= 0 {
		value = value[:index]
	}

	result := make([]byte, 0, min(len(value), maxTextFieldLength))

	for index := 0; index < len(value) && len(result) < maxTextFieldLength; index++ {
		valueByte := value[index]
		if valueByte >= ' ' && valueByte <= '~' {
			result = append(result, valueByte)
		}
	}

	return string(result)
}

func normalizeUnicodeTextFieldValue(value string) string {
	result := make([]rune, 0, min(utf8.RuneCountInString(value), maxTextFieldLength))

	for _, valueRune := range value {
		if valueRune == '\r' || valueRune == '\n' {
			break
		}
		if unicode.IsPrint(valueRune) && valueRune != '\t' {
			result = append(result, valueRune)
			if len(result) == maxTextFieldLength {
				break
			}
		}
	}

	return string(result)
}

func printableTextKey(key string) (string, bool) {
	if key == "space" {
		return " ", true
	}

	if len(key) != 1 || key[0] < ' ' || key[0] > '~' {
		return "", false
	}

	return key, true
}

func printableUnicodeTextKey(key string) (string, bool) {
	if key == "space" {
		return " ", true
	}

	runes := []rune(key)

	if len(runes) != 1 || !unicode.IsPrint(runes[0]) || runes[0] == '\t' {
		return "", false
	}

	return key, true
}
