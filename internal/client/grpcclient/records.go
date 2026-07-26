package grpcclient

import (
	"context"
	"errors"
	"fmt"

	"github.com/xhrobj/gopherkeeper/internal/model"
	gopherkeeperpb "github.com/xhrobj/gopherkeeper/internal/proto"
)

// CreateRecord создаёт запись выбранного типа на Сервере.
func (c *Client) CreateRecord(
	ctx context.Context,
	accessToken string,
	title string,
	payload model.RecordPayload,
) (model.Record, error) {
	protoPayload, err := newProtoRecordPayload(payload)
	if err != nil {
		return model.Record{}, err
	}
	request := &gopherkeeperpb.CreateRecordRequest{}
	request.SetTitle(title)
	request.SetPayload(protoPayload)

	callCtx, cancel := withRequestTimeout(authorizedContext(ctx, accessToken))
	defer cancel()

	response, err := c.records.CreateRecord(callCtx, request)
	if err != nil {
		return model.Record{}, mapRPCError("create record", err, recordErrorCause(err))
	}
	return recordFromProtoResponse("create record", response)
}

// ListRecords возвращает metadata приватных записей текущего пользователя.
func (c *Client) ListRecords(ctx context.Context, accessToken string) ([]model.RecordMetadata, error) {
	callCtx, cancel := withRequestTimeout(authorizedContext(ctx, accessToken))
	defer cancel()

	response, err := c.records.ListRecords(callCtx, &gopherkeeperpb.ListRecordsRequest{})
	if err != nil {
		return nil, mapListRecordsError(err)
	}

	if response == nil {
		return nil, invalidResponseError("list records", errors.New("response is nil"))
	}

	listed := response.GetRecords()
	records := make([]model.RecordMetadata, 0, len(listed))
	for index, value := range listed {
		metadata, err := recordMetadataFromProto(value)
		if err != nil {
			return nil, invalidResponseError(
				"list records",
				fmt.Errorf("record %d: %w", index, err),
			)
		}
		records = append(records, metadata)
	}

	return records, nil
}

// GetRecord возвращает запись текущего пользователя с payload согласно её типу.
func (c *Client) GetRecord(ctx context.Context, accessToken string, recordID string) (model.Record, error) {
	request := &gopherkeeperpb.GetRecordRequest{}
	request.SetId(recordID)

	callCtx, cancel := withRequestTimeout(authorizedContext(ctx, accessToken))
	defer cancel()

	response, err := c.records.GetRecord(callCtx, request)
	if err != nil {
		return model.Record{}, mapRPCError("get record", err, recordErrorCause(err))
	}
	return recordFromProtoResponse("get record", response)
}

// UpdateRecord изменяет запись на Сервере с проверкой ожидаемой ревизии.
func (c *Client) UpdateRecord(
	ctx context.Context,
	accessToken string,
	recordID string,
	expectedRevision int64,
	title string,
	payload model.RecordPayload,
) (model.Record, error) {
	protoPayload, err := newProtoRecordPayload(payload)
	if err != nil {
		return model.Record{}, err
	}
	request := &gopherkeeperpb.UpdateRecordRequest{}
	request.SetId(recordID)
	request.SetExpectedRevision(expectedRevision)
	request.SetTitle(title)
	request.SetPayload(protoPayload)

	callCtx, cancel := withRequestTimeout(authorizedContext(ctx, accessToken))
	defer cancel()

	response, err := c.records.UpdateRecord(callCtx, request)
	if err != nil {
		return model.Record{}, mapRPCError("update record", err, recordErrorCause(err))
	}
	return recordFromProtoResponse("update record", response)
}

// DeleteRecord удаляет запись на Сервере с проверкой ожидаемой ревизии.
func (c *Client) DeleteRecord(
	ctx context.Context,
	accessToken string,
	recordID string,
	expectedRevision int64,
) error {
	request := &gopherkeeperpb.DeleteRecordRequest{}
	request.SetId(recordID)
	request.SetExpectedRevision(expectedRevision)

	callCtx, cancel := withRequestTimeout(authorizedContext(ctx, accessToken))
	defer cancel()

	response, err := c.records.DeleteRecord(callCtx, request)
	if err != nil {
		return mapRPCError("delete record", err, recordErrorCause(err))
	}
	if response == nil {
		return invalidResponseError("delete record", errors.New("response is nil"))
	}

	return nil
}

func recordFromProtoResponse(operation string, response *gopherkeeperpb.Record) (model.Record, error) {
	record, err := recordFromProto(response)
	if err != nil {
		return model.Record{}, invalidResponseError(operation, err)
	}
	return record, nil
}
