package config

import (
	"os"
	"path/filepath"
	"strings"
	"testing"
)

func TestDefault(t *testing.T) {
	want := Config{Address: defaultAddress}
	if got := Default(); got != want {
		t.Errorf("Default() = %+v, want %+v", got, want)
	}
}

func TestResolve(t *testing.T) {
	configFile := writeConfigFile(t, `{
  "address": "localhost:8081",
  "ca_cert_file": "file-ca.pem",
  "session_file": "file-session.json"
}`)

	tests := []struct {
		name       string
		configFile string
		overrides  Overrides
		want       Config
	}{
		{
			name: "defaults",
			want: Config{Address: defaultAddress},
		},
		{
			name:       "config file",
			configFile: configFile,
			want: Config{
				Address:     "localhost:8081",
				CACertFile:  "file-ca.pem",
				SessionFile: "file-session.json",
			},
		},
		{
			name:       "overrides > config file",
			configFile: configFile,
			overrides: Overrides{
				Address:     stringPointer("localhost:8082"),
				CACertFile:  stringPointer("override-ca.pem"),
				SessionFile: stringPointer("override-session.json"),
			},
			want: Config{
				Address:     "localhost:8082",
				CACertFile:  "override-ca.pem",
				SessionFile: "override-session.json",
			},
		},
		{
			name:       "empty override clears file value",
			configFile: configFile,
			overrides: Overrides{
				CACertFile: stringPointer(""),
			},
			want: Config{
				Address:     "localhost:8081",
				SessionFile: "file-session.json",
			},
		},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			got, err := Resolve(tt.configFile, tt.overrides)
			if err != nil {
				t.Fatalf("Resolve() error = %v", err)
			}

			if got != tt.want {
				t.Errorf("Resolve() = %+v, want %+v", got, tt.want)
			}
		})
	}
}

func TestResolve_ReturnsConfigFileError(t *testing.T) {
	tests := []struct {
		name       string
		configFile string
		content    string
		overrides  Overrides
		wantError  string
	}{
		{
			name:       "missing explicit file",
			configFile: filepath.Join(t.TempDir(), "missing.json"),
			wantError:  "read client config file",
		},
		{
			name:      "invalid JSON",
			content:   `{`,
			wantError: "decode client config file",
		},
		{
			name:      "unknown field",
			content:   `{"address":"localhost:8080","token":"secret"}`,
			wantError: "decode client config file",
		},
		{
			name:      "multiple JSON values",
			content:   `{} {}`,
			wantError: "multiple JSON values",
		},
		{
			name:      "empty address in file",
			content:   `{"address":""}`,
			wantError: "server address is required",
		},
		{
			name: "empty address override",
			overrides: Overrides{
				Address: stringPointer(""),
			},
			wantError: "server address is required",
		},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			configFile := tt.configFile
			if tt.content != "" {
				configFile = writeConfigFile(t, tt.content)
			}

			_, err := Resolve(configFile, tt.overrides)
			if err == nil {
				t.Fatal("Resolve() error = nil, want config error")
			}

			if !strings.Contains(err.Error(), tt.wantError) {
				t.Fatalf("Resolve() error = %q, want substring %q", err, tt.wantError)
			}
		})
	}
}

func writeConfigFile(t *testing.T, content string) string {
	t.Helper()

	path := filepath.Join(t.TempDir(), "client.json")
	if err := os.WriteFile(path, []byte(content), 0o600); err != nil {
		t.Fatalf("write config file: %v", err)
	}

	return path
}

func stringPointer(value string) *string {
	return &value
}
