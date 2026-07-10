package service

import (
	"context"
	"fmt"

	"github.com/xhrobj/gopherkeeper/internal/model"
)

// CreateCredentialsRecordRequest содержит входные данные для создания credentials-записи.
type CreateCredentialsRecordRequest struct {
	// UserID содержит идентификатор владельца создаваемой записи.
	UserID int64

	// Title содержит открытое название создаваемой записи.
	Title string

	// Payload содержит приватные credentials-данные создаваемой записи.
	Payload model.CredentialsPayload
}

// UpdateCredentialsRecordRequest содержит входные данные для изменения credentials-записи.
type UpdateCredentialsRecordRequest struct {
	// UserID содержит идентификатор владельца изменяемой записи.
	UserID int64

	// RecordID содержит UUID изменяемой записи.
	RecordID string

	// ExpectedRevision содержит ревизию, ожидаемую Клиентом.
	ExpectedRevision int64

	// Title содержит новое открытое название записи.
	Title string

	// Payload содержит новые приватные credentials-данные записи.
	Payload model.CredentialsPayload
}

// CredentialsRecord содержит открытую metadata и расшифрованный credentials payload.
type CredentialsRecord struct {
	// Metadata содержит открытые поля credentials-записи.
	Metadata model.RecordMetadata

	// Payload содержит расшифрованные приватные credentials-данные.
	Payload model.CredentialsPayload
}

// CreateCredentials создаёт credentials-запись через единый сценарий создания записей.
func (s *RecordService) CreateCredentials(
	ctx context.Context,
	request CreateCredentialsRecordRequest,
) (CredentialsRecord, error) {
	record, err := s.Create(ctx, CreateRecordRequest{
		UserID:  request.UserID,
		Title:   request.Title,
		Payload: &request.Payload,
	})
	if err != nil {
		return CredentialsRecord{}, err
	}

	payload, ok := credentialsPayloadValue(record.Payload)
	if !ok {
		return CredentialsRecord{}, fmt.Errorf("%w: credentials payload", errInvalidStoredRecord)
	}

	return CredentialsRecord{Metadata: record.Metadata, Payload: payload}, nil
}

// GetCredentials возвращает credentials-запись через единый сценарий чтения записей.
func (s *RecordService) GetCredentials(
	ctx context.Context,
	userID int64,
	recordID string,
) (CredentialsRecord, error) {
	expectedType := model.RecordTypeCredentials
	record, err := s.getRecord(ctx, userID, recordID, &expectedType)
	if err != nil {
		return CredentialsRecord{}, err
	}

	payload, ok := credentialsPayloadValue(record.Payload)
	if !ok {
		return CredentialsRecord{}, model.ErrRecordTypeUnsupported
	}

	return CredentialsRecord{Metadata: record.Metadata, Payload: payload}, nil
}

// UpdateCredentials изменяет credentials-запись через единый сценарий изменения записей.
func (s *RecordService) UpdateCredentials(
	ctx context.Context,
	request UpdateCredentialsRecordRequest,
) (CredentialsRecord, error) {
	record, err := s.Update(ctx, UpdateRecordRequest{
		UserID:           request.UserID,
		RecordID:         request.RecordID,
		ExpectedRevision: request.ExpectedRevision,
		Title:            request.Title,
		Payload:          &request.Payload,
	})
	if err != nil {
		return CredentialsRecord{}, err
	}

	payload, ok := credentialsPayloadValue(record.Payload)
	if !ok {
		return CredentialsRecord{}, fmt.Errorf("%w: credentials payload", errInvalidStoredRecord)
	}

	return CredentialsRecord{Metadata: record.Metadata, Payload: payload}, nil
}

func credentialsPayloadValue(payload model.RecordPayload) (model.CredentialsPayload, bool) {
	value, ok := payload.(*model.CredentialsPayload)
	if !ok || value == nil {
		return model.CredentialsPayload{}, false
	}

	return *value, true
}
