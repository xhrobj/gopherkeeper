package service

import (
	"context"
	"fmt"

	"github.com/xhrobj/gopherkeeper/internal/model"
)

// CreateCardRecordRequest содержит входные данные для создания card-записи.
type CreateCardRecordRequest struct {
	// UserID содержит идентификатор владельца создаваемой записи.
	UserID int64

	// Title содержит открытое название создаваемой записи.
	Title string

	// Payload содержит приватные данные банковской карты.
	Payload model.CardPayload
}

// UpdateCardRecordRequest содержит входные данные для изменения card-записи.
type UpdateCardRecordRequest struct {
	// UserID содержит идентификатор владельца изменяемой записи.
	UserID int64

	// RecordID содержит UUID изменяемой записи.
	RecordID string

	// ExpectedRevision содержит ревизию, ожидаемую Клиентом.
	ExpectedRevision int64

	// Title содержит новое открытое название записи.
	Title string

	// Payload содержит новые приватные данные банковской карты.
	Payload model.CardPayload
}

// CardRecord содержит открытую metadata и расшифрованный card payload.
type CardRecord struct {
	// Metadata содержит открытые поля card-записи.
	Metadata model.RecordMetadata

	// Payload содержит расшифрованные приватные данные банковской карты.
	Payload model.CardPayload
}

// CreateCard создаёт card-запись через единый сценарий создания записей.
func (s *RecordService) CreateCard(
	ctx context.Context,
	request CreateCardRecordRequest,
) (CardRecord, error) {
	record, err := s.Create(ctx, CreateRecordRequest{
		UserID:  request.UserID,
		Title:   request.Title,
		Payload: &request.Payload,
	})
	if err != nil {
		return CardRecord{}, err
	}

	payload, ok := cardPayloadValue(record.Payload)
	if !ok {
		return CardRecord{}, fmt.Errorf("%w: card payload", errInvalidStoredRecord)
	}

	return CardRecord{Metadata: record.Metadata, Payload: payload}, nil
}

// GetCard возвращает card-запись через единый сценарий чтения записей.
func (s *RecordService) GetCard(
	ctx context.Context,
	userID int64,
	recordID string,
) (CardRecord, error) {
	expectedType := model.RecordTypeCard
	record, err := s.getRecord(ctx, userID, recordID, &expectedType)
	if err != nil {
		return CardRecord{}, err
	}

	payload, ok := cardPayloadValue(record.Payload)
	if !ok {
		return CardRecord{}, model.ErrRecordTypeUnsupported
	}

	return CardRecord{Metadata: record.Metadata, Payload: payload}, nil
}

// UpdateCard изменяет card-запись через единый сценарий изменения записей.
func (s *RecordService) UpdateCard(
	ctx context.Context,
	request UpdateCardRecordRequest,
) (CardRecord, error) {
	record, err := s.Update(ctx, UpdateRecordRequest{
		UserID:           request.UserID,
		RecordID:         request.RecordID,
		ExpectedRevision: request.ExpectedRevision,
		Title:            request.Title,
		Payload:          &request.Payload,
	})
	if err != nil {
		return CardRecord{}, err
	}

	payload, ok := cardPayloadValue(record.Payload)
	if !ok {
		return CardRecord{}, fmt.Errorf("%w: card payload", errInvalidStoredRecord)
	}

	return CardRecord{Metadata: record.Metadata, Payload: payload}, nil
}

func cardPayloadValue(payload model.RecordPayload) (model.CardPayload, bool) {
	value, ok := payload.(*model.CardPayload)
	if !ok || value == nil {
		return model.CardPayload{}, false
	}

	return *value, true
}
