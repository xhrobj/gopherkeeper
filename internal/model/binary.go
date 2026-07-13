package model

import (
	"strings"
	"unicode/utf8"
)

// BinaryPayload содержит приватные бинарные данные записи и связанные метаданные.
type BinaryPayload struct {
	// Filename содержит исходное имя файла.
	Filename string `json:"filename"`

	// Data содержит исходные байты файла.
	Data []byte `json:"data"`

	// ContentType содержит необязательный пользовательский тип содержимого.
	ContentType string `json:"content_type,omitempty"`

	// Metadata содержит необязательную произвольную текстовую метаинформацию.
	Metadata string `json:"metadata,omitempty"`
}

// Validate проверяет обязательные поля и ограничения binary payload.
func (payload *BinaryPayload) Validate() error {
	if payload == nil {
		return ErrInvalidBinaryPayload
	}

	if strings.TrimSpace(payload.Filename) == "" ||
		!utf8.ValidString(payload.Filename) ||
		!utf8.ValidString(payload.ContentType) {
		return ErrInvalidBinaryPayload
	}
	if payload.Data == nil {
		return ErrInvalidBinaryPayload
	}
	if len(payload.Data) > BinaryPayloadMaxSize {
		return ErrPayloadTooLarge
	}

	return validatePayloadMetadata(payload.Metadata, ErrInvalidBinaryPayload)
}

// RecordType возвращает тип binary-записи.
func (*BinaryPayload) RecordType() RecordType {
	return RecordTypeBinary
}

func (*BinaryPayload) recordPayload() {
	// запрещает реализацию RecordPayload вне пакета model
}
