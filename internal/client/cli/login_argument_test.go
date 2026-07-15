package cli

import (
	"errors"
	"strings"
	"testing"

	"github.com/xhrobj/gopherkeeper/internal/model"
)

func TestCanonicalizeLoginArgument(t *testing.T) {
	tests := []struct {
		name    string
		login   string
		want    string
		wantErr error
	}{
		{name: "canonical login", login: "alice", want: "alice"},
		{name: "trimmed login", login: " Alice ", want: "alice"},
		{name: "allowed separators", login: "Alice.Dev-4_2", want: "alice.dev-4_2"},
		{name: "empty login", login: "", wantErr: errInvalidLoginArgument},
		{name: "blank login", login: "  \t  ", wantErr: errInvalidLoginArgument},
		{name: "too short", login: "ev", wantErr: errInvalidLoginArgument},
		{name: "too long", login: strings.Repeat("e", model.MaxLoginLength+1), wantErr: errInvalidLoginArgument},
		{name: "starts with hyphen", login: "-eve", wantErr: errInvalidLoginArgument},
		{name: "contains space", login: "e ve", wantErr: errInvalidLoginArgument},
		{name: "contains unsupported character", login: "eve@example", wantErr: errInvalidLoginArgument},
		{name: "contains Cyrillic characters", login: "ева", wantErr: errInvalidLoginArgument},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			got, err := canonicalizeLoginArgument(tt.login)
			if !errors.Is(err, tt.wantErr) {
				t.Fatalf("canonicalizeLoginArgument() error = %v, want %v", err, tt.wantErr)
			}
			if got != tt.want {
				t.Errorf("canonicalizeLoginArgument() = %q, want %q", got, tt.want)
			}
		})
	}
}
