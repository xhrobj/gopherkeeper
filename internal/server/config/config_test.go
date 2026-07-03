package config

import (
	"encoding/base64"
	"reflect"
	"strings"
	"testing"
	"time"
)

var testJWTSecret = []byte("0123456789abcdef0123456789abcdef")

func TestParse(t *testing.T) {
	tests := []struct {
		name string
		env  Config
		args []string
		want Config
	}{
		{
			name: "default address and JWT TTL",
			env: Config{
				DatabaseDSN: "postgres://env",
				TLSCertFile: "env-server.pem",
				TLSKeyFile:  "env-server-key.pem",
				JWTSecret:   testJWTSecret,
			},
			want: Config{
				Address:     defaultAddress,
				DatabaseDSN: "postgres://env",
				TLSCertFile: "env-server.pem",
				TLSKeyFile:  "env-server-key.pem",
				JWTSecret:   testJWTSecret,
				JWTTTL:      defaultJWTTTL,
			},
		},
		{
			name: "environment",
			env: Config{
				Address:     "localhost:8081",
				DatabaseDSN: "postgres://env",
				TLSCertFile: "env-server.pem",
				TLSKeyFile:  "env-server-key.pem",
				JWTSecret:   testJWTSecret,
				JWTTTL:      30 * time.Minute,
			},
			want: Config{
				Address:     "localhost:8081",
				DatabaseDSN: "postgres://env",
				TLSCertFile: "env-server.pem",
				TLSKeyFile:  "env-server-key.pem",
				JWTSecret:   testJWTSecret,
				JWTTTL:      30 * time.Minute,
			},
		},
		{
			name: "flags",
			env: Config{
				JWTSecret: testJWTSecret,
			},
			args: []string{
				"-a", "localhost:8082",
				"--database-dsn", "postgres://flag",
				"--tls-cert", "flag-server.pem",
				"--tls-key", "flag-server-key.pem",
				"--jwt-ttl", "45m",
			},
			want: Config{
				Address:     "localhost:8082",
				DatabaseDSN: "postgres://flag",
				TLSCertFile: "flag-server.pem",
				TLSKeyFile:  "flag-server-key.pem",
				JWTSecret:   testJWTSecret,
				JWTTTL:      45 * time.Minute,
			},
		},
		{
			name: "flags > environment",
			env: Config{
				Address:     "localhost:8081",
				DatabaseDSN: "postgres://env",
				TLSCertFile: "env-server.pem",
				TLSKeyFile:  "env-server-key.pem",
				JWTSecret:   testJWTSecret,
				JWTTTL:      30 * time.Minute,
			},
			args: []string{
				"-a", "localhost:8082",
				"--database-dsn", "postgres://flag",
				"--tls-cert", "flag-server.pem",
				"--tls-key", "flag-server-key.pem",
				"--jwt-ttl", "45m",
			},
			want: Config{
				Address:     "localhost:8082",
				DatabaseDSN: "postgres://flag",
				TLSCertFile: "flag-server.pem",
				TLSKeyFile:  "flag-server-key.pem",
				JWTSecret:   testJWTSecret,
				JWTTTL:      45 * time.Minute,
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

			if !reflect.DeepEqual(cfg, tt.want) {
				t.Errorf("Parse() = %+v, want %+v", cfg, tt.want)
			}
		})
	}
}

func TestParse_ReturnsRequiredValueError(t *testing.T) {
	tests := []struct {
		name      string
		env       Config
		args      []string
		wantError string
	}{
		{
			name: "TLS certificate",
			env: Config{
				JWTSecret: testJWTSecret,
			},
			args: []string{
				"--database-dsn", "postgres://test",
				"--tls-key", "server-key.pem",
			},
			wantError: "tls certificate file is required",
		},
		{
			name: "TLS private key",
			env: Config{
				JWTSecret: testJWTSecret,
			},
			args: []string{
				"--database-dsn", "postgres://test",
				"--tls-cert", "server.pem",
			},
			wantError: "tls private key file is required",
		},
		{
			name: "database DSN",
			env: Config{
				JWTSecret: testJWTSecret,
			},
			args: []string{
				"--tls-cert", "server.pem",
				"--tls-key", "server-key.pem",
			},
			wantError: "database DSN is required",
		},
		{
			name: "JWT secret",
			args: []string{
				"--database-dsn", "postgres://test",
				"--tls-cert", "server.pem",
				"--tls-key", "server-key.pem",
			},
			wantError: "JWT secret is required",
		},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			setEnvironment(t, tt.env)

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

func TestParse_ReturnsInvalidJWTSecretError(t *testing.T) {
	tests := []struct {
		name      string
		secret    string
		wantError string
	}{
		{
			name:      "invalid base64",
			secret:    "not-base64",
			wantError: "decode JWT secret",
		},
		{
			name:      "wrong size",
			secret:    base64.StdEncoding.EncodeToString([]byte("too-short")),
			wantError: "JWT secret must decode to 32 bytes",
		},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			setEnvironment(t, Config{})
			t.Setenv("JWT_SECRET", tt.secret)

			_, err := Parse([]string{
				"--database-dsn", "postgres://test",
				"--tls-cert", "server.pem",
				"--tls-key", "server-key.pem",
			})
			if err == nil {
				t.Fatal("Parse() error = nil, want JWT secret error")
			}

			if !strings.Contains(err.Error(), tt.wantError) {
				t.Fatalf("Parse() error = %q, want substring %q", err, tt.wantError)
			}
		})
	}
}

func TestParse_ReturnsInvalidJWTTTLError(t *testing.T) {
	tests := []struct {
		name      string
		envTTL    string
		args      []string
		wantError string
	}{
		{
			name:      "environment format",
			envTTL:    "fifteen minutes",
			wantError: "parse JWT TTL",
		},
		{
			name:      "environment zero",
			envTTL:    "0s",
			wantError: "JWT TTL must be positive",
		},
		{
			name: "flag zero",
			args: []string{
				"--jwt-ttl", "0s",
			},
			wantError: "JWT TTL must be positive",
		},
		{
			name: "flag negative",
			args: []string{
				"--jwt-ttl", "-1s",
			},
			wantError: "JWT TTL must be positive",
		},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			setEnvironment(t, Config{
				DatabaseDSN: "postgres://test",
				TLSCertFile: "server.pem",
				TLSKeyFile:  "server-key.pem",
				JWTSecret:   testJWTSecret,
			})
			t.Setenv("JWT_TTL", tt.envTTL)

			_, err := Parse(tt.args)
			if err == nil {
				t.Fatal("Parse() error = nil, want JWT TTL error")
			}

			if !strings.Contains(err.Error(), tt.wantError) {
				t.Fatalf("Parse() error = %q, want substring %q", err, tt.wantError)
			}
		})
	}
}

func TestParse_ReturnsFlagError(t *testing.T) {
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

	jwtSecret := ""
	if len(cfg.JWTSecret) > 0 {
		jwtSecret = base64.StdEncoding.EncodeToString(cfg.JWTSecret)
	}
	t.Setenv("JWT_SECRET", jwtSecret)

	jwtTTL := ""
	if cfg.JWTTTL > 0 {
		jwtTTL = cfg.JWTTTL.String()
	}
	t.Setenv("JWT_TTL", jwtTTL)
}
