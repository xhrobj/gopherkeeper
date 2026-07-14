package cachecrypto

import (
	"errors"
	"reflect"
	"testing"
	"time"

	"github.com/xhrobj/gopherkeeper/internal/model"
)

func TestRecordCodec_RoundTrip(t *testing.T) {
	expiryMonth := 7
	expiryYear := 2028
	createdAt := time.Date(2026, time.July, 14, 12, 0, 0, 0, time.UTC)
	updatedAt := createdAt.Add(time.Hour)

	tests := []struct {
		name    string
		typeID  model.RecordType
		payload model.RecordPayload
	}{
		{name: "text", typeID: model.RecordTypeText, payload: &model.TextPayload{Text: "secret", Metadata: "note"}},
		{name: "credentials", typeID: model.RecordTypeCredentials, payload: &model.CredentialsPayload{Login: "alice", Password: "password", URL: "https://example.com"}},
		{name: "card", typeID: model.RecordTypeCard, payload: &model.CardPayload{Number: "2013061420200619", ExpiryMonth: &expiryMonth, ExpiryYear: &expiryYear, CVV: "123"}},
		{name: "binary", typeID: model.RecordTypeBinary, payload: &model.BinaryPayload{Filename: "backup.bin", Data: []byte{0, 1, 2, 255}, ContentType: "application/octet-stream"}},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			record := model.Record{
				Metadata: model.RecordMetadata{
					ID:        testRecordID,
					Type:      tt.typeID,
					Title:     "test record",
					Revision:  2,
					CreatedAt: createdAt,
					UpdatedAt: updatedAt,
				},
				Payload: tt.payload,
			}

			encoded, err := EncodeRecord(record)
			if err != nil {
				t.Fatalf("EncodeRecord() error = %v", err)
			}
			decoded, err := DecodeRecord(encoded)
			if err != nil {
				t.Fatalf("DecodeRecord() error = %v", err)
			}
			if !reflect.DeepEqual(decoded, record) {
				t.Fatalf("DecodeRecord() = %#v, want %#v", decoded, record)
			}
		})
	}
}

func TestDecodeRecord_RejectsUnsupportedOrInvalidFormat(t *testing.T) {
	tests := []struct {
		name string
		data string
		want error
	}{
		{
			name: "unsupported version",
			data: `{"format_version":69}`,
			want: ErrUnsupportedRecordFormatVersion,
		},
		{
			name: "unknown field",
			data: `{"format_version":1,"unknown":true}`,
			want: ErrInvalidRecordFormat,
		},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			if _, err := DecodeRecord([]byte(tt.data)); !errors.Is(err, tt.want) {
				t.Fatalf("DecodeRecord() error = %v, want %v", err, tt.want)
			}
		})
	}
}
