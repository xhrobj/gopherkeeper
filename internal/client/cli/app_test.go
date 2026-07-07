package cli

import (
	"bytes"
	"context"
	"io"
	"os"
	"strings"
	"testing"

	"github.com/xhrobj/gopherkeeper/internal/client/config"
)

func TestRun_RootHelpContainsBanner(t *testing.T) {
	tests := []struct {
		name string
		args []string
	}{
		{
			name: "without arguments",
			args: []string{"gopherkeeper"},
		},
		{
			name: "short help flag",
			args: []string{"gopherkeeper", "-h"},
		},
		{
			name: "long help flag",
			args: []string{"gopherkeeper", "--help"},
		},
		{
			name: "help command",
			args: []string{"gopherkeeper", "help"},
		},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			isolateClientConfig(t)

			var output bytes.Buffer

			err := runWithHealthRunner(
				t,
				tt.args,
				&output,
				io.Discard,
				unexpectedHealthRunner(t),
			)
			if err != nil {
				t.Fatalf("run() error = %v", err)
			}

			assertContainsAll(
				t,
				output.String(),
				banner,
				"COMMANDS:",
				"health",
				"register",
				"login",
				"whoami",
			)
		})
	}
}

func TestRun_HealthHelpDoesNotContainBanner(t *testing.T) {
	tests := []struct {
		name string
		args []string
	}{
		{
			name: "help flag",
			args: []string{"gopherkeeper", "health", "--help"},
		},
		{
			name: "help command",
			args: []string{"gopherkeeper", "help", "health"},
		},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			isolateClientConfig(t)

			var output bytes.Buffer

			err := runWithHealthRunner(
				t,
				tt.args,
				&output,
				io.Discard,
				unexpectedHealthRunner(t),
			)
			if err != nil {
				t.Fatalf("run() error = %v", err)
			}

			got := output.String()
			if strings.Contains(got, banner) {
				t.Errorf("health help contains banner: %q", got)
			}

			if !strings.Contains(got, "gopherkeeper health") {
				t.Errorf("help = %q, want health command name", got)
			}
		})
	}
}

func TestRun_VersionContainsBannerAndBuildInfo(t *testing.T) {
	tests := []struct {
		name string
		args []string
	}{
		{
			name: "short version flag",
			args: []string{"gopherkeeper", "-v"},
		},
		{
			name: "long version flag",
			args: []string{"gopherkeeper", "--version"},
		},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			isolateClientConfig(t)

			var output bytes.Buffer

			err := runWithHealthRunner(
				t,
				tt.args,
				&output,
				io.Discard,
				unexpectedHealthRunner(t),
			)
			if err != nil {
				t.Fatalf("run() error = %v", err)
			}

			got := output.String()
			if !strings.Contains(got, banner) {
				t.Errorf("version does not contain banner: %q", got)
			}

			assertContainsAll(
				t,
				got,
				"Build version: v0.4.2",
				"Build date: 2026-06-30",
				"Build commit: deadbeef",
			)

			if strings.Contains(got, "COMMANDS:") {
				t.Errorf("version contains help text: %q", got)
			}
		})
	}
}

func TestRun_HealthCommandOutputDoesNotContainBanner(t *testing.T) {
	isolateClientConfig(t)

	var output bytes.Buffer

	err := runWithHealthRunner(
		t,
		[]string{"gopherkeeper", "health"},
		&output,
		io.Discard,
		func(_ context.Context, _ config.Config, output io.Writer) error {
			_, err := io.WriteString(output, "Server status: ok\n")
			return err
		},
	)
	if err != nil {
		t.Fatalf("run() error = %v", err)
	}

	got := output.String()
	if got != "Server status: ok\n" {
		t.Errorf("health output = %q, want %q", got, "Server status: ok\n")
	}
}

func TestRun_HealthCommandConfiguration(t *testing.T) {
	tests := []struct {
		name           string
		envAddress     string
		envCACert      string
		envSessionFile string
		envConfig      string
		args           []string
		want           config.Config
	}{
		{
			name: "defaults",
			args: []string{"gopherkeeper", "health"},
			want: config.Config{
				Address: "localhost:8080",
			},
		},
		{
			name:      "config file",
			envConfig: writeClientConfig(t, `{"address":"localhost:8081","ca_cert_file":"file-ca.pem","session_file":"file-session.json"}`),
			args:      []string{"gopherkeeper", "health"},
			want: config.Config{
				Address:     "localhost:8081",
				CACertFile:  "file-ca.pem",
				SessionFile: "file-session.json",
			},
		},
		{
			name:           "environment",
			envAddress:     "localhost:8081",
			envCACert:      "env-ca.pem",
			envSessionFile: "env-session.json",
			args:           []string{"gopherkeeper", "health"},
			want: config.Config{
				Address:     "localhost:8081",
				CACertFile:  "env-ca.pem",
				SessionFile: "env-session.json",
			},
		},
		{
			name:           "environment > config file",
			envConfig:      writeClientConfig(t, `{"address":"localhost:8081","ca_cert_file":"file-ca.pem","session_file":"file-session.json"}`),
			envAddress:     "localhost:8082",
			envCACert:      "env-ca.pem",
			envSessionFile: "env-session.json",
			args:           []string{"gopherkeeper", "health"},
			want: config.Config{
				Address:     "localhost:8082",
				CACertFile:  "env-ca.pem",
				SessionFile: "env-session.json",
			},
		},
		{
			name:           "flags > environment",
			envAddress:     "localhost:8081",
			envCACert:      "env-ca.pem",
			envSessionFile: "env-session.json",
			args: []string{
				"gopherkeeper",
				"health",
				"-a", "localhost:8082",
				"--ca-cert", "flag-ca.pem",
				"--session-file", "flag-session.json",
			},
			want: config.Config{
				Address:     "localhost:8082",
				CACertFile:  "flag-ca.pem",
				SessionFile: "flag-session.json",
			},
		},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			isolateClientConfig(t)

			t.Setenv("ADDRESS", tt.envAddress)
			t.Setenv("CA_CERT_FILE", tt.envCACert)
			t.Setenv("SESSION_FILE", tt.envSessionFile)
			t.Setenv("CONFIG", tt.envConfig)

			var got config.Config
			err := runWithHealthRunner(
				t,
				tt.args,
				io.Discard,
				io.Discard,
				func(_ context.Context, cfg config.Config, _ io.Writer) error {
					got = cfg
					return nil
				},
			)
			if err != nil {
				t.Fatalf("run() error = %v", err)
			}

			if got != tt.want {
				t.Errorf("configuration = %+v, want %+v", got, tt.want)
			}
		})
	}
}

func runWithHealthRunner(
	t *testing.T,
	args []string,
	output io.Writer,
	errorOutput io.Writer,
	health outputRunner,
) error {
	t.Helper()

	return runTestCommand(t, args, strings.NewReader(""), output, errorOutput, commandRunners{health: health})
}

func assertContainsAll(t *testing.T, got string, wants ...string) {
	t.Helper()

	for _, want := range wants {
		if !strings.Contains(got, want) {
			t.Errorf("output = %q, want %q", got, want)
		}
	}
}

func writeClientConfig(t *testing.T, content string) string {
	t.Helper()

	path := t.TempDir() + "/client.json"
	if err := os.WriteFile(path, []byte(content), 0o600); err != nil {
		t.Fatalf("write client config: %v", err)
	}

	return path
}
