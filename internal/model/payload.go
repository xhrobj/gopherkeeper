package model

import "unicode/utf8"

// RecordPayload описывает типизированный приватный payload записи.
//
// Реализации интерфейса определены только в пакете model, чтобы тип записи
// всегда соответствовал конкретной структуре payload.
type RecordPayload interface {
	// Validate проверяет доменные ограничения payload.
	Validate() error

	// RecordType возвращает тип записи, которому соответствует payload.
	RecordType() RecordType

	recordPayload()
}

// NewRecordPayload создаёт пустой payload для поддерживаемого типа записи.
func NewRecordPayload(recordType RecordType) (RecordPayload, error) {
	switch recordType {
	case RecordTypeText:
		return &TextPayload{}, nil
	case RecordTypeCredentials:
		return &CredentialsPayload{}, nil
	case RecordTypeCard:
		return &CardPayload{}, nil
	case RecordTypeBinary:
		return &BinaryPayload{}, nil
	default:
		return nil, ErrRecordTypeUnsupported
	}
}

func validatePayloadMetadata(metadata string, invalidPayloadError error) error {
	if !utf8.ValidString(metadata) {
		return invalidPayloadError
	}
	if len(metadata) > MetadataMaxSize {
		return ErrPayloadTooLarge
	}

	return nil
}
