package cachecrypto

import (
	"bytes"
	"encoding/json"
	"errors"
	"fmt"
	"io"
	"time"

	"github.com/xhrobj/gopherkeeper/internal/model"
)

const (
	// RecordFormatVersion содержит текущую версию plaintext-формата локальной записи.
	RecordFormatVersion uint8 = 1
)

var (
	// ErrUnsupportedRecordFormatVersion сообщает, что версия plaintext-формата записи не поддерживается.
	ErrUnsupportedRecordFormatVersion = errors.New("unsupported local cache record format version")

	// ErrInvalidRecordFormat сообщает, что расшифрованная локальная запись имеет некорректный формат.
	ErrInvalidRecordFormat = errors.New("invalid local cache record format")
)

type recordEnvelope struct {
	FormatVersion uint8            `json:"format_version"`
	ID            string           `json:"id"`
	Type          model.RecordType `json:"type"`
	Title         string           `json:"title"`
	Revision      int64            `json:"revision"`
	CreatedAt     time.Time        `json:"created_at"`
	UpdatedAt     time.Time        `json:"updated_at"`
	Payload       json.RawMessage  `json:"payload"`
}

// EncodeRecord сериализует полную приватную запись перед локальным шифрованием.
func EncodeRecord(record model.Record) ([]byte, error) {
	if err := record.Validate(); err != nil {
		return nil, fmt.Errorf("%w: %v", ErrInvalidRecordFormat, err)
	}

	payload, err := json.Marshal(record.Payload)
	if err != nil {
		return nil, fmt.Errorf("%w: encode payload", ErrInvalidRecordFormat)
	}

	envelope := recordEnvelope{
		FormatVersion: RecordFormatVersion,
		ID:            record.Metadata.ID,
		Type:          record.Metadata.Type,
		Title:         record.Metadata.Title,
		Revision:      record.Metadata.Revision,
		CreatedAt:     record.Metadata.CreatedAt,
		UpdatedAt:     record.Metadata.UpdatedAt,
		Payload:       payload,
	}

	encoded, err := json.Marshal(envelope)
	if err != nil {
		return nil, fmt.Errorf("%w: encode record", ErrInvalidRecordFormat)
	}

	return encoded, nil
}

// DecodeRecord восстанавливает и валидирует полную приватную запись после локального расшифрования.
func DecodeRecord(encoded []byte) (model.Record, error) {
	var envelope recordEnvelope
	if err := decodeStrictJSON(encoded, &envelope); err != nil {
		return model.Record{}, fmt.Errorf("%w: decode record", ErrInvalidRecordFormat)
	}
	if envelope.FormatVersion != RecordFormatVersion {
		return model.Record{}, fmt.Errorf(
			"%w: %d",
			ErrUnsupportedRecordFormatVersion,
			envelope.FormatVersion,
		)
	}

	payload, err := model.NewRecordPayload(envelope.Type)
	if err != nil {
		return model.Record{}, fmt.Errorf("%w: record type", ErrInvalidRecordFormat)
	}
	if err := decodeStrictJSON(envelope.Payload, payload); err != nil {
		return model.Record{}, fmt.Errorf("%w: decode payload", ErrInvalidRecordFormat)
	}

	record := model.Record{
		Metadata: model.RecordMetadata{
			ID:        envelope.ID,
			Type:      envelope.Type,
			Title:     envelope.Title,
			Revision:  envelope.Revision,
			CreatedAt: envelope.CreatedAt,
			UpdatedAt: envelope.UpdatedAt,
		},
		Payload: payload,
	}
	if err := record.Validate(); err != nil {
		return model.Record{}, fmt.Errorf("%w: %v", ErrInvalidRecordFormat, err)
	}

	return record, nil
}

func decodeStrictJSON(data []byte, destination any) error {
	decoder := json.NewDecoder(bytes.NewReader(data))
	decoder.DisallowUnknownFields()

	if err := decoder.Decode(destination); err != nil {
		return err
	}
	if err := decoder.Decode(&struct{}{}); !errors.Is(err, io.EOF) {
		return errors.New("unexpected trailing JSON value")
	}

	return nil
}
