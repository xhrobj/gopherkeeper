package model

import (
	"errors"
	"strings"
	"unicode/utf8"
)

var (
	ErrInvalidCredentialsPayload = errors.New("invalid credentials payload")
)

type CredentialsPayload struct {
	Login    string `json:"login"`
	Password string `json:"password"`
	URL      string `json:"url,omitempty"`
	Metadata string `json:"metadata,omitempty"`
}

func (payload CredentialsPayload) Validate() error {
	if !utf8.ValidString(payload.Login) ||
		!utf8.ValidString(payload.Password) ||
		!utf8.ValidString(payload.URL) ||
		!utf8.ValidString(payload.Metadata) {
		return ErrInvalidCredentialsPayload
	}

	if strings.TrimSpace(payload.Login) == "" || strings.TrimSpace(payload.Password) == "" {
		return ErrInvalidCredentialsPayload
	}

	if len(payload.Metadata) > MetadataMaxSize {
		return ErrPayloadTooLarge
	}

	return nil
}
