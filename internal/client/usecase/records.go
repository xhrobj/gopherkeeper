package usecase

import (
	"context"
	"errors"
	"fmt"

	"github.com/xhrobj/gopherkeeper/internal/client/httpclient"
	"github.com/xhrobj/gopherkeeper/internal/model"
)

var errUnexpectedRecordPayload = errors.New("unexpected record payload")

type recordClient interface {
	CreateRecord(ctx context.Context, accessToken string, request httpclient.CreateRecordRequest) (httpclient.Record, error)
	ListRecords(ctx context.Context, accessToken string) ([]model.RecordMetadata, error)
	GetRecord(ctx context.Context, accessToken string, recordID string) (httpclient.Record, error)
	UpdateRecord(
		ctx context.Context,
		accessToken string,
		recordID string,
		expectedRevision int64,
		request httpclient.UpdateRecordRequest,
	) (httpclient.Record, error)
	DeleteRecord(ctx context.Context, accessToken string, recordID string, expectedRevision int64) error
}

// CreateRecordRequest содержит входные данные клиентского сценария создания записи.
type CreateRecordRequest struct {
	// Title содержит открытое название записи.
	Title string

	// Payload содержит типизированный приватный payload.
	Payload model.RecordPayload
}

// UpdateRecordRequest содержит входные данные клиентского сценария изменения записи.
type UpdateRecordRequest struct {
	// RecordID содержит идентификатор изменяемой записи.
	RecordID string

	// ExpectedRevision содержит ревизию, которую пользователь ожидает изменить.
	ExpectedRevision int64

	// Title содержит новое открытое название записи.
	Title string

	// Payload содержит новый типизированный приватный payload.
	Payload model.RecordPayload
}

// DeleteRecordRequest содержит входные данные клиентского сценария удаления записи.
type DeleteRecordRequest struct {
	// RecordID содержит идентификатор удаляемой записи.
	RecordID string

	// ExpectedRevision содержит ревизию, которую пользователь ожидает удалить.
	ExpectedRevision int64
}

// Record содержит открытую metadata и расшифрованный типизированный payload.
type Record struct {
	// Metadata содержит открытые поля записи.
	Metadata model.RecordMetadata

	// Payload содержит приватный payload согласно типу записи.
	Payload model.RecordPayload
}

// CreateRecord создаёт запись выбранного типа в online-режиме.
func (a *Application) CreateRecord(ctx context.Context, request CreateRecordRequest) (Record, error) {
	if err := model.ValidateRecordTitle(request.Title); err != nil {
		return Record{}, err
	}
	if request.Payload == nil {
		return Record{}, errUnexpectedRecordPayload
	}
	if err := request.Payload.Validate(); err != nil {
		return Record{}, err
	}

	storedSession, err := a.loadSession()
	if err != nil {
		return Record{}, err
	}

	record, err := a.records.CreateRecord(ctx, storedSession.AccessToken, httpclient.CreateRecordRequest{
		Title:   request.Title,
		Payload: request.Payload,
	})
	if err != nil {
		return Record{}, mapRecordClientError(fmt.Sprintf("create %s record", request.Payload.RecordType()), err)
	}

	return recordFromClient(record)
}

// ListRecords возвращает metadata приватных записей текущего пользователя в online-режиме.
func (a *Application) ListRecords(ctx context.Context) ([]model.RecordMetadata, error) {

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

// GetRecord возвращает запись текущего пользователя с payload согласно её типу.
func (a *Application) GetRecord(ctx context.Context, recordID string) (Record, error) {
	if err := model.ValidateRecordID(recordID); err != nil {
		return Record{}, err
	}

	storedSession, err := a.loadSession()
	if err != nil {
		return Record{}, err
	}

	record, err := a.records.GetRecord(ctx, storedSession.AccessToken, recordID)
	if err != nil {
		return Record{}, mapRecordClientError("get record", err)
	}

	return recordFromClient(record)
}

// UpdateRecord изменяет запись выбранного типа в online-режиме.
func (a *Application) UpdateRecord(ctx context.Context, request UpdateRecordRequest) (Record, error) {
	if err := model.ValidateRecordID(request.RecordID); err != nil {
		return Record{}, err
	}
	if err := model.ValidateRecordRevision(request.ExpectedRevision); err != nil {
		return Record{}, err
	}
	if err := model.ValidateRecordTitle(request.Title); err != nil {
		return Record{}, err
	}
	if request.Payload == nil {
		return Record{}, errUnexpectedRecordPayload
	}
	if err := request.Payload.Validate(); err != nil {
		return Record{}, err
	}

	storedSession, err := a.loadSession()
	if err != nil {
		return Record{}, err
	}

	record, err := a.records.UpdateRecord(
		ctx,
		storedSession.AccessToken,
		request.RecordID,
		request.ExpectedRevision,
		httpclient.UpdateRecordRequest{
			Title:   request.Title,
			Payload: request.Payload,
		},
	)
	if err != nil {
		return Record{}, mapRecordClientError(fmt.Sprintf("update %s record", request.Payload.RecordType()), err)
	}

	return recordFromClient(record)
}

// DeleteRecord удаляет запись в online-режиме.
func (a *Application) DeleteRecord(ctx context.Context, request DeleteRecordRequest) error {
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

func recordFromClient(record httpclient.Record) (Record, error) {
	if record.Payload == nil || record.Metadata.Type != record.Payload.RecordType() {
		return Record{}, fmt.Errorf("record payload: %w", errUnexpectedRecordPayload)
	}

	return Record{
		Metadata: record.Metadata,
		Payload:  record.Payload,
	}, nil
}
