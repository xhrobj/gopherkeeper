package cli

import (
	"bytes"
	"context"
	"io"
	"testing"

	"github.com/xhrobj/gopherkeeper/internal/client/config"
	"github.com/xhrobj/gopherkeeper/internal/model"
)

func TestWhoamiCommand_Configuration(t *testing.T) {
	isolateClientConfig(t)

	var gotConfig config.Config
	app := newApplicationStub(t)
	app.whoami = func(context.Context) (model.User, error) {
		return model.User{Login: "alice"}, nil
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
			"whoami",
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
	if got := output.String(); got != "alice\n" {
		t.Errorf("output = %q, want alice", got)
	}
}
