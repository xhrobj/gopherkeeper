package config

import "testing"

func TestLoad(t *testing.T) {
	tests := []struct {
		name       string
		envAddress string
		envCACert  string
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
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			t.Setenv("ADDRESS", tt.envAddress)
			t.Setenv("CA_CERT_FILE", tt.envCACert)

			if got := Load(); got != tt.want {
				t.Errorf("Load() = %+v, want %+v", got, tt.want)
			}
		})
	}
}
