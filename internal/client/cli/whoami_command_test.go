package cli

import (
	"bytes"
	"context"
	"io"
	"testing"

	"github.com/xhrobj/gopherkeeper/internal/client/config"
)

func TestWhoamiCommand_Configuration(t *testing.T) {
	var gotConfig config.Config
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
		commandRunners{
			whoami: func(_ context.Context, cfg config.Config, _ io.Writer) error {
				gotConfig = cfg
				return nil
			},
		},
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
}
