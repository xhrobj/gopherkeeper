package model

import "errors"

var (
	// ErrInvalidCredentialsPayload сообщает, что credentials payload некорректен.
	ErrInvalidCredentialsPayload = errors.New("invalid credentials payload")
)

// CredentialsPayload содержит приватную пару login/password и связанные данные.
type CredentialsPayload struct {
	// Login содержит имя пользователя или идентификатор учётной записи.
	Login string `json:"login"`

	// Password содержит секрет аутентификации учётной записи.
	Password string `json:"password"`

	// URL содержит необязательный адрес ресурса, к которому относятся credentials.
	URL string `json:"url,omitempty"`

	// Metadata содержит необязательную произвольную текстовую метаинформацию.
	Metadata string `json:"metadata,omitempty"`
}

// Validate проверяет обязательные поля и ограничения credentials payload.
func (payload *CredentialsPayload) Validate() error {
	if payload == nil ||
		!validateRequiredSingleLine(payload.Login, CredentialsFieldMaxSize) ||
		!validateRequiredSingleLine(payload.Password, CredentialsFieldMaxSize) ||
		!validateOptionalSingleLine(payload.URL, CredentialsFieldMaxSize) {
		return ErrInvalidCredentialsPayload
	}

	return validatePayloadMetadata(payload.Metadata, ErrInvalidCredentialsPayload)
}

// RecordType возвращает тип credentials-записи.
func (*CredentialsPayload) RecordType() RecordType {
	return RecordTypeCredentials
}

func (*CredentialsPayload) recordPayload() {
	// запрещает реализацию RecordPayload вне пакета model
}
