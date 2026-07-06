package cli

import (
	"errors"
	"strings"
	"testing"
)

func TestValidateLoginArgument(t *testing.T) {
	tests := []struct {
		name    string
		login   string
		wantErr error
	}{
		{
			name:  "canonical login",
			login: "alice",
		},
		{
			name:  "trimmed login",
			login: " Alice ",
		},
		{
			name:  "allowed separators",
			login: "alice.dev-4_2",
		},
		{
			name:    "empty login",
			login:   "",
			wantErr: errInvalidLoginArgument,
		},
		{
			name:    "blank login",
			login:   "  \t  ",
			wantErr: errInvalidLoginArgument,
		},
		{
			name:    "too short",
			login:   "ev",
			wantErr: errInvalidLoginArgument,
		},
		{
			name:    "too long",
			login:   strings.Repeat("e", maxLoginLength+1),
			wantErr: errInvalidLoginArgument,
		},
		{
			name:    "starts with hyphen",
			login:   "-eve",
			wantErr: errInvalidLoginArgument,
		},
		{
			name:    "contains space",
			login:   "e ve",
			wantErr: errInvalidLoginArgument,
		},
		{
			name:    "contains unsupported character",
			login:   "eve@example",
			wantErr: errInvalidLoginArgument,
		},
		{
			name:    "contains Cyrillic characters",
			login:   "ева",
			wantErr: errInvalidLoginArgument,
		},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			err := validateLoginArgument(tt.login)
			if !errors.Is(err, tt.wantErr) {
				t.Fatalf("validateLoginArgument() error = %v, want %v", err, tt.wantErr)
			}
		})
	}
}
