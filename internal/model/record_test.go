package model

import (
	"errors"
	"strings"
	"testing"
	"time"

	"github.com/google/uuid"
)

func TestRecordType_Validate(t *testing.T) {
	validTypes := []RecordType{
		RecordTypeCredentials,
		RecordTypeCard,
		RecordTypeText,
		RecordTypeBinary,
	}

	for _, recordType := range validTypes {
		t.Run(string(recordType), func(t *testing.T) {
			if err := recordType.Validate(); err != nil {
				t.Fatalf("Validate() error = %v", err)
			}
		})
	}

	if err := RecordType("unknown").Validate(); !errors.Is(err, ErrRecordTypeUnsupported) {
		t.Fatalf("Validate() error = %v, want ErrRecordTypeUnsupported", err)
	}
}

func TestTextPayload_Validate(t *testing.T) {
	tests := []struct {
		name    string
		payload TextPayload
		wantErr error
	}{
		{
			name: "valid text",
			payload: TextPayload{
				Text:     "secret note",
				Metadata: "private metadata",
			},
		},
		{
			name: "metadata is optional",
			payload: TextPayload{
				Text: "secret note",
			},
		},
		{
			name: "empty text",
			payload: TextPayload{
				Text: "",
			},
			wantErr: ErrInvalidTextPayload,
		},
		{
			name: "invalid UTF-8 text",
			payload: TextPayload{
				Text: string([]byte{0xff}),
			},
			wantErr: ErrInvalidTextPayload,
		},
		{
			name: "invalid UTF-8 metadata",
			payload: TextPayload{
				Text:     "secret note",
				Metadata: string([]byte{0xff}),
			},
			wantErr: ErrInvalidTextPayload,
		},
		{
			name: "text too large",
			payload: TextPayload{
				Text: strings.Repeat("a", TextPayloadMaxSize+1),
			},
			wantErr: ErrPayloadTooLarge,
		},
		{
			name: "metadata too large",
			payload: TextPayload{
				Text:     "secret note",
				Metadata: strings.Repeat("a", MetadataMaxSize+1),
			},
			wantErr: ErrPayloadTooLarge,
		},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			err := tt.payload.Validate()
			if !errors.Is(err, tt.wantErr) {
				t.Fatalf("Validate() error = %v, want %v", err, tt.wantErr)
			}
		})
	}
}

func TestRecord_Metadata(t *testing.T) {
	createdAt := time.Date(2026, time.July, 8, 12, 0, 0, 0, time.UTC)
	updatedAt := time.Date(2026, time.July, 8, 12, 5, 0, 0, time.UTC)
	record := Record{
		ID:            "550e8400-e29b-41d4-a716-446655440000",
		UserID:        42,
		Type:          RecordTypeText,
		Title:         "secret note",
		Revision:      5,
		CreatedAt:     createdAt,
		UpdatedAt:     updatedAt,
		CryptoVersion: 1,
		KeyID:         "primary",
		Nonce:         []byte("nonce"),
		Ciphertext:    []byte("ciphertext"),
	}

	metadata := record.Metadata()
	want := RecordMetadata{
		ID:        record.ID,
		Type:      record.Type,
		Title:     record.Title,
		Revision:  record.Revision,
		CreatedAt: createdAt,
		UpdatedAt: updatedAt,
	}
	if metadata != want {
		t.Fatalf("Metadata() = %+v, want %+v", metadata, want)
	}
}

func TestValidateRecordTitle(t *testing.T) {
	tests := []struct {
		name    string
		title   string
		wantErr error
	}{
		{name: "valid", title: "github note"},
		{name: "Cyrillic", title: "секретная заметка"},
		{name: "empty", title: "", wantErr: ErrInvalidRecordTitle},
		{name: "blank", title: " \t\n", wantErr: ErrInvalidRecordTitle},
		{name: "invalid UTF-8", title: string([]byte{0xff}), wantErr: ErrInvalidRecordTitle},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			err := ValidateRecordTitle(tt.title)
			if !errors.Is(err, tt.wantErr) {
				t.Fatalf("ValidateRecordTitle() error = %v, want %v", err, tt.wantErr)
			}
		})
	}
}

func TestNewRecordID(t *testing.T) {
	id, err := NewRecordID()
	if err != nil {
		t.Fatalf("NewRecordID() error = %v", err)
	}

	parsed, err := uuid.Parse(id)
	if err != nil {
		t.Fatalf("NewRecordID() id = %q is not UUID: %v", id, err)
	}
	if parsed.Version() != 4 {
		t.Fatalf("NewRecordID() version = %d, want 4", parsed.Version())
	}
}

func TestValidateRecordID(t *testing.T) {
	if err := ValidateRecordID("550e8400-e29b-41d4-a716-446655440000"); err != nil {
		t.Fatalf("ValidateRecordID() error = %v", err)
	}

	if err := ValidateRecordID("not-a-uuid"); !errors.Is(err, ErrInvalidRecordID) {
		t.Fatalf("ValidateRecordID() error = %v, want ErrInvalidRecordID", err)
	}
}

func TestValidateRecordRevision(t *testing.T) {
	tests := []struct {
		name     string
		revision int64
		wantErr  error
	}{
		{name: "valid", revision: 42},
		{name: "zero", revision: 0, wantErr: ErrInvalidRecordRevision},
		{name: "negative", revision: -1, wantErr: ErrInvalidRecordRevision},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			err := ValidateRecordRevision(tt.revision)
			if !errors.Is(err, tt.wantErr) {
				t.Fatalf("ValidateRecordRevision() error = %v, want %v", err, tt.wantErr)
			}
		})
	}
}
