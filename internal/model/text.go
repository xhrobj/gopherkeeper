package model

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

	valid, tooLarge := validateRequiredMultilineBytes(payload.Text, TextPayloadMaxSize)
	if tooLarge {
		return ErrPayloadTooLarge
	}
	if !valid {
		return ErrInvalidTextPayload
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
