package cli

import (
	"bytes"
	"context"
	"io"
	"strings"
	"testing"

	"github.com/xhrobj/gopherkeeper/internal/client/config"
	"github.com/xhrobj/gopherkeeper/internal/model"
)

func TestRegisterCommand_ConfigurationAndInput(t *testing.T) {
	isolateClientConfig(t)

	input := strings.NewReader(testRegistrationPassword + "\n")
	var gotConfig config.Config
	var gotLogin string
	var gotPassword string

	app := newApplicationStub(t)
	app.register = func(_ context.Context, login, password string) (model.User, error) {
		gotLogin = login
		gotPassword = password
		return model.User{Login: login}, nil
	}

	factory := newClientFactoryStub(t)
	factory.newApplication = func(cfg config.Config) (application, error) {
		gotConfig = cfg
		return app, nil
	}

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
		factory,
	)
	if err != nil {
		t.Fatalf("run() error = %v", err)
	}

	wantConfig := config.Config{Address: "localhost:8082", CACertFile: "flag-ca.pem"}
	if gotConfig != wantConfig {
		t.Errorf("configuration = %+v, want %+v", gotConfig, wantConfig)
	}
	if gotLogin != "alice" {
		t.Errorf("login = %q, want alice", gotLogin)
	}
	if gotPassword != testRegistrationPassword {
		t.Errorf("password = %q, want stdin password", gotPassword)
	}
	if got := output.String(); got != "User alice registered successfully.\n" {
		t.Errorf("output = %q, want registration result", got)
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
		nil,
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
		nil,
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
