package config

import "testing"

func TestParse(t *testing.T) {
	tests := []struct {
		name       string
		envAddress string
		args       []string
		want       string
	}{
		{
			name: "default",
			want: "localhost:8080",
		},
		{
			name:       "environment",
			envAddress: "localhost:8081",
			want:       "localhost:8081",
		},
		{
			name: "flag",
			args: []string{"-a", "localhost:8082"},
			want: "localhost:8082",
		},
		{
			name:       "flag > environment",
			envAddress: "localhost:8081",
			args:       []string{"-a", "localhost:8082"},
			want:       "localhost:8082",
		},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			t.Setenv("ADDRESS", tt.envAddress)

			cfg, err := Parse(tt.args)
			if err != nil {
				t.Fatalf("Parse() error = %v", err)
			}

			if cfg.Address != tt.want {
				t.Errorf("Parse() Address = %q, want %q", cfg.Address, tt.want)
			}
		})
	}
}

func TestParseReturnsFlagError(t *testing.T) {
	t.Setenv("ADDRESS", "")

	_, err := Parse([]string{"--unknown-flag"})
	if err == nil {
		t.Fatal("Parse() error = nil, want flag parsing error")
	}
}
