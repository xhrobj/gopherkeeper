package grpcserver

import (
	"bytes"
	"errors"
	"reflect"
	"testing"
	"time"

	"github.com/xhrobj/gopherkeeper/internal/model"
	gopherkeeperpb "github.com/xhrobj/gopherkeeper/internal/proto"
)

func TestRecordPayloadConversion_RoundTrip(t *testing.T) {
	month := 12
	year := 30
	tests := []struct {
		name    string
		payload model.RecordPayload
	}{
		{
			name: "credentials",
			payload: &model.CredentialsPayload{
				Login:    "alice",
				Password: "correct-horse-battery-staple",
				URL:      "https://example.com",
				Metadata: "main account",
			},
		},
		{
			name: "card",
			payload: &model.CardPayload{
				Number:      "4111111111111111",
				Cardholder:  "JOEL MILLER",
				ExpiryMonth: &month,
				ExpiryYear:  &year,
				CVV:         "014",
				Metadata:    "main card",
			},
		},
		{
			name: "text",
			payload: &model.TextPayload{
				Text:     "private note",
				Metadata: "notes",
			},
		},
		{
			name: "binary",
			payload: &model.BinaryPayload{
				Filename: "secret.bin",
				Data:     []byte{0x00, 0x2a, 0xff},
				Metadata: "backup",
			},
		},
		{
			name: "empty binary",
			payload: &model.BinaryPayload{
				Filename: "empty.bin",
				Data:     []byte{},
			},
		},
	}

	for _, test := range tests {
		t.Run(test.name, func(t *testing.T) {
			encoded, err := newProtoRecordPayload(test.payload)
			if err != nil {
				t.Fatalf("newProtoRecordPayload() error = %v", err)
			}
			if binary := encoded.GetBinary(); binary != nil && !binary.HasData() {
				t.Fatal("newProtoRecordPayload() binary data presence = false")
			}

			decoded, err := recordPayloadFromProto(encoded)
			if err != nil {
				t.Fatalf("recordPayloadFromProto() error = %v", err)
			}
			if !reflect.DeepEqual(decoded, test.payload) {
				t.Fatalf("decoded payload = %#v, want %#v", decoded, test.payload)
			}
		})
	}
}

func TestRecordPayloadConversion_CopiesBinaryData(t *testing.T) {
	originalData := []byte{0x01, 0x02, 0x03}
	protoPayload, err := newProtoRecordPayload(&model.BinaryPayload{
		Filename: "secret.bin",
		Data:     originalData,
	})
	if err != nil {
		t.Fatalf("newProtoRecordPayload() error = %v", err)
	}
	protoPayload.GetBinary().GetData()[0] = 0xff
	if originalData[0] != 0x01 {
		t.Fatal("newProtoRecordPayload() retained binary data alias")
	}

	protoData := []byte{0x04, 0x05, 0x06}
	modelPayload, err := recordPayloadFromProto(newProtoBinaryPayload("secret.bin", protoData, true))
	if err != nil {
		t.Fatalf("recordPayloadFromProto() error = %v", err)
	}
	modelPayload.(*model.BinaryPayload).Data[0] = 0xff
	if !bytes.Equal(protoData, []byte{0x04, 0x05, 0x06}) {
		t.Fatal("recordPayloadFromProto() retained binary data alias")
	}
}

func TestNewProtoRecord(t *testing.T) {
	createdAt := time.Date(2026, time.July, 24, 12, 0, 0, 0, time.UTC)
	record := model.Record{
		Metadata: model.RecordMetadata{
			ID:        "00000000-0000-4000-8000-000000000042",
			Type:      model.RecordTypeText,
			Title:     "Private note",
			Revision:  42,
			CreatedAt: createdAt,
			UpdatedAt: createdAt.Add(time.Minute),
		},
		Payload: &model.TextPayload{Text: "secret", Metadata: "note"},
	}

	converted, err := newProtoRecord(record)
	if err != nil {
		t.Fatalf("newProtoRecord() error = %v", err)
	}
	if converted.GetMetadata().GetId() != record.Metadata.ID ||
		converted.GetMetadata().GetType() != gopherkeeperpb.RecordType_RECORD_TYPE_TEXT ||
		converted.GetMetadata().GetRevision() != 42 {
		t.Fatalf("converted metadata = %#v", converted.GetMetadata())
	}
	if got := converted.GetPayload().GetText().GetText(); got != "secret" {
		t.Fatalf("converted text = %q, want secret", got)
	}
}

func TestRecordPayloadFromProto_RejectsInvalidPayload(t *testing.T) {
	invalidCredentials := &gopherkeeperpb.RecordPayload{}
	invalidCredentials.SetCredentials(&gopherkeeperpb.CredentialsPayload{})

	tests := []struct {
		name    string
		payload *gopherkeeperpb.RecordPayload
	}{
		{name: "missing payload"},
		{name: "missing oneof value", payload: &gopherkeeperpb.RecordPayload{}},
		{name: "invalid credentials", payload: invalidCredentials},
		{name: "binary data absent", payload: newProtoBinaryPayload("empty.bin", nil, false)},
	}

	for _, test := range tests {
		t.Run(test.name, func(t *testing.T) {
			if _, err := recordPayloadFromProto(test.payload); err == nil {
				t.Fatal("recordPayloadFromProto() error = nil")
			}
		})
	}
}

func TestNewProtoRecordPayload_RejectsNilPayload(t *testing.T) {
	tests := []struct {
		name    string
		payload model.RecordPayload
		wantErr error
	}{
		{name: "credentials", payload: (*model.CredentialsPayload)(nil), wantErr: model.ErrInvalidCredentialsPayload},
		{name: "card", payload: (*model.CardPayload)(nil), wantErr: model.ErrInvalidCardPayload},
		{name: "text", payload: (*model.TextPayload)(nil), wantErr: model.ErrInvalidTextPayload},
		{name: "binary", payload: (*model.BinaryPayload)(nil), wantErr: model.ErrInvalidBinaryPayload},
	}

	for _, test := range tests {
		t.Run(test.name, func(t *testing.T) {
			_, err := newProtoRecordPayload(test.payload)
			if !errors.Is(err, test.wantErr) {
				t.Fatalf("newProtoRecordPayload() error = %v, want %v", err, test.wantErr)
			}
		})
	}
}
