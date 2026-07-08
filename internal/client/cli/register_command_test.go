package cli

import (
	"bytes"
	"context"
	"io"
	"strings"
	"testing"

	"github.com/xhrobj/gopherkeeper/internal/client/config"
)

func TestRegisterCommand_ConfigurationAndInput(t *testing.T) {
	isolateClientConfig(t)

	input := strings.NewReader(testRegistrationPassword + "\n")
	var gotConfig config.Config
	var gotInput io.Reader
	var gotLogin string
	var gotPasswordStdin bool
	var output bytes.Buffer

	err := runTestCommand(
		t,
		[]string{
			"gkeep",
			"register",
			"-l", "alice",
			"--password-stdin",
			"--address", "localhost:8082",
			"--ca-cert", "flag-ca.pem",
		},
		input,
		&output,
		io.Discard,
		commandRunners{
			register: func(
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
		t.Fatalf("run() error = %v", err)
	}

	wantConfig := config.Config{
		Address:    "localhost:8082",
		CACertFile: "flag-ca.pem",
	}
	if gotConfig != wantConfig {
		t.Errorf("configuration = %+v, want %+v", gotConfig, wantConfig)
	}
	if gotInput != input {
		t.Error("register command did not receive standard input")
	}
	if gotLogin != "alice" {
		t.Errorf("login = %q, want alice", gotLogin)
	}
	if !gotPasswordStdin {
		t.Error("password-stdin = false, want true")
	}
}

func TestRegisterCommand_RequiresLogin(t *testing.T) {
	isolateClientConfig(t)

	err := runTestCommand(
		t,
		[]string{"gkeep", "register", "--password-stdin"},
		strings.NewReader(testRegistrationPassword+"\n"),
		io.Discard,
		io.Discard,
		commandRunners{
			register: func(
				context.Context,
				config.Config,
				io.Reader,
				io.Writer,
				io.Writer,
				string,
				bool,
			) error {
				t.Fatal("register runner was called without login")
				return nil
			},
		},
	)
	if err == nil {
		t.Fatal("run() error = nil, want required login error")
	}
}

func TestRegisterCommand_HelpDoesNotOfferPasswordFlag(t *testing.T) {
	isolateClientConfig(t)

	var output bytes.Buffer

	err := runTestCommand(
		t,
		[]string{"gkeep", "register", "--help"},
		strings.NewReader(""),
		&output,
		io.Discard,
		commandRunners{
			register: func(
				context.Context,
				config.Config,
				io.Reader,
				io.Writer,
				io.Writer,
				string,
				bool,
			) error {
				t.Fatal("register runner was called for help")
				return nil
			},
		},
	)
	if err != nil {
		t.Fatalf("run() error = %v", err)
	}

	help := output.String()
	if !strings.Contains(help, "--password-stdin") {
		t.Errorf("register help = %q, want password-stdin flag", help)
	}
	if strings.Contains(help, "--password string") {
		t.Errorf("register help exposes password flag: %q", help)
	}
}
