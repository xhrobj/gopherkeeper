package model

import (
	"errors"
	"strings"
	"testing"
)

func TestCredentialsPayload_Validate(t *testing.T) {
	valid := CredentialsPayload{
		Login:    "alice",
		Password: "correct-horse-battery-staple",
		URL:      "https://example.com",
		Metadata: "personal account",
	}

	tests := []struct {
		name    string
		mutate  func(*CredentialsPayload)
		wantErr error
	}{
		{name: "valid"},
		{name: "valid Unicode", mutate: func(value *CredentialsPayload) {
			value.Login = "алиса"
			value.Password = "секретный пароль"
			value.URL = "https://пример.рф"
			value.Metadata = "личный аккаунт"
		}},
		{name: "optional fields empty", mutate: func(value *CredentialsPayload) {
			value.URL = ""
			value.Metadata = ""
		}},
		{name: "surrounding whitespace allowed", mutate: func(value *CredentialsPayload) {
			value.Login = " alice "
			value.Password = " correct-horse-battery-staple "
			value.URL = " https://example.com "
		}},
		{name: "fields at Unicode limit", mutate: func(value *CredentialsPayload) {
			value.Login = strings.Repeat("я", CredentialsFieldMaxSize)
			value.Password = strings.Repeat("界", CredentialsFieldMaxSize)
			value.URL = strings.Repeat("é", CredentialsFieldMaxSize)
			value.Metadata = strings.Repeat("я", MetadataMaxSize)
		}},
		{name: "empty login", mutate: func(value *CredentialsPayload) { value.Login = "" }, wantErr: ErrInvalidCredentialsPayload},
		{name: "blank login", mutate: func(value *CredentialsPayload) { value.Login = "   " }, wantErr: ErrInvalidCredentialsPayload},
		{name: "empty password", mutate: func(value *CredentialsPayload) { value.Password = "" }, wantErr: ErrInvalidCredentialsPayload},
		{name: "blank password", mutate: func(value *CredentialsPayload) { value.Password = "   " }, wantErr: ErrInvalidCredentialsPayload},
		{name: "login too long", mutate: func(value *CredentialsPayload) {
			value.Login = strings.Repeat("я", CredentialsFieldMaxSize+1)
		}, wantErr: ErrInvalidCredentialsPayload},
		{name: "password too long", mutate: func(value *CredentialsPayload) {
			value.Password = strings.Repeat("界", CredentialsFieldMaxSize+1)
		}, wantErr: ErrInvalidCredentialsPayload},
		{name: "URL too long", mutate: func(value *CredentialsPayload) {
			value.URL = strings.Repeat("é", CredentialsFieldMaxSize+1)
		}, wantErr: ErrInvalidCredentialsPayload},
		{name: "metadata too long", mutate: func(value *CredentialsPayload) {
			value.Metadata = strings.Repeat("я", MetadataMaxSize+1)
		}, wantErr: ErrInvalidCredentialsPayload},
		{name: "control in login", mutate: func(value *CredentialsPayload) { value.Login = "alice\twork" }, wantErr: ErrInvalidCredentialsPayload},
		{name: "control in password", mutate: func(value *CredentialsPayload) { value.Password = "secret\nvalue" }, wantErr: ErrInvalidCredentialsPayload},
		{name: "control in URL", mutate: func(value *CredentialsPayload) { value.URL = "https://example.com\x1b" }, wantErr: ErrInvalidCredentialsPayload},
		{name: "forbidden control in metadata", mutate: func(value *CredentialsPayload) { value.Metadata = "note\ncontinued" }, wantErr: ErrInvalidCredentialsPayload},
		{name: "invalid UTF-8 login", mutate: func(value *CredentialsPayload) { value.Login = string([]byte{0xff}) }, wantErr: ErrInvalidCredentialsPayload},
		{name: "invalid UTF-8 password", mutate: func(value *CredentialsPayload) { value.Password = string([]byte{0xff}) }, wantErr: ErrInvalidCredentialsPayload},
		{name: "invalid UTF-8 URL", mutate: func(value *CredentialsPayload) { value.URL = string([]byte{0xff}) }, wantErr: ErrInvalidCredentialsPayload},
		{name: "invalid UTF-8 metadata", mutate: func(value *CredentialsPayload) { value.Metadata = string([]byte{0xff}) }, wantErr: ErrInvalidCredentialsPayload},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			payload := valid
			if tt.mutate != nil {
				tt.mutate(&payload)
			}
			if err := payload.Validate(); !errors.Is(err, tt.wantErr) {
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
