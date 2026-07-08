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

func mapRecordClientError(operation string, err error) error {
	var apiError *httpclient.APIError
	if errors.As(err, &apiError) {
		switch apiError.Code {
		case "unauthorized":
			return newUserError("not logged in", errors.Join(ErrNotLoggedIn, err))
		case "record_not_found":
			return newUserError("record not found", err)
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
