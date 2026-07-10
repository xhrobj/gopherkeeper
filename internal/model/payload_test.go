package model

import (
	"errors"
	"testing"
)

func TestNewRecordPayload(t *testing.T) {
	tests := []struct {
		name       string
		recordType RecordType
		wantType   RecordType
		wantErr    error
	}{
		{name: "text", recordType: RecordTypeText, wantType: RecordTypeText},
		{name: "credentials", recordType: RecordTypeCredentials, wantType: RecordTypeCredentials},
		{name: "card is not implemented", recordType: RecordTypeCard, wantErr: ErrRecordTypeUnsupported},
		{name: "binary is not implemented", recordType: RecordTypeBinary, wantErr: ErrRecordTypeUnsupported},
		{name: "unknown", recordType: RecordType("unknown"), wantErr: ErrRecordTypeUnsupported},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			payload, err := NewRecordPayload(tt.recordType)
			if !errors.Is(err, tt.wantErr) {
				t.Fatalf("NewRecordPayload() error = %v, want %v", err, tt.wantErr)
			}
			if tt.wantErr != nil {
				if payload != nil {
					t.Fatalf("NewRecordPayload() payload = %T, want nil", payload)
				}
				return
			}
			if payload.RecordType() != tt.wantType {
				t.Fatalf("RecordType() = %q, want %q", payload.RecordType(), tt.wantType)
			}
		})
	}
}

func TestRecordPayload_RecordType(t *testing.T) {
	tests := []struct {
		name    string
		payload RecordPayload
		want    RecordType
	}{
		{name: "text", payload: &TextPayload{Text: "secret"}, want: RecordTypeText},
		{
			name: "credentials",
			payload: &CredentialsPayload{
				Login:    "alice",
				Password: "correct-horse-battery-staple",
			},
			want: RecordTypeCredentials,
		},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			if got := tt.payload.RecordType(); got != tt.want {
				t.Fatalf("RecordType() = %q, want %q", got, tt.want)
			}
		})
	}
}
