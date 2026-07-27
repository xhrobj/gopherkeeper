package grpcserver

import (
	"context"

	"github.com/xhrobj/gopherkeeper/internal/model"
	gopherkeeperpb "github.com/xhrobj/gopherkeeper/internal/proto"
	"github.com/xhrobj/gopherkeeper/internal/server/service"
)

type recordService struct {
	gopherkeeperpb.UnimplementedRecordServiceServer
	records RecordManager
}

func newRecordService(records RecordManager) *recordService {
	return &recordService{records: records}
}

func (server *recordService) CreateRecord(
	ctx context.Context,
	request *gopherkeeperpb.CreateRecordRequest,
) (*gopherkeeperpb.Record, error) {
	userID, ok := userIDFromContext(ctx)
	if !ok {
		return nil, unauthenticatedError()
	}

	if request == nil {
		return nil, transportError(errInvalidRequest)
	}
	if server.records == nil {
		return nil, transportError(errInvalidDependencies)
	}

	payload, err := recordPayloadFromProto(request.GetPayload())
	if err != nil {
		return nil, transportError(err)
	}

	record, err := server.records.Create(ctx, service.CreateRecordRequest{
		UserID:  userID,
		Title:   request.GetTitle(),
		Payload: payload,
	})
	if err != nil {
		return nil, transportError(err)
	}

	response, err := newProtoRecord(record)
	if err != nil {
		return nil, transportError(err)
	}

	return response, nil
}

func (server *recordService) ListRecords(
	ctx context.Context,
	request *gopherkeeperpb.ListRecordsRequest,
) (*gopherkeeperpb.ListRecordsResponse, error) {
	userID, ok := userIDFromContext(ctx)
	if !ok {
		return nil, unauthenticatedError()
	}

	if request == nil {
		return nil, transportError(errInvalidRequest)
	}
	if server.records == nil {
		return nil, transportError(errInvalidDependencies)
	}

	items, err := server.records.List(ctx, userID)
	if err != nil {
		return nil, transportError(err)
	}

	records := make([]*gopherkeeperpb.RecordMetadata, 0, len(items))
	for _, item := range items {
		metadata, err := newProtoRecordMetadata(item)
		if err != nil {
			return nil, transportError(err)
		}
		records = append(records, metadata)
	}

	response := &gopherkeeperpb.ListRecordsResponse{}
	response.SetRecords(records)

	return response, nil
}

func (server *recordService) GetRecord(
	ctx context.Context,
	request *gopherkeeperpb.GetRecordRequest,
) (*gopherkeeperpb.Record, error) {
	userID, ok := userIDFromContext(ctx)
	if !ok {
		return nil, unauthenticatedError()
	}

	if request == nil {
		return nil, transportError(errInvalidRequest)
	}
	if server.records == nil {
		return nil, transportError(errInvalidDependencies)
	}

	record, err := server.records.Get(ctx, userID, request.GetId())
	if err != nil {
		return nil, transportError(err)
	}

	response, err := newProtoRecord(record)
	if err != nil {
		return nil, transportError(err)
	}

	return response, nil
}

func (server *recordService) UpdateRecord(
	ctx context.Context,
	request *gopherkeeperpb.UpdateRecordRequest,
) (*gopherkeeperpb.Record, error) {
	userID, ok := userIDFromContext(ctx)
	if !ok {
		return nil, unauthenticatedError()
	}

	if request == nil {
		return nil, transportError(errInvalidRequest)
	}
	if server.records == nil {
		return nil, transportError(errInvalidDependencies)
	}
	if request.GetExpectedRevision() <= 0 {
		return nil, transportError(model.ErrRecordPreconditionRequired)
	}

	payload, err := recordPayloadFromProto(request.GetPayload())
	if err != nil {
		return nil, transportError(err)
	}

	record, err := server.records.Update(ctx, service.UpdateRecordRequest{
		UserID:           userID,
		RecordID:         request.GetId(),
		ExpectedRevision: request.GetExpectedRevision(),
		Title:            request.GetTitle(),
		Payload:          payload,
	})
	if err != nil {
		return nil, transportError(err)
	}

	response, err := newProtoRecord(record)
	if err != nil {
		return nil, transportError(err)
	}

	return response, nil
}

func (server *recordService) DeleteRecord(
	ctx context.Context,
	request *gopherkeeperpb.DeleteRecordRequest,
) (*gopherkeeperpb.DeleteRecordResponse, error) {
	userID, ok := userIDFromContext(ctx)
	if !ok {
		return nil, unauthenticatedError()
	}

	if request == nil {
		return nil, transportError(errInvalidRequest)
	}
	if server.records == nil {
		return nil, transportError(errInvalidDependencies)
	}
	if request.GetExpectedRevision() <= 0 {
		return nil, transportError(model.ErrRecordPreconditionRequired)
	}

	if err := server.records.Delete(ctx, service.DeleteRecordRequest{
		UserID:           userID,
		RecordID:         request.GetId(),
		ExpectedRevision: request.GetExpectedRevision(),
	}); err != nil {
		return nil, transportError(err)
	}

	return &gopherkeeperpb.DeleteRecordResponse{}, nil
}
