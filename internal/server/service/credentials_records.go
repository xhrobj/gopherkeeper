package service

import (
	"context"

	"github.com/xhrobj/gopherkeeper/internal/model"
)

// CreateCredentialsRecordRequest содержит входные данные для создания credentials-записи.
type CreateCredentialsRecordRequest struct {
	UserID  int64
	Title   string
	Payload model.CredentialsPayload
}

// UpdateCredentialsRecordRequest содержит входные данные для изменения credentials-записи.
type UpdateCredentialsRecordRequest struct {
	UserID           int64
	RecordID         string
	ExpectedRevision int64
	Title            string
	Payload          model.CredentialsPayload
}

// CredentialsRecord содержит открытую metadata и расшифрованный credentials payload.
type CredentialsRecord struct {
	Metadata model.RecordMetadata
	Payload  model.CredentialsPayload
}

// CreateCredentials создаёт credentials-запись, шифрует payload и сохраняет encrypted record.
func (s *RecordService) CreateCredentials(
	ctx context.Context,
	request CreateCredentialsRecordRequest,
) (CredentialsRecord, error) {
	record, err := s.createRecord(
		ctx,
		request.UserID,
		request.Title,
		model.RecordTypeCredentials,
		request.Payload,
	)
	if err != nil {
		return CredentialsRecord{}, err
	}

	return CredentialsRecord{
		Metadata: record.Metadata(),
		Payload:  request.Payload,
	}, nil
}

// GetCredentials возвращает credentials-запись пользователя и расшифровывает её payload.
func (s *RecordService) GetCredentials(
	ctx context.Context,
	userID int64,
	recordID string,
) (CredentialsRecord, error) {
	var payload model.CredentialsPayload

	record, err := s.getRecord(ctx, userID, recordID, model.RecordTypeCredentials, &payload)
	if err != nil {
		return CredentialsRecord{}, err
	}

	return CredentialsRecord{
		Metadata: record.Metadata(),
		Payload:  payload,
	}, nil
}

// UpdateCredentials изменяет credentials-запись при совпадении ожидаемой ревизии.
func (s *RecordService) UpdateCredentials(
	ctx context.Context,
	request UpdateCredentialsRecordRequest,
) (CredentialsRecord, error) {
	updated, err := s.updateRecord(
		ctx,
		request.UserID,
		request.RecordID,
		request.ExpectedRevision,
		request.Title,
		model.RecordTypeCredentials,
		request.Payload,
	)
	if err != nil {
		return CredentialsRecord{}, err
	}

	return CredentialsRecord{
		Metadata: updated.Metadata(),
		Payload:  request.Payload,
	}, nil
}
