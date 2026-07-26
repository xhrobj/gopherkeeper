package config

import (
	"encoding/json"
	"os"
	"path/filepath"
	"strings"
	"testing"
)

func TestDefault(t *testing.T) {
	want := Config{
		Transport:   TransportHTTPS,
		Address:     defaultAddress,
		GRPCAddress: defaultGRPCAddress,
	}
	if got := Default(); got != want {
		t.Errorf("Default() = %+v, want %+v", got, want)
	}
}

func TestResolve(t *testing.T) {
	configFile := writeConfigFile(t, `{
  "transport": "grpc",
  "address": "localhost:8081",
  "grpc_address": "localhost:50052",
  "ca_cert_file": "file-ca.pem",
  "session_dir": "file-session",
  "cache_dir": "file-cache"
}`)

	tests := []struct {
		name       string
		configFile string
		overrides  Overrides
		want       Config
	}{
		{
			name: "defaults",
			want: Config{
				Transport:   TransportHTTPS,
				Address:     defaultAddress,
				GRPCAddress: defaultGRPCAddress,
			},
		},
		{
			name:       "config file",
			configFile: configFile,
			want: Config{
				Transport:   TransportGRPC,
				Address:     "localhost:8081",
				GRPCAddress: "localhost:50052",
				CACertFile:  "file-ca.pem",
				SessionDir:  "file-session",
				CacheDir:    "file-cache",
			},
		},
		{
			name:       "overrides > config file",
			configFile: configFile,
			overrides: Overrides{
				Transport:   transportPointer(TransportHTTPS),
				Address:     stringPointer("localhost:8082"),
				GRPCAddress: stringPointer("localhost:50053"),
				CACertFile:  stringPointer("override-ca.pem"),
				SessionDir:  stringPointer("override-session"),
				CacheDir:    stringPointer("override-cache"),
			},
			want: Config{
				Transport:   TransportHTTPS,
				Address:     "localhost:8082",
				GRPCAddress: "localhost:50053",
				CACertFile:  "override-ca.pem",
				SessionDir:  "override-session",
				CacheDir:    "override-cache",
			},
		},
		{
			name:       "empty override clears inactive values",
			configFile: configFile,
			overrides: Overrides{
				Transport:  transportPointer(TransportHTTPS),
				CACertFile: stringPointer(""),
				CacheDir:   stringPointer(""),
			},
			want: Config{
				Transport:   TransportHTTPS,
				Address:     "localhost:8081",
				GRPCAddress: "localhost:50052",
				SessionDir:  "file-session",
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

func TestResolveNormalizesNetworkAddressesOnly(t *testing.T) {
	configFile := writeConfigFile(t, `{
  "transport": "grpc",
  "address": "  vault.example:8443  ",
  "grpc_address": "  vault.example:9443  ",
  "ca_cert_file": " certs/ca file.pem "
}`)

	got, err := Resolve(configFile, Overrides{
		GRPCAddress: stringPointer("  override.example:9555  "),
	})
	if err != nil {
		t.Fatalf("Resolve() error = %v", err)
	}

	if got.Address != "vault.example:8443" {
		t.Fatalf("HTTPS address = %q, want trimmed address", got.Address)
	}
	if got.GRPCAddress != "override.example:9555" {
		t.Fatalf("gRPC address = %q, want trimmed override", got.GRPCAddress)
	}
	if got.CACertFile != " certs/ca file.pem " {
		t.Fatalf("CA certificate path = %q, want spaces preserved", got.CACertFile)
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
			name:      "unknown transport",
			content:   `{"transport":"quic"}`,
			wantError: "unsupported client transport",
		},
		{
			name:      "empty HTTPS address",
			content:   `{"transport":"https","address":""}`,
			wantError: "HTTPS server address is required",
		},
		{
			name:      "empty gRPC address",
			content:   `{"transport":"grpc","grpc_address":""}`,
			wantError: "gRPC server address is required",
		},
		{
			name: "empty HTTPS address override",
			overrides: Overrides{
				Address: stringPointer(""),
			},
			wantError: "HTTPS server address is required",
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

func TestActiveAddress(t *testing.T) {
	cfg := Config{
		Transport:   TransportGRPC,
		Address:     "localhost:8080",
		GRPCAddress: "localhost:50051",
	}
	if got := cfg.ActiveAddress(); got != "localhost:50051" {
		t.Fatalf("ActiveAddress() = %q", got)
	}

	cfg.Transport = TransportHTTPS
	if got := cfg.ActiveAddress(); got != "localhost:8080" {
		t.Fatalf("ActiveAddress() = %q", got)
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

func transportPointer(value Transport) *Transport {
	return &value
}

func TestSaveWritesClientConfig(t *testing.T) {
	path := filepath.Join(t.TempDir(), "client.json")
	want := Config{
		Transport:   TransportGRPC,
		Address:     "vault.example:9443",
		GRPCAddress: "vault.example:9444",
		CACertFile:  "certs/ca.pem",
		SessionDir:  "session",
		CacheDir:    "cache",
	}

	if err := Save(path, want); err != nil {
		t.Fatalf("Save() error = %v", err)
	}

	got, err := Resolve(path, Overrides{})
	if err != nil {
		t.Fatalf("Resolve() saved config error = %v", err)
	}
	if got != want {
		t.Fatalf("saved config = %#v, want %#v", got, want)
	}

	data, err := os.ReadFile(path)
	if err != nil {
		t.Fatalf("read saved config: %v", err)
	}
	if !strings.HasSuffix(string(data), "\n") {
		t.Fatal("saved config does not end with newline")
	}
}

func TestSaveNormalizesNetworkAddressesOnly(t *testing.T) {
	path := filepath.Join(t.TempDir(), "client.json")
	cfg := Config{
		Transport:   TransportGRPC,
		Address:     "  vault.example:8443  ",
		GRPCAddress: "  vault.example:9443  ",
		CACertFile:  " certs/ca file.pem ",
	}

	if err := Save(path, cfg); err != nil {
		t.Fatalf("Save() error = %v", err)
	}

	data, err := os.ReadFile(path)
	if err != nil {
		t.Fatalf("read saved config: %v", err)
	}

	var saved struct {
		Address     string `json:"address"`
		GRPCAddress string `json:"grpc_address"`
		CACertFile  string `json:"ca_cert_file"`
	}
	if err := json.Unmarshal(data, &saved); err != nil {
		t.Fatalf("decode saved config: %v", err)
	}

	if saved.Address != "vault.example:8443" {
		t.Fatalf("saved HTTPS address = %q, want trimmed address", saved.Address)
	}
	if saved.GRPCAddress != "vault.example:9443" {
		t.Fatalf("saved gRPC address = %q, want trimmed address", saved.GRPCAddress)
	}
	if saved.CACertFile != " certs/ca file.pem " {
		t.Fatalf("saved CA certificate path = %q, want spaces preserved", saved.CACertFile)
	}
}

func TestSaveRejectsMissingPathTransportAndSelectedAddress(t *testing.T) {
	if err := Save("", Default()); err == nil {
		t.Fatal("Save() path error = nil")
	}
	if err := Save(filepath.Join(t.TempDir(), "client.json"), Config{}); err == nil {
		t.Fatal("Save() transport error = nil")
	}
	if err := Save(filepath.Join(t.TempDir(), "client.json"), Config{
		Transport: TransportHTTPS,
	}); err == nil {
		t.Fatal("Save() HTTPS address error = nil")
	}
	if err := Save(filepath.Join(t.TempDir(), "client.json"), Config{
		Transport: TransportGRPC,
	}); err == nil {
		t.Fatal("Save() gRPC address error = nil")
	}
}
