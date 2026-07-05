package service

import (
	"errors"
	"strings"
	"testing"
)

func TestValidateCredentials(t *testing.T) {
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
			password:  testRegistrationPassword,
			wantLogin: "alice",
		},
		{
			name:      "surrounding whitespace",
			login:     "\t\u00a0 Alice \n",
			password:  testRegistrationPassword,
			wantLogin: "alice",
		},
		{
			name:      "allowed login separators",
			login:     "king.of-andals_1st-men",
			password:  testRegistrationPassword,
			wantLogin: "king.of-andals_1st-men",
		},
		{
			name:      "minimum login length",
			login:     "bob",
			password:  testRegistrationPassword,
			wantLogin: "bob",
		},
		{
			name:      "maximum login length",
			login:     strings.Repeat("b", maxLoginLength),
			password:  testRegistrationPassword,
			wantLogin: strings.Repeat("b", maxLoginLength),
		},
		{
			name:      "minimum password length",
			login:     "alice",
			password:  strings.Repeat("z", minPasswordLength),
			wantLogin: "alice",
		},
		{
			name:      "maximum password length",
			login:     "bob",
			password:  strings.Repeat("z", maxPasswordLength),
			wantLogin: "bob",
		},
		{
			name:      "password with boundary printable ASCII symbols",
			login:     "bob",
			password:  "!A0~",
			wantLogin: "bob",
		},
		{
			name:     "login too short",
			login:    "ev",
			password: testRegistrationPassword,
			wantErr:  ErrInvalidLogin,
		},
		{
			name:     "login too long",
			login:    strings.Repeat("e", maxLoginLength+1),
			password: testRegistrationPassword,
			wantErr:  ErrInvalidLogin,
		},
		{
			name:     "login starts with dot",
			login:    ".eve",
			password: testRegistrationPassword,
			wantErr:  ErrInvalidLogin,
		},
		{
			name:     "login starts with underscore",
			login:    "_eve",
			password: testRegistrationPassword,
			wantErr:  ErrInvalidLogin,
		},
		{
			name:     "login starts with hyphen",
			login:    "-eve",
			password: testRegistrationPassword,
			wantErr:  ErrInvalidLogin,
		},
		{
			name:     "login contains space",
			login:    "e ve",
			password: testRegistrationPassword,
			wantErr:  ErrInvalidLogin,
		},
		{
			name:     "login contains unsupported character",
			login:    "eve@example",
			password: testRegistrationPassword,
			wantErr:  ErrInvalidLogin,
		},
		{
			name:     "login contains Cyrillic characters",
			login:    "ева",
			password: testRegistrationPassword,
			wantErr:  ErrInvalidLogin,
		},
		{
			name:     "Cyrillic C disguised as ASCII C",
			login:    "Сoder",
			password: testRegistrationPassword,
			wantErr:  ErrInvalidLogin,
		},
		{
			name:     "Kelvin sign must not become ASCII K",
			login:    "Kirill",
			password: testRegistrationPassword,
			wantErr:  ErrInvalidLogin,
		},
		{
			name:     "password too short",
			login:    "eve",
			password: strings.Repeat("a", minPasswordLength-1),
			wantErr:  ErrPasswordTooShort,
		},
		{
			name:     "password too long",
			login:    "eve",
			password: strings.Repeat("z", maxPasswordLength+1),
			wantErr:  ErrPasswordTooLong,
		},
		{
			name:     "password contains internal space",
			login:    "eve",
			password: "forbidden fruit",
			wantErr:  ErrInvalidPassword,
		},
		{
			name:     "password contains leading space",
			login:    "eve",
			password: " forbidden-fruit",
			wantErr:  ErrInvalidPassword,
		},
		{
			name:     "password contains trailing space",
			login:    "eve",
			password: "forbidden-fruit ",
			wantErr:  ErrInvalidPassword,
		},
		{
			name:     "password contains tab",
			login:    "eve",
			password: "forbidden\tfruit",
			wantErr:  ErrInvalidPassword,
		},
		{
			name:     "password contains newline",
			login:    "eve",
			password: "forbidden\nfruit",
			wantErr:  ErrInvalidPassword,
		},
		{
			name:     "password contains non ASCII characters",
			login:    "eve",
			password: "запретный-плод",
			wantErr:  ErrInvalidPassword,
		},
		{
			name:     "password contains emoji",
			login:    "eve",
			password: "forbidden🍎fruit",
			wantErr:  ErrInvalidPassword,
		},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			gotLogin, err := validateCredentials(tt.login, tt.password)
			if !errors.Is(err, tt.wantErr) {
				t.Fatalf(
					"validateCredentials() error = %v, want %v",
					err,
					tt.wantErr,
				)
			}

			if gotLogin != tt.wantLogin {
				t.Errorf(
					"validateCredentials() login = %q, want %q",
					gotLogin,
					tt.wantLogin,
				)
			}
		})
	}
}

func TestValidateCredentials_DoesNotExposePassword(t *testing.T) {
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
			password: "forbidden fruit",
		},
		{
			name:     "password contains Unicode",
			password: "запретный-плод",
		},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			_, err := validateCredentials("eve", tt.password)
			if err == nil {
				t.Fatal("validateCredentials() error = nil")
			}

			if strings.Contains(err.Error(), tt.password) {
				t.Errorf("validation error contains password: %q", err)
			}
		})
	}
}
