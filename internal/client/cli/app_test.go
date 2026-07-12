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
		{name: "without arguments", args: []string{"gopherkeeper"}},
		{name: "short help flag", args: []string{"gopherkeeper", "-h"}},
		{name: "long help flag", args: []string{"gopherkeeper", "--help"}},
		{name: "help command", args: []string{"gopherkeeper", "help"}},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			isolateClientConfig(t)

			var output bytes.Buffer
			if err := runTestCommand(t, tt.args, nil, &output, io.Discard, nil); err != nil {
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
				"logout",
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
		{name: "help flag", args: []string{"gopherkeeper", "health", "--help"}},
		{name: "help command", args: []string{"gopherkeeper", "help", "health"}},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			isolateClientConfig(t)

			var output bytes.Buffer
			if err := runTestCommand(t, tt.args, nil, &output, io.Discard, nil); err != nil {
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
		{name: "short version flag", args: []string{"gopherkeeper", "-v"}},
		{name: "long version flag", args: []string{"gopherkeeper", "--version"}},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			isolateClientConfig(t)

			var output bytes.Buffer
			if err := runTestCommand(t, tt.args, nil, &output, io.Discard, nil); err != nil {
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

	factory := newClientFactoryStub(t)
	factory.newHealthClient = func(config.Config) (healthClient, error) {
		return healthClientStub{health: func(context.Context) (string, error) {
			return "ok", nil
		}}, nil
	}

	var output bytes.Buffer
	if err := runTestCommand(
		t,
		[]string{"gopherkeeper", "health"},
		nil,
		&output,
		io.Discard,
		factory,
	); err != nil {
		t.Fatalf("run() error = %v", err)
	}

	if got := output.String(); got != "Server status: ok\n" {
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
			want: config.Config{Address: "localhost:8080"},
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
			name:      "config flag before subcommand > CONFIG environment",
			envConfig: writeClientConfig(t, `{"address":"env-file:8080"}`),
			args: []string{
				"gopherkeeper",
				"--config",
				writeClientConfig(t, `{"address":"flag-file-before:8080"}`),
				"health",
			},
			want: config.Config{Address: "flag-file-before:8080"},
		},
		{
			name:      "config flag after subcommand > CONFIG environment",
			envConfig: writeClientConfig(t, `{"address":"env-file:8080"}`),
			args: []string{
				"gopherkeeper",
				"health",
				"-c",
				writeClientConfig(t, `{"address":"flag-file-after:8080"}`),
			},
			want: config.Config{Address: "flag-file-after:8080"},
		},
		{
			name:           "flags before subcommand > environment",
			envAddress:     "localhost:8081",
			envCACert:      "env-ca.pem",
			envSessionFile: "env-session.json",
			args: []string{
				"gopherkeeper",
				"-a", "localhost:8082",
				"--ca-cert", "flag-ca.pem",
				"--session-file", "flag-session.json",
				"health",
			},
			want: config.Config{
				Address:     "localhost:8082",
				CACertFile:  "flag-ca.pem",
				SessionFile: "flag-session.json",
			},
		},
		{
			name:           "flags after subcommand > environment",
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
		{
			name: "inline flag values",
			args: []string{
				"gopherkeeper",
				"health",
				"--address=localhost:8083",
				"--ca-cert=inline-ca.pem",
				"--session-file=inline-session.json",
			},
			want: config.Config{
				Address:     "localhost:8083",
				CACertFile:  "inline-ca.pem",
				SessionFile: "inline-session.json",
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
			factory := newClientFactoryStub(t)
			factory.newHealthClient = func(cfg config.Config) (healthClient, error) {
				got = cfg
				return healthClientStub{health: func(context.Context) (string, error) {
					return "ok", nil
				}}, nil
			}

			if err := runTestCommand(t, tt.args, nil, io.Discard, io.Discard, factory); err != nil {
				t.Fatalf("run() error = %v", err)
			}
			if got != tt.want {
				t.Errorf("configuration = %+v, want %+v", got, tt.want)
			}
		})
	}
}

func TestRun_ReturnsConfigurationError(t *testing.T) {
	tests := []struct {
		name      string
		envConfig string
		args      []string
		wantError string
	}{
		{
			name:      "missing CONFIG file",
			envConfig: t.TempDir() + "/missing.json",
			args:      []string{"gopherkeeper", "health"},
			wantError: "read client config file",
		},
		{
			name:      "empty address flag",
			args:      []string{"gopherkeeper", "health", "--address="},
			wantError: "server address is required",
		},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			isolateClientConfig(t)
			t.Setenv("CONFIG", tt.envConfig)

			factory := newClientFactoryStub(t)
			factory.newHealthClient = func(config.Config) (healthClient, error) {
				t.Fatal("health client must not be created after configuration error")
				return nil, nil
			}

			err := runTestCommand(t, tt.args, nil, io.Discard, io.Discard, factory)
			if err == nil {
				t.Fatal("run() error = nil, want configuration error")
			}
			if !strings.Contains(err.Error(), tt.wantError) {
				t.Fatalf("run() error = %q, want substring %q", err, tt.wantError)
			}
		})
	}
}

func TestRun_ReturnsFlagParsingError(t *testing.T) {
	isolateClientConfig(t)

	factory := newClientFactoryStub(t)
	factory.newHealthClient = func(config.Config) (healthClient, error) {
		t.Fatal("health client must not be created after flag parsing error")
		return nil, nil
	}

	err := runTestCommand(
		t,
		[]string{"gopherkeeper", "health", "--config"},
		nil,
		io.Discard,
		io.Discard,
		factory,
	)
	if err == nil {
		t.Fatal("run() error = nil, want flag parsing error")
	}
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
