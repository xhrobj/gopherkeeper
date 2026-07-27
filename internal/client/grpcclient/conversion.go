package grpcclient

import (
	"errors"
	"fmt"
	"time"

	"github.com/xhrobj/gopherkeeper/internal/grpcpayload"
	"github.com/xhrobj/gopherkeeper/internal/model"
	gopherkeeperpb "github.com/xhrobj/gopherkeeper/internal/proto"
	"google.golang.org/protobuf/types/known/timestamppb"
)

var (
	errInvalidUserResponse   = errors.New("invalid user response")
	errInvalidRecordResponse = errors.New("invalid record response")
	errRecordPayloadRequired = errors.New("record payload is required")
	errTimestampRequired     = errors.New("timestamp is required")
)

func userFromProto(value *gopherkeeperpb.User) (model.User, error) {
	if value == nil || value.GetId() <= 0 {
		return model.User{}, errInvalidUserResponse
	}

	if err := model.ValidateCanonicalLogin(value.GetLogin()); err != nil {
		return model.User{}, fmt.Errorf("%w: %w", errInvalidUserResponse, err)
	}

	createdAt, err := timeFromProto(value.GetCreatedAt())
	if err != nil {
		return model.User{}, fmt.Errorf("%w: created at: %w", errInvalidUserResponse, err)
	}

	return model.User{
		ID:        value.GetId(),
		Login:     value.GetLogin(),
		CreatedAt: createdAt,
	}, nil
}

func recordFromProto(value *gopherkeeperpb.Record) (model.Record, error) {
	if value == nil || value.GetMetadata() == nil || value.GetPayload() == nil {
		return model.Record{}, errInvalidRecordResponse
	}

	metadata, err := recordMetadataFromProto(value.GetMetadata())
	if err != nil {
		return model.Record{}, err
	}

	payload, err := recordPayloadFromProto(value.GetPayload())
	if err != nil {
		return model.Record{}, err
	}

	record := model.Record{Metadata: metadata, Payload: payload}
	if err := record.Validate(); err != nil {
		return model.Record{}, fmt.Errorf("validate record response: %w", err)
	}

	return record, nil
}

func recordMetadataFromProto(value *gopherkeeperpb.RecordMetadata) (model.RecordMetadata, error) {
	if value == nil {
		return model.RecordMetadata{}, errInvalidRecordResponse
	}

	recordType, err := recordTypeFromProto(value.GetType())
	if err != nil {
		return model.RecordMetadata{}, err
	}

	createdAt, err := timeFromProto(value.GetCreatedAt())
	if err != nil {
		return model.RecordMetadata{}, fmt.Errorf("convert record creation time: %w", err)
	}

	updatedAt, err := timeFromProto(value.GetUpdatedAt())
	if err != nil {
		return model.RecordMetadata{}, fmt.Errorf("convert record update time: %w", err)
	}

	metadata := model.RecordMetadata{
		ID:        value.GetId(),
		Type:      recordType,
		Title:     value.GetTitle(),
		Revision:  value.GetRevision(),
		CreatedAt: createdAt,
		UpdatedAt: updatedAt,
	}
	if err := metadata.Validate(); err != nil {
		return model.RecordMetadata{}, fmt.Errorf("validate record metadata response: %w", err)
	}
	return metadata, nil
}

func recordTypeFromProto(value gopherkeeperpb.RecordType) (model.RecordType, error) {
	switch value {
	case gopherkeeperpb.RecordType_RECORD_TYPE_CREDENTIALS:
		return model.RecordTypeCredentials, nil
	case gopherkeeperpb.RecordType_RECORD_TYPE_CARD:
		return model.RecordTypeCard, nil
	case gopherkeeperpb.RecordType_RECORD_TYPE_TEXT:
		return model.RecordTypeText, nil
	case gopherkeeperpb.RecordType_RECORD_TYPE_BINARY:
		return model.RecordTypeBinary, nil
	default:
		return "", model.ErrRecordTypeUnsupported
	}
}

func newProtoRecordPayload(payload model.RecordPayload) (*gopherkeeperpb.RecordPayload, error) {
	if payload == nil {
		return nil, errRecordPayloadRequired
	}

	if err := payload.Validate(); err != nil {
		return nil, err
	}

	return grpcpayload.ToProto(payload)
}

func recordPayloadFromProto(payload *gopherkeeperpb.RecordPayload) (model.RecordPayload, error) {
	if payload == nil {
		return nil, errRecordPayloadRequired
	}

	result, err := grpcpayload.FromProto(payload)
	if err != nil {
		return nil, err
	}

	if err := result.Validate(); err != nil {
		return nil, err
	}

	return result, nil
}

func timeFromProto(value *timestamppb.Timestamp) (time.Time, error) {
	if value == nil {
		return time.Time{}, errTimestampRequired
	}

	if err := value.CheckValid(); err != nil {
		return time.Time{}, err
	}

	return value.AsTime().UTC(), nil
}
