package service

import (
	"context"
	"fmt"

	"github.com/xhrobj/gopherkeeper/internal/model"
)

// CreateTextRecordRequest содержит входные данные для создания text-записи.
type CreateTextRecordRequest struct {
	// UserID содержит идентификатор владельца создаваемой записи.
	UserID int64

	// Title содержит открытое название создаваемой записи.
	Title string

	// Payload содержит приватные текстовые данные создаваемой записи.
	Payload model.TextPayload
}

// UpdateTextRecordRequest содержит входные данные для изменения text-записи.
type UpdateTextRecordRequest struct {
	// UserID содержит идентификатор владельца изменяемой записи.
	UserID int64

	// RecordID содержит UUID изменяемой записи.
	RecordID string

	// ExpectedRevision содержит ревизию, ожидаемую Клиентом.
	ExpectedRevision int64

	// Title содержит новое открытое название записи.
	Title string

	// Payload содержит новые приватные текстовые данные записи.
	Payload model.TextPayload
}

// TextRecord содержит открытую metadata и расшифрованный text payload.
type TextRecord struct {
	// Metadata содержит открытые поля text-записи.
	Metadata model.RecordMetadata

	// Payload содержит расшифрованные приватные текстовые данные.
	Payload model.TextPayload
}

// CreateText создаёт text-запись через единый сценарий создания записей.
func (s *RecordService) CreateText(ctx context.Context, request CreateTextRecordRequest) (TextRecord, error) {
	record, err := s.Create(ctx, CreateRecordRequest{
		UserID:  request.UserID,
		Title:   request.Title,
		Payload: &request.Payload,
	})
	if err != nil {
		return TextRecord{}, err
	}

	payload, ok := textPayloadValue(record.Payload)
	if !ok {
		return TextRecord{}, fmt.Errorf("%w: text payload", errInvalidStoredRecord)
	}

	return TextRecord{Metadata: record.Metadata, Payload: payload}, nil
}

// GetText возвращает text-запись через единый сценарий чтения записей.
func (s *RecordService) GetText(ctx context.Context, userID int64, recordID string) (TextRecord, error) {
	expectedType := model.RecordTypeText
	record, err := s.getRecord(ctx, userID, recordID, &expectedType)
	if err != nil {
		return TextRecord{}, err
	}

	payload, ok := textPayloadValue(record.Payload)
	if !ok {
		return TextRecord{}, model.ErrRecordTypeUnsupported
	}

	return TextRecord{Metadata: record.Metadata, Payload: payload}, nil
}

// UpdateText изменяет text-запись через единый сценарий изменения записей.
func (s *RecordService) UpdateText(ctx context.Context, request UpdateTextRecordRequest) (TextRecord, error) {
	record, err := s.Update(ctx, UpdateRecordRequest{
		UserID:           request.UserID,
		RecordID:         request.RecordID,
		ExpectedRevision: request.ExpectedRevision,
		Title:            request.Title,
		Payload:          &request.Payload,
	})
	if err != nil {
		return TextRecord{}, err
	}

	payload, ok := textPayloadValue(record.Payload)
	if !ok {
		return TextRecord{}, fmt.Errorf("%w: text payload", errInvalidStoredRecord)
	}

	return TextRecord{Metadata: record.Metadata, Payload: payload}, nil
}

func textPayloadValue(payload model.RecordPayload) (model.TextPayload, bool) {
	value, ok := payload.(*model.TextPayload)
	if !ok || value == nil {
		return model.TextPayload{}, false
	}

	return *value, true
}
