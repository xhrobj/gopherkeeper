package grpcserver

import (
	"bytes"
	"fmt"
	"time"

	"github.com/xhrobj/gopherkeeper/internal/model"
	gopherkeeperpb "github.com/xhrobj/gopherkeeper/internal/proto"
	"google.golang.org/protobuf/types/known/timestamppb"
	"google.golang.org/protobuf/types/known/wrapperspb"
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
	result := &gopherkeeperpb.RecordPayload{}

	switch value := payload.(type) {
	case *model.CredentialsPayload:
		if value == nil {
			return nil, model.ErrInvalidCredentialsPayload
		}

		credentials := &gopherkeeperpb.CredentialsPayload{}
		credentials.SetLogin(value.Login)
		credentials.SetPassword(value.Password)
		credentials.SetUrl(value.URL)
		credentials.SetMetadata(value.Metadata)
		result.SetCredentials(credentials)
	case *model.CardPayload:
		if value == nil {
			return nil, model.ErrInvalidCardPayload
		}

		card := &gopherkeeperpb.CardPayload{}
		card.SetNumber(value.Number)
		card.SetCardholder(value.Cardholder)
		card.SetCvv(value.CVV)
		card.SetMetadata(value.Metadata)
		if value.ExpiryMonth != nil {
			card.SetExpiryMonth(wrapperspb.Int32(int32(*value.ExpiryMonth)))
		}
		if value.ExpiryYear != nil {
			card.SetExpiryYear(wrapperspb.Int32(int32(*value.ExpiryYear)))
		}
		result.SetCard(card)
	case *model.TextPayload:
		if value == nil {
			return nil, model.ErrInvalidTextPayload
		}

		text := &gopherkeeperpb.TextPayload{}
		text.SetText(value.Text)
		text.SetMetadata(value.Metadata)
		result.SetText(text)
	case *model.BinaryPayload:
		if value == nil {
			return nil, model.ErrInvalidBinaryPayload
		}

		binary := &gopherkeeperpb.BinaryPayload{}
		binary.SetFilename(value.Filename)
		binary.SetData(bytes.Clone(value.Data))
		binary.SetMetadata(value.Metadata)
		result.SetBinary(binary)
	default:
		return nil, model.ErrRecordTypeUnsupported
	}

	return result, nil
}

func recordPayloadFromProto(payload *gopherkeeperpb.RecordPayload) (model.RecordPayload, error) {
	if payload == nil {
		return nil, model.ErrRecordTypeUnsupported
	}

	var result model.RecordPayload

	switch payload.WhichValue() {
	case gopherkeeperpb.RecordPayload_Credentials_case:
		value := payload.GetCredentials()
		if value == nil {
			return nil, model.ErrInvalidCredentialsPayload
		}
		result = &model.CredentialsPayload{
			Login:    value.GetLogin(),
			Password: value.GetPassword(),
			URL:      value.GetUrl(),
			Metadata: value.GetMetadata(),
		}
	case gopherkeeperpb.RecordPayload_Card_case:
		value := payload.GetCard()
		if value == nil {
			return nil, model.ErrInvalidCardPayload
		}
		card := &model.CardPayload{
			Number:     value.GetNumber(),
			Cardholder: value.GetCardholder(),
			CVV:        value.GetCvv(),
			Metadata:   value.GetMetadata(),
		}
		if value.GetExpiryMonth() != nil {
			month := int(value.GetExpiryMonth().GetValue())
			card.ExpiryMonth = &month
		}
		if value.GetExpiryYear() != nil {
			year := int(value.GetExpiryYear().GetValue())
			card.ExpiryYear = &year
		}
		result = card
	case gopherkeeperpb.RecordPayload_Text_case:
		value := payload.GetText()
		if value == nil {
			return nil, model.ErrInvalidTextPayload
		}
		result = &model.TextPayload{
			Text:     value.GetText(),
			Metadata: value.GetMetadata(),
		}
	case gopherkeeperpb.RecordPayload_Binary_case:
		value := payload.GetBinary()
		if value == nil || !value.HasData() {
			return nil, model.ErrInvalidBinaryPayload
		}
		result = &model.BinaryPayload{
			Filename: value.GetFilename(),
			Data:     bytes.Clone(value.GetData()),
			Metadata: value.GetMetadata(),
		}
	default:
		return nil, model.ErrRecordTypeUnsupported
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
