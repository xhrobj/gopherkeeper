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

func TestTextPayload_ValidateNil(t *testing.T) {
	var payload *TextPayload

	if err := payload.Validate(); !errors.Is(err, ErrInvalidTextPayload) {
		t.Fatalf("Validate() error = %v, want %v", err, ErrInvalidTextPayload)
	}
}

func TestEncryptedRecord_Metadata(t *testing.T) {
	createdAt := time.Date(2026, time.July, 8, 12, 0, 0, 0, time.UTC)
	updatedAt := time.Date(2026, time.July, 8, 12, 5, 0, 0, time.UTC)
	record := EncryptedRecord{
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
		{name: "maximum size", title: strings.Repeat("я", RecordTitleMaxSize/2)},
		{name: "empty", title: "", wantErr: ErrInvalidRecordTitle},
		{name: "blank", title: " \t\n", wantErr: ErrInvalidRecordTitle},
		{name: "too large", title: strings.Repeat("a", RecordTitleMaxSize+1), wantErr: ErrInvalidRecordTitle},
		{name: "tab", title: "GitHub\tpersonal", wantErr: ErrInvalidRecordTitle},
		{name: "line feed", title: "GitHub\npersonal", wantErr: ErrInvalidRecordTitle},
		{name: "escape", title: "GitHub\x1b[31m", wantErr: ErrInvalidRecordTitle},
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

func TestRecordMetadata_Validate(t *testing.T) {
	valid := RecordMetadata{
		ID:       "550e8400-e29b-41d4-a716-446655440000",
		Type:     RecordTypeText,
		Title:    "Private note",
		Revision: 1,
	}
	tests := []struct {
		name     string
		metadata RecordMetadata
		wantErr  error
	}{
		{
			name:     "valid",
			metadata: valid,
		},
		{
			name: "invalid ID",
			metadata: recordMetadataWith(valid, func(value *RecordMetadata) {
				value.ID = "invalid"
			}),
			wantErr: ErrInvalidRecordID,
		},
		{
			name: "unsupported type",
			metadata: recordMetadataWith(valid, func(value *RecordMetadata) {
				value.Type = "unknown"
			}),
			wantErr: ErrRecordTypeUnsupported,
		},
		{
			name: "invalid title",
			metadata: recordMetadataWith(valid, func(value *RecordMetadata) {
				value.Title = "line\nbreak"
			}),
			wantErr: ErrInvalidRecordTitle,
		},
		{
			name: "invalid revision",
			metadata: recordMetadataWith(valid, func(value *RecordMetadata) {
				value.Revision = 0
			}),
			wantErr: ErrInvalidRecordRevision,
		},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			err := tt.metadata.Validate()
			if !errors.Is(err, tt.wantErr) {
				t.Fatalf("Validate() error = %v, want %v", err, tt.wantErr)
			}
		})
	}
}

func TestRecord_Validate(t *testing.T) {
	validMetadata := RecordMetadata{
		ID:       "550e8400-e29b-41d4-a716-446655440000",
		Type:     RecordTypeText,
		Title:    "Private note",
		Revision: 1,
	}
	tests := []struct {
		name    string
		record  Record
		wantErr error
	}{
		{
			name: "valid",
			record: Record{
				Metadata: validMetadata,
				Payload:  &TextPayload{Text: "secret"},
			},
		},
		{
			name: "invalid metadata",
			record: Record{
				Metadata: recordMetadataWith(validMetadata, func(value *RecordMetadata) {
					value.ID = "invalid"
				}),
				Payload: &TextPayload{Text: "secret"},
			},
			wantErr: ErrInvalidRecordID,
		},
		{
			name:    "missing payload",
			record:  Record{Metadata: validMetadata},
			wantErr: ErrInvalidRecordData,
		},
		{
			name: "typed nil payload",
			record: Record{
				Metadata: validMetadata,
				Payload:  (*TextPayload)(nil),
			},
			wantErr: ErrInvalidTextPayload,
		},
		{
			name: "invalid payload",
			record: Record{
				Metadata: validMetadata,
				Payload:  &TextPayload{},
			},
			wantErr: ErrInvalidTextPayload,
		},
		{
			name: "payload type mismatch",
			record: Record{
				Metadata: validMetadata,
				Payload: &CredentialsPayload{
					Login:    "alice",
					Password: "correct-horse-battery-staple",
				},
			},
			wantErr: ErrInvalidRecordData,
		},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			err := tt.record.Validate()
			if !errors.Is(err, tt.wantErr) {
				t.Fatalf("Validate() error = %v, want %v", err, tt.wantErr)
			}
		})
	}
}

func recordMetadataWith(metadata RecordMetadata, change func(*RecordMetadata)) RecordMetadata {
	change(&metadata)
	return metadata
}
