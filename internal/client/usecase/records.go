package usecase

import (
	"context"
	"errors"
	"fmt"

	"github.com/xhrobj/gopherkeeper/internal/client/httpclient"
	"github.com/xhrobj/gopherkeeper/internal/model"
)

type recordClient interface {
	CreateTextRecord(ctx context.Context, accessToken string, request httpclient.CreateTextRecordRequest) (httpclient.TextRecord, error)
	ListRecords(ctx context.Context, accessToken string) ([]model.RecordMetadata, error)
	GetTextRecord(ctx context.Context, accessToken string, recordID string) (httpclient.TextRecord, error)
	UpdateTextRecord(
		ctx context.Context,
		accessToken string,
		recordID string,
		expectedRevision int64,
		request httpclient.UpdateTextRecordRequest,
	) (httpclient.TextRecord, error)
	DeleteRecord(ctx context.Context, accessToken string, recordID string, expectedRevision int64) error
}

// CreateTextRecordRequest содержит входные данные клиентского сценария создания text-записи.
type CreateTextRecordRequest struct {
	// Title содержит открытое название записи.
	Title string

	// Text содержит приватный текст записи.
	Text string

	// Metadata содержит необязательную приватную метаинформацию записи.
	Metadata string
}

// UpdateTextRecordRequest содержит входные данные клиентского сценария изменения text-записи.
type UpdateTextRecordRequest struct {
	// RecordID содержит идентификатор изменяемой записи.
	RecordID string

	// ExpectedRevision содержит ревизию, которую пользователь ожидает изменить.
	ExpectedRevision int64

	// Title содержит новое открытое название записи.
	Title string

	// Text содержит новый приватный текст записи.
	Text string

	// Metadata содержит новую необязательную приватную метаинформацию записи.
	Metadata string
}

// DeleteRecordRequest содержит входные данные клиентского сценария удаления записи.
type DeleteRecordRequest struct {
	// RecordID содержит идентификатор удаляемой записи.
	RecordID string

	// ExpectedRevision содержит ревизию, которую пользователь ожидает удалить.
	ExpectedRevision int64
}

// TextRecord содержит открытую metadata и расшифрованный text payload.
type TextRecord struct {
	// Metadata содержит открытые поля записи.
	Metadata model.RecordMetadata

	// Payload содержит приватный text payload.
	Payload model.TextPayload
}

// CreateTextRecord создаёт text-запись в online-режиме.
func (a *Application) CreateTextRecord(ctx context.Context, request CreateTextRecordRequest) (TextRecord, error) {
	if a.records == nil {
		return TextRecord{}, errors.New("record client is not configured")
	}
	if err := model.ValidateRecordTitle(request.Title); err != nil {
		return TextRecord{}, err
	}

	payload := model.TextPayload{
		Text:     request.Text,
		Metadata: request.Metadata,
	}
	if err := payload.Validate(); err != nil {
		return TextRecord{}, err
	}

	storedSession, err := a.loadSession()
	if err != nil {
		return TextRecord{}, err
	}

	record, err := a.records.CreateTextRecord(ctx, storedSession.AccessToken, httpclient.CreateTextRecordRequest{
		Title:   request.Title,
		Payload: payload,
	})
	if err != nil {
		return TextRecord{}, mapRecordClientError("create text record", err)
	}

	return textRecordFromClient(record), nil
}

// ListRecords возвращает metadata приватных записей текущего пользователя в online-режиме.
func (a *Application) ListRecords(ctx context.Context) ([]model.RecordMetadata, error) {
	if a.records == nil {
		return nil, errors.New("record client is not configured")
	}

	storedSession, err := a.loadSession()
	if err != nil {
		return nil, err
	}

	records, err := a.records.ListRecords(ctx, storedSession.AccessToken)
	if err != nil {
		return nil, mapRecordClientError("list records", err)
	}

	return records, nil
}

// GetTextRecord возвращает text-запись текущего пользователя в online-режиме.
func (a *Application) GetTextRecord(ctx context.Context, recordID string) (TextRecord, error) {
	if a.records == nil {
		return TextRecord{}, errors.New("record client is not configured")
	}
	if err := model.ValidateRecordID(recordID); err != nil {
		return TextRecord{}, err
	}

	storedSession, err := a.loadSession()
	if err != nil {
		return TextRecord{}, err
	}

	record, err := a.records.GetTextRecord(ctx, storedSession.AccessToken, recordID)
	if err != nil {
		return TextRecord{}, mapRecordClientError("get text record", err)
	}

	return textRecordFromClient(record), nil
}

// UpdateTextRecord изменяет text-запись в online-режиме.
func (a *Application) UpdateTextRecord(ctx context.Context, request UpdateTextRecordRequest) (TextRecord, error) {
	if a.records == nil {
		return TextRecord{}, errors.New("record client is not configured")
	}
	if err := model.ValidateRecordID(request.RecordID); err != nil {
		return TextRecord{}, err
	}
	if err := model.ValidateRecordRevision(request.ExpectedRevision); err != nil {
		return TextRecord{}, err
	}
	if err := model.ValidateRecordTitle(request.Title); err != nil {
		return TextRecord{}, err
	}

	payload := model.TextPayload{
		Text:     request.Text,
		Metadata: request.Metadata,
	}
	if err := payload.Validate(); err != nil {
		return TextRecord{}, err
	}

	storedSession, err := a.loadSession()
	if err != nil {
		return TextRecord{}, err
	}

	record, err := a.records.UpdateTextRecord(
		ctx,
		storedSession.AccessToken,
		request.RecordID,
		request.ExpectedRevision,
		httpclient.UpdateTextRecordRequest{
			Title:   request.Title,
			Payload: payload,
		},
	)
	if err != nil {
		return TextRecord{}, mapRecordClientError("update text record", err)
	}

	return textRecordFromClient(record), nil
}

// DeleteRecord удаляет запись в online-режиме.
func (a *Application) DeleteRecord(ctx context.Context, request DeleteRecordRequest) error {
	if a.records == nil {
		return errors.New("record client is not configured")
	}
	if err := model.ValidateRecordID(request.RecordID); err != nil {
		return err
	}
	if err := model.ValidateRecordRevision(request.ExpectedRevision); err != nil {
		return err
	}

	storedSession, err := a.loadSession()
	if err != nil {
		return err
	}

	if err := a.records.DeleteRecord(ctx, storedSession.AccessToken, request.RecordID, request.ExpectedRevision); err != nil {
		return mapRecordClientError("delete record", err)
	}

	return nil
}

func mapRecordClientError(operation string, err error) error {
	var apiError *httpclient.APIError
	if errors.As(err, &apiError) {
		switch apiError.Code {
		case "unauthorized":
			return newUserError("not logged in", errors.Join(ErrNotLoggedIn, err))
		case "record_not_found":
			return newUserError("record not found", err)
		case "record_revision_conflict":
			return newUserError("record revision conflict", err)
		case "precondition_required":
			return newUserError("record revision is required", err)
		case "payload_too_large":
			return newUserError("payload is too large", err)
		case "invalid_request":
			return newUserError("invalid record data", err)
		}
	}

	return fmt.Errorf("%s: %w", operation, err)
}

func textRecordFromClient(record httpclient.TextRecord) TextRecord {
	return TextRecord{
		Metadata: record.Metadata,
		Payload:  record.Payload,
	}
}
