package grpcclient

import (
	"bytes"
	"errors"
	"reflect"
	"testing"

	"github.com/xhrobj/gopherkeeper/internal/model"
	gopherkeeperpb "github.com/xhrobj/gopherkeeper/internal/proto"
	"google.golang.org/protobuf/types/known/timestamppb"
)

func TestRecordPayloadRoundTrip(t *testing.T) {
	month, year := 12, 30
	tests := []struct {
		name    string
		payload model.RecordPayload
	}{
		{
			name: "credentials",
			payload: &model.CredentialsPayload{
				Login: "alice", Password: "secret", URL: "https://example.com", Metadata: "notes",
			},
		},
		{
			name: "card",
			payload: &model.CardPayload{
				Number: "4111111111111111", Cardholder: "JOEL MILLER",
				ExpiryMonth: &month, ExpiryYear: &year, CVV: "014", Metadata: "notes",
			},
		},
		{name: "text", payload: &model.TextPayload{Text: "secret", Metadata: "notes"}},
		{name: "binary", payload: &model.BinaryPayload{Filename: "secret.bin", Data: []byte{0, 1, 2}, Metadata: "notes"}},
		{name: "empty binary", payload: &model.BinaryPayload{Filename: "empty.bin", Data: []byte{}, Metadata: "notes"}},
	}

	for _, test := range tests {
		t.Run(test.name, func(t *testing.T) {
			encoded, err := newProtoRecordPayload(test.payload)
			if err != nil {
				t.Fatalf("newProtoRecordPayload() error = %v", err)
			}
			if test.name == "empty binary" && !encoded.GetBinary().HasData() {
				t.Fatal("empty binary data is not marked present")
			}

			decoded, err := recordPayloadFromProto(encoded)
			if err != nil {
				t.Fatalf("recordPayloadFromProto() error = %v", err)
			}
			if !reflect.DeepEqual(decoded, test.payload) {
				t.Fatalf("round trip payload = %#v, want %#v", decoded, test.payload)
			}
		})
	}
}

func TestNewProtoRecordPayloadCopiesBinaryData(t *testing.T) {
	data := []byte{1, 2, 3}
	encoded, err := newProtoRecordPayload(&model.BinaryPayload{Filename: "secret.bin", Data: data})
	if err != nil {
		t.Fatalf("newProtoRecordPayload() error = %v", err)
	}
	data[0] = 9
	if bytes.Equal(encoded.GetBinary().GetData(), data) {
		t.Fatal("newProtoRecordPayload() retained source binary slice")
	}
}

func TestRecordPayloadFromProtoRejectsMissingBinaryData(t *testing.T) {
	binary := &gopherkeeperpb.BinaryPayload{}
	binary.SetFilename("empty.bin")
	payload := &gopherkeeperpb.RecordPayload{}
	payload.SetBinary(binary)

	_, err := recordPayloadFromProto(payload)
	if !errors.Is(err, model.ErrInvalidBinaryPayload) {
		t.Fatalf("recordPayloadFromProto() error = %v, want invalid binary", err)
	}
}

func TestRecordTypeFromProtoRejectsUnknownType(t *testing.T) {
	_, err := recordTypeFromProto(gopherkeeperpb.RecordType(69))
	if !errors.Is(err, model.ErrRecordTypeUnsupported) {
		t.Fatalf("recordTypeFromProto() error = %v, want unsupported type", err)
	}
}

func TestUserFromProtoRejectsInvalidTimestamp(t *testing.T) {
	user := protoUser()
	user.SetCreatedAt(&timestamppb.Timestamp{Seconds: 253402300800})
	_, err := userFromProto(user)
	if err == nil {
		t.Fatal("userFromProto() error = nil, want invalid timestamp")
	}
}

func TestRecordFromProto(t *testing.T) {
	got, err := recordFromProto(protoTextRecord())
	if err != nil {
		t.Fatalf("recordFromProto() error = %v", err)
	}
	want := modelTextRecord()
	if !reflect.DeepEqual(got, want) {
		t.Fatalf("recordFromProto() = %#v, want %#v", got, want)
	}
}

func TestRecordFromProtoRejectsMissingParts(t *testing.T) {
	_, err := recordFromProto(&gopherkeeperpb.Record{})
	if !errors.Is(err, errInvalidRecordResponse) {
		t.Fatalf("recordFromProto() error = %v, want invalid response", err)
	}
}
