package cli

import (
	"bytes"
	"context"
	"io"
	"strings"
	"testing"

	"github.com/xhrobj/gopherkeeper/internal/buildinfo"
	"github.com/xhrobj/gopherkeeper/internal/client/config"
)

var testBuildInfo = buildinfo.Info{
	Version: "v1.2.3",
	Date:    "2026-06-30",
	Commit:  "deadbeef",
}

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
			var output bytes.Buffer

			err := run(
				context.Background(),
				tt.args,
				&output,
				io.Discard,
				testBuildInfo,
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
			var output bytes.Buffer

			err := run(
				context.Background(),
				tt.args,
				&output,
				io.Discard,
				testBuildInfo,
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
			var output bytes.Buffer

			err := run(
				context.Background(),
				tt.args,
				&output,
				io.Discard,
				testBuildInfo,
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
				"Build version: v1.2.3",
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
	var output bytes.Buffer

	err := run(
		context.Background(),
		[]string{"gopherkeeper", "health"},
		&output,
		io.Discard,
		testBuildInfo,
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
		name       string
		envAddress string
		envCACert  string
		args       []string
		want       config.Config
	}{
		{
			name: "defaults",
			args: []string{"gopherkeeper", "health"},
			want: config.Config{
				Address: "localhost:8080",
			},
		},
		{
			name:       "environment",
			envAddress: "localhost:8081",
			envCACert:  "env-ca.pem",
			args:       []string{"gopherkeeper", "health"},
			want: config.Config{
				Address:    "localhost:8081",
				CACertFile: "env-ca.pem",
			},
		},
		{
			name:       "flags over environment",
			envAddress: "localhost:8081",
			envCACert:  "env-ca.pem",
			args: []string{
				"gopherkeeper",
				"health",
				"-a", "localhost:8082",
				"--ca-cert", "flag-ca.pem",
			},
			want: config.Config{
				Address:    "localhost:8082",
				CACertFile: "flag-ca.pem",
			},
		},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			t.Setenv("ADDRESS", tt.envAddress)
			t.Setenv("CA_CERT_FILE", tt.envCACert)

			var got config.Config
			err := run(
				context.Background(),
				tt.args,
				io.Discard,
				io.Discard,
				testBuildInfo,
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

func assertContainsAll(t *testing.T, got string, wants ...string) {
	t.Helper()

	for _, want := range wants {
		if !strings.Contains(got, want) {
			t.Errorf("output = %q, want %q", got, want)
		}
	}
}

func unexpectedHealthRunner(t *testing.T) healthRunner {
	t.Helper()

	return func(context.Context, config.Config, io.Writer) error {
		t.Fatal("health command must not run")
		return nil
	}
}
