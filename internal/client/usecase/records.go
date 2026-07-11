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

// CreateTextRecordRequest содержит входные данные клиентского сценария создания text-записи.
type CreateTextRecordRequest struct {
	// Title содержит открытое название записи.
	Title string

	// Text содержит приватный текст записи.
	Text string

	// Metadata содержит необязательную приватную метаинформацию записи.
	Metadata string
}

// CreateCredentialsRecordRequest содержит входные данные клиентского сценария создания credentials-записи.
type CreateCredentialsRecordRequest struct {
	// Title содержит открытое название записи.
	Title string

	// Login содержит приватный login учётной записи.
	Login string

	// Password содержит приватный password учётной записи.
	Password string

	// URL содержит необязательный приватный адрес ресурса.
	URL string

	// Metadata содержит необязательную приватную метаинформацию записи.
	Metadata string
}

// CreateCardRecordRequest содержит входные данные клиентского сценария создания card-записи.
type CreateCardRecordRequest struct {
	// Title содержит открытое название записи.
	Title string

	// Number содержит приватный номер карты.
	Number string

	// Cardholder содержит необязательное приватное имя владельца карты.
	Cardholder string

	// ExpiryMonth содержит необязательный приватный месяц окончания срока действия.
	ExpiryMonth *int

	// ExpiryYear содержит необязательный приватный год окончания срока действия.
	ExpiryYear *int

	// CVV содержит необязательный приватный код безопасности карты.
	CVV string

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

// UpdateCredentialsRecordRequest содержит входные данные клиентского сценария изменения credentials-записи.
type UpdateCredentialsRecordRequest struct {
	// RecordID содержит идентификатор изменяемой записи.
	RecordID string

	// ExpectedRevision содержит ревизию, которую пользователь ожидает изменить.
	ExpectedRevision int64

	// Title содержит новое открытое название записи.
	Title string

	// Login содержит новый приватный login учётной записи.
	Login string

	// Password содержит новый приватный password учётной записи.
	Password string

	// URL содержит новый необязательный приватный адрес ресурса.
	URL string

	// Metadata содержит новую необязательную приватную метаинформацию записи.
	Metadata string
}

// UpdateCardRecordRequest содержит входные данные клиентского сценария изменения card-записи.
type UpdateCardRecordRequest struct {
	// RecordID содержит идентификатор изменяемой записи.
	RecordID string

	// ExpectedRevision содержит ревизию, которую пользователь ожидает изменить.
	ExpectedRevision int64

	// Title содержит новое открытое название записи.
	Title string

	// Number содержит новый приватный номер карты.
	Number string

	// Cardholder содержит новое необязательное приватное имя владельца карты.
	Cardholder string

	// ExpiryMonth содержит новый необязательный приватный месяц окончания срока действия.
	ExpiryMonth *int

	// ExpiryYear содержит новый необязательный приватный год окончания срока действия.
	ExpiryYear *int

	// CVV содержит новый необязательный приватный код безопасности карты.
	CVV string

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

// Record содержит открытую metadata и расшифрованный типизированный payload.
type Record struct {
	// Metadata содержит открытые поля записи.
	Metadata model.RecordMetadata

	// Payload содержит приватный payload согласно типу записи.
	Payload model.RecordPayload
}

// TextRecord содержит открытую metadata и расшифрованный text payload.
type TextRecord struct {
	// Metadata содержит открытые поля записи.
	Metadata model.RecordMetadata

	// Payload содержит приватный text payload.
	Payload model.TextPayload
}

// CredentialsRecord содержит открытую metadata и расшифрованный credentials payload.
type CredentialsRecord struct {
	// Metadata содержит открытые поля записи.
	Metadata model.RecordMetadata

	// Payload содержит приватный credentials payload.
	Payload model.CredentialsPayload
}

// CardRecord содержит открытую metadata и расшифрованный card payload.
type CardRecord struct {
	// Metadata содержит открытые поля записи.
	Metadata model.RecordMetadata

	// Payload содержит приватный card payload.
	Payload model.CardPayload
}

// CreateTextRecord создаёт text-запись в online-режиме.
func (a *Application) CreateTextRecord(ctx context.Context, request CreateTextRecordRequest) (TextRecord, error) {
	payload := &model.TextPayload{
		Text:     request.Text,
		Metadata: request.Metadata,
	}

	record, err := a.createRecord(ctx, request.Title, payload, "create text record")
	if err != nil {
		return TextRecord{}, err
	}

	return textRecordFromClient(record)
}

// CreateCredentialsRecord создаёт credentials-запись в online-режиме.
func (a *Application) CreateCredentialsRecord(
	ctx context.Context,
	request CreateCredentialsRecordRequest,
) (CredentialsRecord, error) {
	payload := &model.CredentialsPayload{
		Login:    request.Login,
		Password: request.Password,
		URL:      request.URL,
		Metadata: request.Metadata,
	}

	record, err := a.createRecord(ctx, request.Title, payload, "create credentials record")
	if err != nil {
		return CredentialsRecord{}, err
	}

	return credentialsRecordFromClient(record)
}

// CreateCardRecord создаёт card-запись в online-режиме.
func (a *Application) CreateCardRecord(
	ctx context.Context,
	request CreateCardRecordRequest,
) (CardRecord, error) {
	payload := &model.CardPayload{
		Number:      request.Number,
		Cardholder:  request.Cardholder,
		ExpiryMonth: request.ExpiryMonth,
		ExpiryYear:  request.ExpiryYear,
		CVV:         request.CVV,
		Metadata:    request.Metadata,
	}

	record, err := a.createRecord(ctx, request.Title, payload, "create card record")
	if err != nil {
		return CardRecord{}, err
	}

	return cardRecordFromClient(record)
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

// GetRecord возвращает запись текущего пользователя с payload согласно её типу.
func (a *Application) GetRecord(ctx context.Context, recordID string) (Record, error) {
	if a.records == nil {
		return Record{}, errors.New("record client is not configured")
	}
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

	return Record{
		Metadata: record.Metadata,
		Payload:  record.Payload,
	}, nil
}

// GetTextRecord возвращает text-запись текущего пользователя в online-режиме.
func (a *Application) GetTextRecord(ctx context.Context, recordID string) (TextRecord, error) {
	record, err := a.GetRecord(ctx, recordID)
	if err != nil {
		return TextRecord{}, err
	}

	return textRecordFromRecord(record)
}

// UpdateTextRecord изменяет text-запись в online-режиме.
func (a *Application) UpdateTextRecord(ctx context.Context, request UpdateTextRecordRequest) (TextRecord, error) {
	payload := &model.TextPayload{
		Text:     request.Text,
		Metadata: request.Metadata,
	}

	record, err := a.updateRecord(
		ctx,
		request.RecordID,
		request.ExpectedRevision,
		request.Title,
		payload,
		"update text record",
	)
	if err != nil {
		return TextRecord{}, err
	}

	return textRecordFromClient(record)
}

// UpdateCredentialsRecord изменяет credentials-запись в online-режиме.
func (a *Application) UpdateCredentialsRecord(
	ctx context.Context,
	request UpdateCredentialsRecordRequest,
) (CredentialsRecord, error) {
	payload := &model.CredentialsPayload{
		Login:    request.Login,
		Password: request.Password,
		URL:      request.URL,
		Metadata: request.Metadata,
	}

	record, err := a.updateRecord(
		ctx,
		request.RecordID,
		request.ExpectedRevision,
		request.Title,
		payload,
		"update credentials record",
	)
	if err != nil {
		return CredentialsRecord{}, err
	}

	return credentialsRecordFromClient(record)
}

// UpdateCardRecord изменяет card-запись в online-режиме.
func (a *Application) UpdateCardRecord(
	ctx context.Context,
	request UpdateCardRecordRequest,
) (CardRecord, error) {
	payload := &model.CardPayload{
		Number:      request.Number,
		Cardholder:  request.Cardholder,
		ExpiryMonth: request.ExpiryMonth,
		ExpiryYear:  request.ExpiryYear,
		CVV:         request.CVV,
		Metadata:    request.Metadata,
	}

	record, err := a.updateRecord(
		ctx,
		request.RecordID,
		request.ExpectedRevision,
		request.Title,
		payload,
		"update card record",
	)
	if err != nil {
		return CardRecord{}, err
	}

	return cardRecordFromClient(record)
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

func (a *Application) createRecord(
	ctx context.Context,
	title string,
	payload model.RecordPayload,
	operation string,
) (httpclient.Record, error) {
	if a.records == nil {
		return httpclient.Record{}, errors.New("record client is not configured")
	}
	if err := model.ValidateRecordTitle(title); err != nil {
		return httpclient.Record{}, err
	}
	if payload == nil {
		return httpclient.Record{}, errUnexpectedRecordPayload
	}
	if err := payload.Validate(); err != nil {
		return httpclient.Record{}, err
	}

	storedSession, err := a.loadSession()
	if err != nil {
		return httpclient.Record{}, err
	}

	record, err := a.records.CreateRecord(ctx, storedSession.AccessToken, httpclient.CreateRecordRequest{
		Title:   title,
		Payload: payload,
	})
	if err != nil {
		return httpclient.Record{}, mapRecordClientError(operation, err)
	}

	return record, nil
}

func (a *Application) updateRecord(
	ctx context.Context,
	recordID string,
	expectedRevision int64,
	title string,
	payload model.RecordPayload,
	operation string,
) (httpclient.Record, error) {
	if a.records == nil {
		return httpclient.Record{}, errors.New("record client is not configured")
	}
	if err := model.ValidateRecordID(recordID); err != nil {
		return httpclient.Record{}, err
	}
	if err := model.ValidateRecordRevision(expectedRevision); err != nil {
		return httpclient.Record{}, err
	}
	if err := model.ValidateRecordTitle(title); err != nil {
		return httpclient.Record{}, err
	}
	if payload == nil {
		return httpclient.Record{}, errUnexpectedRecordPayload
	}
	if err := payload.Validate(); err != nil {
		return httpclient.Record{}, err
	}

	storedSession, err := a.loadSession()
	if err != nil {
		return httpclient.Record{}, err
	}

	record, err := a.records.UpdateRecord(
		ctx,
		storedSession.AccessToken,
		recordID,
		expectedRevision,
		httpclient.UpdateRecordRequest{
			Title:   title,
			Payload: payload,
		},
	)
	if err != nil {
		return httpclient.Record{}, mapRecordClientError(operation, err)
	}

	return record, nil
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

func textRecordFromClient(record httpclient.Record) (TextRecord, error) {
	return textRecordFromRecord(Record{
		Metadata: record.Metadata,
		Payload:  record.Payload,
	})
}

func textRecordFromRecord(record Record) (TextRecord, error) {
	payload, ok := record.Payload.(*model.TextPayload)
	if !ok || payload == nil || record.Metadata.Type != model.RecordTypeText {
		return TextRecord{}, fmt.Errorf("text record payload: %w", errUnexpectedRecordPayload)
	}

	return TextRecord{
		Metadata: record.Metadata,
		Payload:  *payload,
	}, nil
}

func credentialsRecordFromClient(record httpclient.Record) (CredentialsRecord, error) {
	payload, ok := record.Payload.(*model.CredentialsPayload)
	if !ok || payload == nil || record.Metadata.Type != model.RecordTypeCredentials {
		return CredentialsRecord{}, fmt.Errorf("credentials record payload: %w", errUnexpectedRecordPayload)
	}

	return CredentialsRecord{
		Metadata: record.Metadata,
		Payload:  *payload,
	}, nil
}

func cardRecordFromClient(record httpclient.Record) (CardRecord, error) {
	payload, ok := record.Payload.(*model.CardPayload)
	if !ok || payload == nil || record.Metadata.Type != model.RecordTypeCard {
		return CardRecord{}, fmt.Errorf("card record payload: %w", errUnexpectedRecordPayload)
	}

	return CardRecord{
		Metadata: record.Metadata,
		Payload:  *payload,
	}, nil
}
