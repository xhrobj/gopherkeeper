package config

import "testing"

func TestParse(t *testing.T) {
	tests := []struct {
		name       string
		envAddress string
		envCACert  string
		args       []string
		want       Config
	}{
		{
			name: "defaults",
			want: Config{
				Address: defaultAddress,
			},
		},
		{
			name:       "environment",
			envAddress: "localhost:8081",
			envCACert:  "env-ca.pem",
			want: Config{
				Address:    "localhost:8081",
				CACertFile: "env-ca.pem",
			},
		},
		{
			name: "flags",
			args: []string{
				"-a", "localhost:8082",
				"--ca-cert", "flag-ca.pem",
			},
			want: Config{
				Address:    "localhost:8082",
				CACertFile: "flag-ca.pem",
			},
		},
		{
			name:       "flags > environment",
			envAddress: "localhost:8081",
			envCACert:  "env-ca.pem",
			args: []string{
				"-a", "localhost:8082",
				"--ca-cert", "flag-ca.pem",
			},
			want: Config{
				Address:    "localhost:8082",
				CACertFile: "flag-ca.pem",
			},
		},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			t.Setenv("ADDRESS", tt.envAddress)
			t.Setenv("CA_CERT_FILE", tt.envCACert)

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

func TestParseReturnsFlagError(t *testing.T) {
	t.Setenv("ADDRESS", "")
	t.Setenv("CA_CERT_FILE", "")

	_, err := Parse([]string{"--unknown-flag"})
	if err == nil {
		t.Fatal("Parse() error = nil, want flag parsing error")
	}
}
