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

func TestLoginCommand_ConfigurationAndInput(t *testing.T) {
	isolateClientConfig(t)

	input := strings.NewReader(testRegistrationPassword + "\n")
	var gotConfig config.Config
	var gotLogin string
	var gotPassword string

	app := newApplicationStub(t)
	app.login = func(_ context.Context, login, password string) (model.User, error) {
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
			"login",
			"-l", "alice",
			"--password-stdin",
			"--address", "localhost:8082",
			"--ca-cert", "flag-ca.pem",
			"--session-file", "flag-session.json",
		},
		input,
		&output,
		io.Discard,
		factory,
	)
	if err != nil {
		t.Fatalf("run() error = %v", err)
	}

	wantConfig := config.Config{
		Address:     "localhost:8082",
		CACertFile:  "flag-ca.pem",
		SessionFile: "flag-session.json",
	}
	if gotConfig != wantConfig {
		t.Errorf("configuration = %+v, want %+v", gotConfig, wantConfig)
	}
	if gotLogin != "alice" {
		t.Errorf("login = %q, want alice", gotLogin)
	}
	if gotPassword != testRegistrationPassword {
		t.Errorf("password = %q, want stdin password", gotPassword)
	}
	if got := output.String(); got != "User alice logged in successfully.\n" {
		t.Errorf("output = %q, want login result", got)
	}
}

func TestLoginCommand_RequiresLogin(t *testing.T) {
	isolateClientConfig(t)

	err := runTestCommand(
		t,
		[]string{"gkeep", "login", "--password-stdin"},
		strings.NewReader(testRegistrationPassword+"\n"),
		io.Discard,
		io.Discard,
		nil,
	)
	if err == nil {
		t.Fatal("run() error = nil, want required login error")
	}
}

func TestLoginCommand_HelpDoesNotOfferPasswordFlag(t *testing.T) {
	isolateClientConfig(t)

	var output bytes.Buffer
	err := runTestCommand(
		t,
		[]string{"gkeep", "login", "--help"},
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
		t.Errorf("login help = %q, want password-stdin flag", help)
	}
	if strings.Contains(help, "--password string") {
		t.Errorf("login help exposes password flag: %q", help)
	}
}
