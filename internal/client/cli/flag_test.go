package cli

import (
	"testing"

	urfavecli "github.com/urfave/cli/v3"
	"github.com/xhrobj/gopherkeeper/internal/client/config"
)

func TestConfigFromCommand(t *testing.T) {
	want := config.Config{Address: "localhost:8080"}
	command := &urfavecli.Command{Metadata: map[string]any{
		clientConfigMetadataKey: want,
	}}

	got, err := configFromCommand(command)
	if err != nil {
		t.Fatalf("configFromCommand() error = %v", err)
	}
	if got != want {
		t.Errorf("configFromCommand() = %+v, want %+v", got, want)
	}
}

func TestConfigFromCommand_ReturnsMissingConfigError(t *testing.T) {
	command := &urfavecli.Command{Metadata: map[string]any{}}

	_, err := configFromCommand(command)
	if err == nil {
		t.Fatal("configFromCommand() error = nil, want client config is missing")
	}
	if got, want := err.Error(), "client config is missing"; got != want {
		t.Errorf("configFromCommand() error = %q, want %q", got, want)
	}
}

func TestConfigFromCommand_ReturnsUnexpectedTypeError(t *testing.T) {
	command := &urfavecli.Command{Metadata: map[string]any{
		clientConfigMetadataKey: "invalid",
	}}

	_, err := configFromCommand(command)
	if err == nil {
		t.Fatal("configFromCommand() error = nil, want client config has unexpected type")
	}
	if got, want := err.Error(), "client config has unexpected type"; got != want {
		t.Errorf("configFromCommand() error = %q, want %q", got, want)
	}
}
