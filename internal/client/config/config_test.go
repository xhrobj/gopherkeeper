package config

import (
	"os"
	"path/filepath"
	"strings"
	"testing"
)

func TestParse(t *testing.T) {
	configFile := writeConfigFile(t, `{
  "address": "localhost:8081",
  "ca_cert_file": "file-ca.pem",
  "session_file": "file-session.json"
}`)

	tests := []struct {
		name           string
		envAddress     string
		envCACert      string
		envSessionFile string
		envConfig      string
		args           []string
		want           Config
	}{
		{
			name: "defaults",
			want: Config{
				Address: defaultAddress,
			},
		},
		{
			name:      "config file",
			envConfig: configFile,
			want: Config{
				Address:     "localhost:8081",
				CACertFile:  "file-ca.pem",
				SessionFile: "file-session.json",
			},
		},
		{
			name:           "environment > config file",
			envConfig:      configFile,
			envAddress:     "localhost:8082",
			envCACert:      "env-ca.pem",
			envSessionFile: "env-session.json",
			want: Config{
				Address:     "localhost:8082",
				CACertFile:  "env-ca.pem",
				SessionFile: "env-session.json",
			},
		},
		{
			name:           "flags > environment",
			envConfig:      configFile,
			envAddress:     "localhost:8082",
			envCACert:      "env-ca.pem",
			envSessionFile: "env-session.json",
			args: []string{
				"health",
				"-a", "localhost:8083",
				"--ca-cert", "flag-ca.pem",
				"--session-file", "flag-session.json",
			},
			want: Config{
				Address:     "localhost:8083",
				CACertFile:  "flag-ca.pem",
				SessionFile: "flag-session.json",
			},
		},
		{
			name:      "config flag > CONFIG environment",
			envConfig: writeConfigFile(t, `{"address":"env-file:8080"}`),
			args:      []string{"health", "--config", configFile},
			want: Config{
				Address:     "localhost:8081",
				CACertFile:  "file-ca.pem",
				SessionFile: "file-session.json",
			},
		},
		{
			name: "inline flag values",
			args: []string{
				"health",
				"--config=" + configFile,
				"--address=localhost:8084",
				"--ca-cert=inline-ca.pem",
				"--session-file=inline-session.json",
			},
			want: Config{
				Address:     "localhost:8084",
				CACertFile:  "inline-ca.pem",
				SessionFile: "inline-session.json",
			},
		},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			isolateConfigEnvironment(t)
			t.Setenv("ADDRESS", tt.envAddress)
			t.Setenv("CA_CERT_FILE", tt.envCACert)
			t.Setenv("SESSION_FILE", tt.envSessionFile)
			t.Setenv("CONFIG", tt.envConfig)

			got, err := Parse(tt.args)
			if err != nil {
				t.Fatalf("Parse() error = %v", err)
			}

			if got != tt.want {
				t.Errorf("Parse() = %+v, want %+v", got, tt.want)
			}
		})
	}
}

func TestParse_IgnoresImplicitUserConfigFile(t *testing.T) {
	home := t.TempDir()
	t.Setenv("HOME", home)
	t.Setenv("XDG_CONFIG_HOME", filepath.Join(home, ".config"))
	t.Setenv("APPDATA", filepath.Join(home, "AppData", "Roaming"))
	t.Setenv("CONFIG", "")

	implicitConfigPath := filepath.Join(home, ".config", "gopherkeeper", "client.json")
	if err := os.MkdirAll(filepath.Dir(implicitConfigPath), 0o700); err != nil {
		t.Fatalf("mkdir implicit config directory: %v", err)
	}
	if err := os.WriteFile(implicitConfigPath, []byte(`{"address":"localhost:8085"}`), 0o600); err != nil {
		t.Fatalf("write implicit config file: %v", err)
	}

	got, err := Parse(nil)
	if err != nil {
		t.Fatalf("Parse() error = %v", err)
	}

	want := Config{Address: defaultAddress}
	if got != want {
		t.Errorf("Parse() = %+v, want %+v", got, want)
	}
}

func TestParse_ReturnsConfigFileError(t *testing.T) {
	tests := []struct {
		name      string
		content   string
		args      []string
		wantError string
	}{
		{
			name:      "missing explicit file",
			args:      []string{"health", "--config", filepath.Join(t.TempDir(), "missing.json")},
			wantError: "read client config file",
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
			name:      "empty address",
			content:   `{"address":""}`,
			wantError: "server address is required",
		},
		{
			name:      "flag without value",
			args:      []string{"health", "--config"},
			wantError: "requires a value",
		},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			isolateConfigEnvironment(t)

			args := tt.args
			if tt.content != "" {
				args = []string{"health", "--config", writeConfigFile(t, tt.content)}
			}

			_, err := Parse(args)
			if err == nil {
				t.Fatal("Parse() error = nil, want config error")
			}

			if !strings.Contains(err.Error(), tt.wantError) {
				t.Fatalf("Parse() error = %q, want substring %q", err, tt.wantError)
			}
		})
	}
}

func TestLoad(t *testing.T) {
	isolateConfigEnvironment(t)
	t.Setenv("ADDRESS", "localhost:8081")
	t.Setenv("CA_CERT_FILE", "env-ca.pem")
	t.Setenv("SESSION_FILE", "env-session.json")
	t.Setenv("CONFIG", "")

	want := Config{
		Address:     "localhost:8081",
		CACertFile:  "env-ca.pem",
		SessionFile: "env-session.json",
	}
	if got := Load(); got != want {
		t.Errorf("Load() = %+v, want %+v", got, want)
	}
}

func isolateConfigEnvironment(t *testing.T) {
	t.Helper()

	home := t.TempDir()
	t.Setenv("HOME", home)
	t.Setenv("XDG_CONFIG_HOME", filepath.Join(home, ".config"))
	t.Setenv("APPDATA", filepath.Join(home, "AppData", "Roaming"))
	t.Setenv("CONFIG", "")
}

func writeConfigFile(t *testing.T, content string) string {
	t.Helper()

	path := filepath.Join(t.TempDir(), "client.json")
	if err := os.WriteFile(path, []byte(content), 0o600); err != nil {
		t.Fatalf("write config file: %v", err)
	}

	return path
}
