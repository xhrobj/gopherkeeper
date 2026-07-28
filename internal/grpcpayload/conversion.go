// Package grpcpayload преобразует payload записей между доменной моделью
// и protobuf-представлением, общим для gRPC-клиента и gRPC-сервера.
package grpcpayload

import (
	"bytes"

	"github.com/xhrobj/gopherkeeper/internal/model"
	gopherkeeperpb "github.com/xhrobj/gopherkeeper/internal/proto"
	"google.golang.org/protobuf/types/known/wrapperspb"
)

// ToProto преобразует доменный payload записи в protobuf-представление.
// Проверка доменных ограничений остаётся ответственностью вызывающего кода.
func ToProto(payload model.RecordPayload) (*gopherkeeperpb.RecordPayload, error) {
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

// FromProto преобразует protobuf-представление payload записи в доменную модель.
// Проверка доменных ограничений остаётся ответственностью вызывающего кода.
func FromProto(payload *gopherkeeperpb.RecordPayload) (model.RecordPayload, error) {
	if payload == nil {
		return nil, model.ErrRecordTypeUnsupported
	}

	switch payload.WhichValue() {
	case gopherkeeperpb.RecordPayload_Credentials_case:
		return credentialsFromProto(payload.GetCredentials())
	case gopherkeeperpb.RecordPayload_Card_case:
		return cardFromProto(payload.GetCard())
	case gopherkeeperpb.RecordPayload_Text_case:
		return textFromProto(payload.GetText())
	case gopherkeeperpb.RecordPayload_Binary_case:
		return binaryFromProto(payload.GetBinary())
	default:
		return nil, model.ErrRecordTypeUnsupported
	}
}

func credentialsFromProto(value *gopherkeeperpb.CredentialsPayload) (model.RecordPayload, error) {
	if value == nil {
		return nil, model.ErrInvalidCredentialsPayload
	}

	return &model.CredentialsPayload{
		Login:    value.GetLogin(),
		Password: value.GetPassword(),
		URL:      value.GetUrl(),
		Metadata: value.GetMetadata(),
	}, nil
}

func cardFromProto(value *gopherkeeperpb.CardPayload) (model.RecordPayload, error) {
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

	return card, nil
}

func textFromProto(value *gopherkeeperpb.TextPayload) (model.RecordPayload, error) {
	if value == nil {
		return nil, model.ErrInvalidTextPayload
	}

	return &model.TextPayload{
		Text:     value.GetText(),
		Metadata: value.GetMetadata(),
	}, nil
}

func binaryFromProto(value *gopherkeeperpb.BinaryPayload) (model.RecordPayload, error) {
	if value == nil || !value.HasData() {
		return nil, model.ErrInvalidBinaryPayload
	}

	return &model.BinaryPayload{
		Filename: value.GetFilename(),
		Data:     bytes.Clone(value.GetData()),
		Metadata: value.GetMetadata(),
	}, nil
}
