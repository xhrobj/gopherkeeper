package model

import (
	"errors"
	"strings"
	"testing"
)

func TestCanonicalizeLogin(t *testing.T) {
	tests := []struct {
		name    string
		login   string
		want    string
		wantErr error
	}{
		{name: "canonical login", login: "alice", want: "alice"},
		{name: "surrounding whitespace", login: "\t\u00a0 Alice \n", want: "alice"},
		{name: "allowed separators", login: "King.Of-Andals_1st-Men", want: "king.of-andals_1st-men"},
		{name: "minimum length", login: "Bob", want: "bob"},
		{name: "maximum length", login: strings.Repeat("A", MaxLoginLength), want: strings.Repeat("a", MaxLoginLength)},
		{name: "too short", login: "ev", wantErr: ErrInvalidLogin},
		{name: "too long", login: strings.Repeat("e", MaxLoginLength+1), wantErr: ErrInvalidLogin},
		{name: "invalid first character", login: ".eve", wantErr: ErrInvalidLogin},
		{name: "internal space", login: "e ve", wantErr: ErrInvalidLogin},
		{name: "unsupported character", login: "eve@example", wantErr: ErrInvalidLogin},
		{name: "Cyrillic", login: "ева", wantErr: ErrInvalidLogin},
		{name: "Cyrillic lookalike", login: "Сoder", wantErr: ErrInvalidLogin},
		{name: "Kelvin sign", login: "Kirill", wantErr: ErrInvalidLogin},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			got, err := CanonicalizeLogin(tt.login)
			if !errors.Is(err, tt.wantErr) {
				t.Fatalf("CanonicalizeLogin() error = %v, want %v", err, tt.wantErr)
			}
			if got != tt.want {
				t.Errorf("CanonicalizeLogin() = %q, want %q", got, tt.want)
			}
		})
	}
}

func TestValidateCanonicalLogin(t *testing.T) {
	tests := []struct {
		name    string
		login   string
		wantErr error
	}{
		{name: "canonical login", login: "alice.dev-4_2"},
		{name: "uppercase", login: "Alice", wantErr: ErrInvalidLogin},
		{name: "surrounding whitespace", login: " alice ", wantErr: ErrInvalidLogin},
		{name: "invalid first character", login: "_alice", wantErr: ErrInvalidLogin},
		{name: "Unicode", login: "алиса", wantErr: ErrInvalidLogin},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			if err := ValidateCanonicalLogin(tt.login); !errors.Is(err, tt.wantErr) {
				t.Fatalf("ValidateCanonicalLogin() error = %v, want %v", err, tt.wantErr)
			}
		})
	}
}
