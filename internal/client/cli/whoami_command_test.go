package cli

import (
	"bytes"
	"context"
	"io"
	"strings"
	"testing"

	"github.com/xhrobj/gopherkeeper/internal/client/config"
)

func TestWhoamiCommand_Configuration(t *testing.T) {
	t.Setenv("ADDRESS", "localhost:8081")
	t.Setenv("CA_CERT_FILE", "env-ca.pem")
	t.Setenv("SESSION_FILE", "env-session.json")

	var gotConfig config.Config
	var output bytes.Buffer

	err := runWithInput(
		context.Background(),
		[]string{
			"gkeep",
			"whoami",
			"--address", "localhost:8082",
			"--ca-cert", "flag-ca.pem",
			"--session-file", "flag-session.json",
		},
		strings.NewReader(""),
		&output,
		io.Discard,
		testBuildInfo,
		commandRunners{
			health:   unexpectedHealthRunner(t),
			register: unexpectedRegisterRunner(t),
			login:    unexpectedLoginRunner(t),
			whoami: func(_ context.Context, cfg config.Config, _ io.Writer) error {
				gotConfig = cfg
				return nil
			},
		},
	)
	if err != nil {
		t.Fatalf("runWithInput() error = %v", err)
	}

	wantConfig := config.Config{
		Address:     "localhost:8082",
		CACertFile:  "flag-ca.pem",
		SessionFile: "flag-session.json",
	}
	if gotConfig != wantConfig {
		t.Errorf("configuration = %+v, want %+v", gotConfig, wantConfig)
	}
}

func TestWhoamiCommand_HelpDoesNotContainBanner(t *testing.T) {
	var output bytes.Buffer

	err := runWithInput(
		context.Background(),
		[]string{"gkeep", "whoami", "--help"},
		strings.NewReader(""),
		&output,
		io.Discard,
		testBuildInfo,
		commandRunners{
			health:   unexpectedHealthRunner(t),
			register: unexpectedRegisterRunner(t),
			login:    unexpectedLoginRunner(t),
			whoami: func(context.Context, config.Config, io.Writer) error {
				t.Fatal("whoami runner was called for help")
				return nil
			},
		},
	)
	if err != nil {
		t.Fatalf("runWithInput() error = %v", err)
	}

	help := output.String()
	if !strings.Contains(help, "gkeep whoami") {
		t.Errorf("whoami help = %q, want command name", help)
	}
	if strings.Contains(help, banner) {
		t.Errorf("whoami help contains root banner: %q", help)
	}
}
