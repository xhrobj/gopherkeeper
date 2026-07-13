package cli

import (
	"testing"

	urfavecli "github.com/urfave/cli/v3"
	"github.com/xhrobj/gopherkeeper/internal/client/config"
)

func TestConfigFromCommand(t *testing.T) {
	want := config.Config{Address: "localhost:8080"}

	tests := []struct {
		name      string
		metadata  map[string]any
		want      config.Config
		wantError string
	}{
		{
			name: "valid config",
			metadata: map[string]any{
				clientConfigMetadataKey: want,
			},
			want: want,
		},
		{
			name:      "missing config",
			metadata:  map[string]any{},
			wantError: "client config is missing",
		},
		{
			name: "unexpected config type",
			metadata: map[string]any{
				clientConfigMetadataKey: "invalid",
			},
			wantError: "client config has unexpected type",
		},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			command := &urfavecli.Command{Metadata: tt.metadata}

			got, err := configFromCommand(command)
			if tt.wantError != "" {
				if err == nil {
					t.Fatalf("configFromCommand() error = nil, want %q", tt.wantError)
				}
				if err.Error() != tt.wantError {
					t.Fatalf("configFromCommand() error = %q, want %q", err, tt.wantError)
				}
				return
			}

			if err != nil {
				t.Fatalf("configFromCommand() error = %v", err)
			}
			if got != tt.want {
				t.Errorf("configFromCommand() = %+v, want %+v", got, tt.want)
			}
		})
	}
}
