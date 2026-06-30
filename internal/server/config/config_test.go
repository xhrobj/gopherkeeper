package config

import (
	"strings"
	"testing"
)

func TestParse(t *testing.T) {
	tests := []struct {
		name string
		env  Config
		args []string
		want Config
	}{
		{
			name: "default address",
			env: Config{
				DatabaseDSN: "postgres://env",
				TLSCertFile: "env-server.pem",
				TLSKeyFile:  "env-server-key.pem",
			},
			want: Config{
				Address:     defaultAddress,
				DatabaseDSN: "postgres://env",
				TLSCertFile: "env-server.pem",
				TLSKeyFile:  "env-server-key.pem",
			},
		},
		{
			name: "environment",
			env: Config{
				Address:     "localhost:8081",
				DatabaseDSN: "postgres://env",
				TLSCertFile: "env-server.pem",
				TLSKeyFile:  "env-server-key.pem",
			},
			want: Config{
				Address:     "localhost:8081",
				DatabaseDSN: "postgres://env",
				TLSCertFile: "env-server.pem",
				TLSKeyFile:  "env-server-key.pem",
			},
		},
		{
			name: "flags",
			args: []string{
				"-a", "localhost:8082",
				"--database-dsn", "postgres://flag",
				"--tls-cert", "flag-server.pem",
				"--tls-key", "flag-server-key.pem",
			},
			want: Config{
				Address:     "localhost:8082",
				DatabaseDSN: "postgres://flag",
				TLSCertFile: "flag-server.pem",
				TLSKeyFile:  "flag-server-key.pem",
			},
		},
		{
			name: "flags > environment",
			env: Config{
				Address:     "localhost:8081",
				DatabaseDSN: "postgres://env",
				TLSCertFile: "env-server.pem",
				TLSKeyFile:  "env-server-key.pem",
			},
			args: []string{
				"-a", "localhost:8082",
				"--database-dsn", "postgres://flag",
				"--tls-cert", "flag-server.pem",
				"--tls-key", "flag-server-key.pem",
			},
			want: Config{
				Address:     "localhost:8082",
				DatabaseDSN: "postgres://flag",
				TLSCertFile: "flag-server.pem",
				TLSKeyFile:  "flag-server-key.pem",
			},
		},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			setEnvironment(t, tt.env)

			cfg, err := Parse(tt.args)
			if err != nil {
				t.Fatalf("Parse() error = %v", err)
			}

			if cfg != tt.want {
				t.Errorf("Parse() = %+v, want %+v", cfg, tt.want)
			}
		})
	}
}

func TestParseReturnsRequiredValueError(t *testing.T) {
	tests := []struct {
		name      string
		args      []string
		wantError string
	}{
		{
			name: "TLS certificate",
			args: []string{
				"--database-dsn", "postgres://test",
				"--tls-key", "server-key.pem",
			},
			wantError: "tls certificate file is required",
		},
		{
			name: "TLS private key",
			args: []string{
				"--database-dsn", "postgres://test",
				"--tls-cert", "server.pem",
			},
			wantError: "tls private key file is required",
		},
		{
			name: "database DSN",
			args: []string{
				"--tls-cert", "server.pem",
				"--tls-key", "server-key.pem",
			},
			wantError: "database DSN is required",
		},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			setEnvironment(t, Config{})

			_, err := Parse(tt.args)
			if err == nil {
				t.Fatal("Parse() error = nil, want required value error")
			}

			if !strings.Contains(err.Error(), tt.wantError) {
				t.Fatalf("Parse() error = %q, want substring %q", err, tt.wantError)
			}
		})
	}
}

func TestParseReturnsFlagError(t *testing.T) {
	setEnvironment(t, Config{})

	_, err := Parse([]string{"--unknown-flag"})
	if err == nil {
		t.Fatal("Parse() error = nil, want flag parsing error")
	}
}

func setEnvironment(t *testing.T, cfg Config) {
	t.Helper()

	t.Setenv("ADDRESS", cfg.Address)
	t.Setenv("DATABASE_DSN", cfg.DatabaseDSN)
	t.Setenv("TLS_CERT_FILE", cfg.TLSCertFile)
	t.Setenv("TLS_KEY_FILE", cfg.TLSKeyFile)
}
