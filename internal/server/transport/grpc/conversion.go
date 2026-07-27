package grpcserver

import (
	"fmt"
	"time"

	"github.com/xhrobj/gopherkeeper/internal/grpcpayload"
	"github.com/xhrobj/gopherkeeper/internal/model"
	gopherkeeperpb "github.com/xhrobj/gopherkeeper/internal/proto"
	"google.golang.org/protobuf/types/known/timestamppb"
)

func newProtoUser(user model.User) (*gopherkeeperpb.User, error) {
	createdAt, err := newProtoTimestamp(user.CreatedAt)
	if err != nil {
		return nil, fmt.Errorf("convert user creation time: %w", err)
	}

	result := &gopherkeeperpb.User{}
	result.SetId(user.ID)
	result.SetLogin(user.Login)
	result.SetCreatedAt(createdAt)

	return result, nil
}

func newProtoRecord(record model.Record) (*gopherkeeperpb.Record, error) {
	if err := record.Validate(); err != nil {
		return nil, fmt.Errorf("validate record response: %w", err)
	}

	metadata, err := newProtoRecordMetadata(record.Metadata)
	if err != nil {
		return nil, err
	}
	payload, err := newProtoRecordPayload(record.Payload)
	if err != nil {
		return nil, err
	}

	result := &gopherkeeperpb.Record{}
	result.SetMetadata(metadata)
	result.SetPayload(payload)

	return result, nil
}

func newProtoRecordMetadata(metadata model.RecordMetadata) (*gopherkeeperpb.RecordMetadata, error) {
	if err := metadata.Validate(); err != nil {
		return nil, fmt.Errorf("validate record metadata response: %w", err)
	}

	recordType, err := newProtoRecordType(metadata.Type)
	if err != nil {
		return nil, err
	}
	createdAt, err := newProtoTimestamp(metadata.CreatedAt)
	if err != nil {
		return nil, fmt.Errorf("convert record creation time: %w", err)
	}
	updatedAt, err := newProtoTimestamp(metadata.UpdatedAt)
	if err != nil {
		return nil, fmt.Errorf("convert record update time: %w", err)
	}

	result := &gopherkeeperpb.RecordMetadata{}
	result.SetId(metadata.ID)
	result.SetType(recordType)
	result.SetTitle(metadata.Title)
	result.SetRevision(metadata.Revision)
	result.SetCreatedAt(createdAt)
	result.SetUpdatedAt(updatedAt)

	return result, nil
}

func newProtoRecordType(recordType model.RecordType) (gopherkeeperpb.RecordType, error) {
	switch recordType {
	case model.RecordTypeCredentials:
		return gopherkeeperpb.RecordType_RECORD_TYPE_CREDENTIALS, nil
	case model.RecordTypeCard:
		return gopherkeeperpb.RecordType_RECORD_TYPE_CARD, nil
	case model.RecordTypeText:
		return gopherkeeperpb.RecordType_RECORD_TYPE_TEXT, nil
	case model.RecordTypeBinary:
		return gopherkeeperpb.RecordType_RECORD_TYPE_BINARY, nil
	default:
		return gopherkeeperpb.RecordType_RECORD_TYPE_UNSPECIFIED, model.ErrRecordTypeUnsupported
	}
}

func newProtoRecordPayload(payload model.RecordPayload) (*gopherkeeperpb.RecordPayload, error) {
	return grpcpayload.ToProto(payload)
}

func recordPayloadFromProto(payload *gopherkeeperpb.RecordPayload) (model.RecordPayload, error) {
	result, err := grpcpayload.FromProto(payload)
	if err != nil {
		return nil, err
	}

	if err := result.Validate(); err != nil {
		return nil, err
	}

	return result, nil
}

func newProtoTimestamp(value time.Time) (*timestamppb.Timestamp, error) {
	timestamp := timestamppb.New(value.UTC())
	if err := timestamp.CheckValid(); err != nil {
		return nil, err
	}

	return timestamp, nil
}
