package service

import (
	"errors"
	"strings"
	"testing"
)

func TestValidateRegistrationCredentials(t *testing.T) {
	const validPassword = "valid-password"

	tests := []struct {
		name      string
		login     string
		password  string
		wantLogin string
		wantErr   error
	}{
		{
			name:      "canonical login",
			login:     " Alice ",
			password:  validPassword,
			wantLogin: "alice",
		},
		{
			name:      "allowed login separators",
			login:     "king.of-andals_1st-men",
			password:  validPassword,
			wantLogin: "king.of-andals_1st-men",
		},
		{
			name:      "minimum login length",
			login:     "a-1",
			password:  validPassword,
			wantLogin: "a-1",
		},
		{
			name:      "maximum login length",
			login:     strings.Repeat("a", maxLoginLength),
			password:  validPassword,
			wantLogin: strings.Repeat("a", maxLoginLength),
		},
		{
			name:      "minimum password length",
			login:     "alice",
			password:  strings.Repeat("z", minPasswordLength),
			wantLogin: "alice",
		},
		{
			name:      "maximum password length",
			login:     "alice",
			password:  strings.Repeat("z", maxPasswordLength),
			wantLogin: "alice",
		},
		{
			name:      "password with boundary printable ASCII symbols",
			login:     "alice",
			password:  "!A0~",
			wantLogin: "alice",
		},
		{
			name:     "login too short",
			login:    "ab",
			password: validPassword,
			wantErr:  ErrInvalidLogin,
		},
		{
			name:     "login too long",
			login:    strings.Repeat("a", maxLoginLength+1),
			password: validPassword,
			wantErr:  ErrInvalidLogin,
		},
		{
			name:     "login starts with dot",
			login:    ".alice",
			password: validPassword,
			wantErr:  ErrInvalidLogin,
		},
		{
			name:     "login starts with underscore",
			login:    "_alice",
			password: validPassword,
			wantErr:  ErrInvalidLogin,
		},
		{
			name:     "login starts with hyphen",
			login:    "-alice",
			password: validPassword,
			wantErr:  ErrInvalidLogin,
		},
		{
			name:     "login contains space",
			login:    "ali ce",
			password: validPassword,
			wantErr:  ErrInvalidLogin,
		},
		{
			name:     "login contains unsupported character",
			login:    "alice@example",
			password: validPassword,
			wantErr:  ErrInvalidLogin,
		},
		{
			name:     "login contains Cyrillic characters",
			login:    "алиса",
			password: validPassword,
			wantErr:  ErrInvalidLogin,
		},
		{
			name:     "Cyrillic C disguised as ASCII C",
			login:    "Сoder",
			password: validPassword,
			wantErr:  ErrInvalidLogin,
		},
		{
			name:     "Kelvin sign must not become ASCII K",
			login:    "Kirill",
			password: validPassword,
			wantErr:  ErrInvalidLogin,
		},
		{
			name:     "password too short",
			login:    "alice",
			password: strings.Repeat("a", minPasswordLength-1),
			wantErr:  ErrPasswordTooShort,
		},
		{
			name:     "password too long",
			login:    "alice",
			password: strings.Repeat("z", maxPasswordLength+1),
			wantErr:  ErrPasswordTooLong,
		},
		{
			name:     "password contains internal space",
			login:    "alice",
			password: "god love sex",
			wantErr:  ErrInvalidPassword,
		},
		{
			name:     "password contains leading space",
			login:    "alice",
			password: " valid-password",
			wantErr:  ErrInvalidPassword,
		},
		{
			name:     "password contains trailing space",
			login:    "alice",
			password: "valid-password ",
			wantErr:  ErrInvalidPassword,
		},
		{
			name:     "password contains tab",
			login:    "alice",
			password: "valid\tpassword",
			wantErr:  ErrInvalidPassword,
		},
		{
			name:     "password contains newline",
			login:    "alice",
			password: "valid\npassword",
			wantErr:  ErrInvalidPassword,
		},
		{
			name:     "password contains non ASCII characters",
			login:    "alice",
			password: "пароль",
			wantErr:  ErrInvalidPassword,
		},
		{
			name:     "password contains emoji",
			login:    "alice",
			password: "valid🔐password",
			wantErr:  ErrInvalidPassword,
		},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			gotLogin, err := validateRegistrationCredentials(tt.login, tt.password)
			if !errors.Is(err, tt.wantErr) {
				t.Fatalf(
					"validateRegistrationCredentials() error = %v, want %v",
					err,
					tt.wantErr,
				)
			}

			if gotLogin != tt.wantLogin {
				t.Errorf(
					"validateRegistrationCredentials() login = %q, want %q",
					gotLogin,
					tt.wantLogin,
				)
			}
		})
	}
}

func TestValidateRegistrationCredentials_DoesNotExposePassword(t *testing.T) {
	tests := []struct {
		name     string
		password string
	}{
		{
			name:     "password too short",
			password: strings.Repeat("a", minPasswordLength-1),
		},
		{
			name:     "password too long",
			password: strings.Repeat("a", maxPasswordLength+1),
		},
		{
			name:     "password contains space",
			password: "god love sex",
		},
		{
			name:     "password contains Unicode",
			password: "суперпароль",
		},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			_, err := validateRegistrationCredentials("alice", tt.password)
			if err == nil {
				t.Fatal("validateRegistrationCredentials() error = nil")
			}

			if strings.Contains(err.Error(), tt.password) {
				t.Errorf("validation error contains password: %q", err)
			}
		})
	}
}
