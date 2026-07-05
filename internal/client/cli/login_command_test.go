package cli

import (
	"bytes"
	"context"
	"io"
	"strings"
	"testing"

	"github.com/xhrobj/gopherkeeper/internal/client/config"
)

func TestLoginCommand_ConfigurationAndInput(t *testing.T) {
	t.Setenv("ADDRESS", "localhost:8081")
	t.Setenv("CA_CERT_FILE", "env-ca.pem")
	t.Setenv("SESSION_FILE", "env-session.json")

	input := strings.NewReader(testRegistrationPassword + "\n")
	var gotConfig config.Config
	var gotInput io.Reader
	var gotLogin string
	var gotPasswordStdin bool
	var output bytes.Buffer

	err := runWithInput(
		context.Background(),
		[]string{
			"gkeep",
			"login",
			"--login", "alice",
			"--password-stdin",
			"--address", "localhost:8082",
			"--ca-cert", "flag-ca.pem",
			"--session-file", "flag-session.json",
		},
		input,
		&output,
		io.Discard,
		testBuildInfo,
		commandRunners{
			health:   unexpectedHealthRunner(t),
			register: unexpectedRegisterRunner(t),
			login: func(
				_ context.Context,
				cfg config.Config,
				commandInput io.Reader,
				_ io.Writer,
				_ io.Writer,
				login string,
				passwordStdin bool,
			) error {
				gotConfig = cfg
				gotInput = commandInput
				gotLogin = login
				gotPasswordStdin = passwordStdin
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
	if gotInput != input {
		t.Error("login command did not receive standard input")
	}
	if gotLogin != "alice" {
		t.Errorf("login = %q, want alice", gotLogin)
	}
	if !gotPasswordStdin {
		t.Error("password-stdin = false, want true")
	}
}

func TestLoginCommand_RequiresLogin(t *testing.T) {
	err := runWithInput(
		context.Background(),
		[]string{"gkeep", "login", "--password-stdin"},
		strings.NewReader(testRegistrationPassword+"\n"),
		io.Discard,
		io.Discard,
		testBuildInfo,
		commandRunners{
			health:   unexpectedHealthRunner(t),
			register: unexpectedRegisterRunner(t),
			login: func(
				context.Context,
				config.Config,
				io.Reader,
				io.Writer,
				io.Writer,
				string,
				bool,
			) error {
				t.Fatal("login runner was called without login")
				return nil
			},
		},
	)
	if err == nil {
		t.Fatal("runWithInput() error = nil, want required login error")
	}
}

func TestLoginCommand_HelpDoesNotOfferPasswordFlag(t *testing.T) {
	var output bytes.Buffer

	err := runWithInput(
		context.Background(),
		[]string{"gkeep", "login", "--help"},
		strings.NewReader(""),
		&output,
		io.Discard,
		testBuildInfo,
		commandRunners{
			health:   unexpectedHealthRunner(t),
			register: unexpectedRegisterRunner(t),
			login: func(
				context.Context,
				config.Config,
				io.Reader,
				io.Writer,
				io.Writer,
				string,
				bool,
			) error {
				t.Fatal("login runner was called for help")
				return nil
			},
		},
	)
	if err != nil {
		t.Fatalf("runWithInput() error = %v", err)
	}

	help := output.String()
	if !strings.Contains(help, "--password-stdin") {
		t.Errorf("login help = %q, want password-stdin flag", help)
	}
	if strings.Contains(help, "--password string") {
		t.Errorf("login help exposes password flag: %q", help)
	}
	if strings.Contains(help, banner) {
		t.Errorf("login help contains root banner: %q", help)
	}
}
