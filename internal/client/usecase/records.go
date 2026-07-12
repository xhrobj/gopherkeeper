package usecase

import (
	"context"
	"errors"
	"fmt"

	"github.com/xhrobj/gopherkeeper/internal/model"
)

var errUnexpectedRecordPayload = errors.New("unexpected record payload")

type recordGateway interface {
	CreateRecord(
		ctx context.Context,
		accessToken string,
		title string,
		payload model.RecordPayload,
	) (model.Record, error)
	ListRecords(ctx context.Context, accessToken string) ([]model.RecordMetadata, error)
	GetRecord(ctx context.Context, accessToken string, recordID string) (model.Record, error)
	UpdateRecord(
		ctx context.Context,
		accessToken string,
		recordID string,
		expectedRevision int64,
		title string,
		payload model.RecordPayload,
	) (model.Record, error)
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

// CreateRecord создаёт запись выбранного типа в online-режиме.
func (a *Application) CreateRecord(ctx context.Context, request CreateRecordRequest) (model.Record, error) {
	if err := model.ValidateRecordTitle(request.Title); err != nil {
		return model.Record{}, err
	}
	if request.Payload == nil {
		return model.Record{}, errUnexpectedRecordPayload
	}
	if err := request.Payload.Validate(); err != nil {
		return model.Record{}, err
	}

	storedSession, err := a.loadSession()
	if err != nil {
		return model.Record{}, err
	}

	record, err := a.records.CreateRecord(
		ctx,
		storedSession.AccessToken,
		request.Title,
		request.Payload,
	)
	if err != nil {
		return model.Record{}, mapRecordClientError(fmt.Sprintf("create %s record", request.Payload.RecordType()), err)
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
func (a *Application) GetRecord(ctx context.Context, recordID string) (model.Record, error) {
	if err := model.ValidateRecordID(recordID); err != nil {
		return model.Record{}, err
	}

	storedSession, err := a.loadSession()
	if err != nil {
		return model.Record{}, err
	}

	record, err := a.records.GetRecord(ctx, storedSession.AccessToken, recordID)
	if err != nil {
		return model.Record{}, mapRecordClientError("get record", err)
	}

	return recordFromClient(record)
}

// UpdateRecord изменяет запись выбранного типа в online-режиме.
func (a *Application) UpdateRecord(ctx context.Context, request UpdateRecordRequest) (model.Record, error) {
	if err := model.ValidateRecordID(request.RecordID); err != nil {
		return model.Record{}, err
	}
	if err := model.ValidateRecordRevision(request.ExpectedRevision); err != nil {
		return model.Record{}, err
	}
	if err := model.ValidateRecordTitle(request.Title); err != nil {
		return model.Record{}, err
	}
	if request.Payload == nil {
		return model.Record{}, errUnexpectedRecordPayload
	}
	if err := request.Payload.Validate(); err != nil {
		return model.Record{}, err
	}

	storedSession, err := a.loadSession()
	if err != nil {
		return model.Record{}, err
	}

	record, err := a.records.UpdateRecord(
		ctx,
		storedSession.AccessToken,
		request.RecordID,
		request.ExpectedRevision,
		request.Title,
		request.Payload,
	)
	if err != nil {
		return model.Record{}, mapRecordClientError(fmt.Sprintf("update %s record", request.Payload.RecordType()), err)
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
	switch {
	case errors.Is(err, model.ErrUnauthorized):
		return newUserError("not logged in", errors.Join(ErrNotLoggedIn, err))
	case errors.Is(err, model.ErrRecordNotFound):
		return newUserError("record not found", err)
	case errors.Is(err, model.ErrRecordRevisionConflict):
		return newUserError("record revision conflict", err)
	case errors.Is(err, model.ErrRecordPreconditionRequired):
		return newUserError("record revision is required", err)
	case errors.Is(err, model.ErrPayloadTooLarge):
		return newUserError("payload is too large", err)
	case errors.Is(err, model.ErrInvalidRecordData):
		return newUserError("invalid record data", err)
	default:
		return fmt.Errorf("%s: %w", operation, err)
	}
}

func recordFromClient(record model.Record) (model.Record, error) {
	if record.Payload == nil {
		return model.Record{}, fmt.Errorf("record payload: %w", errUnexpectedRecordPayload)
	}
	if err := record.Payload.Validate(); err != nil || record.Metadata.Type != record.Payload.RecordType() {
		return model.Record{}, fmt.Errorf("record payload: %w", errUnexpectedRecordPayload)
	}

	return model.Record{
		Metadata: record.Metadata,
		Payload:  record.Payload,
	}, nil
}
