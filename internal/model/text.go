package model

import "unicode/utf8"

// TextPayload содержит приватный текстовый payload записи.
type TextPayload struct {
	// Text содержит приватные текстовые данные записи.
	Text string `json:"text"`

	// Metadata содержит необязательную произвольную текстовую метаинформацию.
	Metadata string `json:"metadata,omitempty"`
}

// Validate проверяет обязательный текст и ограничения размера text payload.
func (payload *TextPayload) Validate() error {
	if payload == nil {
		return ErrInvalidTextPayload
	}

	if payload.Text == "" || !utf8.ValidString(payload.Text) {
		return ErrInvalidTextPayload
	}

	if len(payload.Text) > TextPayloadMaxSize {
		return ErrPayloadTooLarge
	}

	return validatePayloadMetadata(payload.Metadata, ErrInvalidTextPayload)
}

// RecordType возвращает тип text-записи.
func (*TextPayload) RecordType() RecordType {
	return RecordTypeText
}

func (*TextPayload) recordPayload() {
	// запрещает реализацию RecordPayload вне пакета model
}
