package model

import "strings"

// BinaryPayload содержит приватные бинарные данные записи и связанные метаданные.
type BinaryPayload struct {
	// Filename содержит исходное имя файла.
	Filename string `json:"filename"`

	// Data содержит исходные байты файла.
	Data []byte `json:"data"`

	// Metadata содержит необязательную произвольную текстовую метаинформацию.
	Metadata string `json:"metadata,omitempty"`
}

// Validate проверяет обязательные поля и ограничения binary payload.
func (payload *BinaryPayload) Validate() error {
	if payload == nil ||
		!validateRequiredSingleLine(payload.Filename, BinaryFilenameMaxSize) ||
		strings.ContainsAny(payload.Filename, `/\\`) ||
		payload.Filename == "." || payload.Filename == ".." {
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
