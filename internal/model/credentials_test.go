package model

import (
	"errors"
	"strings"
	"testing"
)

func TestCredentialsPayload_Validate(t *testing.T) {
	tests := []struct {
		name    string
		payload CredentialsPayload
		wantErr error
	}{
		{
			name: "valid payload",
			payload: CredentialsPayload{
				Login:    "alice",
				Password: "correct-horse-battery-staple",
				URL:      "https://example.com",
				Metadata: "personal account",
			},
		},
		{
			name: "valid Unicode",
			payload: CredentialsPayload{
				Login:    "алиса",
				Password: "секретный-пароль",
				URL:      "https://пример.рф",
				Metadata: "личный аккаунт",
			},
		},
		{
			name: "optional fields are empty",
			payload: CredentialsPayload{
				Login:    "alice",
				Password: "correct-horse-battery-staple",
			},
		},
		{
			name: "surrounding whitespace is allowed",
			payload: CredentialsPayload{
				Login:    " alice ",
				Password: " correct-horse-battery-staple ",
				URL:      " https://example.com ",
			},
		},
		{
			name: "metadata at limit",
			payload: CredentialsPayload{
				Login:    "alice",
				Password: "correct-horse-battery-staple",
				Metadata: strings.Repeat("a", MetadataMaxSize),
			},
		},
		{
			name: "empty login",
			payload: CredentialsPayload{
				Password: "correct-horse-battery-staple",
			},
			wantErr: ErrInvalidCredentialsPayload,
		},
		{
			name: "blank login",
			payload: CredentialsPayload{
				Login:    " \t\n",
				Password: "correct-horse-battery-staple",
			},
			wantErr: ErrInvalidCredentialsPayload,
		},
		{
			name: "empty password",
			payload: CredentialsPayload{
				Login: "alice",
			},
			wantErr: ErrInvalidCredentialsPayload,
		},
		{
			name: "blank password",
			payload: CredentialsPayload{
				Login:    "alice",
				Password: " \t\n",
			},
			wantErr: ErrInvalidCredentialsPayload,
		},
		{
			name: "invalid UTF-8 login",
			payload: CredentialsPayload{
				Login:    string([]byte{0xff}),
				Password: "correct-horse-battery-staple",
			},
			wantErr: ErrInvalidCredentialsPayload,
		},
		{
			name: "invalid UTF-8 password",
			payload: CredentialsPayload{
				Login:    "alice",
				Password: string([]byte{0xff}),
			},
			wantErr: ErrInvalidCredentialsPayload,
		},
		{
			name: "invalid UTF-8 URL",
			payload: CredentialsPayload{
				Login:    "alice",
				Password: "correct-horse-battery-staple",
				URL:      string([]byte{0xff}),
			},
			wantErr: ErrInvalidCredentialsPayload,
		},
		{
			name: "invalid UTF-8 metadata",
			payload: CredentialsPayload{
				Login:    "alice",
				Password: "correct-horse-battery-staple",
				Metadata: string([]byte{0xff}),
			},
			wantErr: ErrInvalidCredentialsPayload,
		},
		{
			name: "metadata too large",
			payload: CredentialsPayload{
				Login:    "alice",
				Password: "correct-horse-battery-staple",
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

func TestCredentialsPayload_ValidateNil(t *testing.T) {
	var payload *CredentialsPayload

	if err := payload.Validate(); !errors.Is(err, ErrInvalidCredentialsPayload) {
		t.Fatalf("Validate() error = %v, want %v", err, ErrInvalidCredentialsPayload)
	}
}

func TestCredentialsPayload_ValidatePreservesValues(t *testing.T) {
	payload := CredentialsPayload{
		Login:    " alice ",
		Password: " correct-horse-battery-staple ",
		URL:      " https://example.com ",
		Metadata: " personal account ",
	}
	want := payload

	if err := payload.Validate(); err != nil {
		t.Fatalf("Validate() error = %v", err)
	}
	if payload != want {
		t.Fatalf("Validate() payload = %+v, want %+v", payload, want)
	}
}
