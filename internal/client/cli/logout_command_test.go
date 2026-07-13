package cli

import (
	"bytes"
	"context"
	"io"
	"testing"

	"github.com/xhrobj/gopherkeeper/internal/client/config"
)

func TestLogoutCommand_Configuration(t *testing.T) {
	isolateClientConfig(t)

	var gotConfig config.Config
	factory := newClientFactoryStub(t)
	factory.newLogoutApplication = func(cfg config.Config) (userLogoutter, error) {
		gotConfig = cfg
		return userLogoutterStub{logout: func(context.Context) error { return nil }}, nil
	}

	var output bytes.Buffer
	err := runTestCommand(
		t,
		[]string{
			"gkeep",
			"logout",
			"--address", "localhost:8082",
			"--ca-cert", "flag-ca.pem",
			"--session-file", "flag-session.json",
		},
		nil,
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
	if got := output.String(); got != "logged out\n" {
		t.Errorf("output = %q, want logged out", got)
	}
}
