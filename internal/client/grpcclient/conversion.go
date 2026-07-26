package grpcclient

import (
	"bytes"
	"errors"
	"fmt"
	"time"

	"github.com/xhrobj/gopherkeeper/internal/model"
	gopherkeeperpb "github.com/xhrobj/gopherkeeper/internal/proto"
	"google.golang.org/protobuf/types/known/timestamppb"
	"google.golang.org/protobuf/types/known/wrapperspb"
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

	result := &gopherkeeperpb.RecordPayload{}
	switch value := payload.(type) {
	case *model.CredentialsPayload:
		credentials := &gopherkeeperpb.CredentialsPayload{}
		credentials.SetLogin(value.Login)
		credentials.SetPassword(value.Password)
		credentials.SetUrl(value.URL)
		credentials.SetMetadata(value.Metadata)
		result.SetCredentials(credentials)
	case *model.CardPayload:
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
		text := &gopherkeeperpb.TextPayload{}
		text.SetText(value.Text)
		text.SetMetadata(value.Metadata)
		result.SetText(text)
	case *model.BinaryPayload:
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
		return nil, errRecordPayloadRequired
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
		result = &model.TextPayload{Text: value.GetText(), Metadata: value.GetMetadata()}
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

func timeFromProto(value *timestamppb.Timestamp) (time.Time, error) {
	if value == nil {
		return time.Time{}, errTimestampRequired
	}
	if err := value.CheckValid(); err != nil {
		return time.Time{}, err
	}
	return value.AsTime().UTC(), nil
}
